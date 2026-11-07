// xvenue_legs.go —— T-0145 跨所 carry 双腿构（自 sizing.go 拆件——棘轮 D-588）。
package execution

import (
	"fmt"
	"math"
	"strconv"
)

// BuildXvenueLegs 跨所 carry 双腿构（T-0145——DM 裁定腿序钦定 leg1=mt5 cfd
// 先证可开受限 venue，leg2=okx perp；回滚成本=OrderClose(ticket) 在案）。
// 向位钦定两态：long_perp_short_cfd→mt5 sell+okx buy；short_perp_long_cfd→
// mt5 buy+okx sell；他向位 fail-closed 拒构。
// cfd 腿先落 lot 网格（粗粒度先量化），perp 腿反推 cfd 实名义额（T-0062
// 对齐律跨 venue 同律——Notional 携各腿实值，hedge_band 验真对齐）。
// pxCfd=cfd 标的价（底层资产行情）；pxPerp=perp 自身最新价（币本位 ctVal
// 换算）。TradeMode=SymbolGroup 活读值（下单时点 FullAccess 断言同
// BuildCfdLegs——CloseOnly/Disabled/单向限拒构，禁台账快照当放行证）。
func BuildXvenueLegs(direction, instID, cfdInst string, notional float64,
	perpM InstMeta, cfdM CfdMeta, pxCfd, pxPerp float64) ([2]Leg, error) {
	var cfdSide, perpSide string
	switch direction {
	case "long_perp_short_cfd":
		cfdSide, perpSide = "sell", "buy"
	case "short_perp_long_cfd":
		cfdSide, perpSide = "buy", "sell"
	default:
		return [2]Leg{}, fmt.Errorf("xvenue 向位 %q 未载映射表 fail-closed", direction)
	}
	if cfdInst == "" {
		return [2]Leg{}, fmt.Errorf("cfd 官名缺 fail-closed（%s 无双腿映射）", instID)
	}
	if cfdM.TradeMode != 4 {
		return [2]Leg{}, fmt.Errorf("cfd %s TradeMode=%d 非 FullAccess（可开性预检拒——broker 动态限，单腿裸露防线）", cfdInst, cfdM.TradeMode)
	}
	half := notional / 2
	if pxCfd <= 0 || pxPerp <= 0 {
		return [2]Leg{}, fmt.Errorf("px 缺 fail-closed（cfd=%.4f perp=%.4f）", pxCfd, pxPerp)
	}
	lots := math.Floor(half/pxCfd/cfdM.ContractSize/cfdM.LotStep) * cfdM.LotStep
	if lots < cfdM.MinLot {
		return [2]Leg{}, fmt.Errorf("cfd 最小手数起不来: lots=%.4f<min=%.4f（规模不达标不裸放禁降阈）", lots, cfdM.MinLot)
	}
	// cfd 实名义反推 perp 腿（T-0062 对齐律——lot 砍残失衡由 perp 侧吸收）。
	cfdUSD := lots * cfdM.ContractSize * pxCfd
	perpRaw, err := perpContracts(cfdUSD, perpM.CtVal, perpM.CtValCcy, pxPerp)
	if err != nil {
		return [2]Leg{}, err
	}
	perpSz := floorLot(perpRaw, perpM.LotSz)
	if perpSz < perpM.MinSz {
		return [2]Leg{}, fmt.Errorf("perp 最小张起不来: sz=%.4f<minSz=%.4f", perpSz, perpM.MinSz)
	}
	perpUSD := perpSz * perpM.CtVal
	if perpM.CtValCcy != "USD" && perpM.CtValCcy != "USDT" {
		perpUSD *= pxPerp
	}
	return [2]Leg{
		{Venue: "mt5", InstID: cfdInst, Side: cfdSide, Sz: strconv.FormatFloat(lots, 'f', -1, 64),
			Notional: cfdUSD, CtVal: cfdM.ContractSize},
		{Venue: "okx", InstID: instID, Side: perpSide, Sz: strconv.FormatFloat(perpSz, 'f', -1, 64),
			Notional: perpUSD},
	}, nil
}
