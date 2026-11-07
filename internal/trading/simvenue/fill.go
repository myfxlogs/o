// fill.go —— venue 量纲语义面（T-0426 自 simvenue.go 拆件——行数棘轮）：
// 乐器面/成交流水/成交计划/量纲消歧/槽④余额判/槽①网格判/币种推导。
package simvenue

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"o/internal/trading/execution"
)

// Instrument 乐器面（/public/instruments 方言——字段与 okxexec.InstrumentMeta
// 同词汇；Px=仿真恒定成交价——行情面由 fixture 供 md_ticks，与 venue 成交价
// 同锚防假 drift）。远期品种面=T-0427 品种主档供（接口留位不预建）。
type Instrument struct {
	InstID   string
	InstType string  // SPOT|SWAP|FUTURES
	LotSz    float64 // 数量步长（SPOT=base 币量/SWAP=张）
	MinSz    float64 // 最小量
	MaxSz    float64 // 量上限（0=不限）
	TickSz   float64 // 价步长
	CtVal    float64 // SWAP 张面值（base 币量/张）
	CtValCcy string  // 面值币种
	TakerFee float64 // taker 费率（基准币内扣——canon cex-close-sz-fee-in-base）
	Px       float64 // 仿真成交价
	State    string  // live|suspend（instruments state 方言）
}

// Fill 成交流水（venue 内部真相——账实对平真源；QuoteAmt 负=现金流出）。
type Fill struct {
	ClOrdID   string
	InstID    string
	Side      string
	QtyBase   float64 // spot 实得/实付币量
	Contracts float64 // SWAP 张数
	QuoteAmt  float64 // 现金流（计价币）
	Fee       float64 // 费
	FeeCcy    string  // 费币种（spot 买=基准币内扣/卖+合约=计价币——bills 方言同构）
	Px        float64
	Ts        time.Time
}

// Plan 成交计划（演习可编程位——部分成交/订单生命周期控制；零值=市价即满成）。
type Plan struct {
	FillRatio   float64 // 每拍成交占比（0<r<1=部分成；0/1=即满成）
	ResolveCap  int     // 补成拍数上限（Resolve 拍数耗尽=余量一次补完；0=冻结部分态）
	MinFillBase float64 // 首拍最小成量（base 币量/张——防占比过小成尘）
	OnlyClose   bool    // 只作用于平仓腿（leg.Close）——开平同乐器异计划位
}

// fillMath 请求量→（venue 单位成量, 现金支出, 张数）量纲消歧（flow.legPx 同款
// sz_unit 语义：SWAP=contract/TgtQuote=quote/余=base；venue 单位=accFillSz
// 方言——SWAP 张/spot base 币量，OrderResult.FillSz 同口径）。
func fillMath(in Instrument, leg execution.Leg, szF float64) (unit, spend, contracts float64) {
	switch {
	case in.InstType == "SWAP" || in.InstType == "FUTURES":
		return szF, 0, szF
	case leg.Side == "buy" && leg.TgtQuote:
		// 委托额含费——实扣=成交额 qtyBase×px<委托额（T-0116 实证口径：差额
		// 留交易户——legGross 成交额锚与此同源，账实可对平）。
		qb := szF * (1 - in.TakerFee) / in.Px
		return qb, qb * in.Px, 0
	case leg.Side == "buy":
		return szF * (1 - in.TakerFee), szF * in.Px, 0
	default:
		return szF, 0, 0
	}
}

// afford 槽④余额判（纯判不动账）：现货买=计价币余额≥支出；现货卖=基准币
// 余额≥卖量；合约减仓=净持≥减量（反向新开防——Close 语义同款）。
func (v *SimVenue) afford(in Instrument, leg execution.Leg, contracts, spend, unit float64) error {
	switch {
	case in.InstType == "SWAP" || in.InstType == "FUTURES":
		if leg.Close {
			// reduceOnly 语义（T-0043 同款）：零仓减仓=无仓可平、超持减仓=
			// 反向新开——皆拒（51008 族，拒单槽序④）。
			if abs := math.Abs(v.pos[leg.InstID]); v.pos[leg.InstID] == 0 || abs < contracts {
				return fmt.Errorf("可用余额不足——净持 %.8g 张<减仓 %.8g 张（零仓/反向新开防）", abs, contracts)
			}
		}
		return nil
	case leg.Side == "buy":
		if v.bal[quoteCcy(in)] < spend {
			return fmt.Errorf("可用余额不足——%s 需 %.8g 实余 %.8g", quoteCcy(in), spend, v.bal[quoteCcy(in)])
		}
		return nil
	default:
		if v.bal[baseCcy(in)] < unit {
			return fmt.Errorf("可用余额不足——%s 需 %.8g 实余 %.8g", baseCcy(in), unit, v.bal[baseCcy(in)])
		}
		return nil
	}
}

// vetGrid 槽①量网格判（纯函数——确定性钉死进测试）：低于 minSz/超 max/不落
// lotSz 网格各携码；quote 委托=base 等当量判（OKX 市价买额域同构）。
func vetGrid(in Instrument, leg execution.Leg, szF float64) (code, msg string) {
	q := szF
	if leg.Side == "buy" && leg.TgtQuote {
		q = szF / in.Px // 额→base 等当量（minSz 域为 base 口径）
	}
	if in.MinSz > 0 && q < in.MinSz {
		return RejBelowMin, fmt.Sprintf("量 %.8g 低于 minSz %.8g", q, in.MinSz)
	}
	if in.MaxSz > 0 && q > in.MaxSz {
		return RejAboveMax, fmt.Sprintf("量 %.8g 超 max %.8g（量上限）", q, in.MaxSz)
	}
	if in.LotSz > 0 {
		if r := q / in.LotSz; math.Abs(r-math.Round(r)) > 1e-6 {
			return RejOffGrid, fmt.Sprintf("量 %.8g 不落 lotSz %.8g 网格", q, in.LotSz)
		}
	}
	return "", ""
}

func baseCcy(in Instrument) string {
	if in.InstType == "SWAP" || in.InstType == "FUTURES" {
		return in.CtValCcy
	}
	return strings.SplitN(in.InstID, "-", 2)[0]
}

func quoteCcy(in Instrument) string {
	if in.InstType == "SWAP" || in.InstType == "FUTURES" {
		return "USDT"
	}
	if p := strings.SplitN(in.InstID, "-", 3); len(p) >= 2 {
		return p[1]
	}
	return "USDT"
}

func fmtPx(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
