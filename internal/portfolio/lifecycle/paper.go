package lifecycle

import (
	"context"
	"fmt"
	"log"
	"time"

	"o/internal/platform/excell"
	"o/internal/research/grammar"
)

// paperAccounting paper 态策略吃 md_funding 新行虚拟记账（空跑=对真事实行结算，
// 不碰真单径）。funding 收入=rate×notional（short_perp 收正租；long_perp 付正租——
// 本票 funding-hedge 模板 direction=short_perp_long_spot）。T-0377 方向符号根修：
// perp 腿方向由 direction 首段判——long_perp 正 rate 付租（符号取反），不再
// 恒按 short_perp 记（xvenue long_perp_short_cfd 案实证伪账风险）。
func (r *Runner) paperAccounting(ctx context.Context) error {
	// 每策略记 last_fill_ts 于 detail 简化为：取 md_funding 行 ts > 最近 funding fill ts。
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.direction, s.notional_usd,
		COALESCE((SELECT max(ts) FROM fin_paper_fill f WHERE f.strategy_id=s.id AND f.kind='funding'), s.entered_at)
		FROM fin_strategy s WHERE s.state='paper'`)
	if err != nil {
		return err
	}
	type st struct {
		id       int64
		inst     string
		dir      string
		notional float64
		last     time.Time
	}
	var sts []st
	for rows.Next() {
		var x st
		if err := rows.Scan(&x.id, &x.inst, &x.dir, &x.notional, &x.last); err != nil {
			rows.Close()
			return err
		}
		sts = append(sts, x)
	}
	rows.Close()
	for _, x := range sts {
		sign := 1.0
		if len(x.dir) >= 9 && x.dir[:9] == "long_perp" { // long_perp 付正租——符号取反
			sign = -1.0
		}
		frows, err := r.pool.Query(ctx, `SELECT ts, funding_rate FROM md_funding
			WHERE inst_id=$1 AND ts>$2 ORDER BY ts`, x.inst, x.last)
		if err != nil {
			return err
		}
		for frows.Next() {
			var ts time.Time
			var rate float64
			if err := frows.Scan(&ts, &rate); err != nil {
				frows.Close()
				return err
			}
			amt := sign * rate * x.notional
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
				VALUES($1,'funding',$2,$3,$4,$5)`,
				x.id, x.inst, amt, []byte(`{"rate":`+ftoa(rate)+`}`), ts); err != nil {
				frows.Close()
				return err
			}
		}
		frows.Close()
	}
	return nil
}

