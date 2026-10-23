// Package funding —— funding 对冲机会枚举+评估卡（研究域首岗；员工 探员/验证官）。
// 净 APR 五拆：毛收 − 双腿费 − 滑点 − 失败成本 − 窗口损；任一无证据=不可评估禁出卡。
package funding

import "fmt"

// Params 五拆判据（全自 app_config 机读——零代码字面量）。
type Params struct {
	TakerFeeBps   float64
	SlippageBps   float64
	FailCostBps   float64
	WindowLossBps float64
	MinNetAPR     float64
	SettlesPerDay float64
}

// Eval 评估卡出件。
type Eval struct {
	InstID    string
	Direction string
	GrossAPR  float64
	NetAPR    float64
	Verdict   string // proposed|rejected|unevaluable
	Legs      map[string]any
	Evidence  map[string]any
	Reason    string
}

// Evaluate 净 APR 五拆（funding>0→空永续收租=short_perp_long_spot；
// funding<0→long_perp_short_spot——现货做空腿当前无支撑=不可执行 rejected）。
func Evaluate(instID string, rate float64, p Params) Eval {
	e := Eval{InstID: instID, Legs: map[string]any{}, Evidence: map[string]any{}}
	if rate >= 0 {
		e.Direction = "short_perp_long_spot"
		e.Legs = map[string]any{"perp": "short", "spot": "long"}
	} else {
		e.Direction = "long_perp_short_spot"
		e.Legs = map[string]any{"perp": "long", "spot": "short"}
	}
	gross := abs(rate) * p.SettlesPerDay * 365
	cost := (2*p.TakerFeeBps + p.SlippageBps + p.FailCostBps + p.WindowLossBps) / 1e4
	net := gross - cost
	e.GrossAPR, e.NetAPR = gross, net
	e.Evidence = map[string]any{
		"funding_rate": rate, "settles_per_day": p.SettlesPerDay,
		"taker_fee_bps": p.TakerFeeBps, "slippage_bps": p.SlippageBps,
		"fail_cost_bps": p.FailCostBps, "window_loss_bps": p.WindowLossBps,
		"min_net_apr": p.MinNetAPR,
	}
	switch {
	case rate < 0:
		e.Verdict, e.Reason = "rejected", "负 funding 反向=现货做空腿无 venue 支撑（不可执行）"
	case net >= p.MinNetAPR:
		e.Verdict = "proposed"
	default:
		e.Verdict, e.Reason = "rejected", fmt.Sprintf("net_apr %.4f < 阈 %.4f", net, p.MinNetAPR)
	}
	return e
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
