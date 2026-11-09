// Package hunger —— 盈亏自治执行环·饥饿响应（T-0376 业主令：「如何去赚」
// 全参数域归机器自主，业主面=注册/凭证/KYC/加本金）。饥饿态机读定义：
// 连续 N 天零提案（dm.hunger.zero_proposal_days，票面种 3）或 paper 池空
// →自动复审序列四步：放行线检校→休眠模板族再评估→低产 live 重组候选→
// venue 候选推进（到凭据边界出条件请求）。每步 ops_event 行落账；参数
// 自调走 config.DMSet 域闸/域内 UPDATE+参数调整行——自主≠无账。
// venue 推进不重复 scout 证据机制（metrics.go maybeRecommend 达阈翻牌
// 在案）——本环只做「recommended 候选的条件请求盒补位」（凭据边界=
// 业主面，fin_condition_gap 请求盒 D-0014）。
package hunger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/condreq"
	"o/internal/platform/config"
)

// hungerSeat 事件行席别（纪元3=探测席——探针族事件行写面判据口径）。
const hungerSeat = "探测席"

// hungerTicket 常设环归属票号（评审拍 T-0360 同例——例行行携环票号）。
const hungerTicket = ""

// Sensor 饥饿感知器（fin 池=SG 钱径面；cfg=app_config 热读）。
type Sensor struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	now  func() time.Time // 测试注桩
	// expandUniverse 宇宙扩件供给件（T-0416——okx.symbols 含 "*" 时订阅
	// 宇宙实读径；装配层挂 okxws.UniverseCatalog.Symbols——探针与
	// collector/粗扫臂同一供给件。nil=扩件未挂：* 形 fail-closed 让位
	// 不虚构标的集）。
	expandUniverse func(ctx context.Context) ([]string, error)
	// coverSeen 腿键首见戳册（T-0451 新挂腿宽限——键入所需集时刻探针
	// 不判盲；hungerTick 单协程串行访问免锁，重启清零=全键重计宽限）。
	coverSeen map[string]time.Time
}

// New 装配（now 缺省=time.Now）。
func New(pool *pgxpool.Pool, cfg *config.Client) *Sensor {
	return &Sensor{pool: pool, cfg: cfg, now: time.Now}
}

// Report 饥饿判定结果（机读定义两触发形——证据随行）。
type Report struct {
	Triggered  bool
	Reasons    []string
	WindowDays int64
}

// Detect 饥饿态机读判定：①连续 N 天零 fin_proposal 新行；②paper 池空
// （fin_strategy state='paper' 计数=0）。窗口键缺/坏=fail-closed 报错
// （检测器自身不可带默认阈滑行——dm.hunger.* 种子迁移在案）。
func (s *Sensor) Detect(ctx context.Context) (*Report, error) {
	days, err := s.cfg.GetInt("dm.hunger.zero_proposal_days")
	if err != nil {
		return nil, fmt.Errorf("饥饿窗键不可读: %w", err)
	}
	if days <= 0 {
		return nil, fmt.Errorf("饥饿窗非正（%d——键值坏 fail-closed）", days)
	}
	rep := &Report{WindowDays: days}
	var last *time.Time
	if err := s.pool.QueryRow(ctx,
		`SELECT max(created_at) FROM fin_proposal`).Scan(&last); err != nil {
		return nil, fmt.Errorf("零提案面读败: %w", err)
	}
	if last == nil || s.now().Sub(*last) > time.Duration(days)*24*time.Hour {
		d := "无提案史"
		if last != nil {
			d = fmt.Sprintf("%.1f天无新提案", s.now().Sub(*last).Hours()/24)
		}
		rep.Triggered = true
		rep.Reasons = append(rep.Reasons, fmt.Sprintf("zero_proposal:%s(窗=%d天)", d, days))
	}
	var paper int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_strategy WHERE state='paper'`).Scan(&paper); err != nil {
		return nil, fmt.Errorf("paper 池读败: %w", err)
	}
	if paper == 0 {
		rep.Triggered = true
		rep.Reasons = append(rep.Reasons, "paper_empty:paper池空(0策略)")
	}
	return rep, nil
}

// StepResult 单步结果（Acted=本步有参数/请求落位；Note=证据与旧新值）。
type StepResult struct {
	Step  string
	Acted bool
	Note  string
}

// Respond 自动复审序列四步（票面钦定序不可换）。步间独立——单步败不
// 中断序列（败情落本步事件行+并入返回错），饥饿响应不因一步滑停摆。
func (s *Sensor) Respond(ctx context.Context, rep *Report) ([]StepResult, error) {
	steps := []struct {
		name string
		run  func(context.Context, *Report) (StepResult, error)
	}{
		{"放行线检校", s.stepGateCheck},
		{"休眠模板族再评估", s.stepTemplateRevival},
		{"低产live重组候选", s.stepLiveCandidates},
		{"venue候选推进", s.stepVenueAdvance},
	}
	var out []StepResult
	var errs []error
	for _, st := range steps {
		res, err := st.run(ctx, rep)
		if err != nil {
			res = StepResult{Step: st.name, Note: "败情:" + err.Error()}
			errs = append(errs, fmt.Errorf("%s: %w", st.name, err))
		}
		if err := s.logStep(ctx, res); err != nil {
			errs = append(errs, fmt.Errorf("%s 事件行: %w", st.name, err))
		}
		out = append(out, res)
	}
	return out, errors.Join(errs...)
}

// logStep 步行落账（event=饥饿响应——note 携步名/Acted/证据）。
func (s *Sensor) logStep(ctx context.Context, res StepResult) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,$2,$3,$4)`,
		hungerSeat, "饥饿响应", hungerTicket,
		fmt.Sprintf("step=%s acted=%v %s", res.Step, res.Acted, res.Note))
	return err
}

