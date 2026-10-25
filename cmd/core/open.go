// open.go —— live 开仓装配缝（T-0050 首仓序④）：lifecycle.Opener 适配——
// risk.Gate.Check（唯一交易前闸）→execution.BuildLegs（perp 短+现货长，乐器
// 元数据决策时点 instruments 现读）→Runner.Execute（token/对冲带/Earn 赎回/
// 双腿原子——与 openfirst 同码径，事件驱自动径）。
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/okxexec"
)

// hedgeOpener funding-hedge 开仓适配（cmd 组合根——投资域不反向依赖交易/风控域）。
type hedgeOpener struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	gate *risk.Gate
	r    *execution.Runner
	cli  *okxexec.Client
}

// Open live 策略开仓：双帽取小→闸→腿构→原子执行。任一环败=err 上抛
// （lifecycle 计数续试——ev_strategy 事件驱非热环）。
func (h hedgeOpener) Open(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	notional := spec.NotionalUSD
	// 双帽取小（openfirst 同径：probe_notional/max_notional——缺键 fail-closed）。
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
	// T-0107 再开冷却：guard 触发平仓后同策略同标的间隔窗（exec.reopen_cooldown_sec
	// 热启；缺键/探测败 fail-closed 拒开）——瞬态触发禁 45s 级连绞再装填。
	blocked, err := reopenCooldownBlocked(ctx, h.cfg, h.pool, spec.StrategyRef, spec.InstID)
	if err != nil {
		return 0, fmt.Errorf("冷却窗探测 fail-closed: %w", err)
	}
	if blocked {
		return 0, fmt.Errorf("reopen_cooldown: guard 平仓后冷却窗内拒再开（%s）", spec.InstID)
	}
	dec := h.gate.Check(ctx, risk.Intent{StrategyRef: spec.StrategyRef, Venue: spec.Venue,
		InstID: spec.InstID, Legs: []string{"perp_short", "spot_long"},
		NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return 0, fmt.Errorf("闸拒: %s", dec.Reason)
	}
	legs, _, err := execution.BuildLegs(ctx, h.pool, h.loadMeta, spec.Venue, spec.InstID, notional)
	if err != nil {
		return 0, fmt.Errorf("腿构: %w", err)
	}
	return h.r.Execute(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: spec.StrategyRef, InstID: spec.InstID, NotionalUSD: notional, Legs: legs})
}

// reopenCooldownBlocked 再开冷却窗判定（T-0107）：同策略同标的存在冷却窗内
// guard 平仓记录（updated_at=任一 guard 态变时点）→true。判据=config 快照件
// GetFloat（缺键 fail-closed true——宁可冻结不再开也不裸奔连绞）；0=显式关闭。
func reopenCooldownBlocked(ctx context.Context, cfg *config.Client, pool *pgxpool.Pool,
	strategyRef int64, instID string) (bool, error) {
	sec, err := cfg.GetFloat("exec.reopen_cooldown_sec")
	if err != nil {
		return true, fmt.Errorf("exec.reopen_cooldown_sec: %w", err)
	}
	if sec <= 0 {
		return false, nil // 冷却显式关闭（0=运维豁免位）
	}
	var blocked bool
	err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fin_guard_close g
		JOIN fin_position p ON p.id=g.position_id
		WHERE p.strategy_ref=$1 AND p.inst_id=$2
		  AND g.updated_at > now()-make_interval(secs=>$3))`,
		strategyRef, instID, sec).Scan(&blocked)
	return blocked, err
}

// loadMeta 乐器元数据现读（/public/instruments——SPOT/SWAP 按后缀分派）。
func (h hedgeOpener) loadMeta(ctx context.Context, iid string) (execution.InstMeta, error) {
	it := "SPOT"
	if strings.HasSuffix(iid, "-SWAP") {
		it = "SWAP"
	}
	m, err := h.cli.InstrumentMeta(ctx, it, iid)
	return execution.InstMeta{CtVal: m.CtVal, CtValCcy: m.CtValCcy, LotSz: m.LotSz, MinSz: m.MinSz}, err
}