// promote paper 满判据→probe（时长+模拟净回报+回撤上界全机读；live 闸委员会开关）。
func (r *Runner) promote(ctx context.Context, minH, minNet, maxDD float64, liveSw bool) error {
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.template_ref, s.direction,
		EXTRACT(EPOCH FROM (now()-s.entered_at))/3600.0,
		COALESCE((SELECT sum(amount_usd) FROM fin_paper_fill f WHERE f.strategy_id=s.id),0),
		COALESCE((SELECT min(cum) FROM (SELECT sum(amount_usd) OVER (ORDER BY ts) cum
			FROM fin_paper_fill f WHERE f.strategy_id=s.id) q),0),
		COALESCE((SELECT p.net_apr FROM fin_proposal p WHERE p.id=s.proposal_ref),0)
		FROM fin_strategy s WHERE s.state='paper'`)
	if err != nil {
		return err
	}
	type cand struct {
		id         int64
		inst, tmpl string
		dir        string
		hours      float64
		net        float64
		minCum     float64
		refAPR     float64
	}
	var cs []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.inst, &c.tmpl, &c.dir, &c.hours, &c.net, &c.minCum, &c.refAPR); err != nil {
			rows.Close()
			return err
		}
		cs = append(cs, c)
	}
	rows.Close()
	for _, c := range cs {
		if c.hours < minH || c.net < minNet || c.minCum < -maxDD {
			continue // 判据未满不晋升（不缺省放行）
		}
		if fam, ok := grammar.LookupFamily(c.tmpl); ok && !fam.AcceptsInst(c.inst) {
			r.rej.Inc("inst_form_mismatch")
			log.Printf("[lifecycle] #%d paper→probe 拦：inst %s 构型不配族 %s 腿构型（工单#15943——期权/远期 inst 不得进非期权/远期族）",
				c.id, c.inst, c.tmpl)
			continue
		}
		if ok, why := r.confGate(ctx, c.id); !ok { // T-0521：置信下界证伪闸
			r.rej.Inc("conf_lb_fail")
			log.Printf("[lifecycle] #%d paper→probe 拦：%s", c.id, why)
			continue
		}
		if ok, why := r.entryCondition(ctx, c.inst, c.tmpl, c.dir, c.refAPR); !ok { // T-0521：进入条件判词
			r.rej.Inc("entry_cond_defer")
			log.Printf("[lifecycle] #%d paper→probe 拦：%s", c.id, why)
			continue
		}
		if err := Transition(ctx, r.pool, c.id, "probe",
			"paper 判据满（时长/净回报/回撤+置信下界+进入条件五机读全过）→probe 观察岗", liveSw); err != nil {
			r.rej.Inc("illegal_transition")
			return err
		}
	}
	// probe→live（T-0050 首仓序④）：观察窗（同 minH 键复用——不另造判据）+
	// paper 净回报续优 + **委员会 live_switch=true**（缺/非 true fail-closed 不晋）。
	if !liveSw {
		return nil
	}
	prows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.template_ref, s.direction,
		EXTRACT(EPOCH FROM (now()-s.entered_at))/3600.0,
		COALESCE((SELECT sum(amount_usd) FROM fin_paper_fill f WHERE f.strategy_id=s.id),0),
		COALESCE((SELECT p.net_apr FROM fin_proposal p WHERE p.id=s.proposal_ref),0)
		FROM fin_strategy s WHERE s.state='probe'`)
	if err != nil {
		return err
	}
	type pc struct {
		id         int64
		inst, tmpl string
		dir        string
		hours      float64
		net        float64
		refAPR     float64
	}
	var pcs []pc
	for prows.Next() {
		var c pc
		if err := prows.Scan(&c.id, &c.inst, &c.tmpl, &c.dir, &c.hours, &c.net, &c.refAPR); err != nil {
			prows.Close()
			return err
		}
		pcs = append(pcs, c)
	}
	prows.Close()
	for _, c := range pcs {
		if c.hours < minH || c.net < minNet {
			continue
		}
		if fam, ok := grammar.LookupFamily(c.tmpl); ok && !fam.AcceptsInst(c.inst) {
			r.rej.Inc("inst_form_mismatch")
			log.Printf("[lifecycle] #%d probe→live 拦：inst %s 构型不配族 %s 腿构型（工单#15943）", c.id, c.inst, c.tmpl)
			continue
		}
		if ok, why := r.confGate(ctx, c.id); !ok { // T-0521：置信下界证伪闸（同 paper→probe 径）
			r.rej.Inc("conf_lb_fail")
			log.Printf("[lifecycle] #%d probe→live 拦：%s", c.id, why)
			continue
		}
		if ok, why := r.entryCondition(ctx, c.inst, c.tmpl, c.dir, c.refAPR); !ok { // T-0521：进入条件判词（live 真钱时位——逆 regime 不开仓）
			r.rej.Inc("entry_cond_defer")
			log.Printf("[lifecycle] #%d probe→live 拦：%s", c.id, why)
			continue
		}
		// T-0426 执行径演习闸（业主令 2026-11-07「每策略毕业前就该检查订单
		// 会不会出错」）：probe→live 判据增「执行径演习通过」——该策略型的
		// 开→平→对账全径仿真跑绿才准 live；判据件缺/演习败=fail-closed 不晋
		// （同经济判据待遇——simdrill 演习绿落凭据后自解）。
		// 消费域收窄（升件#15291——canon simdrill-gate-domain-coverage-
		// mismatch）：策略级演习无产能径的模板不进凭据闸（xvenue 腿形
		// Drill.Run 拒载——T-0145 真执行另票接通前 probe 滞留
		// exec_drill_missing 至 armed_ttl=死径占位耗 TTL）；谓词与 Run
		// 派路同出 excell.LegShapeOf 单源，消费域=产能域按构成立。
		if !excell.Drillable(c.tmpl) {
			r.rej.Inc("exec_drill_unshaped")
			log.Printf("[lifecycle] #%d probe→live 拦：演习面无策略级凭据产能（template=%s——Drill.Run 拒载族，钻径接通前毕业链止于此）", c.id, c.tmpl)
			continue
		}
		pass, drilled, err := r.execDrillPass(ctx, c.id)
		if err != nil {
			r.rej.Inc("exec_drill_err")
			return fmt.Errorf("probe→live 演习凭据读败（断库非断凭——fail-closed 上抛不误记缺件）: %w", err)
		}
		if !drilled {
			r.rej.Inc("exec_drill_missing")
			log.Printf("[lifecycle] #%d probe→live 拦：执行径演习凭据缺（fail-closed——跑演习落凭据后自解）", c.id)
			continue
		}
		if !pass {
			r.rej.Inc("exec_drill_failed")
			log.Printf("[lifecycle] #%d probe→live 拦：执行径演习败（fail-closed——修病灶重演习）", c.id)
			continue
		}
		if err := Transition(ctx, r.pool, c.id, "live",
			"probe 观察窗满+净回报续优+执行径演习绿+委员会开关 true→live 真钱开仓岗", liveSw); err != nil {
			r.rej.Inc("illegal_transition")
			return err
		}
	}
	return nil
}
