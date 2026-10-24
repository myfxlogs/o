package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/events"
	"phase/internal/platform/notify"
)

// startEventPush 经营事件报（T-0061：三里程碑实时可见——与告警分径独立开关；
// 事件驱动=ev_position/ev_cashflow 脊，禁周期环）。
func startEventPush(ctx context.Context, _ chan<- error, pool *pgxpool.Pool,
	cfg *config.Client, sender notify.Sender) {
	evPush := notify.NewEventPush(pool, sender, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.events.enabled"); return v })
	go events.ListenLoop(ctx, pool, "event_push",
		[]string{"ev_position", "ev_cashflow"}, evPush.Scan)
}
