// swapdialect.go —— MT5 swap 单位方言换算件（FIX-02 自 bridge.go 拆出——行规模门禁
// D-077/D-588 棘轮；与 swapdialect_test.go 同族对位：Points/PercCurPrice 年化
// 正典式+缺席如实语义。unit 方言判据出处=skills/mt5-quotes-history+proto 实读）。
package mt5

import (
	mt5pb "o/internal/gen/mtapi/mt5"
)

// SwapAnnualBPS swap→空腿年化 bps 换算件（T-0143 点数方言解锁闸）。
// 单位方言判据（SwapType 全谱——skills/mt5-quotes-history+proto 实读）：
//
//	PercCurPrice（5）=报价币百分比/天→swap_short×365×10⁴ bps 直年化；
//	Points（1）=点数/天→币值/手/天=swap_points×TickValue×Points/TickSize，
//	  年化 bps=币值×365/(mid×ContractSize)×10⁴（换算链任一环零=不可年化如实）；
//	其余单位（币量/保证金）=缺换算件（false——val=NULL 走缺席如实）。
func SwapAnnualBPS(si *mt5pb.SymbolInfo, sg *mt5pb.SymGroup, mid float64) (float64, bool) {
	if sg == nil {
		return 0, false
	}
	return swapAnnualSide(si, sg, mid, sg.GetSwapShort())
}

// SwapAnnualBoth 双腿年化（T-0144 返修①——swap_long/short 各经同换算核：
// 任一腿不可年化=ok false 整体缺席——双腿经济学须双端齐）。
func SwapAnnualBoth(si *mt5pb.SymbolInfo, sg *mt5pb.SymGroup, mid float64) (float64, float64, bool) {
	if sg == nil {
		return 0, 0, false
	}
	long, okL := swapAnnualSide(si, sg, mid, sg.GetSwapLong())
	short, okS := swapAnnualSide(si, sg, mid, sg.GetSwapShort())
	return long, short, okL && okS
}

// swapAnnualSide 单腿换算核（swap=长/短腿字段值——单位方言判据同正典式）。
func swapAnnualSide(si *mt5pb.SymbolInfo, sg *mt5pb.SymGroup, mid, swap float64) (float64, bool) {
	switch sg.GetSwapType() {
	case mt5pb.SwapType_SwapType_PercCurPrice:
		return swap * 365 * 10000, true
	case mt5pb.SwapType_SwapType_InPoints:
		// T-0143 二返修正典式（venue 实证 2026-10-26：BTCUSDm/ETHUSDm
		// tick_value/tick_size=0 不填、points=0.01/contract_size=1 实填——
		// tick 双字段降为候选佐证非必需环，0=不阻断）：
		//   点值=points×contract_size（报价币/点/手）
		//   币值/手/天=swap×points×cs；年化 bps=×365/(mid×cs)×10⁴（cs 约掉量纲自洽）。
		if si == nil {
			return 0, false
		}
		pts, cs := si.GetPoints(), si.GetContractSize()
		if pts <= 0 || cs <= 0 || mid <= 0 {
			return 0, false // 换算链缺环——缺席如实
		}
		ccyPerLot := swap * pts * cs
		return ccyPerLot * 365 / (mid * cs) * 10000, true
	default:
		return 0, false
	}
}
