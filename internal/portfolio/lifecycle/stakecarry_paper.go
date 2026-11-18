// stakecarry_paper.go —— stake-carry 双头族 paper 记账臂（工单#21612 参考
// 模型面：TRX 链上质押率+CMC/OKX 空腿付费面=外部收益参考簿记，非公司业绩
// 簿记——venue 真执行臂另案）。双腿计息：
//
//	① stake 腿 = trx_onchain 现率按日摊（fin_trx_vote_apr.current_rate +
//	   fin_trx_txs rental_income 近7d实收/fin_trx_account.energy_delegated
//	   折年——hypothesis SpotYield 同工口径）；
//	② 空腿 = venue 方言：okx→md_funding TRX-USDT-SWAP 行逐条 +rate*notional
//	   （空腿收 funding 正号——basis.go perp_long 付费同式翻向）；
//	   cmc→fin_cmc_history OVERNIGHT_SETTLEMENT 30d 日均实付率按日摊
//	   （venue 实收意愿证据——hypothesis.cmcRealizedSwapAnn 同式）。
//	率缺/样本不足/表域缺席=该腿让位不造 fill（fail-closed 同 dcd 径）。
package lifecycle

import (
	"context"
	"time"
)

// stakeCarryAccounting 双腿记账拍（挂 scan 非致命位——记账败不拖开户/晋升拍）。
func (r *Runner) stakeCarryAccounting(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.notional_usd
		FROM fin_strategy s WHERE s.state='paper' AND s.template_ref='stake-carry'`)
	if err != nil {
		return err
	}
	type st struct {
		id       int64
		inst     string
		notional float64
	}
	var sts []st
	for rows.Next() {
		var x st
		if err := rows.Scan(&x.id, &x.inst, &x.notional); err != nil {
			rows.Close()
			return err
		}
		sts = append(sts, x)
	}
	rows.Close()
	for _, x := range sts {
		if err := r.stakeLegAccrue(ctx, x.id, x.inst, x.notional); err != nil {
			return err
		}
		if err := r.shortLegAccrue(ctx, x.id, x.inst, x.notional); err != nil {
			return err
		}
	}
	return nil
}

// stakeLegAccrue 质押腿按日摊——fill inst 键=inst+":stake"（与空腿分键各记
// 各 last-ts，basis 双腿分键同式）。率=链上现率；行龄陈=让位（stale_hours
// 闸读 research.stake.stale_hours——枚举臂同键值域）。
func (r *Runner) stakeLegAccrue(ctx context.Context, sid int64, inst string, notional float64) error {
	var last time.Time
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT max(ts) FROM fin_paper_fill f
		WHERE f.strategy_id=$1 AND f.kind='funding' AND f.inst_id=$2), s.entered_at)
		FROM fin_strategy s WHERE s.id=$1`, sid, inst+":stake").Scan(&last); err != nil {
		return err
	}
	days := time.Since(last).Hours() / 24
	if days < 1.0 {
		return nil // 日摊节拍同 dcd 径
	}
	apr, ok := r.trxOnchainApr(ctx)
	if !ok {
		r.rej.Inc("stake_yield_stale")
		return nil // 率缺/陈=让位不虚构（SpotYield absent 如实同款）
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
		VALUES($1,'funding',$2,$3,$4,now())`,
		sid, inst+":stake", notional*apr*days/365,
		[]byte(`{"apr":`+ftoa(apr)+`,"leg":"stake","src":"trx_onchain"}`)); err != nil {
		return err
	}
	return nil
}

// trxOnchainApr 链上质押现率=vote 年化(fin_trx_vote_apr.current_rate%)
// +rental 年化(近7d rental_income 实收/energy_delegated 快照×365/7)——
// hypothesis funding.trxOnchainYield 同口径；vote 行缺/陈=缺席如实。
func (r *Runner) trxOnchainApr(ctx context.Context) (float64, bool) {
	staleH := 0.0
	if v, err := r.cfg.GetFloat("research.stake.stale_hours"); err == nil && v > 0 {
		staleH = v
	}
	var votePct float64
	var vts time.Time
	if err := r.pool.QueryRow(ctx, `SELECT current_rate, fetched_at FROM fin_trx_vote_apr
		ORDER BY fetched_at DESC LIMIT 1`).Scan(&votePct, &vts); err != nil {
		return 0, false
	}
	if staleH > 0 && time.Since(vts) > time.Duration(staleH*float64(time.Hour)) {
		return 0, false
	}
	rentalPct := 0.0
	var delegated int64
	if err := r.pool.QueryRow(ctx, `SELECT energy_delegated FROM fin_trx_account
		ORDER BY fetched_at DESC LIMIT 1`).Scan(&delegated); err == nil && delegated > 0 {
		var sun float64
		if err := r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_sun),0) FROM fin_trx_txs
			WHERE kind='rental_income' AND ts>now()-interval '7 days'`).Scan(&sun); err == nil && sun > 0 {
			rentalPct = (sun / 1e6 / 7 * 365) / (float64(delegated) / 1e6) * 100
		}
	}
	return (votePct + rentalPct) / 100, true
}

