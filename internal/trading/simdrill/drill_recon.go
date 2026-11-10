// drill_recon.go —— 双腿径账实对拍一族（T-0522 棘轮拆件自 drill.go）：
// recon 四锚（全径闭环/drift 假报/残袋残仓腿型分判/现金对平）+legNotional
// 腿名义+balanced 均衡判——funding/basis 通用件（腿型按 InstType 分派）。
package simdrill

import (
	"context"
	"fmt"
	"math"

	"o/internal/trading/simvenue"
)

// recon 账实对拍四锚（detail=人读结论；verdict=fail=病面已升件）。
// long=多头腿（SPOT=裸袋挂账锚/合约=净持锚）deriv=卖出衍生腿（净持锚）。
func (d *Drill) recon(ctx context.Context, res Result, long, deriv simvenue.Instrument, openBalanced bool) (string, string) {
	var closed bool
	var posID int64
	if err := d.Pool.QueryRow(ctx, `SELECT id,closed_at IS NOT NULL FROM fin_position
		WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`, res.OpenExecID).Scan(&posID, &closed); err != nil {
		return d.fail(ctx, res, "演习仓行缺（开仓事实不可读）"), "fail"
	}
	// 锚⓪全径闭环：仓未平=平仓径败（拒单/执行败）——演习未走完即 fail 禁假绿。
	if !closed {
		return d.fail(ctx, res,
			fmt.Sprintf("演习未平（触发=%s close_exec=#%d）——平仓径拒单/执行败", orDash(res.Trigger), res.CloseExecID)), "fail"
	}
	// 锚①drift 假报：开仓后 venue 真相均衡却报 delta_drift=假报（升件）。
	if res.Trigger == "delta_drift" && openBalanced {
		return d.fail(ctx, res,
			"delta_drift 假报——venue 真相开仓双腿均衡（出生粒度域）"), "fail"
	}
	// 锚②残袋/残仓按腿型分判：SPOT 多头腿=仓关而 venue 货在且未挂账
	// （挂账=账实同步显形非盲）；合约多头腿=净持归零（非资产挂账域）。
	var bagQty float64
	residLabel := ""
	resid := 0.0
	if long.InstType == "SPOT" {
		_ = d.Pool.QueryRow(ctx, `SELECT COALESCE(bare_bag_qty,0) FROM fin_position
			WHERE id=$1`, posID).Scan(&bagQty)
		resid = d.Venue.Balance(baseOf(long))
		residLabel = baseOf(long)
		dust := math.Max(long.LotSz, 1e-9)
		if resid > math.Max(dust, bagQty)+1e-9 {
			return d.fail(ctx, res,
				fmt.Sprintf("残袋非零未挂账——venue 实余 %.8g %s>挂账 %.8g（closed 了但货还在）", resid, residLabel, bagQty)), "fail"
		}
	} else if math.Abs(d.Venue.Position(long.InstID)) > long.LotSz/2+1e-9 {
		return d.fail(ctx, res,
			fmt.Sprintf("合约残仓 %.8g 张——多头腿 %s 平未净敞口", d.Venue.Position(long.InstID), long.InstID)), "fail"
	}
	// 锚③衍生腿净持归零：平仓后净持≠0=敞口残（升件——非资产挂账域）。
	if math.Abs(d.Venue.Position(deriv.InstID)) > deriv.LotSz/2+1e-9 {
		return d.fail(ctx, res,
			fmt.Sprintf("合约残仓 %.8g 张——衍生腿 %s 平未净敞口", d.Venue.Position(deriv.InstID), deriv.InstID)), "fail"
	}
	// 锚④现金对平：venue USDT 流水 Δ==Σfin_cashflow（费行入账后——口径差
	// >分位=账实背离升件）。
	var sum float64
	if err := d.Pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE position_ref=$1 AND kind NOT IN ('earn_in','earn_out')`, posID).Scan(&sum); err != nil {
		return d.fail(ctx, res, fmt.Sprintf("fin_cashflow 归仓和读败: %v", err)), "fail"
	}
	var venueSum float64
	for _, f := range d.Venue.Fills() {
		venueSum += f.QuoteAmt // 计价币流水（fixture 域 USDT 计价——venue 真相）
	}
	if math.Abs(venueSum-sum) > 0.01 {
		return d.fail(ctx, res,
			fmt.Sprintf("现金账实不平——venue 流水 %.6f vs 台账 %.6f（差 %.6f>分位）", venueSum, sum, venueSum-sum)), "fail"
	}
	return fmt.Sprintf("全径绿（open=#%d close=#%d 触发=%s 现金对平 %.6f 残余 %s=%.8g）",
		res.OpenExecID, res.CloseExecID, orDash(res.Trigger), sum, residLabel, resid), "pass"
}

// legNotional 腿名义（SPOT=余额×价 合约=净持×面值×价——venue 真相口径）。
func (d *Drill) legNotional(in simvenue.Instrument) float64 {
	if in.InstType == "SPOT" {
		return d.Venue.Balance(baseOf(in)) * in.Px
	}
	return math.Abs(d.Venue.Position(in.InstID)) * in.CtVal * in.Px
}

// balanced venue 真相均衡判（开仓后双腿名义对齐——腿型按 InstType 分判
// SPOT=余额域/合约=净持域,funding/basis 通用）。容差=50bps 与衍生腿
// 1 张名义的大者——合约张数取整天然残差（$50 名义 BTC 域 1 张≈$4=16%，
// 粒度内不算失衡；超 1 张=真失衡才算 venue 真相不均衡）。
func (d *Drill) balanced(long, deriv simvenue.Instrument) bool {
	ln, dn := d.legNotional(long), d.legNotional(deriv)
	avg := (ln + dn) / 2
	if avg <= 0 {
		return false
	}
	tol := math.Max(50, deriv.CtVal*deriv.Px/avg*1e4)
	return math.Abs(ln-dn)/avg*1e4 <= tol
}
