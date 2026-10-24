package main

import (
	"context"
	"time"

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

	// 财经日历事件报（T-0070：[财经日历] 独立开关+幂等去重——高影响级预推）。
	cal := notify.NewCalendar(pool, sender, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.calendar.enabled"); return v },
		func() string { v, _ := cfg.GetString("calendar.min_impact"); return v },
		func() int64 { v, _ := cfg.GetInt("calendar.lead_minutes"); return int64(v) })
	go func() {
		// sched-exempt-ok 外部 feed 拉取——FF 周历 XML 无推送面，周期由外部源
		// 更新节律决定非决策节拍（DB 扫描环禁令不涉）。
		tk := time.NewTicker(15 * time.Minute) // sched-exempt-ok 外部源拉取
		defer tk.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				_ = cal.Scan(ctx)
			}
		}
	}()
}