// stepGateCheck ①放行线检校（零写——判据键在册可读+取值如实落账；
// 不可读键=检校异常显形，fail-closed 非假绿）。
func (s *Sensor) stepGateCheck(ctx context.Context, rep *Report) (StepResult, error) {
	keys := []string{
		"invest.lifecycle.live_switch",
		"invest.lifecycle.paper_min_hours",
		"invest.lifecycle.paper_min_net_usd",
		"invest.lifecycle.paper_max_dd_usd",
	}
	var parts []string
	for _, k := range keys {
		raw, err := s.cfg.Get(k)
		if err != nil {
			return StepResult{Step: "放行线检校"},
				fmt.Errorf("判据键 %s 不可读: %w", k, err)
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, string(raw)))
	}
	note := "判据值在册可读（检校零写）——" + strings.Join(parts, " ")
	return StepResult{Step: "放行线检校", Note: note}, nil
}

// stepTemplateRevival ②休眠模板族再评估：休眠族窗口内有 verdict='proposed'
// 评估=机会活着族睡着——带证据直启（DM 全权：模板族启停）；零证据=如实
// 记不启。每次启落 ops_event 参数调整行（旧值/新值/证据锚）。
func (s *Sensor) stepTemplateRevival(ctx context.Context, rep *Report) (StepResult, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,template,venue,enabled,state
		FROM fin_opp_template WHERE enabled=false OR state<>'enabled'`)
	if err != nil {
		return StepResult{Step: "休眠模板族再评估"}, err
	}
	type dormant struct {
		id              int64
		template, venue string
		enabled         bool
		state           string
	}
	var all []dormant
	for rows.Next() {
		var d dormant
		if err := rows.Scan(&d.id, &d.template, &d.venue, &d.enabled, &d.state); err != nil {
			rows.Close()
			return StepResult{Step: "休眠模板族再评估"}, err
		}
		all = append(all, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return StepResult{Step: "休眠模板族再评估"}, err
	}
	res := StepResult{Step: "休眠模板族再评估", Note: fmt.Sprintf("休眠族%d个", len(all))}
	for _, d := range all {
		var n int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
			WHERE template=$1 AND verdict='proposed' AND ts>=now()-make_interval(days=>$2)`,
			d.template, rep.WindowDays).Scan(&n); err != nil {
			return res, fmt.Errorf("复活证据读败 template=%s: %w", d.template, err)
		}
		if n == 0 {
			continue // 零证据不启——机会没醒族继续睡（证据行如实记）
		}
		old := fmt.Sprintf("enabled=%v/state=%s", d.enabled, d.state)
		if _, err := s.pool.Exec(ctx,
			`UPDATE fin_opp_template SET enabled=true,state='enabled' WHERE id=$1`, d.id); err != nil {
			return res, fmt.Errorf("族启败 template=%s: %w", d.template, err)
		}
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,$2,$3,$4)`,
			hungerSeat, "参数调整", hungerTicket,
			fmt.Sprintf("target=fin_opp_template key=%s@%s 旧=%s 新=enabled=true/state=enabled 证据=proposed_eval×%d@窗%d天",
				d.template, d.venue, old, n, rep.WindowDays)); err != nil {
			return res, err
		}
		res.Acted = true
		res.Note += fmt.Sprintf(" 启:%s@%s(证据×%d)", d.template, d.venue, n)
	}
	if !res.Acted {
		res.Note += " 零复活证据不启"
	}
	return res, nil
}

// stepLiveCandidates ③低产 live 重组候选（advisory——候选列如实录行，
// 重组动作经 lifecycle 退役闸判据径非本环直改）。
func (s *Sensor) stepLiveCandidates(ctx context.Context, rep *Report) (StepResult, error) {
	rows, err := s.pool.Query(ctx, `SELECT s.id,s.inst_id,p.settled_amount
		FROM fin_strategy s JOIN fin_position p ON p.strategy_ref=s.id
		WHERE s.state='live' AND p.settled_amount IS NOT NULL AND p.settled_amount<=0
		  AND s.entered_at<now()-make_interval(days=>$1) LIMIT 20`, rep.WindowDays)
	if err != nil {
		return StepResult{Step: "低产live重组候选"}, err
	}
	defer rows.Close()
	var cands []string
	for rows.Next() {
		var id int64
		var inst string
		var amt float64
		if err := rows.Scan(&id, &inst, &amt); err != nil {
			return StepResult{Step: "低产live重组候选"}, err
		}
		cands = append(cands, fmt.Sprintf("#%d:%s(净%.2f)", id, inst, amt))
	}
	if err := rows.Err(); err != nil {
		return StepResult{Step: "低产live重组候选"}, err
	}
	if len(cands) == 0 {
		return StepResult{Step: "低产live重组候选", Note: "零候选（live 窗内无低产仓）"}, nil
	}
	return StepResult{Step: "低产live重组候选", Acted: true,
		Note: fmt.Sprintf("候选%d个（advisory——重组经退役闸判据径）:%s",
			len(cands), strings.Join(cands, ","))}, nil
}

// stepVenueAdvance ④venue 候选推进到凭据边界：recommended 候选的条件
// 请求盒补位（kind=credential——注册/凭证=业主面边界；pending 同键幂等
// 去重 fin_0072 闸）。翻牌机制=scout maybeRecommend 证据判据在案不重复。
func (s *Sensor) stepVenueAdvance(ctx context.Context, rep *Report) (StepResult, error) {
	rows, err := s.pool.Query(ctx, `SELECT venue FROM fin_venue_candidate
		WHERE state='recommended' AND kyc_ok=true`)
	if err != nil {
		return StepResult{Step: "venue候选推进"}, err
	}
	defer rows.Close()
	var venues []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return StepResult{Step: "venue候选推进"}, err
		}
		venues = append(venues, v)
	}
	if err := rows.Err(); err != nil {
		return StepResult{Step: "venue候选推进"}, err
	}
	res := StepResult{Step: "venue候选推进",
		Note: fmt.Sprintf("recommended%d个", len(venues))}
	for _, v := range venues {
		payload := fmt.Sprintf(`{"source":"hunger","ticket":"%s","at":"%s"}`,
			hungerTicket, s.now().Format(time.RFC3339))
		if err := condreq.Request(ctx, s.pool, "credential", v, "", payload); err != nil {
			return res, fmt.Errorf("条件请求败 venue=%s: %w", v, err)
		}
		res.Acted = true
		res.Note += fmt.Sprintf(" 请求:credential@%s", v)
	}
	if !res.Acted {
		res.Note += " 零 recommended=零推进"
	}
	return res, nil
}
