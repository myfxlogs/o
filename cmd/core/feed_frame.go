// feed_frame.go —— 在办帧：「此刻谁在干什么」白话一行，冠于 O·动 每批
// （业主令 2026-11-15：主动报在办+说人话非流水黑话）。
// 数据面：ops_headless 心跳<100s=在飞（会话 60s 一拍,死透自然陈旧
// ——心跳按日键行在而活性过期）；认领行<2h=在办租约（与工单桌同契）；
// 行内工单#N→取票题白话显形。查败=帧面降级省行（帧=解说非主载,
// 不阻推送主径）。
package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type frameSeat struct {
	seat    string // 唤知席/施工席/审计席/独白席
	ticket  int64  // 认领工单 id（0=在跑未领）
	subject string // 票题（工单行 note 首段）
	claimTs time.Time
}

var frameTicketRe = regexp.MustCompile(`工单#([0-9]+)`)

// feedFrame 组帧——活跃席+未办两计数一行白话。
func (s *feedPush) feedFrame(ctx context.Context) string {
	seats, err := s.frameSeats(ctx)
	if err != nil {
		return "" // 降级省行不阻主径
	}
	var esc, tickets, claimable int64
	_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_esc_pending`).Scan(&esc)
	// 工单两计数：OPEN 全谱(未销)与可拣面(剔挂起缓期窗/在办认领——
	// 与施工桌 dispatch_worker TICKET_N 同契)。
	_ = s.pool.QueryRow(ctx, `WITH open_t AS (
		SELECT t.id,t.ts,t.note FROM ops_event t
		WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='处置'
		    AND x.ts>t.ts AND x.note ~ ('工单#'||t.id::text||'([^0-9]|$)')
		    AND x.note NOT LIKE '%挂起条件=%'
		    AND x.note !~ ('工单#'||t.id::text||' ?接续')))
		SELECT (SELECT count(*) FROM open_t),
		  (SELECT count(*) FROM open_t t
		    WHERE NOT EXISTS (SELECT 1 FROM ops_event s WHERE s.event='处置'
		      AND s.ts>t.ts AND s.note ~ ('工单#'||t.id::text||'([^0-9]|$)')
		      AND s.note LIKE '%挂起条件=%'
		      AND s.ts + COALESCE((substring(s.note from '挂起期限=([0-9]+)h'))::int,
		        COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,72))*interval '1 hour' > now())
		    AND NOT EXISTS (SELECT 1 FROM ops_event c WHERE c.event='认领'
		      AND c.ts>t.ts AND c.ts>now()-interval '2 hours'
		      AND c.note ~ ('工单#'||t.id::text||'([^0-9]|$)')))`).Scan(&tickets, &claimable)
	return frameText(seats, esc, tickets, claimable)
}

// frameSeats 在飞席+其认领票——心跳<100s 取活（死透行陈旧自然剔除）。
func (s *feedPush) frameSeats(ctx context.Context) ([]frameSeat, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT seat FROM ops_headless
		WHERE state='alive' AND last_activity>now()-interval '100 seconds'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var seats []frameSeat
	for rows.Next() {
		var fs frameSeat
		if err := rows.Scan(&fs.seat); err != nil {
			return seats, err
		}
		seats = append(seats, fs)
	}
	if err := rows.Err(); err != nil {
		return seats, err
	}
	for i := range seats { // 各席末认领<2h→票id+题+认领龄
		var note string
		err := s.pool.QueryRow(ctx, `SELECT note,ts FROM ops_event
			WHERE event='认领' AND seat=$1 AND ts>now()-interval '2 hours'
			ORDER BY id DESC LIMIT 1`, seats[i].seat).Scan(&note, &seats[i].claimTs)
		if err != nil {
			continue // 无认领=在跑未领
		}
		if m := frameTicketRe.FindStringSubmatch(note); m != nil {
			fmt.Sscanf(m[1], "%d", &seats[i].ticket)
			_ = s.pool.QueryRow(ctx, `SELECT left(note,30) FROM ops_event
				WHERE id=$1`, seats[i].ticket).Scan(&seats[i].subject)
		}
	}
	return seats, nil
}

// frameText 帧文纯件——白话一行（在办: <谁干什么> | 信号N · 工单M(可拣K)）。
func frameText(seats []frameSeat, esc, tickets, claimable int64) string {
	var parts []string
	for _, fs := range seats {
		switch {
		case fs.ticket > 0:
			sub := fs.subject
			if sub != "" {
				sub = "「" + sub + "」"
			}
			parts = append(parts, fmt.Sprintf("%s在修工单#%d%s(%s)",
				fs.seat, fs.ticket, sub, frameAge(time.Since(fs.claimTs))))
		default:
			parts = append(parts, fs.seat+"在跑(未领件)")
		}
	}
	head := "在办: " + strings.Join(parts, " | ")
	if len(parts) == 0 {
		head += "全员空闲"
	}
	return fmt.Sprintf("%s | 信号%d · 工单%d(可拣%d)", head, esc, tickets, claimable)
}

// frameAge 龄白话（<1分=刚领;<60分=N分;≥1h=NhM分）。
func frameAge(d time.Duration) string {
	if d < time.Minute {
		return "刚领"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%d分", int(d.Hours()), int(d.Minutes())%60)
}
