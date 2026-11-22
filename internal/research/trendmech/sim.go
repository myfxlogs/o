// sim.go —— 机械趋势纸面模拟核（工单O-32604 假想卡②——手术刀=单范式
// 单参数，禁加指标/过滤层）。纯函数件：bars(1D确认棒正序)+funding 结算
// 行→信号序列+手续费/资金费账+日终净值曲线。未来函数断言=结构性：
// 信号只用 bars[..i] 收盘判（通道窗不含当棒），成交恒在次根 bars[i+1].O
// ——t 收盘信号→t+1 开盘假设成交，无未来数据可达成交价。
//
// 规则钦定（spec 单参数化）：收盘价破前 20 日高→开多 / 破前 20 日低→
// 开空（通道=max/min(H/L) 前 N 根不含当棒）；持仓中收盘破反向 10 日通道
// →次根开盘平仓；无加仓无止损叠加。perp 单腿 1x notional——qty=notional/
// 成交价名义定量。
//
// 账本=线性永续现金流模型：pos 带符qty(+多/-空)，cash 含本金+已结算现金流；
// 每笔成交 cash-=dq*px+|dq|*px*fee（开多付名义+费/开空收名义-费/平反向）；
// funding 结算 cash-=pos*markPx*rate（多头 rate>0 付/空头收——markPx 取
// 结算落在当日棒收盘近似）；equity=cash+pos*markPx。gross 平行账=同仓位
// 只扣价差不扣费/息（gross_apr 对照面）。
package trendmech

import (
	"math"
	"time"
)

const (
	// Inst 评估标的（spec 钦定 BTC-USDT-SWAP 日级）。
	Inst = "BTC-USDT-SWAP"
	// Template eval 卡 template 键=假想卡 surface 同名（trend_mech）。
	Template = "trend_mech"
	// EntryN 入场 Donchian 通道窗（日，spec 钦定 20）。
	EntryN = 20
	// ExitN 反向平仓通道窗（日，spec 钦定 10）。
	ExitN = 10
	// bar K线粒度（spec 钦定日级）。
	bar = "1D"
)

// Bar 1D 确认棒（o/h/l/c/ts——信号用 c 判突破、成交用 o、盯市用 c）。
type Bar struct {
	Ts time.Time
	O  float64
	H  float64
	L  float64
	C  float64
}

// Fund funding 结算行（ts=结算时刻,rate=结算率——多头付空头收惯例）。
type Fund struct {
	Ts   time.Time
	Rate float64
}

// Fill 一笔成交（Px=价,DQ=带符量(+买/-卖),Fee=费额,Why=因）。
type Fill struct {
	Ts  time.Time
	Px  float64
	DQ  float64
	Fee float64
	Why string
}

// Trade 一轮完整往返（entry/exit 配对——胜率/往返账面）。
type Trade struct {
	OpenTs, CloseTs time.Time
	Dir             int // +1多/-1空
	EntryPx         float64
	ExitPx          float64
	Qty             float64 // 名义定量 BTC 数（开仓量）
	Pnl             float64 // 往返净额 USDT=价差损益−双边费+持仓期资金费现金流
}

// Result 模拟全产出（净值曲线+成交+往返账+年化判词原料）。
type Result struct {
	Equity   []float64 // 日终净值（USDT），对齐 Days
	Days     []time.Time
	GrossEq  []float64 // 毛净值（不扣费息——gross 对照面）
	Fills    []Fill
	Trades   []Trade
	NetRet   float64 // 期末净值/本金-1
	GrossRet float64
	NetAPR   float64 // NetRet*365/窗日——年化判词原料（窗≤0=0）
	GrossAPR float64
	FinalEq  float64
}

func extreme(bars []Bar, i, n int, f func(Bar) float64, wantMax bool) float64 {
	m := f(bars[i-n])
	for j := i - n + 1; j < i; j++ {
		v := f(bars[j])
		if (wantMax && v > m) || (!wantMax && v < m) {
			m = v
		}
	}
	return m
}

