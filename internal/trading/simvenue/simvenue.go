// Package simvenue —— 合成 venue 仿真层（T-0426 执行径演习——venue 语义纸面跑）。
//
// 只替 venue IO 缝：实现 execution.Executor（下单面）+ guard.VenueSizer 三方法
// （结构化隐式满足——不 import guard，方向闸零边）；消费件（flow 腿落库/guard
// drift 判定/sizing 平仓计划/bag 残袋对拍/settle 账实对平）一律产线同码——禁影子实现。
//
// 方言词汇源（禁拍脑袋编）：
//   - sz_unit(base/quote/contract)/accFillSz=qty_base/腿装配语义 = fin_0047+
//     fin_order_leg 实约（flow.legPx 同款量纲）；
//   - Instrument 字段 = /public/instruments 实约（okxexec.InstrumentMeta 同词汇：
//     ctVal/lotSz/minSz/tickSz/state——skills/okx-api-spec 卡）；
//   - 拒码 = 51xxx 业务拒族：51008/51001 仓内实证（okxexec/order.go+skills/
//     okx-exec 卡），51006/51007/51020 = OKX docs-v5 错误码总表域；仿真域冻结。
//
// 拒单槽序（业主令钦定冻结——首命中即拒，确定性顺序写死进测试）：
//
//	① lotSz/minSz/max 网格 → ② 价格带（挂价腿） → ③ not_marketable → ④ insufficient_balance
package simvenue

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"o/internal/trading/execution"
)

// RulesVersion 引擎指纹（RULES_VERSION 族）：演习凭据/拒单审计携本值——
// 「哪套规则判的」可验；规则语义变更必翻版（旧凭据可审计为旧规）。
const RulesVersion = "simvenue/1"

// 拒因码（venue 方言回执——VenueCode 透传 fin_order_leg.venue_err_code，禁吞）。
const (
	RejBelowMin  = "51006" // 量低于 minSz
	RejAboveMax  = "51007" // 量高于 max（量上限）
	RejOffGrid   = "51005" // 量不落 lotSz 网格
	RejPxBand    = "51020" // 委托价出价格带（挂价腿）
	RejNotMkt    = "51001" // not_marketable——乐器不在册/state≠live（终局族实证码）
	RejInsuffBal = "51008" // 可用余额不足（真户实锤码——okxexec/order.go 实测）
)

type order struct {
	leg     execution.Leg
	clOrdID string
	state   string // filled|partially_filled|cancelled
	fillSz  float64
	px      float64
	plan    Plan
	rounds  int
	totalSz float64 // 请求总量（base 币量/张口径）
}

// SimVenue 合成 venue（纸面钱域——余额/持仓/订单态机/成交流水全内持；
// SWAP 保证金面不模：开仓保证金无限纸面钱，减仓超净持=拒——反向新开防同 T-0043 语义）。
type SimVenue struct {
	mu     sync.Mutex
	instr  map[string]Instrument
	bal    map[string]float64 // ccy→可用余额（现货域）
	pos    map[string]float64 // instID→净持仓张数（合约域，带号）
	orders map[string]*order
	done   map[string]float64 // clOrdID→已落账累计（补成拍增量记账锚）
	plans  map[string]Plan    // instID→成交计划
	fills  []Fill
	pxBand float64 // 价格带宽度（比例——0.1=±10%）
}

// New 合成 venue 构造。
func New() *SimVenue {
	return &SimVenue{
		instr:  map[string]Instrument{},
		bal:    map[string]float64{},
		pos:    map[string]float64{},
		orders: map[string]*order{},
		done:   map[string]float64{},
		plans:  map[string]Plan{},
		pxBand: 0.1,
	}
}

// WithInstrument 乐器入册（链式）。
func (v *SimVenue) WithInstrument(i Instrument) *SimVenue {
	v.mu.Lock()
	defer v.mu.Unlock()
	if i.State == "" {
		i.State = "live"
	}
	v.instr[i.InstID] = i
	return v
}

