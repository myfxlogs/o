// open.go —— live 开仓装配缝（T-0050 首仓序④）：lifecycle.Opener 适配——
// risk.Gate.Check（唯一交易前闸）→execution.BuildLegs（perp 短+现货长，乐器
// 元数据决策时点 instruments 现读）→Runner.Execute（token/对冲带/Earn 赎回/
// 双腿原子——与 openfirst 同码径，事件驱自动径）。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	legs, _, lerr := execution.BuildLegs(ctx, h.pool, h.loadMeta, spec.Venue, spec.InstID, notional)
	if lerr != nil {
		// T-0147 终局分类收口（错误冒泡点统一——腿构 meta/校验径与下单提交
		// 径同闸）：51001 在腿构 spot 元数据拉取段返回（实案 286078），
		// TerminalInstErr 分类不得只钉 order-submit 出口。
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
	// T-0142 终局分类：instrument 终局拒（51001/51003 族）=venue 拒语终局事实
	// ——黑名单入册+策略退（retired,venue_untradeable）+同 inst armed 提案退。
	// 策略退后 openLive 查询（state='live'）不再选中=续试风暴根除（禁事件续试）。
	if okxexec.TerminalInstErr(xerr) {
		if herr := h.untradeable(ctx, spec, xerr); herr != nil {
			return eid, fmt.Errorf("终局处置败（须人查）: %w", herr)
		}
		return eid, fmt.Errorf("venue_untradeable: %w", xerr)
	}
	return eid, xerr
}

// untradeable 终局处置三件（T-0142）：黑名单入册（学费一次）→策略退
// （Transition 留痕同决策席手动处置写径）→同 inst armed 提案退 fin_retired。
func (h hedgeOpener) untradeable(ctx context.Context, spec lifecycle.OpenSpec, xerr error) error {
	var ae *okxexec.APIError
	_ = errors.As(xerr, &ae)
	code, msg := "", ""
	if ae != nil {
		code, msg = ae.Code, ae.Msg
	}
	if _, err := h.pool.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
		VALUES($1,$2,$3)
		ON CONFLICT (inst_id) DO UPDATE SET code=EXCLUDED.code, msg=EXCLUDED.msg`,
		spec.InstID, code, msg); err != nil {
		return err
	}
	log.Printf("ALERT [open] 终局拒 %s（%s %s）——黑名单入册+策略 %d 退+armed 提案退",
		spec.InstID, code, msg, spec.StrategyRef)
	if err := lifecycle.Transition(ctx, h.pool, spec.StrategyRef, "retired",
		"venue_untradeable:"+code+" "+msg, false); err != nil {
		return fmt.Errorf("策略退: %w", err)
	}
	_, err := h.pool.Exec(ctx, `UPDATE fin_proposal SET state='fin_retired'
		WHERE inst_id=$1 AND state='fin_armed'`, spec.InstID)
	return err
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
