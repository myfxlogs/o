// Package lifecycle —— 生命周期状态机+paper 沙箱（投资域首岗；员工「命官」）。
// 迁移闭集：proposed→paper、paper→probe、paper→degraded/retired、probe→live/
// degraded/retired、live→degraded/retired。paper→live 越级禁绝；live 迁移另须
// 投资委员会开关 invest.lifecycle.live_switch=true（缺键/非 true fail-closed）。
package lifecycle

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

var legal = map[string][]string{
	"proposed": {"paper", "retired"},
	"paper":    {"probe", "degraded", "retired"},
	"probe":    {"live", "degraded", "retired"},
	"live":     {"degraded", "retired"},
	"degraded": {"retired", "paper"}, // 降级观察后可回 paper 复评或退役
	"retired":  {},
}

// Legal 迁移合法性判（非法=false——调用方拒+计数）。
func Legal(from, to string) bool {
	for _, t := range legal[from] {
		if t == to {
			return true
		}
	}
	return false
}

// Transition 迁移执行：合法性校验→state 翻转+fin_strategy_transition 留痕（同事务）。
// live 迁移须 liveSwitch=true（G4 真钱闸——委员会开关位，非本机自裁）。
func Transition(ctx context.Context, pool *pgxpool.Pool, strategyID int64, to, reason string, liveSwitch bool) error {
	var from string
	if err := pool.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE id=$1", strategyID).Scan(&from); err != nil {
		return err
	}
	if !Legal(from, to) {
		return fmt.Errorf("非法迁移 %s→%s（闭集拒）", from, to)
	}
	if to == "live" && !liveSwitch {
		return fmt.Errorf("live 迁移须 invest.lifecycle.live_switch=true（委员会开关缺位 fail-closed）")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "UPDATE fin_strategy SET state=$2,entered_at=now() WHERE id=$1", strategyID, to); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
		VALUES($1,$2,$3,$4)`, strategyID, from, to, reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
