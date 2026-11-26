// simgrid_engine.go —— simgrid 网格引擎内件（O-41370 拆件——棘轮 300
// 行界）：档位价网格+买低卖高对称挂单+成交翻转补位（买成→上档补卖/
// 卖成→下档补买，出界不补=网格边缘耗尽诚实语义）。
package okxgrid

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

type simOrder struct {
	OrdID  string  `json:"ordId"`
	Side   string  `json:"side"` // buy|sell
	Px     float64 `json:"px"`
	Sz     float64 `json:"sz"`
	State  string  `json:"state"` // live|filled|cancelled
	AvgPx  float64 `json:"avgPx"`
	AccSz  float64 `json:"accFillSz"`
	Fee    float64 `json:"fee"`
	FeeCcy string  `json:"feeCcy"`
	Pnl    float64 `json:"pnl"`
	CTime  string  `json:"cTime"`
}

type simBot struct {
	algoID, instID, state, runType, tag, clOrdID string
	maxPx, minPx, runPx, quoteSz                 float64
	gridNum                                      int
	orders                                       []*simOrder
	tradeNum, arbNum                             int
	gridProfit, baseHeld, quoteSpent             float64
	avgBuyPx                                     float64
	buyQty                                       float64
	ct                                           time.Time
}

// levels 算术网格档位价（gridNum 档→gridNum+1 价点——OKX 算术型同形）。
func (b *simBot) levels() []float64 {
	out := make([]float64, b.gridNum+1)
	step := (b.maxPx - b.minPx) / float64(b.gridNum)
	for i := range out {
		out[i] = b.minPx + float64(i)*step
	}
	return out
}

// placeInitial 首挂：runPx 下方买单/上方卖单（经典网格初始态）。
func (b *simBot) placeInitial() {
	lv := b.levels()
	perQuote := b.quoteSz / float64(b.gridNum)
	for i, p := range lv {
		if p >= b.runPx {
			continue
		}
		sz := perQuote / p
		b.orders = append(b.orders, &simOrder{OrdID: fmt.Sprintf("%s-L%d", b.algoID, i),
			Side: "buy", Px: p, Sz: sz, State: "live", FeeCcy: "USDT", CTime: b.ct.Format(time.RFC3339)})
	}
	for i, p := range lv {
		if p <= b.runPx {
			continue
		}
		sz := perQuote / p
		b.orders = append(b.orders, &simOrder{OrdID: fmt.Sprintf("%s-L%d", b.algoID, i),
			Side: "sell", Px: p, Sz: sz, State: "live", FeeCcy: "USDT", CTime: b.ct.Format(time.RFC3339)})
	}
}

// tickLocked 喂价结算：买单 px>=价=跌破成交、卖单 px<=价=涨破成交；
// 成交后翻转补位（买成→上移一档补卖；卖成→下移一档补买——出界不补）。
func (b *simBot) tickLocked(px float64) {
	lv := b.levels()
	for _, o := range b.orders {
		if o.State != "live" {
			continue
		}
		if o.Side == "buy" && px <= o.Px || o.Side == "sell" && px >= o.Px {
			o.State, o.AvgPx, o.AccSz = "filled", o.Px, o.Sz
			b.tradeNum++
			b.arbNum++
			o.Fee = o.Px * o.Sz * 0.001
			if o.Side == "buy" {
				b.baseHeld += o.Sz
				b.quoteSpent += o.Px * o.Sz
				if b.buyQty > 0 {
					b.avgBuyPx = (b.avgBuyPx*b.buyQty + o.Px*o.Sz) / (b.buyQty + o.Sz)
				} else {
					b.avgBuyPx = o.Px
				}
				b.buyQty += o.Sz
			} else {
				// 卖出配对最早买入成本→已实现利（FIFO 近似——sim 域诚实下界）。
				match := math.Min(o.Sz, b.buyQty)
				if match > 0 {
					b.gridProfit += (o.Px - b.avgBuyPx) * match
					b.buyQty -= match
					if b.buyQty <= 1e-12 {
						b.buyQty, b.avgBuyPx = 0, 0
					}
				}
				b.baseHeld -= o.Sz
			}
			b.requoteLocked(o, lv)
		}
	}
	sort.SliceStable(b.orders, func(i, j int) bool { return b.orders[i].Px < b.orders[j].Px })
}

// requoteLocked 成交翻转补位（档位序号定位→买成上移/卖成下移一档）。
func (b *simBot) requoteLocked(o *simOrder, lv []float64) {
	idx := -1
	for i, p := range lv {
		if math.Abs(p-o.Px) < 1e-12 {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	ni := idx + 1 // 买成→上档补卖；卖成→下档补买（ni 由 side 定）
	if o.Side == "sell" {
		ni = idx - 1
	}
	if ni < 0 || ni >= len(lv) {
		return // 出界不补=网格边缘耗尽（诚实语义——非隐藏失败）
	}
	p := lv[ni]
	side := "sell"
	if o.Side == "sell" {
		side = "buy"
	}
	sz := (b.quoteSz / float64(b.gridNum)) / p
	b.orders = append(b.orders, &simOrder{OrdID: fmt.Sprintf("%s-L%d-%d", b.algoID, ni, b.arbNum),
		Side: side, Px: p, Sz: sz, State: "live", FeeCcy: "USDT", CTime: time.Now().Format(time.RFC3339)})
}

// detailsJSON Details 行（okxgrid.Details json 词汇同键——消费面产线同解）。
func (b *simBot) detailsJSON() map[string]any {
	floatPnl := 0.0
	if b.buyQty > 0 {
		mark := b.runPx
		if mark <= 0 {
			mark = (b.minPx + b.maxPx) / 2
		}
		floatPnl = (mark - b.avgBuyPx) * b.buyQty
	}
	st := b.state
	if st == "" {
		st = "running"
	}
	return map[string]any{
		"algoId": b.algoID, "algoClOrdId": b.clOrdID, "instId": b.instID,
		"instType": "SPOT", "state": st, "stopResult": "0",
		"investment": numStr(b.quoteSz), "gridProfit": numStr(b.gridProfit),
		"floatProfit": numStr(floatPnl), "totalPnl": numStr(b.gridProfit + floatPnl),
		"pnlRatio": "0", "arbitrageNum": strconv.Itoa(b.arbNum),
		"tradeNum": strconv.Itoa(b.tradeNum), "annualizedRate": "0",
		"totalAnnualizedRate": "0", "perMaxProfitRate": "0", "perMinProfitRate": "0",
		"runPx": numStr(b.runPx), "gridNum": strconv.Itoa(b.gridNum),
		"maxPx": numStr(b.maxPx), "minPx": numStr(b.minPx),
		"quoteSz": numStr(b.quoteSz), "baseSz": "0",
		"curBaseSz": numStr(b.baseHeld), "curQuoteSz": numStr(b.quoteSz - b.quoteSpent),
		"singleAmt": numStr(b.quoteSz / float64(b.gridNum)), "fee": "0", "feeCcy": "USDT",
		"fundingFee": "0", "cTime": b.ct.Format("2006-01-02T15:04:05.000Z"),
		"uTime": time.Now().Format("2006-01-02T15:04:05.000Z"),
	}
}

func numStr(f float64) string { return strconv.FormatFloat(f, 'f', 8, 64) }
