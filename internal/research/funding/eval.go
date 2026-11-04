// Package funding —— funding 对冲机会枚举+评估卡（研究域首岗；员工 探员/验证官）。
// 净 APR 五拆：毛收 − 往返摩擦年化摊销（双腿费+滑点+失败成本+窗口损，T-0297
// 口径根修——一次性费用按 guard.max_hold_hours 持仓期摊年化，与毛收同单位相减）；
// 任一无证据=不可评估禁出卡。
package funding

import (
	"fmt"
	"math"
)

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
	// HoldDays 预测持仓期天（guard.max_hold_hours/24——T-0297：与 opener
	// breakevenOK 同一时间地平线一源，费用年化摊销分母）。
	HoldDays float64
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

// Yield 现货腿质押/Earn 收益分量（T-0320 双头 carry 口径根修——short_
// perp_long_spot 向长现货腿若可质押（能力面 staking 维注册），质押/Earn
// 年化并入毛收；旧口径裸持=质押分量系统性蒸发，双头结构被按单头算分）。
// APR=年化小数（0.033=3.3%）；Source=率源（注册行 rate_source 字段）。
type Yield struct {
	APR    float64
	Source string
}

// Evaluate 净 APR 五拆（funding>0→空永续收租=short_perp_long_spot；
// funding<0→long_perp_short_spot——现货做空腿当前无支撑=不可执行 rejected；
// sy 零值=无质押面/不可质押如实——gross 退化原单头形）。
func Evaluate(instID string, rate float64, p Params, sy Yield) Eval {
	e := Eval{InstID: instID, Legs: map[string]any{}, Evidence: map[string]any{}}
	if rate >= 0 {
		e.Direction = "short_perp_long_spot"
		e.Legs = map[string]any{"perp": "short", "spot": "long"}
	} else {
		e.Direction = "long_perp_short_spot"
		e.Legs = map[string]any{"perp": "long", "spot": "short"}
		sy = Yield{} // 现货做空腿无质押分量（方向闸——空 spot 不记收益）
	}
	// 动态比较底（D-0006）：floor=max(静态底, Earn 现率)——停泊收益是机会成本底线。
	floor := p.MinNetAPR
	floorSrc := "static"
	if p.EarnRateAPR != nil && *p.EarnRateAPR > floor {
		floor = *p.EarnRateAPR
		floorSrc = "earn"
	}
	gross := abs(rate)*p.SettlesPerDay*365 + sy.APR // T-0320：质押腿入总分
	// T-0238 分费制：maker_fee_bps 在册=maker 腿×maker 费+降级腿×taker 费
	// 复合估（一条 maker 一条降级保守面）；缺键=原双 taker 形。
	feeBps := 2 * p.TakerFeeBps
	if p.MakerFeeSet { // 在册按真值（0=免 maker 费档，负=返佣计入 cost 负项——哨兵不揉）
		feeBps = p.TakerFeeBps + p.MakerFeeBps
	}
	// T-0297 费用往返口径＋年化摊销（根修：旧式一次性费用与年化毛收直减
	// 低估 ~120 倍）——cost_rt_bps=四次成交往返摩擦，cost_apr=按持仓期摊年化。
	costRtBps := RoundTripBps(p)
	costAPR := CostAPR(p)
	net := gross - costAPR
	e.GrossAPR, e.NetAPR = gross, net
	e.Evidence = map[string]any{
		"funding_rate": rate, "settles_per_day": p.SettlesPerDay,
		"fee_bps": feeBps, "taker_fee_bps": p.TakerFeeBps, "maker_fee_bps": p.MakerFeeBps, "maker_fee_set": p.MakerFeeSet, "slippage_bps": p.SlippageBps,
		"fail_cost_bps": p.FailCostBps, "window_loss_bps": p.WindowLossBps,
		"cost_rt_bps": costRtBps, "hold_days": p.HoldDays, "cost_apr": costAPR,
		"min_net_apr": p.MinNetAPR, "floor_apr": floor, "floor_source": floorSrc,
		"spot_leg_yield_apr": sy.APR, // T-0320 分量字段——0=无质押面如实
	}
	if p.EarnRateAPR != nil {
		e.Evidence["earn_rate_apr"] = *p.EarnRateAPR
	}
	if sy.Source != "" {
		e.Evidence["spot_yield_source"] = sy.Source
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

// RoundTripBps 往返摩擦（bps，T-0297 导出——opener breakevenOK 同源调用面）：
// 四次成交=2×双腿费 + 2×滑点 + 失败成本 + 窗口损。maker 分层沿 T-0247：
// 在册=taker+maker 复合，缺=2×taker。
func RoundTripBps(p Params) float64 {
	feeBps := 2 * p.TakerFeeBps
	if p.MakerFeeSet {
		feeBps = p.TakerFeeBps + p.MakerFeeBps
	}
	return 2*feeBps + 2*p.SlippageBps + p.FailCostBps + p.WindowLossBps
}

// CostAPR 往返摩擦年化摊销（比率）=cost_rt_bps/1e4×365/hold_days。
// HoldDays<=0=判据缺位——+Inf fail-closed（净 APR 必沉底，不虚构放行）。
func CostAPR(p Params) float64 {
	if p.HoldDays <= 0 {
		return math.Inf(1)
	}
	return RoundTripBps(p) / 1e4 * 365 / p.HoldDays
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
