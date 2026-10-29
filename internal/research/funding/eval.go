// Package funding —— funding 对冲机会枚举+评估卡（研究域首岗；员工 探员/验证官）。
// 净 APR 五拆：毛收 − 双腿费 − 滑点 − 失败成本 − 窗口损；任一无证据=不可评估禁出卡。
package funding

import "fmt"

// Params 五拆判据（全自 app_config 机读——零代码字面量）。
type Params struct {
	TakerFeeBps   float64
	MakerFeeBps   float64
	MakerFeeSet   bool // T-0247 分层：在册（含 0/负）→真值复合；键缺→双 taker 保守
	SlippageBps   float64
	FailCostBps   float64
	WindowLossBps float64
	MinNetAPR     float64
	SettlesPerDay float64
	// EarnRateAPR Earn 现率年化（D-0006 动态比较底；nil=缺失/陈旧——降级静态底+告警计数）。
	EarnRateAPR *float64
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
	// 动态比较底（D-0006）：floor=max(静态底, Earn 现率)——停泊收益是机会成本底线。
	floor := p.MinNetAPR
	floorSrc := "static"
	if p.EarnRateAPR != nil && *p.EarnRateAPR > floor {
		floor = *p.EarnRateAPR
		floorSrc = "earn"
	}
	gross := abs(rate) * p.SettlesPerDay * 365
	// T-0238 分费制：maker_fee_bps 在册=maker 腿×maker 费+降级腿×taker 费
	// 复合估（一条 maker 一条降级保守面）；缺键=原双 taker 形。
	feeBps := 2 * p.TakerFeeBps
	if p.MakerFeeSet { // 在册按真值（0=免 maker 费档，负=返佣计入 cost 负项——哨兵不揉）
		feeBps = p.TakerFeeBps + p.MakerFeeBps
	}
	cost := (feeBps + p.SlippageBps + p.FailCostBps + p.WindowLossBps) / 1e4
	net := gross - cost
	e.GrossAPR, e.NetAPR = gross, net
	e.Evidence = map[string]any{
		"funding_rate": rate, "settles_per_day": p.SettlesPerDay,
		"fee_bps": feeBps, "taker_fee_bps": p.TakerFeeBps, "maker_fee_bps": p.MakerFeeBps, "maker_fee_set": p.MakerFeeSet, "slippage_bps": p.SlippageBps,
		"fail_cost_bps": p.FailCostBps, "window_loss_bps": p.WindowLossBps,
		"min_net_apr": p.MinNetAPR, "floor_apr": floor, "floor_source": floorSrc,
	}
	if p.EarnRateAPR != nil {
		e.Evidence["earn_rate_apr"] = *p.EarnRateAPR
	}
	switch {
	case rate < 0:
		e.Verdict, e.Reason = "rejected", "负 funding 反向=现货做空腿无 venue 支撑（不可执行）"
	case net >= floor:
		e.Verdict = "proposed"
	case net < p.MinNetAPR:
		e.Verdict, e.Reason = "rejected", fmt.Sprintf("net_apr %.4f < 阈 %.4f", net, p.MinNetAPR)
	default:
		// 动态底杀：静态底过但停泊收益更高——below_earn（verdict/reason 面机读）。
		e.Verdict, e.Reason = "rejected", fmt.Sprintf("below_earn: net_apr %.4f < Earn 底 %.4f（静态底 %.4f 过）", net, floor, p.MinNetAPR)
	}
	e.Evidence["reason"] = e.Reason // 拒因入 evidence 落卡（fin_opp_eval 无独立 reason 列——机读面=evidence jsonb）
	return e
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
