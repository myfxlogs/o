// dcd_paper.go —— dcd 族 paper 记账（自 paper.go 拆出——棘轮 300 行
// 顶线；语义零改纯位移）。
package lifecycle

import (
	"context"
	"strings"
	"time"
)

// dcdPaperAccounting dcd_call paper 记账（T-0160 ①paper 径实装：md_dcd_product
// 最新快照 annualized_yield 按日摊→fin_paper_fill kind='funding'——
// amount=notional*apr*days/365；quote 缺/过期=fail-closed 跳过不造零）。
// T-0171 价格源律：现货腿价改读 md_ticks WS 行——目录 spot 字段非价格源
// （COALESCE(spot,0) 消费撤）；鲜度闸同 T-0128 语义（exec.tick_max_age_s
// 缺省 120s，陈=fill 让位 fail-closed 不记账）。活跃集订阅面含 dcd 现货腿
// （subscribe.go activeSet——SG 本地 tickers 供价，T-0171 裁本地订阅径）。
// T-0353 件②：dcd_put 复役入册——权利金摊提公式方向无关（declared APR
// 按日摊，C/P 同式），模板列表补词即通。
func (r *Runner) dcdPaperAccounting(ctx context.Context) error {
	// T-0215：vol-sell 两族同径记账——single_call 单腿键；strangle 复合键
	// 「Cprod+Pprod」拆双腿各摊半名义（构型=双边权利金等权名义合并收割）。
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.notional_usd,
		COALESCE((SELECT max(ts) FROM fin_paper_fill f WHERE f.strategy_id=s.id AND f.kind='funding'), s.entered_at)
		FROM fin_strategy s WHERE s.state='paper' AND s.template_ref IN
		('dcd_call','dcd_put','vol_sell_single_call','vol_sell_strangle')`)
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
			WHERE inst_id=regexp_replace($1,'-USD(G)?$','-USDT') AND venue<>'sim'`, ulys[0].uly).Scan(&spot, &tickTs)
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
