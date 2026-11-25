// desklife.go —— 票桌生命周期探针（工单O-39400——「只进不出」票桌
// 根修细胞，hungerTick 同槽 ProbeBreedStall 后）：
//
// ①delivered-sweep 交付实据全域销：未销∧未挂起∧非裁定票存在契形回执
//
//	（完工|候审|施工收口|工单完工|回执事件族 ∨ 处置行@sha——谓词与
//	land_queue.sh CLOSESET 逐字同源）→ 落地件同形销行
//	（seat=探测席 处置「工单#<id> 销——…」→ops_act_parse act='close'→
//	desk.closed 派生真）。CLOSESET 只在列车落成拍对当批票号集生效，
//	交付实据早于机制/走旁路落成的票永挂——本件=同谓词全域连续版。
//
// ②premise-death 前提死亡：票 note 引 venue=<v> 且
//
//	fin_venue_candidate.state='rejected'（state CHECK 闭集硬终态）→
//	直销（venue 死=票前提死，无裁量空间）；note 含已退役
//	template_ref（fin_strategy.state='retired' 归一 _↔-）→
//	premise_dead 升件交裁——前提存疑只升件不直杀（spec 禁误杀）。
//
// ③decay 陈腐衰减阀：deadline+dm.hunger.desk_decay_days 零活动
//
//	（last_active<deadline——死后无动作）→ desk_stale 升件一次
//	（无窗去重=单次，不刷屏）。
//
// 禁误杀硬条款：claimed/rulable/susp_until 非空票一律不碰（在办/待裁/
// 挂起面归原径）；delivered-sweep 与 CLOSESET 同闸（不查 claimed——
// 交付实据在=工作已完，claim 语义已兑现）。fail-closed：
// dm.hunger.desk_decay_days 缺/非正=报错不探（迁移 fin_0254 落键）。
package hunger

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	deskPremiseEscClass = "premise_dead"
	deskStaleEscClass   = "desk_stale"
)

var deskTokenRe = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9_-]{4,}`)

// ProbeDeskLifecycle 票桌生命周期一拍：三面顺序扫，单面败上抛归
// hungerTick 自报位（同 ProbeBreedStall 败情隔离约定）。
func (s *Sensor) ProbeDeskLifecycle(ctx context.Context) error {
	days, err := s.cfg.GetInt("dm.hunger.desk_decay_days")
	if err != nil {
		return fmt.Errorf("票桌衰减窗键不可读: %w", err)
	}
	if days <= 0 {
		return fmt.Errorf("票桌衰减窗非正（%d——键值坏 fail-closed）", days)
	}
	if err := s.deskDeliveredSweep(ctx); err != nil {
		return err
	}
	if err := s.deskPremiseDeath(ctx); err != nil {
		return err
	}
	return s.deskDecay(ctx, days)
}

// deskDeliveredSweep ①交付实据销——谓词与 land_queue.sh CLOSESET 的
// EXISTS 支逐字同源（完工|候审|施工收口|工单完工|回执 事件族 ∨
// note 头 40 字内完工回执/完工推支候审/候审回执/施工收口 ∨
// 处置行^O-<id>…@<sha>；复审过 sha=/复审席/落地件 豁免；票号三源
// O-<id>/工单#<id>/ticket 列）。全域版=不限当批 TSET。
func (s *Sensor) deskDeliveredSweep(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT d.id FROM ops_ticket_desk d
		WHERE d.closed IS NOT TRUE AND d.susp_until IS NULL AND d.rulable IS NOT TRUE
		  AND EXISTS (SELECT 1 FROM ops_event r
		    WHERE ((r.event ~ '完工|候审|施工收口|工单完工|回执'
		      OR r.note ~ '^.{0,40}(完工回执|完工推支候审|候审回执|施工收口)'
		      OR (r.event='处置' AND r.note ~ '^O-[0-9]+[^@]*@[0-9a-f]{9,40}'))
		      AND r.note !~ '^复审过[[:space:]]*sha=' AND r.seat NOT IN ('复审席','落地件')
		      AND (r.note ~ ('O-'||d.id||'([^0-9]|$)') OR r.note ~ ('工单#'||d.id||'([^0-9]|$)')
		        OR r.ticket ~ ('^(O-|工单#)?'||d.id||'$'))))`)
	if err != nil {
		return fmt.Errorf("交付实据面读败: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("交付实据扫败: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("交付实据迭代败: %w", err)
	}
	for _, id := range ids {
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note)
			 VALUES($1,'处置','工单',$2)`,
			"落地件",
			fmt.Sprintf("工单#%d 销——交付实据销（契形回执/处置@sha 在案——desklife全域版）", id)); err != nil {
			return fmt.Errorf("交付实据销行写败 id=%d: %w", id, err)
		}
	}
	return nil
}

// deskPremiseDeath ②前提死亡——venue=rejected 硬终态直销；retired
// template_ref 命中升件交裁。claimed/rulable/susp 票不碰。
func (s *Sensor) deskPremiseDeath(ctx context.Context) error {
	// venue 臂：note venue=<v> 引用 ∧ candidate.state='rejected'。
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT d.id, v[1]
		FROM ops_ticket_desk d,
		     regexp_matches(d.note, 'venue=''?([a-zA-Z0-9_-]+)''?', 'g') AS m(v)
		JOIN fin_venue_candidate c ON c.venue = v[1]
		WHERE d.closed IS NOT TRUE AND d.claimed IS NOT TRUE
		  AND d.susp_until IS NULL AND d.rulable IS NOT TRUE
		  AND c.state='rejected'`)
	if err != nil {
		return fmt.Errorf("前提死亡 venue 面读败: %w", err)
	}
	type vp struct {
		id    int64
		venue string
	}
	var hits []vp
	for rows.Next() {
		var h vp
		if err := rows.Scan(&h.id, &h.venue); err != nil {
			rows.Close()
			return fmt.Errorf("前提死亡 venue 扫败: %w", err)
		}
		hits = append(hits, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("前提死亡 venue 迭代败: %w", err)
	}
	for _, h := range hits {
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note)
			 VALUES($1,'处置','工单',$2)`,
			"落地件",
			fmt.Sprintf("工单#%d 销——premise-dead:venue=%s state=rejected（封轴/判死终态——前提死票随销）", h.id, h.venue)); err != nil {
			return fmt.Errorf("前提死亡销行写败 id=%d: %w", h.id, err)
		}
	}
	// 策略臂：note 含 retired template_ref（_↔- 归一——eval/strategy
	// 方言实证同 breedstall）→ premise_dead 升件交裁不直杀。
	retired, err := s.deskRetiredRefs(ctx)
	if err != nil {
		return err
	}
	if len(retired) == 0 {
		return nil
	}
	trows, err := s.pool.Query(ctx, `SELECT id, note FROM ops_ticket_desk
		WHERE closed IS NOT TRUE AND claimed IS NOT TRUE
		  AND susp_until IS NULL AND rulable IS NOT TRUE`)
	if err != nil {
		return fmt.Errorf("前提死亡策略面读败: %w", err)
	}
	defer trows.Close()
	for trows.Next() {
		var id int64
		var note string
		if err := trows.Scan(&id, &note); err != nil {
			return fmt.Errorf("前提死亡策略扫败: %w", err)
		}
		for _, tok := range deskTokenRe.FindAllString(note, -1) {
			if _, dead := retired[strings.ReplaceAll(tok, "-", "_")]; dead {
				if err := s.deskEscOnce(ctx, deskPremiseEscClass,
					strconv.FormatInt(id, 10),
					fmt.Sprintf("票桌前提疑似死亡：工单#%d note 命中已退役 template_ref（%s——退役≠死刑裁定存疑——交裁不直杀）",
						id, tok), droughtDedupHours); err != nil {
					return err
				}
				break
			}
		}
	}
	return trows.Err()
}

// deskRetiredRefs 退役 template_ref 集（_ 归一键——note 词元 _↔-
// 双向归一同 breedstall 硬条款）。
func (s *Sensor) deskRetiredRefs(ctx context.Context) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT replace(template_ref,'-','_')
		FROM fin_strategy WHERE state='retired' AND template_ref IS NOT NULL
		  AND template_ref<>''`)
	if err != nil {
		return nil, fmt.Errorf("退役面读败: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, fmt.Errorf("退役面扫败: %w", err)
		}
		out[r] = true
	}
	return out, rows.Err()
}

