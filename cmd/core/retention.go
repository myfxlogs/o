// retention.go —— 数据清道夫日拍装配（T-0153：fin_data_retention 账行驱动；
// 未入账表报缺 fail-closed；哨兵告警走 humanize kind）。
package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/ledger/retention"
	"o/internal/platform/notify"
	"o/internal/platform/roster"
)

// startRetention 清道夫起岗（日拍——bounded 巡检非业务轮询；D-0008 豁免注）。
func startRetention(ctx context.Context, errc chan<- error, pool *pgxpool.Pool, al *notify.Alerter) {
	if err := roster.Register(ctx, pool, "清道夫", "ledger", "数据保留清道", nil); err != nil {
		errc <- err
		return
	}
	sw := retention.New(pool, func(ctx context.Context, msg string) {
		if al != nil {
			al.Alert(ctx, "retention_volume_alert", msg)
		}
	})
	tick := time.NewTicker(24 * time.Hour) // sched-exempt-ok：日拍清道（保留账巡面非业务轮询——节拍级让位）
	defer tick.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if err := sw.Sweep(ctx); err != nil {
					errc <- err
					return
				}
			}
		}
	}()
}