// signal 当棒收盘判下一根开盘动作（只读 bars[..i]——当棒收盘 c[i] 参与
// 判突破但成交价永不取自当棒之后之外；通道窗=前 N 根不含当棒）。
// 返 (exit,dir)：持仓中反向通道破→exit=true dir=平仓向；空仓 20 日通道
// 破→dir=±1 入场向。
func signal(bars []Bar, i int, pos float64) (bool, int) {
	c := bars[i].C
	if pos > 0 {
		if i >= ExitN && c < extreme(bars, i, ExitN, func(b Bar) float64 { return b.L }, false) {
			return true, -1
		}
		return false, 0
	}
	if pos < 0 {
		if i >= ExitN && c > extreme(bars, i, ExitN, func(b Bar) float64 { return b.H }, true) {
			return true, 1
		}
		return false, 0
	}
	if i < EntryN {
		return false, 0
	}
	if c > extreme(bars, i, EntryN, func(b Bar) float64 { return b.H }, true) {
		return false, 1
	}
	if c < extreme(bars, i, EntryN, func(b Bar) float64 { return b.L }, false) {
		return false, -1
	}
	return false, 0
}

// Simulate 全窗重放模拟（确定性径——每拍全窗重算无状态漂移；evidence
// 载窗界供消费面对龄）。bars 须 ts 正序；feeBps=taker 费基点（双边各
// 收一次）；notional=单腿名义 USDT（1x 上限=qty*px=notional 取等）。
func Simulate(bars []Bar, funds []Fund, feeBps, notional float64) Result {
	r := Result{FinalEq: notional}
	if len(bars) < 2 || notional <= 0 {
		return r
	}
	fee := feeBps / 1e4
	var pos, cash, gCash float64 = 0, notional, notional
	var fundAcc, openFee float64 // 持仓期资金费现金流累计+开仓费（进 trade.Pnl）
	var pendExit bool
	var pendDir int
	var openTrade *Trade
	fi := 0 // funding 游标（funds 按 ts 正序）
	for i := 1; i < len(bars); i++ {
		b := bars[i]
		// ①次根开盘成交（信号属 bars[i-1] 收盘判——无未来函数）
		if pendDir != 0 || pendExit {
			var dq float64
			why := "entry_20d_break"
			if pendExit {
				dq, why = -pos, "exit_10d_reverse"
			} else {
				dq = float64(pendDir) * notional / b.O
			}
			if dq != 0 {
				f := math.Abs(dq) * b.O * fee
				cash -= dq*b.O + f
				gCash -= dq * b.O
				pos += dq
				r.Fills = append(r.Fills, Fill{Ts: b.Ts, Px: b.O, DQ: dq, Fee: f, Why: why})
				if pendExit && openTrade != nil {
					openTrade.CloseTs = b.Ts
					openTrade.ExitPx = b.O
					openTrade.Pnl = float64(openTrade.Dir)*(b.O-openTrade.EntryPx)*openTrade.Qty -
						openFee - f + fundAcc
					r.Trades = append(r.Trades, *openTrade)
					openTrade, fundAcc, openFee = nil, 0, 0
				} else if !pendExit {
					openFee = f
					openTrade = &Trade{OpenTs: b.Ts, Dir: pendDir, EntryPx: b.O, Qty: math.Abs(dq)}
				}
			}
			pendExit, pendDir = false, 0
		}
		// ②funding 结算（(bars[i-1].Ts,b.Ts] 窗内逐行——持仓才付息）
		for fi < len(funds) && !funds[fi].Ts.After(b.Ts) {
			if funds[fi].Ts.After(bars[i-1].Ts) && pos != 0 {
				fc := -pos * b.C * funds[fi].Rate
				cash += fc
				fundAcc += fc
			}
			fi++
		}
		// ③日终盯市
		r.Equity = append(r.Equity, cash+pos*b.C)
		r.GrossEq = append(r.GrossEq, gCash+pos*b.C)
		r.Days = append(r.Days, b.Ts)
		// ④当棒收盘判次根动作（通道窗不含当棒——结构性无未来函数）
		if i+1 < len(bars) {
			pendExit, pendDir = signal(bars, i, pos)
		}
	}
	// 窗尾未平仓如实记浮盈账（不强平——paper 窗读数=持仓市值口径）。
	r.NetRet = r.Equity[len(r.Equity)-1]/notional - 1
	r.GrossRet = r.GrossEq[len(r.GrossEq)-1]/notional - 1
	if span := r.Days[len(r.Days)-1].Sub(r.Days[0]).Hours() / 24; span > 0 {
		r.NetAPR = r.NetRet * 365 / span
		r.GrossAPR = r.GrossRet * 365 / span
	}
	r.FinalEq = r.Equity[len(r.Equity)-1]
	return r
}
