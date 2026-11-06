// Package lifecycle —— 生命周期状态机+paper 沙箱（投资域首岗；员工「命官」）。
// 迁移闭集：proposed→paper、paper→probe、paper→degraded/retired、probe→live/
// degraded/retired、live→degraded/retired。paper→live 越级禁绝；live 迁移另须
// 投资委员会开关 invest.lifecycle.live_switch=true（缺键/非 true fail-closed）。
package lifecycle

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/decision"
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
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	// T-0415：ops_decision 记忆行继事实行落账（退役=close 墓志铭——触发因携
	// reason，载荷 expected_net_apr/realized_usd 实现的vs期望的；余迁=adjust）。
	// 记忆写不阻钱径——写败走升件自报禁静默（decision 写径纪律②）。
	act := "adjust"
	if to == "retired" {
		act = "close"
	}
	payload := map[string]any{"from": from, "to": to}
	for k, v := range decision.MoneyFacts(ctx, pool, strategyID, 0) {
		payload[k] = v
	}
	decision.WriteEsc(ctx, pool, decision.Row{
		Domain: "fin", Actor: "lifecycle", RefID: strconv.FormatInt(strategyID, 10),
		Action: act, Reason: reason, Payload: payload})
	return nil
}
