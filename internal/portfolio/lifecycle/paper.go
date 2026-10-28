package lifecycle

import (
	"context"
	"strconv"
	"strings"
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

// dcdPaperAccounting dcd_call paper 记账（T-0160 ①paper 径实装：md_dcd_product
// 最新快照 annualized_yield 按日摊→fin_paper_fill kind='funding'——
// amount=notional*apr*days/365；quote 缺/过期=fail-closed 跳过不造零）。
// T-0171 价格源律：现货腿价改读 md_ticks WS 行——目录 spot 字段非价格源
// （COALESCE(spot,0) 消费撤）；鲜度闸同 T-0128 语义（exec.tick_max_age_s
// 缺省 120s，陈=fill 让位 fail-closed 不记账）。活跃集订阅面含 dcd 现货腿
// （subscribe.go activeSet——SG 本地 tickers 供价，T-0171 裁本地订阅径）。
func (r *Runner) dcdPaperAccounting(ctx context.Context) error {
	// T-0215：vol-sell 两族同径记账——single_call 单腿键；strangle 复合键
	// 「Cprod+Pprod」拆双腿各摊半名义（构型=双边权利金等权名义合并收割）。
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.notional_usd,
		COALESCE((SELECT max(ts) FROM fin_paper_fill f WHERE f.strategy_id=s.id), s.entered_at)
		FROM fin_strategy s WHERE s.state='paper' AND s.template_ref IN
		('dcd_call','vol_sell_single_call','vol_sell_strangle')`)
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
	maxAge := 120.0
	if v, err := r.cfg.GetFloat("exec.tick_max_age_s"); err == nil && v > 0 {
		maxAge = v
	}
	for _, x := range sts {
		// T-0215：strangle 复合键拆双腿——各腿 APR×半名义各自摊，任一腿
		// quote 缺/到期=该腿跳记不虚构（fail-closed 同单腿径）。
		var aprs, ulys []struct {
			apr float64
			exp time.Time
			uly string
		}
		legOK := true
		for _, pid := range strings.Split(x.inst, "+") {
			var a struct {
				apr float64
				exp time.Time
				uly string
			}
			err := r.pool.QueryRow(ctx, `SELECT COALESCE(annualized_yield,0),expiry,uly FROM md_dcd_product
				WHERE product_id=$1 ORDER BY ts DESC LIMIT 1`, pid).Scan(&a.apr, &a.exp, &a.uly)
			if err != nil || a.apr <= 0 || time.Now().After(a.exp) {
				r.rej.Inc("dcd_quote_missing")
				legOK = false // quote 面缺/到期=fail-closed 不造零 fill
				break
			}
			aprs = append(aprs, a)
			ulys = append(ulys, a)
		}
		if !legOK {
			continue
		}
		// 现货腿现价=md_ticks WS 行；uly 方言归一（BTC-USD/USDG→BTC-USDT，
		// 表达式与 subscribe.go activeSet 同源）。
		var spot float64
		var tickTs time.Time
		err = r.pool.QueryRow(ctx, `SELECT last,ts FROM md_ticks
			WHERE inst_id=regexp_replace($1,'-USD(G)?$','-USDT')`, ulys[0].uly).Scan(&spot, &tickTs)
		if err != nil || spot <= 0 || time.Since(tickTs) > time.Duration(maxAge*float64(time.Second)) {
			r.rej.Inc("dcd_spot_stale")
			continue // 现货腿价缺/陈=事实缺 fail-closed 让位不造 fill（T-0171）
		}
		days := time.Since(x.last).Hours() / 24
		if days < 1.0 {
			continue
		}
		var amt float64
		for _, a := range aprs {
			amt += a.apr * days / 365 * x.notional / float64(len(aprs))
		}
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
			VALUES($1,'funding',$2,$3,$4,now())`,
			x.id, x.inst, amt,
			[]byte(`{"apr":`+ftoa(aprs[0].apr)+`,"spot":`+ftoa(spot)+`}`)); err != nil {
			return err
		}
	}
	return nil
}
