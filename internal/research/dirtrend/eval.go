// eval.go —— 评估数学核（纯函数件：信号+判词+证据载荷；run.go 持拍面，
// persist.go 持落表面——369 行硬阻断拆分，语义零动）。
package dirtrend

import (
	"math"
	"time"
)

const wilsonZ = 1.96 // Wilson 95% 下界 z

// card 一 inst 一拍评估中间件（信号+判词+证据载荷）。
type card struct {
	inst, sig, opp, verdict, reason string
	n, wins                         int
	phat, lb                        float64
	sigmaD, sigmaAnn, eD            float64
	grossAPR, netAPR                float64
	bars                            int
	lastTs                          time.Time
}

// channel Donchian 通道=max/min（win 非空——调用侧结构保证）。
func channel(win []float64) (hi, lo float64) {
	hi, lo = win[0], win[0]
	for _, v := range win[1:] {
		if v > hi {
			hi = v
		}
		if v < lo {
			lo = v
		}
	}
	return hi, lo
}

// wilsonLB Wilson score 下界（保守单调——小样本不虚高；N=0→0）。
func wilsonLB(wins, n int, z float64) float64 {
	if n <= 0 {
		return 0
	}
	p := float64(wins) / float64(n)
	zn := z * z / float64(n)
	denom := 1 + zn
	center := (p + zn/2) / denom
	half := z * math.Sqrt(p*(1-p)/float64(n)+z*z/(4*float64(n)*float64(n))) / denom
	lb := center - half
	if lb < 0 {
		return 0
	}
	return lb
}

// evalCard 单 inst 信号+判词（纯函数——cl=末 n 根确认棒收盘正序，
// n=lookback+replay+1；判词比较写 !(x>=阈) 反形——NaN 病态 fail-closed
// 落真因非穿透 proposed；reject 真因序=派工单列序
// no_breakout→thin_samples→lb_weak→net_thin→sigma_zero）。
func evalCard(inst string, cl []float64, lastTs time.Time, p Params) card {
	n := len(cl)
	c := card{inst: inst, sig: "no_breakout", bars: n, lastTs: lastTs,
		verdict: "rejected", reason: "no_breakout"}
	rets := make([]float64, 0, n-1)
	for j := 1; j < n; j++ {
		rets = append(rets, math.Log(cl[j]/cl[j-1]))
	}
	var mean float64
	for _, x := range rets {
		mean += x
	}
	mean /= float64(len(rets))
	var sd float64
	for _, x := range rets {
		sd += (x - mean) * (x - mean)
	}
	c.sigmaD = math.Sqrt(sd / float64(len(rets)-1))
	c.sigmaAnn = c.sigmaD * math.Sqrt(365)

	// 今棒信号：通道=cl[n-1-lookback..n-2]（末棒不入通道）。
	hi, lo := channel(cl[n-1-p.Lookback : n-1])
	switch last := cl[n-1]; {
	case last > hi:
		c.sig, c.opp = "long_perp", "short_perp"
	case last < lo:
		c.sig, c.opp = "short_perp", "long_perp"
	}

	// winrate-CI 回放：i∈[lookback,n-2]，i 收盘破前 lookback 通道记样本
	// （多破 obs=r_{i+1}，空破 obs=−r_{i+1}——多空同构合并）。
	var obs []float64
	for i := p.Lookback; i <= n-2; i++ {
		h, l := channel(cl[i-p.Lookback : i])
		switch ci := cl[i]; {
		case ci > h:
			obs = append(obs, math.Log(cl[i+1]/cl[i]))
		case ci < l:
			obs = append(obs, -math.Log(cl[i+1]/cl[i]))
		}
	}
	c.n = len(obs)
	var sum float64
	for _, o := range obs {
		sum += o
		if o > 0 {
			c.wins++
		}
	}
	if c.n > 0 {
		c.eD = sum / float64(c.n)
		c.phat = float64(c.wins) / float64(c.n)
		c.lb = wilsonLB(c.wins, c.n, wilsonZ)
	}
	c.grossAPR = c.eD * 365
	c.netAPR = (c.eD - p.FrictionRT/p.HoldDays) * 365

	switch {
	case c.sig == "no_breakout":
		// 已置 rejected/no_breakout——无信号向可挂。
	case c.n < p.MinSamples:
		c.reason = "thin_samples"
	case !(c.lb >= p.WinrateLB):
		c.reason = "lb_weak"
	case !(c.netAPR >= p.MinNetAPR):
		c.reason = "net_thin"
	case !(c.sigmaD > 0):
		c.reason = "sigma_zero"
	default:
		c.verdict, c.reason = "proposed", ""
	}
	return c
}
