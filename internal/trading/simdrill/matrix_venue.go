// matrix_venue.go —— T-0461 矩阵仿真 venue 装配+乐器键派生（行数棘轮
// 拆件自 matrix.go）：inst 键对/baseCcy/cfdCanonical/pxOf 确定性价/
// newOKXVenue/newMT5Venue 本格乐器册/dcdSim DCD 产品仿真 venue。
package simdrill

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"

	"o/internal/trading/mt5sim"
	"o/internal/trading/simvenue"
)

// ---- 仿真 venue 装配 ----

// instPair 乐器键对（inst→base/quote 拆分——X-USDT-SWAP→(X,USDT)；
// X-USDT-YYMMDD→(X,USDT)；裸 X-USDT→(X,USDT)）。
func instPair(inst string) (base, quote string) {
	parts := strings.Split(inst, "-")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return inst, "USDT"
}

// baseCcy 乐器基键（instPair[0]——现货裸袋判锚）。
func baseCcy(inst string) string { b, _ := instPair(inst); return b }

// cfdCanonical okx perp 键→CFD 官话 canonical（X-USDT-SWAP→XUSD——mt5 腿
// InstID=官名唯一天，venue 方言名由 mt5sim.Dialect 译件承担）。
func cfdCanonical(inst string) string { return baseCcy(inst) + "USD" }

// pxOf 确定性仿真价（fnv 散列基键→[100,10000)——同基键同价跨格一致，
// 异基键价差防同值串判；名义换算全由 legs 携带不依赖绝对价）。
func pxOf(inst string) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(baseCcy(inst)))
	return 100 + float64(h.Sum32()%9900)
}

// newOKXVenue 本格 OKX 仿真 venue（乐器册按格 inst 派生——derived spot
// 仅当 OKX 实有面在册才登记（T-0482 实有面校真）；SWAP/本格合约全量入册；
// SWAP+基键-SWAP 双登幂等；钱域 USDT 1e7 播种）。
func newOKXVenue(inst string) *simvenue.SimVenue {
	base, quote := instPair(inst)
	px := pxOf(inst)
	v := simvenue.New().WithBalance("USDT", 1e7)
	spotID := base + "-" + quote
	swapID := spotID + "-SWAP"
	// 实有面无此对不登记——腿构「现货乐器不在册」fail 机理对齐产线
	// loadMeta instruments 空 rows fail-closed（T-0481 判词 sim 缺陷根修）。
	if okxSpotLive(spotID) {
		v.WithInstrument(simvenue.Instrument{InstID: spotID, InstType: "SPOT",
			LotSz: 0.0001, MinSz: 0.0001, TickSz: 0.0001, TakerFee: 0.001, Px: px})
	}
	// 本格合约乐器（SWAP 常规键 / FUTURES=-YYMMDD 期货键——basis_open.go 同域）。
	itype := "SWAP"
	if datedFutRe.MatchString(inst) {
		itype = "FUTURES"
	}
	v.WithInstrument(simvenue.Instrument{InstID: inst, InstType: itype,
		LotSz: 0.001, MinSz: 0.001, TickSz: 0.01, CtVal: 0.001, CtValCcy: "USDT",
		TakerFee: 0.0005, Px: px})
	// 基键 perp（perp_quarter_spread 多腿/xvenue perp 腿共用——幂等重登无害）。
	if inst != swapID {
		v.WithInstrument(simvenue.Instrument{InstID: swapID, InstType: "SWAP",
			LotSz: 0.001, MinSz: 0.001, TickSz: 0.01, CtVal: 0.001, CtValCcy: "USDT",
			TakerFee: 0.0005, Px: px})
	}
	return v
}

// newMT5Venue 本格 MT5 CFD 仿真 venue（canonical↔venue 方言对——
// TradeMode=FullAccess(4) 产线可开性断言同值；钱域名义面 1e7 播种）。
func newMT5Venue(inst string) *mt5sim.Sim {
	canon := cfdCanonical(inst)
	px := pxOf(inst)
	return mt5sim.New().WithBalance(1e7).WithInstrument(mt5sim.Instrument{
		VenueSymbol: canon + "m", Canonical: canon,
		ContractSize: 0.001, MinLot: 0.001, LotStep: 0.001,
		TradeMode: 4, Px: px, State: "live"})
}

// defaultInst 空 InstID 格代表乐器（钦定 xvenue 空格=$50 单档——BTC
// 基准键确定性取，演习域 fixture 价位同 pxOf 派生）。
func defaultInst() string { return "BTC-USDT-SWAP" }

// dcdSim DCD 产品仿真 venue（dcdexec.Venue 三步缝桩——quote/trade/
// order-status 产线同签名；状态机 LIVE→SETTLED=到期结算径）。
type dcdSim struct {
	productID string
	notional  float64
	ordID     string
	state     string
	settled   bool
}

func newDcdSim(productID string, notional float64) *dcdSim {
	return &dcdSim{productID: productID, notional: notional,
		ordID: "SIM-ORD-" + productID}
}

func (s *dcdSim) DcdQuote(_ context.Context, productID, _, _ string) (string, error) {
	if productID != s.productID {
		return "", fmt.Errorf("dcdsim: product %s 不在册", productID)
	}
	return "SIM-QUOTE-" + productID, nil
}

func (s *dcdSim) DcdTrade(_ context.Context, quoteID string) (json.RawMessage, error) {
	if quoteID != "SIM-QUOTE-"+s.productID {
		return nil, fmt.Errorf("dcdsim: quote %s 不在册", quoteID)
	}
	s.state = "LIVE"
	return json.RawMessage(fmt.Sprintf(
		`[{"ordId":%q,"sCode":"0","sMsg":""}]`, s.ordID)), nil
}

func (s *dcdSim) DcdOrderStatus(_ context.Context, ordID string) (json.RawMessage, error) {
	if ordID != s.ordID {
		return nil, fmt.Errorf("dcdsim: ordId %s 不在册", ordID)
	}
	return json.RawMessage(fmt.Sprintf(`[{"ordId":%q,"state":%q}]`, ordID, s.state)), nil
}

// 钱径摆渡缝三桩——仿真域恒满仓（funding 余=名义额全覆盖,摆渡径
// 天然跳过;桩签名保 Venue 接口满实现）。

func (s *dcdSim) FundingAvail(_ context.Context, _ string) (float64, error) {
	return s.notional, nil
}

func (s *dcdSim) AssetTransfer(_ context.Context, _ string, _ float64, _, _ string) (string, string, error) {
	return "", "", nil
}

func (s *dcdSim) AssetTransferState(_ context.Context, _, _ string) (string, error) {
	return "success", nil
}

// Settle 到期结算（强平径——venue 侧态迁 SETTLED）。
func (s *dcdSim) Settle() { s.state = "SETTLED"; s.settled = true }

// OrderState 当前 order-status 态（hold/结算断言锚）。
func (s *dcdSim) OrderState() string { return s.state }

// OpenExposure 在敞口（未结算=名义额；结算后=0——残留对拍锚）。
func (s *dcdSim) OpenExposure() float64 {
	if s.settled {
		return 0
	}
	return s.notional
}
