package lifecycle

import (
	"context"
	"strconv"
	"time"
)

// paperAccounting paper 态策略吃 md_funding 新行虚拟记账（空跑=对真事实行结算，
// 不碰真单径）。funding 收入=rate×notional（short_perp 收正租；long_perp 付正租——
// 本票 funding-hedge 模板 direction=short_perp_long_spot）。
func (r *Runner) paperAccounting(ctx context.Context) error {
	// 每策略记 last_fill_ts 于 detail 简化为：取 md_funding 行 ts > 最近 funding fill ts。
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.notional_usd,
		COALESCE((SELECT max(ts) FROM fin_paper_fill f WHERE f.strategy_id=s.id AND f.kind='funding'), s.entered_at)
		FROM fin_strategy s WHERE s.state='paper'`)
	if err != nil {
		return err
	}
	type st struct {
		id       int64
		inst     string
		notional float64
		last     time.Time
	}
	var sts []st
	for rows.Next() {
		var x st
		if err := rows.Scan(&x.id, &x.inst, &x.notional, &x.last); err != nil {
			rows.Close()
			return err
		}
		sts = append(sts, x)
	}
	rows.Close()
	for _, x := range sts {
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
			amt := rate * x.notional
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
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id,
		EXTRACT(EPOCH FROM (now()-s.entered_at))/3600.0,
		COALESCE((SELECT sum(amount_usd) FROM fin_paper_fill f WHERE f.strategy_id=s.id),0),
		COALESCE((SELECT min(cum) FROM (SELECT sum(amount_usd) OVER (ORDER BY ts) cum
			FROM fin_paper_fill f WHERE f.strategy_id=s.id) q),0)
		FROM fin_strategy s WHERE s.state='paper'`)
	if err != nil {
		return err
	}
	type cand struct {
		id     int64
		inst   string
		hours  float64
		net    float64
		minCum float64
	}
	var cs []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.inst, &c.hours, &c.net, &c.minCum); err != nil {
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
		if err := Transition(ctx, r.pool, c.id, "probe",
			"paper 判据满（时长/净回报/回撤三机读全过）→probe 观察岗", liveSw); err != nil {
			r.rej.Inc("illegal_transition")
			return err
		}
	}
	// probe→live（T-0050 首仓序④）：观察窗（同 minH 键复用——不另造判据）+
	// paper 净回报续优 + **委员会 live_switch=true**（缺/非 true fail-closed 不晋）。
	if !liveSw {
		return nil
	}
	prows, err := r.pool.Query(ctx, `SELECT s.id,
		EXTRACT(EPOCH FROM (now()-s.entered_at))/3600.0,
		COALESCE((SELECT sum(amount_usd) FROM fin_paper_fill f WHERE f.strategy_id=s.id),0)
		FROM fin_strategy s WHERE s.state='probe'`)
	if err != nil {
		return err
	}
	type pc struct {
		id    int64
		hours float64
		net   float64
	}
	var pcs []pc
	for prows.Next() {
		var c pc
		if err := prows.Scan(&c.id, &c.hours, &c.net); err != nil {
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
		if err := Transition(ctx, r.pool, c.id, "live",
			"probe 观察窗满+净回报续优+委员会开关 true→live 真钱开仓岗", liveSw); err != nil {
			r.rej.Inc("illegal_transition")
			return err
		}
	}
	return nil
}

func ftoa(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}
