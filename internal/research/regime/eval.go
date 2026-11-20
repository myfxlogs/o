// eval.go —— regime 分类核（纯函数件：OHLC 窗→五态判词+证据载荷；
// run.go 持拍面，persist.go 持读面/落表面——工单#25750 SSOT③钦定判据
// 落地式）。
package regime

import "math"

// kline 单根确认棒（分类消费 h/l/c——h/l 现行仅占位备查，判据全吃 c；
// 与 gridfit 同形三件——宇宙同 md_klines 1D 面）。
type kline struct{ h, l, c float64 }

// feat 单 inst 分类证据（态账/诊断面载荷——非落表字段）。
type feat struct {
	z, sigma, vr, cont float64
	bars               int
}

// variance 样本方差（n<2→0——病态窗不虚构）。
func variance(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var m float64
	for _, x := range xs {
		m += x
	}
	m /= float64(len(xs))
	var s float64
	for _, x := range xs {
		s += (x - m) * (x - m)
	}
	return s / float64(len(xs)-1)
}

// classify 单 inst regime 判词（纯函数——ks=末 lookback+1 根确认棒
// 正序；判词序=SSOT③列序 trend→chop→quiet→volatile→mixed 首中即返）：
//
//	|z|≥z_thr→trend（z=dirmom 同式 mom/(sigma_d·√lookback)）；
//	vr=Var(r2)/(2·Var(r1))<vr_thr ∧ 含带率≥cont_thr→chop——r1=日对数
//	  收益、r2=两日对数收益（重叠全样本 i∈[2,n)）；含带率=收盘含于
//	  既往通道占比（c_i∈[min(c_0..i-1),max(c_0..i-1)] 不破前高前低，
//	  首棒无既往通道不计——chop=区间束缚形态特征，票面未钉细节自裁）；
//	sigma_d<quiet_thr→quiet；sigma_d>vol_thr→volatile；余=mixed。
//
// sigma_d=0（零波动）→z 置 0 不入 trend——落 quiet 归「死市静态」实。
func classify(ks []kline, p Params) (string, feat) {
	n := len(ks)
	cl := make([]float64, n)
	for i, k := range ks {
		cl[i] = k.c
	}
	rets := make([]float64, 0, n-1)
	for i := 1; i < n; i++ {
		rets = append(rets, math.Log(cl[i]/cl[i-1]))
	}
	sigma := math.Sqrt(variance(rets))
	z := 0.0
	if sigma > 0 {
		z = math.Log(cl[n-1]/cl[0]) / (sigma * math.Sqrt(float64(p.Lookback)))
	}
	r2 := make([]float64, 0, n-2)
	for i := 2; i < n; i++ {
		r2 = append(r2, math.Log(cl[i]/cl[i-2]))
	}
	v1, v2 := variance(rets), variance(r2)
	vr := math.Inf(1)
	if v1 > 0 {
		vr = v2 / (2 * v1)
	}
	// 含带率：收盘含于既往收盘通道（不破前高前低）棒占比。
	inside, runLo, runHi := 0, cl[0], cl[0]
	for i := 1; i < n; i++ {
		if cl[i] >= runLo && cl[i] <= runHi {
			inside++
		}
		if cl[i] < runLo {
			runLo = cl[i]
		}
		if cl[i] > runHi {
			runHi = cl[i]
		}
	}
	cont := float64(inside) / float64(n-1)
	f := feat{z: z, sigma: sigma, vr: vr, cont: cont, bars: n}
	switch {
	case math.Abs(z) >= p.ZThr:
		return "trend", f
	case vr < p.VrThr && cont >= p.ContThr:
		return "chop", f
	case sigma < p.QuietThr:
		return "quiet", f
	case sigma > p.VolThr:
		return "volatile", f
	default:
		return "mixed", f
	}
}
