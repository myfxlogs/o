// notify.go —— kill_switch 状态变迁推（T-0048 事件化：LISTEN config_changed 醒后
// 查现值，true/false 各一报不重复——自主文件拆分，行数棘轮）。
package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/notify"
)

func killSwitchNotifier(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client, alerter *notify.Alerter) {
	prev, _ := cfg.GetBool("risk.kill_switch")
	for {
		conn, err := pool.Acquire(ctx)
		if err != nil {
			return
		}
		if _, err := conn.Exec(ctx, "LISTEN config_changed"); err != nil {
			conn.Release()
			return
		}
		n, err := conn.Conn().WaitForNotification(ctx)
		conn.Release()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			continue // 断线重听（事件化=真事件脊非轮询）
		}
		_ = n
		v, err := cfg.GetBool("risk.kill_switch")
		if err == nil && v != prev {
			alerter.Alert(ctx, "kill_switch", fmt.Sprintf("kill_switch %v→%v", prev, v))
			prev = v
		}
	}
}
