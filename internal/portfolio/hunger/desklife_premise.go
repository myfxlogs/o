package hunger

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var deskVenueRe = regexp.MustCompile(`venue='?([a-zA-Z0-9_-]+)'?`)
var deskTokenRe = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9_-]{4,}`)

// deskTicketRow 前提死亡票面行（O-39410 锚投影列——sem 锚键经
// desk 视图投影；manual/legacy=无锚票走 note 兜底只升件）。
type deskTicketRow struct {
	id         int64
	anchorType string
	anchorRef  string
	note       string
}

// deskVenueStates venue 名→state 批量读面（锚径+兜底径共用——缺席
// =对象不可判，按 spec 禁线升件不直杀）。
func (s *Sensor) deskVenueStates(ctx context.Context, venues []string) (map[string]string, error) {
	out := map[string]string{}
	if len(venues) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT venue,state FROM fin_venue_candidate WHERE venue=ANY($1)`, venues)
	if err != nil {
		return nil, fmt.Errorf("venue 态面读败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v, st string
		if err := rows.Scan(&v, &st); err != nil {
			return nil, fmt.Errorf("venue 态扫败: %w", err)
		}
		out[v] = st
	}
	return out, rows.Err()
}

// deskPremiseDeath ②前提死亡（O-39410 锚优先径）：
// 锚径——anchor_type='premise'∧anchor_ref='venue:<v>' 结构化硬引用：
//
//	candidate.state='rejected'=硬终态→直销；venue 行缺席=锚指对象
//	读败→升件（spec 禁线）；余态=前提活不动作。
//
// 兜底径——无锚票（manual/legacy/非 venue 锚）note venue= 解析命中
//
//	rejected→premise_dead 升件交裁（只升件不直销——判归席）；
//	retired template_ref 命中→premise_dead 升件（退役≠死刑存疑）。
//
// claimed/rulable/susp 票不碰。
func (s *Sensor) deskPremiseDeath(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT id, anchor_type, anchor_ref, note
		FROM ops_ticket_desk
		WHERE closed IS NOT TRUE AND claimed IS NOT TRUE
		  AND susp_until IS NULL AND rulable IS NOT TRUE`)
	if err != nil {
		return fmt.Errorf("前提死亡票面读败: %w", err)
	}
	var tickets []deskTicketRow
	for rows.Next() {
		var t deskTicketRow
		if err := rows.Scan(&t.id, &t.anchorType, &t.anchorRef, &t.note); err != nil {
			rows.Close()
			return fmt.Errorf("前提死亡票面扫败: %w", err)
		}
		tickets = append(tickets, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("前提死亡票面迭代败: %w", err)
	}
	// venue 名集=锚径 venue: 引用 ∪ 兜底径 note 解析引用（去重批读）
	var venues []string
	seen := map[string]bool{}
	addV := func(v string) {
		if v != "" && !seen[v] {
			seen[v] = true
			venues = append(venues, v)
		}
	}
	for _, t := range tickets {
		if t.anchorType == "premise" && strings.HasPrefix(t.anchorRef, "venue:") {
			addV(strings.TrimPrefix(t.anchorRef, "venue:"))
			continue // 锚票不走 note 兜底——锚=唯一判据
		}
		for _, m := range deskVenueRe.FindAllStringSubmatch(t.note, -1) {
			addV(m[1])
		}
	}
	states, err := s.deskVenueStates(ctx, venues)
	if err != nil {
		return err
	}
	// 锚径：premise/venue: 硬引用逐票判
	for _, t := range tickets {
		if t.anchorType != "premise" || !strings.HasPrefix(t.anchorRef, "venue:") {
			continue
		}
		v := strings.TrimPrefix(t.anchorRef, "venue:")
		st, ok := states[v]
		if !ok {
			if err := s.deskEscOnce(ctx, deskPremiseEscClass,
				strconv.FormatInt(t.id, 10),
				fmt.Sprintf("票桌前提锚指对象缺席：工单#%d anchor_ref=venue:%s——fin_venue_candidate 无此行=锚不可判——交裁不直杀",
					t.id, v), droughtDedupHours); err != nil {
				return err
			}
			continue
		}
		if st != "rejected" {
			continue // 前提活——venue 未判死
		}
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note)
			 VALUES($1,'处置','工单',$2)`,
			"落地件",
			fmt.Sprintf("工单#%d 销——premise-dead:anchor venue=%s state=rejected（锚径硬终态直销——O-39410）", t.id, v)); err != nil {
			return fmt.Errorf("前提死亡销行写败 id=%d: %w", t.id, err)
		}
	}
	// 兜底径+策略臂：无锚票 note venue= 命中 rejected→只升件；
	// retired template_ref 命中→升件（全票面——升件不直杀天然安全）。
	retired, err := s.deskRetiredRefs(ctx)
	if err != nil {
		return err
	}
	for _, t := range tickets {
		if !(t.anchorType == "premise" && strings.HasPrefix(t.anchorRef, "venue:")) {
			for _, m := range deskVenueRe.FindAllStringSubmatch(t.note, -1) {
				if states[m[1]] == "rejected" {
					if err := s.deskEscOnce(ctx, deskPremiseEscClass,
						strconv.FormatInt(t.id, 10),
						fmt.Sprintf("票桌前提疑似死亡（无锚兜底只升件）：工单#%d note 引 venue=%s state=rejected——判归席裁销",
							t.id, m[1]), droughtDedupHours); err != nil {
						return err
					}
					break
				}
			}
		}
		for _, tok := range deskTokenRe.FindAllString(t.note, -1) {
			if _, dead := retired[strings.ReplaceAll(tok, "-", "_")]; dead {
				if err := s.deskEscOnce(ctx, deskPremiseEscClass,
					strconv.FormatInt(t.id, 10),
					fmt.Sprintf("票桌前提疑似死亡：工单#%d note 命中已退役 template_ref（%s——退役≠死刑裁定存疑——交裁不直杀）",
						t.id, tok), droughtDedupHours); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
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
