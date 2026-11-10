// matrix_legs.go —— T-0461 矩阵腿形构法（行数棘轮拆件自 matrix.go）：
// funding/basis/xvenue 三族腿构+flat 残留判+TgtQuote 网格对齐——腿序/
// 名义/量纲全产线同径（execution.BuildXvenueLegs 直用）。
package simdrill

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"o/internal/platform/excell"
	"o/internal/trading/execution"
	"o/internal/trading/mt5sim"
	"o/internal/trading/simvenue"
)

// isBasisTemplate basis 族判定（模板族——basis_open.go 同域；inst 键形
// 由 basisLegs 内 datedFutRe 断言 fail-closed）。
func isBasisTemplate(tmpl string) bool {
	return tmpl == "quarter_basis_long" || tmpl == "perp_quarter_spread" || tmpl == "quarter_basis_short"
}

// spotQuoteSz TgtQuote 委托额网格对齐——simvenue vetGrid 判
// q=sz/px 须落 lotSz 网格（floor 到格内最大额；实成略低于 half，
// leg.Notional 声明值不变 hedge_band 对齐律不受影响）。
func spotQuoteSz(quote, px, lotSz float64) float64 {
	if px <= 0 || lotSz <= 0 {
		return quote
	}
	return math.Floor(quote/(px*lotSz)+1e-9) * lotSz * px
}

// fundingLegs funding-hedge 族腿构（现货 TgtQuote 买+SWAP 卖——Drill.Run
// 同型同名义径；flat=现货裸袋归零+合约净持归零双锚）。
func fundingLegs(t excell.Tier, inst string, vs *simvenue.SimVenue) ([2]execution.Leg, func() bool, error) {
	spotID := strings.TrimSuffix(inst, "-SWAP")
	sp, ok1 := vs.Instrument(spotID)
	sw, ok2 := vs.Instrument(inst)
	if !ok1 || !ok2 {
		return [2]execution.Leg{}, nil, fmt.Errorf("演习乐器不在册 %s（spot/swap 缺）", inst)
	}
	half := t.Notional / 2
	contracts := math.Floor(half/(sw.CtVal*sw.Px)/sw.LotSz+1e-9) * sw.LotSz
	if contracts <= 0 {
		return [2]execution.Leg{}, nil, fmt.Errorf("合成腿张数≤0（notional=%.2f）", half)
	}
	legs := [2]execution.Leg{
		{Venue: "okx", InstID: spotID, Side: "buy", TgtQuote: true,
			Sz: strconv.FormatFloat(spotQuoteSz(half, sp.Px, sp.LotSz), 'f', -1, 64), Notional: half},
		{Venue: "okx", InstID: inst, Side: "sell",
			Sz: strconv.FormatFloat(contracts, 'f', -1, 64), Notional: half},
	}
	return legs, func() bool {
		return math.Abs(vs.Position(inst)) <= sw.LotSz/2+1e-9 &&
			vs.Balance(baseCcy(spotID)) <= math.Max(sp.LotSz, 1e-6)
	}, nil
}

