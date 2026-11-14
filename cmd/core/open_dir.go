// open_dir.go —— dir_perp_mom live 开仓装配（棒5c 单腿真户径）：
// 方向族=一张 perp 腿全载信念——双帽取小同 hedge 径，腿构=BuildDirLeg
// （张数=minSz 闸 fail-closed），执行=ExecuteDir（令牌/live 闸/赎回内建）。
// 平仓不由此径——live 出场=仓巡 tend（stop/take/TTL/翻面 dirTrig→单腿 CloseExec）。
package main

import (
	"context"
	"fmt"

	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/okxexec"
)

func (h hedgeOpener) dirOpen(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
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
	legLabel, lerr := dirLegLabel(spec.Direction)
	if lerr != nil {
		return 0, lerr
	}
	dec := h.gate.Check(ctx, risk.Intent{StrategyRef: spec.StrategyRef, Venue: spec.Venue,
		InstID: spec.InstID, Legs: []string{legLabel},
		NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return 0, fmt.Errorf("闸拒: %s", dec.Reason)
	}
	leg, lerr := execution.BuildDirLeg(ctx, h.pool, h.loadMeta, spec.Venue, spec.InstID, spec.Direction, notional)
	if lerr != nil {
		if okxexec.TerminalInstErr(lerr) {
			if herr := h.untradeable(ctx, spec, lerr); herr != nil {
				return 0, fmt.Errorf("终局处置败（须人查）: %w", herr)
			}
			return 0, fmt.Errorf("venue_untradeable: %w", lerr)
		}
		return 0, fmt.Errorf("腿构: %w", lerr)
	}
	eid, xerr := h.r.ExecuteDir(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: spec.StrategyRef, InstID: spec.InstID, NotionalUSD: notional,
		Legs: [2]execution.Leg{leg}})
	if okxexec.TerminalInstErr(xerr) {
		if herr := h.untradeable(ctx, spec, xerr); herr != nil {
			return eid, fmt.Errorf("终局处置败（须人查）: %w", herr)
		}
		return eid, fmt.Errorf("venue_untradeable: %w", xerr)
	}
	return eid, xerr
}

// dirLegLabel direction→闸腿签（long_perp=perp_long/short_perp=perp_short——
// 风险册腿域标签方言，与 hedge 径 ["perp_short","spot_long"] 同集）。
func dirLegLabel(direction string) (string, error) {
	switch direction {
	case "long_perp":
		return "perp_long", nil
	case "short_perp":
		return "perp_short", nil
	}
	return "", fmt.Errorf("direction 非闭集: %s", direction)
}
