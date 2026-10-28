// basis_open.go —— basis 族 live 开仓 executor（T-0216——quarter_basis_long/
// perp_quarter_spread 同 funding-hedge 开闸构：cap→冷却→闸→腿构→Execute
// 原子双腿；衍生腿=-YYMMDD 到期日期货=FUTURES 乐器域）。quarter_basis_short
// 不入此面——spot_short 腿 margin 能力缺口（T-0216 条件缺口件）。
package main

import (
	"context"
	"fmt"
	"regexp"

	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/okxexec"
)

// datedFutRe 到期日期货键（-YYMMDD 尾——grammar.evalBasis 同表达式）。
var datedFutRe = regexp.MustCompile(`^([A-Z0-9]+-[A-Z0-9]+)-[0-9]{6}$`)

// basisOpen basis 族开仓（inst_id=季度期货键；多头腿键按族派生——
// quarter_basis_long→基键现货 / perp_quarter_spread→基键+-SWAP）。
func (h hedgeOpener) basisOpen(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	m := datedFutRe.FindStringSubmatch(spec.InstID)
	if m == nil {
		return 0, fmt.Errorf("basis inst 非到期日期货键: %s", spec.InstID)
	}
	longInst := m[1]
	longSpot := spec.Template == "quarter_basis_long"
	if !longSpot {
		longInst += "-SWAP"
	}
	notional := spec.NotionalUSD
	cap1, e1 := h.cfg.GetFloat("exec.probe_notional_usd")
	cap2, e2 := h.cfg.GetFloat("risk.max_notional_usd")
	if e1 != nil || e2 != nil {
		return 0, fmt.Errorf("cap 键缺 fail-closed: %v/%v", e1, e2)
	}
	notional = min(notional, cap1, cap2)
	if notional <= 0 {
		return 0, fmt.Errorf("notional≤0 拒（strategy=%.2f caps=%.2f/%.2f）",
			spec.NotionalUSD, cap1, cap2)
	}
	blocked, err := reopenCooldownBlocked(ctx, h.cfg, h.pool, spec.StrategyRef, spec.InstID)
	if err != nil {
		return 0, fmt.Errorf("冷却窗探测 fail-closed: %w", err)
	}
	if blocked {
		return 0, fmt.Errorf("reopen_cooldown: guard 平仓后冷却窗内拒再开（%s）", spec.InstID)
	}
	dec := h.gate.Check(ctx, risk.Intent{StrategyRef: spec.StrategyRef, Venue: spec.Venue,
		InstID: spec.InstID, Legs: []string{"quarter_future_short", "basis_long"},
		NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return 0, fmt.Errorf("闸拒: %s", dec.Reason)
	}
	legs, lerr := execution.BuildBasisLegs(ctx, h.pool, h.loadMeta, spec.Venue,
		spec.InstID, longInst, longSpot, notional)
	if lerr != nil {
		if okxexec.TerminalInstErr(lerr) {
			if herr := h.untradeable(ctx, spec, lerr); herr != nil {
				return 0, fmt.Errorf("终局处置败（须人查）: %w", herr)
			}
			return 0, fmt.Errorf("venue_untradeable: %w", lerr)
		}
		return 0, fmt.Errorf("腿构: %w", lerr)
	}
	eid, xerr := h.r.Execute(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: spec.StrategyRef, InstID: spec.InstID, NotionalUSD: notional, Legs: legs})
	if okxexec.TerminalInstErr(xerr) {
		if herr := h.untradeable(ctx, spec, xerr); herr != nil {
			return eid, fmt.Errorf("终局处置败（须人查）: %w", herr)
		}
		return eid, fmt.Errorf("venue_untradeable: %w", xerr)
	}
	return eid, xerr
}
