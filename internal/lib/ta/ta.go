// Package ta —— 技术指标壳（T-0310 阶段二特征层备料——ark internal/service/ta
// vendored 吸收关键件：RSI/MACD/BB/ATR 纯函数零状态；出处 Apache-2.0）。
package ta

// SMA 简单均线。
func SMA(xs []float64, n int) float64 {
	if len(xs) < n || n <= 0 {
		return 0
	}
	s := 0.0
	for _, v := range xs[len(xs)-n:] {
		s += v
	}
	return s / float64(n)
}

// EMA 指数均线（Wilder 平滑）。
func EMA(xs []float64, n int) float64 {
	if len(xs) < n || n <= 0 {
		return 0
	}
	k := 2.0 / float64(n+1)
	e := SMA(xs[:n], n)
	for _, v := range xs[n:] {
		e = v*k + e*(1-k)
	}
	return e
}

// RSI 相对强弱（14 期 Wilder——末值）。
func RSI(closes []float64, n int) float64 {
	if len(closes) <= n {
		return 50
	}
	var ag, al float64
	for i := 1; i <= n; i++ {
		d := closes[len(closes)-n+i-1] - closes[len(closes)-n+i-2]
		if d > 0 {
			ag += d
		} else {
			al -= d
		}
	}
	ag, al = ag/float64(n), al/float64(n)
	for i := len(closes) - n + 1; i < len(closes); i++ {
		d := closes[i] - closes[i-1]
		if d > 0 {
			ag = (ag*float64(n-1) + d) / float64(n)
		} else {
			al = (al*float64(n-1) - d) / float64(n)
		}
	}
	if al == 0 {
		return 100
	}
	return 100 - 100/(1+ag/al)
}

// MACD (12,26,9)——返回 dif/dea/hist。
func MACD(closes []float64) (dif, dea, hist float64) {
	if len(closes) < 35 {
		return 0, 0, 0
	}
	// 全序列 EMA 差→信号线（末值口径——备料非交易信号）。
	var difs []float64
	for i := 26; i <= len(closes); i++ {
		difs = append(difs, EMA(closes[:i], 12)-EMA(closes[:i], 26))
	}
	dif = difs[len(difs)-1]
	dea = EMA(difs, 9)
	return dif, dea, dif - dea
}

// Bollinger 布林带（20,2）——mid/up/lo。
func Bollinger(closes []float64, n int, k float64) (mid, up, lo float64) {
	if len(closes) < n || n <= 0 {
		return 0, 0, 0
	}
	mid = SMA(closes, n)
	var ss float64
	for _, v := range closes[len(closes)-n:] {
		ss += (v - mid) * (v - mid)
	}
	sd := (ss / float64(n))
	if sd > 0 {
		sd = sqrt(sd)
	}
	return mid, mid + k*sd, mid - k*sd
}

// ATR 真实波幅（14 期 Wilder——末值）。
func ATR(high, low, close []float64, n int) float64 {
	if len(high) <= n || len(high) != len(low) || len(low) != len(close) {
		return 0
	}
	var tr float64
	for i := 1; i <= n; i++ {
		tr += trange(high[i], low[i], close[i-1])
	}
	atr := tr / float64(n)
	for i := n + 1; i < len(high); i++ {
		atr = (atr*float64(n-1) + trange(high[i], low[i], close[i-1])) / float64(n)
	}
	return atr
}

func trange(h, l, pc float64) float64 {
	a := h - l
	if b := abs(h - pc); b > a {
		a = b
	}
	if c := abs(l - pc); c > a {
		a = c
	}
	return a
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 50; i++ {
		z = (z + x/z) / 2
	}
	return z
}
