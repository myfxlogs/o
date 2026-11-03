package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/notify"
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

	// T-0312 regime 判词 O 群推（第九桥镜像后 ev_regime 帧驱动——
	// 非轮询钦定 LISTEN 首选径；md_regime_push 幂等锚不重推）。
	rg := notify.NewRegime(pool, sender, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.regime.enabled"); return v })
	go events.ListenLoop(ctx, pool, "regime_push",
		[]string{"ev_regime"}, rg.Scan)

	// T-0312 第二刀④：venue 注册请求 O 群推（recommended 翻牌 ev_venue_req
	// 帧驱动；(venue,request_at) 锚对 30d 重发各推一次）。
	vq := notify.NewVenueReq(pool, sender, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.venue_req.enabled"); return v })
	go events.ListenLoop(ctx, pool, "venue_req_push",
		[]string{"ev_venue_req"}, vq.Scan)

	// T-0314 经济案 digest（日拍首版不 LISTEN——fin_econ_lesson pending 批推
	// 【经济案】待裁统计；sched-exempt-ok 日拍节律——同 calendar 先例）。
	ec := notify.NewEcon(pool, sender, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.econ_digest.enabled"); return v })
	go func() {
		tk := time.NewTicker(24 * time.Hour) // sched-exempt-ok 日拍 digest
		defer tk.Stop()
		_ = ec.Scan(ctx) // 起拍首扫
		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				_ = ec.Scan(ctx)
			}
		}
	}()

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
