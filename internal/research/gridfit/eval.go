// eval.go —— 网格仿真核（纯函数件：OHLC 回放→realized/unrealized/判词；
// run.go 持拍面，persist.go 持落表面——工单#25737 SSOT③④钦定判据落地式）。
package gridfit

import "time"

// bar 单根确认棒（仿真消费 h/l/c——o 不入判据径）。
type kline struct{ h, l, c float64 }

// lineState 买线档位（每线一仓——filled 持仓中；卖出落袋后该线可再买，
// 经典网格「一档一仓」语义）。
type lineState struct {
	px     float64 // 买线价 c0*(1-step)^k
	filled bool
	buyPx  float64
	qty    float64
	sellPx float64
}

// card 一 inst 一拍评估中间件（仿真证据+判词载荷）。
type card struct {
	inst, verdict, reason string
	c0, bandLo, bandHi    float64
	lastClose             float64
	realized, unrealized  float64
	fees, capital         float64
	grossAPR, netAPR      float64
	nLines, nBuys, nSells int
	bars                  int
	lastTs                time.Time
}

// evalGrid 单 inst 网格回放+判词（纯函数——bars=末 lookback+1 根确认棒
// 正序；bars[0].c=c0 锚定带上下沿，bars[1:]=仿真窗=lookback 棒，与
// net_apr 年化分母 365/lookback 自洽）。
// 买线 line_k=c0*(1-step)^k≥band_lo（k=1,2,…——锚收盘下方档，k=0=锚上
// 非买档；params 下界 band_pct>step_pct 保 ≥1 线）。逐棒先卖后买
// （SSOT③序）：卖=持仓∧bar.high≥sell_px→realized+=qty*(sell-buy) 毛利、
// 费记 Σfee；买=未满线∧bar.low≤line→qty=line_notional/line、
// sell_px=line*(1+step)。末持仓按末棒收盘 MTM 入 unrealized。
// 判词 proposed⟺net_apr≥min_net_apr∧last≥band_lo*(1-blown_tol)（SSOT④）；
// 拒因序=band_blown→net_thin（跌穿带=结构死因先于薄利——thin_bars 在
// scan 层 skip 不入卡）。
func evalGrid(inst string, bars []kline, lastTs time.Time, p Params) card {
	n := len(bars)
	c0 := bars[0].c
	c := card{inst: inst, verdict: "rejected", reason: "net_thin",
		c0: c0, bandLo: c0 * (1 - p.BandPct), bandHi: c0 * (1 + p.BandPct),
		bars: n, lastTs: lastTs}
	var lines []lineState
	for px := c0 * (1 - p.StepPct); px >= c.bandLo; px *= (1 - p.StepPct) {
		lines = append(lines, lineState{px: px})
	}
	c.nLines = len(lines)
	c.capital = float64(len(lines)) * p.LineNotional
	// 逐棒仿真（bars[1:]——首棒只产 c0 锚不入场）。
	for _, b := range bars[1:] {
		for i := range lines { // 先卖：持仓档 high≥sell_px 落袋
			l := &lines[i]
			if l.filled && b.h >= l.sellPx {
				c.realized += l.qty * (l.sellPx - l.buyPx)
				c.fees += l.qty * l.sellPx * p.FeeRT
				l.filled = false
				c.nSells++
			}
		}
		for i := range lines { // 后买：未满档 low≤line 补仓
			l := &lines[i]
			if !l.filled && b.l <= l.px {
				l.filled, l.buyPx = true, l.px
				l.qty = p.LineNotional / l.px
				l.sellPx = l.px * (1 + p.StepPct)
				c.fees += p.LineNotional * p.FeeRT // 买侧名义费（qty*px=名义恒等）
				c.nBuys++
			}
		}
	}
	c.lastClose = bars[n-1].c
	for i := range lines { // 末持仓按末棒收盘 MTM
		if lines[i].filled {
			c.unrealized += lines[i].qty * (c.lastClose - lines[i].buyPx)
		}
	}
	ann := 365 / float64(p.Lookback)
	c.grossAPR = (c.realized + c.unrealized) / c.capital * ann
	c.netAPR = (c.realized + c.unrealized - c.fees) / c.capital * ann
	switch {
	case c.lastClose < c.bandLo*(1-p.BlownTol):
		c.reason = "band_blown"
	case !(c.netAPR >= p.MinNetAPR):
		c.reason = "net_thin"
	default:
		c.verdict, c.reason = "proposed", ""
	}
	return c
}
