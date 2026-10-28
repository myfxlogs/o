// volsell.go —— vol-sell 族实评（T-0215 executor 落位——executor_missing 钉
// 撤换为真判词）。自 run.go 拆出（check-lines 棘轮——行数闸硬线 450/棘轮 300）。
package grammar

import (
	"context"
	"fmt"
	"log"
	"strings"

	"o/internal/research/dcd"
)

// evalVolSell vol-sell 族实评（T-0215 executor 落位——executor_missing 钉
// 撤换为真判词）：
//   - vol_sell_single_call=call 单腿（dcd_call 同判据域——dist/term/avail/apr
//     顺闸，拒因首触）；
//   - vol_sell_strangle=同 uly+同 expiry 的 C+P 双腿对（inst_id=「Cprod+Pprod」
//     复合键——executor 依键拆腿各走 quote→trade→status）。
//
// 判据键复用 research.dcd.*（同卖权 carry 族阈域）；spot/avail 同 dcd 径——
// 判据/读件缺=unevaluable fail-closed 不虚构。
func (r *Runner) evalVolSell(ctx context.Context, f Family) error {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_dcd_product
		WHERE ts>now()-interval '2 days' AND expiry>now()`).Scan(&n); err != nil || n == 0 {
		return r.writeEval(ctx, "-", f.Template, f.Direction, "unevaluable", "product_rows_missing",
			map[string]any{"kind": f.Kind})
	}
	p, err := r.volParams(ctx)
	if err != nil {
		r.rej.Inc("missing_param")
		return r.writeEval(ctx, "-", f.Template, f.Direction, "unevaluable", "missing_param",
			map[string]any{"kind": f.Kind, "err": err.Error()})
	}
	rows, err := r.pool.Query(ctx, `SELECT product_id,uly,direction,COALESCE(strike,0),
		expiry,COALESCE(annualized_yield,0),COALESCE(min_size,0),COALESCE(max_size,0),raw->>'quoteCcy'
		FROM md_dcd_product WHERE ts>now()-interval '2 days' AND expiry>now()`)
	if err != nil {
		return err
	}
	var prods []dcd.Product
	for rows.Next() {
		var pr dcd.Product
		var qccy *string
		if err := rows.Scan(&pr.ProductID, &pr.Uly, &pr.Direction, &pr.Strike,
			&pr.Expiry, &pr.AnnualizedYield, &pr.MinSize, &pr.MaxSize, &qccy); err != nil {
			rows.Close()
			return err
		}
		if qccy != nil {
			pr.QuoteCcy = *qccy
		}
		prods = append(prods, pr)
	}
	rows.Close()
	var coins map[string]float64
	if r.coins != nil {
		if coins, err = r.coins(ctx); err != nil {
			r.rej.Inc("coins_absent")
			coins = nil // 读件败=闸缺位——Evaluate avail<0 判 unevaluable
		}
	}
	if f.Template == "vol_sell_single_call" {
		for _, pr := range prods {
			if pr.Direction != "C" {
				continue // single_call 单腿=C（LockedCcyCall）
			}
			e := r.evalLeg(ctx, pr, coins, p)
			if err := r.persistLegEval(ctx, e, f, pr.ProductID); err != nil {
				return err
			}
		}
		return nil
	}
	// strangle：同 uly+同 expiry C×P 配对（每 uly/expiry 各取双侧最优 APR 腿——
	// 组合 APR=双腿等权名义均值，构型语义=双边权利金合并收割）。
	byKey := map[string][]dcd.Product{}
	for _, pr := range prods {
		byKey[pr.Uly+"|"+pr.Expiry.Format("2006-01-02")] = append(byKey[pr.Uly+"|"+pr.Expiry.Format("2006-01-02")], pr)
	}
	for _, grp := range byKey {
		var call, put *dcd.Product
		for i := range grp {
			if grp[i].Direction == "C" && (call == nil || grp[i].AnnualizedYield > call.AnnualizedYield) {
				call = &grp[i]
			}
			if grp[i].Direction == "P" && (put == nil || grp[i].AnnualizedYield > put.AnnualizedYield) {
				put = &grp[i]
			}
		}
		if call == nil || put == nil {
			continue // 单侧无腿=对不成——该组不评（下一组续）
		}
		ec := r.evalLeg(ctx, *call, coins, p)
		ep := r.evalLeg(ctx, *put, coins, p)
		inst := call.ProductID + "+" + put.ProductID
		verdict, reason := "proposed", ""
		apr := (ec.APR + ep.APR) / 2 // 双腿等权名义合并 APR
		switch {
		case ec.Verdict == "unevaluable":
			verdict, reason = "unevaluable", "call_"+ec.Reason
		case ep.Verdict == "unevaluable":
			verdict, reason = "unevaluable", "put_"+ep.Reason
		case ec.Verdict == "rejected":
			verdict, reason = "rejected", "call_"+ec.Reason
		case ep.Verdict == "rejected":
			verdict, reason = "rejected", "put_"+ep.Reason
		case apr < p.MinAPR:
			verdict, reason = "rejected", "apr_thin"
		}
		if err := r.persistLegEval(ctx, dcd.Eval{Product: *call, APR: apr, Verdict: verdict, Reason: reason},
			f, inst); err != nil {
			return err
		}
	}
	return nil
}

// volParams 判据键复用 research.dcd.* 阈域（任一缺=fail-closed）。
func (r *Runner) volParams(ctx context.Context) (dcd.Params, error) {
	var p dcd.Params
	for _, kv := range []struct {
		k string
		v *float64
	}{
		{"research.dcd.min_dist_pct", &p.MinDistPct},
		{"research.dcd.max_term_days", &p.MaxTermDays},
		{"research.dcd.min_apr", &p.MinAPR},
		{"research.dcd.notional_pct", &p.NotionalPct},
	} {
		v, err := r.cfg.GetFloat(kv.k)
		if err != nil {
			return p, fmt.Errorf("%s: %w", kv.k, err)
		}
		*kv.v = v
	}
	return p, nil
}

// evalLeg 单腿评估（dcd.Evaluate 同判据域；spot=uly 方言归一 ticks/mark 行，
// avail=锁仓侧余量——coins 缺/读败=avail<0 unevaluable）。
func (r *Runner) evalLeg(ctx context.Context, pr dcd.Product, coins map[string]float64, p dcd.Params) dcd.Eval {
	avail := -1.0
	if coins != nil {
		if pr.Direction == "P" {
			cc := pr.QuoteCcy
			if cc == "" {
				cc = "USDT"
			}
			avail = coins[cc]
		} else {
			avail = coins[ulyBase(pr.Uly)]
		}
	}
	return dcd.Evaluate(pr, r.spotOf(ctx, pr.Uly), avail, p)
}

// persistLegEval 评估卡落+proposed 且无 open 提案→fin_proposal（dcd
// persistEval 同径幂等：1h 防抖窗在 writeEval 内）。
func (r *Runner) persistLegEval(ctx context.Context, e dcd.Eval, f Family, inst string) error {
	if err := r.writeEval(ctx, inst, f.Template, f.Direction, e.Verdict, e.Reason,
		e.Evidence()); err != nil {
		return err
	}
	if e.Verdict != "proposed" {
		return nil
	}
	var evalID int64
	if err := r.pool.QueryRow(ctx, `SELECT id FROM fin_opp_eval
		WHERE inst_id=$1 AND template=$2 AND direction=$3`, inst, f.Template, f.Direction).Scan(&evalID); err != nil {
		return err
	}
	var open int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE inst_id=$1 AND state='fin_proposed'", inst).Scan(&open); err != nil {
		return err
	}
	if open > 0 {
		return nil
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,$2,$3,$4)`, evalID, inst, f.Direction, e.APR); err != nil {
		r.rej.Inc("proposal_write_err")
		return err
	}
	log.Printf("[research/grammar] vol-sell 提案落卡 %s %s apr=%.4f", inst, f.Template, e.APR)
	return nil
}

// ulyBase uly 首段（BTC-USD→BTC——锁仓基币方言）。
func ulyBase(uly string) string {
	if i := strings.Index(uly, "-"); i > 0 {
		return uly[:i]
	}
	return uly
}

// spotOf uly 现价（md_ticks∪md_mark_index 末行——uly 方言归一同 dcd.eval
// 序：uly→uly+T→uly+T-SWAP→uly-SWAP，USDG 尾补 USDT 变体，首命中即回）。
func (r *Runner) spotOf(ctx context.Context, uly string) float64 {
	cands := []string{uly, uly + "T", uly + "T-SWAP", uly + "-SWAP"}
	if alt := strings.Replace(uly, "USDG", "USDT", 1); alt != uly {
		cands = append(cands, alt, alt+"-SWAP")
	}
	var last *float64
	for _, inst := range cands {
		_ = r.pool.QueryRow(ctx, `SELECT v FROM (SELECT last v,ts FROM md_ticks WHERE inst_id=$1
			UNION ALL SELECT px,ts FROM md_mark_index WHERE inst_id=$1) q
			ORDER BY ts DESC LIMIT 1`, inst).Scan(&last)
		if last != nil {
			return *last
		}
	}
	return 0
}