// basisLegs basis 族腿构（basis_open.go 同派生律：衍生腿=-YYMMDD 期货空；
// 多头腿 quarter_basis_long→基键现货买 / 余→基键+-SWAP 买；flat=全腿
// venue 真相归零——期货净持+多头腿残位双锚）。
func basisLegs(c excell.Cell, t excell.Tier, inst string, vs *simvenue.SimVenue) ([2]execution.Leg, func() bool, error) {
	if c.Template == "quarter_basis_short" {
		// spot_short 腿=保证金借币卖——margin-borrow 面 enabled=false（T-0216
		// 条件缺口/parked 域）；原映射错塞 SWAP 多腿构——显式拒防错腿产凭据。
		return [2]execution.Leg{}, nil, fmt.Errorf("quarter_basis_short 演习不支持（spot_short 腿 margin-borrow 面缺——fail-closed 不构错腿）")
	}
	mm := datedFutRe.FindStringSubmatch(inst)
	if mm == nil {
		return [2]execution.Leg{}, nil, fmt.Errorf("basis inst 非到期日期货键: %s", inst)
	}
	base := mm[1] // 基键=BTC-USDT（现货键）
	fut, ok := vs.Instrument(inst)
	if !ok {
		return [2]execution.Leg{}, nil, fmt.Errorf("期货乐器不在册 %s", inst)
	}
	half := t.Notional / 2
	futContracts := math.Floor(half/(fut.CtVal*fut.Px)/fut.LotSz+1e-9) * fut.LotSz
	if futContracts <= 0 {
		return [2]execution.Leg{}, nil, fmt.Errorf("期货腿张数≤0")
	}
	futLeg := execution.Leg{Venue: "okx", InstID: inst, Side: "sell",
		Sz: strconv.FormatFloat(futContracts, 'f', -1, 64), Notional: half}
	flat := func() bool {
		return math.Abs(vs.Position(inst)) <= fut.LotSz/2+1e-9
	}
	if c.Template == "quarter_basis_long" {
		// 现货腿 TgtQuote 买（quote 计价）。现货键走 hedgeSpotKey 归一
		// （产线 lifecycle.basisSpotLeg 同律——-USD 期货对冲=USDT 现货）：
		// 基键直查 BTC-USD 类实有无对假「不在册」（2026-11-10 产线坑）。
		spotID := hedgeSpotKey(base)
		sp, ok := vs.Instrument(spotID)
		if !ok {
			return [2]execution.Leg{}, nil, fmt.Errorf("现货乐器不在册 %s", spotID)
		}
		legs := [2]execution.Leg{futLeg, {Venue: "okx", InstID: spotID, Side: "buy",
			Sz:       strconv.FormatFloat(spotQuoteSz(half, sp.Px, sp.LotSz), 'f', -1, 64),
			Notional: half, TgtQuote: true}}
		inner := flat
		return legs, func() bool {
			return inner() && vs.Balance(baseCcy(spotID)) <= math.Max(sp.LotSz, 1e-6)
		}, nil
	}
	// perp_quarter_spread→基键+-SWAP 多腿。
	swID := base + "-SWAP"
	sw, ok := vs.Instrument(swID)
	if !ok {
		return [2]execution.Leg{}, nil, fmt.Errorf("perp 乐器不在册 %s", swID)
	}
	contracts := math.Floor(half/(sw.CtVal*sw.Px)/sw.LotSz+1e-9) * sw.LotSz
	if contracts <= 0 {
		return [2]execution.Leg{}, nil, fmt.Errorf("perp 多腿张数≤0")
	}
	legs := [2]execution.Leg{futLeg, {Venue: "okx", InstID: swID, Side: "buy",
		Sz: strconv.FormatFloat(contracts, 'f', -1, 64), Notional: half}}
	inner := flat
	return legs, func() bool {
		return inner() && math.Abs(vs.Position(swID)) <= sw.LotSz/2+1e-9
	}, nil
}

// xvenueLegs xvenue 双腿构（产线同径 execution.BuildXvenueLegs——腿序
// mt5 CFD 先+okx perp 后钉死；meta/px 全仿真册值注入）。
func xvenueLegs(c excell.Cell, t excell.Tier, inst string, mt *mt5sim.Sim) ([2]execution.Leg, error) {
	if c.Direction != excell.DirLongPerpShortCFD && c.Direction != excell.DirShortPerpLongCFD {
		return [2]execution.Leg{}, fmt.Errorf("xvenue 向位 %q 未载映射表", c.Direction)
	}
	canon := cfdCanonical(inst)
	cs, minLot, lotStep, tradeMode, ok := mt.Meta(canon)
	if !ok {
		return [2]execution.Leg{}, fmt.Errorf("cfd 乐器不在册 %s", canon)
	}
	px := pxOf(inst)
	perpM := execution.InstMeta{CtVal: 0.001, CtValCcy: "USDT", LotSz: 0.001, MinSz: 0.001}
	return execution.BuildXvenueLegs(c.Direction, inst, canon, t.Notional,
		perpM, execution.CfdMeta{ContractSize: cs, MinLot: minLot,
			LotStep: lotStep, TradeMode: tradeMode}, px, px)
}
