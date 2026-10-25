// eval.go —— 双币赢评估+阈闸（T-0102 钦定初版，判据全 config 热启）：
// dist→term→notional→单 uly 方向不叠→apr 顺闸，拒因=首触（evidence.reason）；
// 判据缺位=unevaluable fail-closed（禁虚构判据）。
package dcd

import (
	"math"
	"time"
)

// Params 阈闸判据（票面初版数值种 0043——热启可调）。
type Params struct {
	MinDistPct  float64 // strike 距现价 ≥N%
	MaxTermDays float64 // 期限 ≤N 天
	MinAPR      float64 // 年化 ≥N
	NotionalPct float64 // 名义 ≤可用币种余量×N%
}

// Eval 单品评估结论。
type Eval struct {
	Product   Product
	Template  string // dcd_call / dcd_put
	Spot      float64
	TermDays  float64
	DistPct   float64
	APR       float64
	Avail     float64 // 锁仓侧币种可动余量（C=基础币、P=USDT）
	Verdict   string  // proposed / rejected / unevaluable
	Reason    string  // 拒因/不可评估因（evidence.reason）
	lockedCcy string  // 锁仓侧币种
}

// lockedCcyOf 锁仓侧币种（C=锁基础币卖 call；P=锁 USDT 卖 put）。
func lockedCcyOf(p Product) string {
	if p.Direction == "C" {
		return firstSeg(p.Uly)
	}
	return "USDT"
}

func firstSeg(uly string) string {
	for i := 0; i < len(uly); i++ {
		if uly[i] == '-' {
			return uly[:i]
		}
	}
	return uly
}

// Evaluate 单品评估（spot=现价 md_ticks 同源；avail=锁仓侧可动余量，
// <0=判据缺位——unevaluable fail-closed）。
func Evaluate(p Product, spot, avail float64, params Params) Eval {
	e := Eval{Product: p, Spot: spot, APR: p.AnnualizedYield,
		Avail: avail, lockedCcy: lockedCcyOf(p)}
	e.Template = "dcd_put" // venue 方言 C/P→模板名 call/put
	if p.Direction == "C" {
		e.Template = "dcd_call"
	}
	if p.Expiry.IsZero() {
		e.Verdict, e.Reason = "unevaluable", "expiry_missing"
		return e
	}
	e.TermDays = math.Ceil(time.Until(p.Expiry).Hours() / 24)
	if spot <= 0 {
		e.Verdict, e.Reason = "unevaluable", "spot_missing"
		return e
	}
	if p.Strike <= 0 {
		e.Verdict, e.Reason = "unevaluable", "strike_missing"
		return e
	}
	e.DistPct = math.Abs(p.Strike-spot) / spot * 100
	// 闸序=票面钦定顺序，拒因取首触。
	if e.DistPct < params.MinDistPct {
		e.Verdict, e.Reason = "rejected", "dist_thin"
		return e
	}
	if e.TermDays > params.MaxTermDays {
		e.Verdict, e.Reason = "rejected", "term_over"
		return e
	}
	if avail < 0 {
		e.Verdict, e.Reason = "unevaluable", "coins_absent"
		return e
	}
	if avail*params.NotionalPct/100 < p.MinSize {
		e.Verdict, e.Reason = "rejected", "insufficient_balance"
		return e
	}
	if e.APR < params.MinAPR {
		e.Verdict, e.Reason = "rejected", "apr_thin"
		return e
	}
	e.Verdict = "proposed"
	return e
}

// Evidence 证据体（评估卡 jsonb——溯源锚：productId/strike/term/dist_pct/spot）。
func (e Eval) Evidence() map[string]any {
	return map[string]any{
		"productId": e.Product.ProductID, "uly": e.Product.Uly,
		"strike": e.Product.Strike, "term": e.TermDays,
		"dist_pct": e.DistPct, "spot": e.Spot, "apr": e.APR,
		"locked_ccy": e.lockedCcy, "avail": e.Avail,
		"min_size": e.Product.MinSize, "reason": e.Reason,
	}
}

// Legs 模板语义标注（legs jsonb——非腿对冲族：锁仓卖权 carry）。
func (e Eval) Legs() map[string]any {
	if e.Product.Direction == "C" {
		return map[string]any{"locked": e.lockedCcy, "sell": "call", "settle": "expiry"}
	}
	return map[string]any{"locked": "USDT", "sell": "put", "settle": "expiry"}
}
