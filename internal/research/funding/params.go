// params.go —— 判据参数装载面（工单#19797 自 run.go 拆出：棘轮 300 线顶）。
package funding

import "fmt"

var paramKeys = []string{
	"research.funding.taker_fee_bps", "research.funding.maker_fee_bps", "research.funding.slippage_bps",
	"research.funding.fail_cost_bps", "research.funding.window_loss_bps",
	"research.funding.min_net_apr", "research.funding.settles_per_day",
	"guard.max_hold_hours", // T-0297：费用年化摊销持仓期源（与 opener breakevenOK 同键一源）
}

// params 判据全自 config 快照读；任一键缺=不可评估面（fail-closed 不虚构判据）。
func (r *Runner) params() (Params, error) {
	var p Params
	get := func(k string) (float64, error) { return r.cfg.GetFloat(k) }
	var err error
	if p.TakerFeeBps, err = get(paramKeys[0]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[0], err)
	}
	if v, e := get(paramKeys[1]); e == nil {
		p.MakerFeeBps, p.MakerFeeSet = v, true // T-0247 分层：在册含 0/负按真值
	}
	if p.SlippageBps, err = get(paramKeys[2]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[1], err)
	}
	if p.FailCostBps, err = get(paramKeys[3]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[3], err)
	}
	if p.WindowLossBps, err = get(paramKeys[4]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[4], err)
	}
	if p.MinNetAPR, err = get(paramKeys[5]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[6], err)
	}
	if p.SettlesPerDay, err = get(paramKeys[6]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[5], err)
	}
	var holdH float64
	if holdH, err = get(paramKeys[7]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[7], err)
	}
	p.HoldDays = holdH / 24
	return p, nil
}

// stakeStale 质押分量率行新鲜度门读（research.stake.stale_hours——
// 缺/坏=0 不闸，慢特征率行如实取；scan 拍内调一次非逐 inst）。
func (r *Runner) stakeStale() float64 {
	if v, err := r.cfg.GetFloat("research.stake.stale_hours"); err == nil && v > 0 {
		return v
	}
	return 0
}
