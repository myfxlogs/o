// drill_basis.go —— basis 族单策略演习径（T-0522 假想卡骨架批——
// quarter_basis_long/perp_quarter_spread 毕业闸凭据产能面；同坑戒
// KB simdrill-gate-domain-coverage-mismatch：闸消费域新增族=演习产能
// 同 commit 补格）。inst=-YYMMDD 期货键；腿构=matrix basisLegs 同件
// （期货空+多头腿现货/永续买）；对拍锚=leggedDrill 通用四锚腿型分判。
package simdrill

import (
	"context"
	"fmt"

	"o/internal/platform/excell"
	"o/internal/trading/execution"
)

// runBasisDrill basis 族全径演习（inst=到期日期货键；腿序/名义=执行器
// basisOpen 同律——衍生腿期货空+多头腿现货（quarter_basis_long）或永续
// （perp_quarter_spread）买）。
func (d *Drill) runBasisDrill(ctx context.Context, res Result, inst, templateRef string, notional float64) (Result, error) {
	// 腿构=矩阵同件（产线执行器派生律同域——basisLegs 内 datedFutRe/
	// quarter_basis_short 显式拒等断言在件）。
	legs, _, err := basisLegs(excell.Cell{Template: templateRef},
		excell.Tier{Notional: notional}, inst, d.Venue)
	if err != nil {
		return res, fmt.Errorf("basis 腿构: %w", err)
	}
	// 对拍锚乐器面——deriv=期货键本身（净持归零锚）；long=多头腿键
	// （SPOT=裸袋挂账锚 / SWAP=合约净持锚——recon 按 InstType 分判）。
	deriv, ok := d.Venue.Instrument(inst)
	if !ok {
		return res, fmt.Errorf("演习乐器不在册: 期货腿 %s", inst)
	}
	longID := legs[1].InstID // basisLegs 腿序=[期货空,多头买]——多头键直取腿件
	long, ok := d.Venue.Instrument(longID)
	if !ok {
		return res, fmt.Errorf("演习乐器不在册: 多头腿 %s", longID)
	}
	return d.leggedDrill(ctx, res, inst, notional, legs,
		basisGateLegs(legs), deriv, long)
}

// basisGateLegs 闸签标签（matrix_exec 同派生律——腿型按键形判：
// -YYMMDD=fut/-SWAP=perp/余=spot + 买卖向；闸语义只验非空,签入决策行供审计）。
func basisGateLegs(legs [2]execution.Leg) []string {
	out := make([]string, 0, 2)
	for _, l := range legs {
		tag := "spot"
		switch {
		case datedFutRe.MatchString(l.InstID):
			tag = "fut"
		case l.Venue == "mt5":
			tag = "cfd"
		case len(l.InstID) > 5 && l.InstID[len(l.InstID)-5:] == "-SWAP":
			tag = "perp"
		}
		dir := "long"
		if l.Side == "sell" {
			dir = "short"
		}
		out = append(out, tag+"_"+dir)
	}
	return out
}