// deskDecay ③陈腐衰减阀——deadline+days 零活动（last_active<deadline）
// → desk_stale 升件一次（无窗去重=单次）。claimed/rulable/susp 不碰。
func (s *Sensor) deskDecay(ctx context.Context, days int64) error {
	rows, err := s.pool.Query(ctx, `SELECT id, tkind, left(note,80) FROM ops_ticket_desk
		WHERE closed IS NOT TRUE AND claimed IS NOT TRUE
		  AND susp_until IS NULL AND rulable IS NOT TRUE
		  AND deadline IS NOT NULL
		  AND deadline + make_interval(days=>$1) < now()
		  AND (last_active IS NULL OR last_active < deadline)`, days)
	if err != nil {
		return fmt.Errorf("票桌陈腐面读败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var tkind, note *string
		if err := rows.Scan(&id, &tkind, &note); err != nil {
			return fmt.Errorf("票桌陈腐扫败: %w", err)
		}
		tk := ""
		if tkind != nil {
			tk = *tkind
		}
		scope := strconv.FormatInt(id, 10)
		var dup int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE event='升件' AND esc_class=$1 AND esc_scope=$2`,
			deskStaleEscClass, scope).Scan(&dup); err != nil {
			return fmt.Errorf("票桌陈腐去重读败 id=%d: %w", id, err)
		}
		if dup > 0 {
			continue // 单次升件——不刷屏
		}
		n := ""
		if note != nil {
			n = *note
		}
		if len(n) > 60 {
			n = n[:60]
		}
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
			 VALUES($1,'升件',$2,$3,$4,$5)`,
			hungerSeat, hungerTicket,
			fmt.Sprintf("票桌陈腐：工单#%d（%s）deadline+%dd 零活动——%s——处置/续期/销案交裁（desklife 衰减阀单次升件）",
				id, tk, days, n),
			deskStaleEscClass, scope); err != nil {
			return fmt.Errorf("票桌陈腐升件写败 id=%d: %w", id, err)
		}
	}
	return rows.Err()
}

// deskEscOnce 升件+同窗去重（droughtDedupHours 同例——hours>0 时窗
// 去重；hours<=0=无窗永不去重仅由调用面控）。
func (s *Sensor) deskEscOnce(ctx context.Context, class, scope, note string, hours int) error {
	var dup int
	q := `SELECT count(*) FROM ops_event WHERE event='升件' AND esc_class=$1 AND esc_scope=$2`
	args := []any{class, scope}
	if hours > 0 {
		q += ` AND ts>now()-make_interval(hours=>$3)`
		args = append(args, hours)
	}
	if err := s.pool.QueryRow(ctx, q, args...).Scan(&dup); err != nil {
		return fmt.Errorf("升件去重读败 scope=%s: %w", scope, err)
	}
	if dup > 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		 VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket, note, class, scope); err != nil {
		return fmt.Errorf("升件写败 scope=%s: %w", scope, err)
	}
	return nil
}