// shortLegAccrue 空腿实付记账——venue 方言派发（inst=venue:ccy 前缀解）。
// okx：md_funding 行逐条 +rate*notional（空腿收——perp_long 付费翻向）；
// cmc：30d 实收日均率按日摊。键面缺席=让位。
func (r *Runner) shortLegAccrue(ctx context.Context, sid int64, inst string, notional float64) error {
	switch {
	case len(inst) > 4 && inst[:4] == "okx:":
		perpInst := inst[4:] + "-USDT-SWAP"
		var flast time.Time
		if err := r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT max(ts) FROM fin_paper_fill f
			WHERE f.strategy_id=$1 AND f.kind='funding' AND f.inst_id=$2), s.entered_at)
			FROM fin_strategy s WHERE s.id=$1`, sid, perpInst).Scan(&flast); err != nil {
			return err
		}
		frows, err := r.pool.Query(ctx, `SELECT ts, funding_rate FROM md_funding
			WHERE inst_id=$1 AND ts>$2 ORDER BY ts`, perpInst, flast)
		if err != nil {
			return err
		}
		defer frows.Close()
		for frows.Next() {
			var ts time.Time
			var rate float64
			if err := frows.Scan(&ts, &rate); err != nil {
				return nil // 行损=本拍让位（已落 fill 不回滚——basis 同款）
			}
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
				VALUES($1,'funding',$2,$3,$4,$5)`,
				sid, perpInst, rate*notional,
				[]byte(`{"rate":`+ftoa(rate)+`,"leg":"perp_short"}`), ts); err != nil {
				return err
			}
		}
		return nil
	case len(inst) > 4 && inst[:4] == "cmc:":
		var last time.Time
		if err := r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT max(ts) FROM fin_paper_fill f
			WHERE f.strategy_id=$1 AND f.kind='funding' AND f.inst_id=$2), s.entered_at)
			FROM fin_strategy s WHERE s.id=$1`, sid, inst+":cmc_swap").Scan(&last); err != nil {
			return err
		}
		days := time.Since(last).Hours() / 24
		if days < 1.0 {
			return nil
		}
		bps, ok := r.cmcRealizedSwapBps(ctx)
		if !ok {
			r.rej.Inc("cmc_swap_stale")
			return nil
		}
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
			VALUES($1,'funding',$2,$3,$4,now())`,
			sid, inst+":cmc_swap", notional*bps/1e4*days/365,
			[]byte(`{"bps":`+ftoa(bps)+`,"leg":"cmc_swap","src":"cmc_realized_30d"}`)); err != nil {
			return err
		}
		return nil
	}
	return nil // venue 方言未注册——该腿让位如实（枚举面未产不虚构）
}

// cmcRealizedSwapBps CMC 空腿实付年化 bps（30d OVERNIGHT_SETTLEMENT 日均
// /positions 空腿名义×365——hypothesis.cmcRealizedSwapAnn 同式；样本<3d/
// 名义缺=缺席如实）。
func (r *Runner) cmcRealizedSwapBps(ctx context.Context) (float64, bool) {
	var daily float64
	var days int
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(AVG(s),0), count(*) FROM (
		SELECT date_trunc('day',transaction_time) d, SUM(amount) s
		FROM fin_cmc_history
		WHERE transaction_type='OVERNIGHT_SETTLEMENT'
		  AND transaction_time>now()-interval '30 days'
		GROUP BY 1) t`).Scan(&daily, &days); err != nil || days < 3 || daily <= 0 {
		return 0, false
	}
	var notional float64
	if err := r.pool.QueryRow(ctx, `SELECT quantity*entry_price FROM fin_cmc_positions
		WHERE direction='Sell' ORDER BY fetched_at DESC LIMIT 1`).Scan(&notional); err != nil || notional <= 0 {
		return 0, false
	}
	return daily / notional * 365 * 1e4, true
}
