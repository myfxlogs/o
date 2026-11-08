// xvenue_retire.go —— T-0145 xvenue 退役/对位面（自 xvenue.go 拆件——棘轮
// D-588）：提案双行对位+终局拒双腿处置+黑名单预检退役；附 reopen 冷却
// 探测件（open.go 迁——funding-hedge/xvenue 两径共用）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/portfolio/lifecycle"
	"o/internal/trading/mtexec"
	"o/internal/trading/okxexec"
)

// xvenueErr 全 err 出口终局分类（T-0145 返修①——venue 终局拒任一出口皆
// 入册联动，非仅 Execute xerr；mtexec/okxexec 双判各入各册在案码径：
// okx 终局→okx inst 入册，mt5 终局→cfd canonical 入册——双腿+提案退役
// 三件共 xvenueUntradeable）。非终局/入册锚缺席（cfd 未对位）=原 err 透传。
func (h hedgeOpener) xvenueErr(ctx context.Context, spec lifecycle.OpenSpec,
	mateID int64, cfd string, err error) error {
	if err == nil {
		return nil
	}
	var inst string
	switch {
	case okxexec.TerminalInstErr(err):
		inst = spec.InstID
	case mtexec.TerminalInstErr(err):
		inst = cfd
	default:
		return err
	}
	if inst == "" {
		return err
	}
	if herr := h.xvenueUntradeable(ctx, spec, mateID, inst, err); herr != nil {
		return fmt.Errorf("终局处置败（须人查）: %w", herr)
	}
	return fmt.Errorf("venue_untradeable: %w", err)
}

// xvenuePair cfd 官名+副腿行 id 对位（提案双行同 proposal_ref 配对——
// xvenuepair.go 建序钦定）；pair 行未落期兜底 bridge.xvenue.map 同映射源
// （双无=fail-closed 不构单腿）。
func (h hedgeOpener) xvenuePair(ctx context.Context, spec lifecycle.OpenSpec) (string, int64, error) {
	var cfd string
	var mateID int64
	err := h.pool.QueryRow(ctx, `SELECT m.id, m.inst_id FROM fin_strategy m
		JOIN fin_strategy s ON s.proposal_ref=m.proposal_ref
		WHERE s.id=$1 AND m.venue='mt5' AND m.id<>s.id
		ORDER BY m.id DESC LIMIT 1`, spec.StrategyRef).Scan(&mateID, &cfd)
	if err == nil {
		return cfd, mateID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", 0, err
	}
	var mapRaw string
	if e := h.pool.QueryRow(ctx, `SELECT value FROM app_config WHERE key='bridge.xvenue.map'`).Scan(&mapRaw); e != nil {
		return "", 0, fmt.Errorf("xvenue 双腿映射缺（pair 行+bridge.xvenue.map 双无）fail-closed: %w", e)
	}
	var m map[string]string
	if e := json.Unmarshal([]byte(mapRaw), &m); e != nil || m[spec.InstID] == "" {
		return "", 0, fmt.Errorf("xvenue.map 无 %s 映射 fail-closed", spec.InstID)
	}
	return m[spec.InstID], 0, nil
}

// xvenueUntradeable 终局拒双腿处置（T-0145③——显式迁移先于入册序位律同
// T-0147：trigger 先至撞「retired→retired」非法迁移）。三件：①双腿显式退役
// （state-guard 幂等）；②提案 fin_retired（挂 okx inst 的 armed 提案）；③终局
// inst 入册（0061/0062 trigger 续扫同 inst 余量——mt5 副行/cfd 提案 trigger 兜）。
func (h hedgeOpener) xvenueUntradeable(ctx context.Context, spec lifecycle.OpenSpec,
	mateID int64, blInst string, xerr error) error {
	code, msg := "", ""
	var me *mtexec.APIError
	var oe *okxexec.APIError
	if errors.As(xerr, &me) {
		code, msg = me.Code, me.Msg
	}
	if errors.As(xerr, &oe) {
		code, msg = oe.Code, oe.Msg
	}
	if code == "" {
		code = "unknown"
	}
	if err := h.xvenuePairRetire(ctx, spec, mateID, code+" "+msg); err != nil {
		return err
	}
	if _, err := h.pool.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
		VALUES($1,$2,$3)
		ON CONFLICT (inst_id) DO UPDATE SET code=EXCLUDED.code, msg=EXCLUDED.msg`,
		blInst, code, msg); err != nil {
		return err
	}
	log.Printf("ALERT [open] xvenue 终局拒 %s（%s %s）——双腿退役+提案退役+入册",
		blInst, code, msg)
	return nil
}

// xvenuePairRetire 双腿+提案退役（预检命中/终局拒共用——state-guard 幂等
// 防 trigger 先至/重复处置撞非法迁移）。
func (h hedgeOpener) xvenuePairRetire(ctx context.Context, spec lifecycle.OpenSpec,
	mateID int64, reason string) error {
	for _, sid := range []int64{spec.StrategyRef, mateID} {
		if sid == 0 {
			continue
		}
		var live int
		if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy
			WHERE id=$1 AND state IN ('live','paper','proposed','probe','degraded')`, sid).Scan(&live); err != nil {
			return err
		}
		if live > 0 {
			if err := lifecycle.Transition(ctx, h.pool, sid, "retired",
				"venue_untradeable:"+reason, false); err != nil {
				return fmt.Errorf("策略退#%d: %w", sid, err)
			}
		}
	}
	if _, err := h.pool.Exec(ctx, `UPDATE fin_proposal SET state='fin_retired',
		retire_reason='venue_untradeable',retired_at=now()
		WHERE inst_id=$1 AND state='fin_armed'`, spec.InstID); err != nil {
		return err
	}
	log.Printf("ALERT [open] xvenue 双腿退役（%s）", reason)
	return nil
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