// WithBalance 现货余额播种（纸面钱）。
func (v *SimVenue) WithBalance(ccy string, amt float64) *SimVenue {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.bal[ccy] = amt
	return v
}

// WithPlan 成交计划注入（per 乐器——演习部分成交/态机路径用）。
func (v *SimVenue) WithPlan(instID string, p Plan) *SimVenue {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.plans[instID] = p
	return v
}

// Place 下单面（execution.Executor 缝）——拒单槽序首命中即拒（业务拒=absent+码，
// OkxExecutor sCode≠0 同语义）；成交即落真相（bal/pos/fills）+态机迁位。
func (v *SimVenue) Place(_ context.Context, leg execution.Leg, clOrdID string) (execution.OrderResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	res := execution.OrderResult{ClOrdID: clOrdID, RulesVersion: RulesVersion}
	in, ok := v.instr[leg.InstID]
	szF, _ := strconv.ParseFloat(leg.Sz, 64)
	// ① lotSz/minSz/max 网格（乐器在册才可判——量纲随 sz_unit 消歧）。
	if ok {
		if code, msg := vetGrid(in, leg, szF); code != "" {
			res.State, res.VenueCode, res.VenueMsg = "absent", code, msg
			return res, nil
		}
		// ② 价格带（挂价腿——市价单无价跳槽）。
		if leg.Maker && leg.LimitPx > 0 && math.Abs(leg.LimitPx-in.Px)/in.Px > v.pxBand {
			res.State, res.VenueCode, res.VenueMsg = "absent", RejPxBand,
				fmt.Sprintf("挂价 %s 出价格带 ±%.0f%%（ref=%.8g）", fmtPx(leg.LimitPx), v.pxBand*100, in.Px)
			return res, nil
		}
	}
	// ③ not_marketable（不在册/state≠live——单未建终局族）。
	if !ok || in.State != "live" {
		st := "不在册"
		if ok {
			st = "state=" + in.State
		}
		res.State, res.VenueCode, res.VenueMsg = "absent", RejNotMkt,
			fmt.Sprintf("Instrument %s not Marketable（%s）", leg.InstID, st)
		return res, nil
	}
	// ④ insufficient_balance（现货域余额判；合约=减仓超净持判）。
	unit, spend, contracts := fillMath(in, leg, szF)
	if err := v.afford(in, leg, contracts, spend, unit); err != nil {
		res.State, res.VenueCode, res.VenueMsg = "absent", RejInsuffBal, err.Error()
		return res, nil
	}
	plan := v.plans[leg.InstID]
	if plan.OnlyClose && !leg.Close {
		plan = Plan{} // 开腿不受 OnlyClose 计划——开平同乐器异生命周期位
	}
	o := &order{leg: leg, clOrdID: clOrdID, totalSz: unit, plan: plan, px: in.Px}
	if r := o.plan.FillRatio; r > 0 && r < 1 {
		o.fillSz = math.Min(o.totalSz, math.Max(o.plan.MinFillBase, o.totalSz*r))
		o.state = "partially_filled"
	} else {
		o.fillSz = o.totalSz
		o.state = "filled"
	}
	v.orders[clOrdID] = o
	v.apply(o, in)
	res.State, res.FillPx, res.FillSz = o.state, in.Px, o.fillSz
	return res, nil
}

// Resolve 定态面（execution.Executor 缝）——部分成交补成拍（plan.ResolveCap
// 拍数耗尽=余量一次补完，态机 partial→filled）；不在册单=absent（51603 幂等键同语义）。
func (v *SimVenue) Resolve(_ context.Context, _ execution.Leg, clOrdID string) (execution.OrderResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	o, ok := v.orders[clOrdID]
	if !ok {
		return execution.OrderResult{ClOrdID: clOrdID, State: "absent", RulesVersion: RulesVersion}, nil
	}
	if o.state == "partially_filled" && o.plan.ResolveCap > 0 {
		o.rounds++
		if o.rounds >= o.plan.ResolveCap {
			o.fillSz = o.totalSz // 补成拍耗尽——余量一次补完
		} else {
			o.fillSz = math.Min(o.totalSz, o.fillSz+math.Max(o.plan.MinFillBase, o.totalSz*o.plan.FillRatio))
		}
		if o.fillSz >= o.totalSz-1e-12 {
			o.fillSz = o.totalSz
			o.state = "filled"
		}
		v.apply(o, v.instr[o.leg.InstID])
	}
	return execution.OrderResult{ClOrdID: clOrdID, State: o.state, FillPx: o.px, FillSz: o.fillSz, RulesVersion: RulesVersion}, nil
}

