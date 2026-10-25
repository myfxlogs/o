package main

import (
	"context"
	"time"

	"o/internal/gen/dashboardv1"
)

// venues 交易所在线四行（T-0088：真事实源禁假——读件缺席=红+「读件缺」，
// 禁零值绿灯；龄律 30s 黄 / 2min 红全行同判）。
func (c coreService) venues(ctx context.Context) []*dashboardv1.VenueGroup {
	age := func(q string) *dashboardv1.VenueChannel {
		var ts *time.Time
		_ = c.pool.QueryRow(ctx, q).Scan(&ts)
		return ageChan(ts)
	}
	okx := &dashboardv1.VenueGroup{Name: "okx"}
	// REST=对拍环实调成功戳（earn_parking 帧稀疏不可当活性——restOKAt 原子戳）。
	restCh := ageChan(nil)
	if t := restOKAt.Load(); t > 0 {
		ts := time.Unix(0, t).UTC()
		restCh = ageChan(&ts)
	}
	okx.Channels = append(okx.Channels,
		// 公开 WS 真流龄=md_ticks∪md_mark_index（T-0132 分级后宇宙面主源=
		// mark/index 心跳；md_funding.ts=费率事件时间可未来值仍不入）。
		withName(age(`SELECT max(ts) FROM (SELECT ts FROM md_ticks WHERE coalesce(venue,'okx')='okx'
			UNION ALL SELECT ts FROM md_mark_index) t`), "公开 WS", "md_ticks+mark_index"),
		withName(age(`SELECT max(ts) FROM md_account_events
			WHERE channel IN ('account','balance_and_position')`), "私有 WS", "md_account_events"),
		withName(restCh, "REST", "funding+savings 实调"))
	// MT5=账号粒度（v2③）：login 枚举自 mt5 事件行（汇市巡 boot 快照带出）；
	// 账号活性=该账号会话活性（汇市巡 beat+mt5 tick 流——一账号一会话）
	// 与账号帧龄取新者。缺账号面=红显「读件缺」不藏。
	var sess *time.Time
	_ = c.pool.QueryRow(ctx, `SELECT max(b) FROM (
		SELECT last_beat b FROM fin_employee_registry WHERE name='汇市巡'
		UNION ALL SELECT max(ts) FROM md_ticks WHERE venue='mt5') t`).Scan(&sess)
	mt5g := &dashboardv1.VenueGroup{Name: "mt5"}
	lrows, lerr := c.pool.Query(ctx, `SELECT data->>'login', max(ts) FROM md_account_events
		WHERE venue='mt5' AND data ? 'login' GROUP BY 1 ORDER BY 1`)
	if lerr == nil {
		defer lrows.Close()
		for lrows.Next() {
			var login string
			var at time.Time
			if lrows.Scan(&login, &at) != nil {
				continue
			}
			if sess != nil && sess.After(at) {
				at = *sess
			}
			mt5g.Channels = append(mt5g.Channels,
				withName(ageChan(&at), "户 "+login, "login "+login))
		}
	}
	if len(mt5g.Channels) == 0 {
		fb := ageChan(sess)
		fb.State, fb.Note = 2, "读件缺" // 账号枚举缺=红显不藏（sess ts 仅作旁注）
		mt5g.Channels = append(mt5g.Channels, withName(fb, "账号面", "mt5 accounts"))
	}
	return []*dashboardv1.VenueGroup{okx, mt5g}
}

func ageChan(ts *time.Time) *dashboardv1.VenueChannel {
	ch := &dashboardv1.VenueChannel{State: 2, Note: "读件缺"}
	if ts != nil {
		ch.LastTs = ts.UTC().Format(time.RFC3339)
		ch.Note = ""
		switch d := time.Since(*ts); {
		case d > 2*time.Minute:
			ch.State = 2
		case d > 30*time.Second:
			ch.State = 1
		default:
			ch.State = 0
		}
	}
	return ch
}

func withName(ch *dashboardv1.VenueChannel, name, src string) *dashboardv1.VenueChannel {
	ch.Name, ch.Channel = name, src
	return ch
}
