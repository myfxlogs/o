// push.go —— 未决请求推送环（main.go >300 棘轮拆分）：pending 扫描→交互卡
// →连续水位推进（O-30635：发败不跳号+requested_at 滞后余量）。
package main

import (
	"context"
	"log"
	"time"
)

// pushPending 未决请求→交互卡（水位=连续成功末行——发败行不再被后位行
// 跳过：水位钉死于败行前，败行下醒追偿、后位已发行凭 b.sent 免重发）。
// requested_at 滞后余量 60s：id 分配序≠提交序（Marten 型——低 id 行未提交
// 期间高 id 行先入水位=低位行永跳）；请求盒非实时径，滞后无感。
func (b *bridge) pushPending(ctx context.Context) {
	wm := readWM()
	rows, err := b.pool.Query(ctx, `SELECT id,kind,coalesce(venue,''),coalesce(surface,''),payload::text
		FROM fin_condition_gap WHERE state='pending' AND id>$1
		  AND requested_at<now()-interval '60 seconds' ORDER BY id`, wm)
	if err != nil {
		log.Printf("feishubridge: pending 查败: %v", err)
		return
	}
	defer rows.Close()
	send := b.sendCard
	if send == nil {
		send = b.fs.CreateInteractive
	}
	blocked := false // 发败闸——水位停于连续成功末行
	for rows.Next() {
		var id int64
		var kind, venue, surface, payload string
		if rows.Scan(&id, &kind, &venue, &surface, &payload) != nil {
			continue
		}
		if _, ok := b.sent[id]; ok {
			if !blocked {
				writeWM(id) // 本程已发重扫——水位续进免重发
			}
			continue
		}
		if !ownerFacing(kind, payload) {
			// 域外件水位可进：原「水位不进」只保尾段 flag 后翻可推——发送序下
			// 其后行必然已跳过该语义，收敛为随水位（判词域闭集变更罕）。
			if !blocked {
				writeWM(id)
			}
			continue
		}
		mid, err := send(ctx, b.chatID, reqCard(id, kind, venue, surface, payload))
		if err != nil {
			log.Printf("feishubridge: 卡发败 req=%d: %v", id, err)
			blocked = true // 水位钉死——后位行照发不推进，败行下响追偿
			continue
		}
		b.sent[id] = mid
		if !blocked {
			writeWM(id)
		}
	}
}

// listenPending condreq_pending 敲门环（fin_0164 触发器——INSERT/UPDATE
// 至 pending 即醒；开机先扫底覆盖停机期漏件；断流退避重连）。
func (b *bridge) listenPending(ctx context.Context) {
	b.pushPending(ctx) // 开机扫底——停机期漏件追偿
	for ctx.Err() == nil {
		c, err := b.pool.Acquire(ctx)
		if err != nil {
			time.Sleep(3 * time.Second) // sched-exempt-ok 断流重连退避（mechwake 同约——非轮询数据）
			continue
		}
		if _, err := c.Exec(ctx, "LISTEN condreq_pending"); err != nil {
			c.Release()
			time.Sleep(3 * time.Second) // sched-exempt-ok 同上
			continue
		}
		for ctx.Err() == nil {
			if _, err := c.Conn().WaitForNotification(ctx); err != nil {
				break // 断流→出内环重连
			}
			b.pushPending(ctx) // notify 只敲门——读仍走表正源
		}
		c.Release()
		time.Sleep(3 * time.Second) // sched-exempt-ok 断流重连退避
	}
}