// Cancel 撤挂面（execution.Executor 缝）——挂单态撤成；已结单=错（降级径
// 不问撤果，Resolve 定真态——产线同语义）。
func (v *SimVenue) Cancel(_ context.Context, _ execution.Leg, clOrdID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	o, ok := v.orders[clOrdID]
	if !ok {
		return nil
	}
	if o.state == "filled" {
		return fmt.Errorf("simvenue: 单已成交不可撤 %s", clOrdID)
	}
	o.state = "cancelled"
	return nil
}

// apply 成交落真相（调用方持锁）——增量记账：delta=本次拍新成量。
func (v *SimVenue) apply(o *order, in Instrument) {
	delta := o.fillSz - v.done[o.clOrdID]
	if delta <= 0 {
		return
	}
	v.done[o.clOrdID] = o.fillSz
	leg := o.leg
	switch {
	case in.InstType == "SWAP" || in.InstType == "FUTURES":
		sign := 1.0
		if leg.Side == "sell" {
			sign = -1.0
		}
		if leg.Close {
			// reduceOnly 语义（T-0043 同款）：平仓腿只向零减——同向持仓上
			// 反向发平单=venue 拒（afford 槽④拦），部分成也只减不反向加。
			if cur := v.pos[leg.InstID]; cur != 0 {
				sign = -1.0
				if cur < 0 {
					sign = 1.0
				}
			}
		}
		v.pos[leg.InstID] += sign * delta
		fee := delta * in.CtVal * o.px * in.TakerFee // 计价币扣（linear bills 同构）
		v.bal[quoteCcy(in)] -= fee
		v.fills = append(v.fills, Fill{ClOrdID: o.clOrdID, InstID: leg.InstID, Side: leg.Side,
			Contracts: delta, QuoteAmt: -fee, Fee: fee, FeeCcy: quoteCcy(in), Px: o.px, Ts: time.Now()})
	case leg.Side == "buy":
		// 费基准币内扣（canon cex-close-sz-fee-in-base）：实得 delta=毛量×(1−费)。
		var spend float64
		if leg.TgtQuote {
			spend = delta * o.px // 委托额含费——实扣=成交额（T-0116 实证口径）
		} else {
			spend = delta / (1 - in.TakerFee) * o.px // 毛额流出（费在基准币内扣）
		}
		ccy := baseCcy(in)
		v.bal[ccy] += delta
		v.bal[quoteCcy(in)] -= spend
		v.fills = append(v.fills, Fill{ClOrdID: o.clOrdID, InstID: leg.InstID, Side: leg.Side,
			QtyBase: delta, QuoteAmt: -spend, Fee: spend - delta*o.px, FeeCcy: baseCcy(in),
			Px: o.px, Ts: time.Now()})
	default:
		// 卖：费计价币内扣——实收=毛额×(1−费)。
		gross := delta * o.px
		fee := gross * in.TakerFee
		v.bal[baseCcy(in)] -= delta
		v.bal[quoteCcy(in)] += gross - fee
		v.fills = append(v.fills, Fill{ClOrdID: o.clOrdID, InstID: leg.InstID, Side: leg.Side,
			QtyBase: delta, QuoteAmt: gross - fee, Fee: fee, FeeCcy: quoteCcy(in),
			Px: o.px, Ts: time.Now()})
	}
}
