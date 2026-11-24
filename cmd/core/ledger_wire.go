// ledger_wire.go —— 账本域装配件（棘轮拆出：report 推送+pusher+ledger
// runner 三件同源接线——main.go 棘轮基线减负）。
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/ledger"
	"o/internal/ledger/report"
	"o/internal/platform/config"
	"o/internal/platform/notify"
)

// wireLedgerReport 账本域首岗装配（report 日推+pusher 凭据解+
// ledger runner 对账环——navFn 闭包同源 svc 装配面）。
func wireLedgerReport(ctx context.Context, errc chan error, pool *pgxpool.Pool,
	cfg *config.Client, navFn func(context.Context) (map[string]float64, error),
	alerter *notify.Alerter) *ledger.Runner {
	mk, _ := config.ReadMasterKey()
	pusher := report.NewPusher(pool, cfg, mk)
	go report.NewRunner(pool, pusher).WithConfig(cfg).WithNav(navFn).Run(ctx, errc)
	lr := ledger.NewRunner(pool, cfg).WithAlerter(alerter)
	go lr.Run(ctx, errc)
	return lr
}
