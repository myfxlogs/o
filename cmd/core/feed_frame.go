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
	// 工单两计数：OPEN 全谱(未销)与可拣面——票态单点=ops_ticket_desk 视图
	// (fin_0190 语义层,与施工桌 dispatch_worker TICKET_N 同源)。旧行内正则
	// 径只认「处置」字面行:sem 显式 act 销(fin_0191 钦定外席径)/无 工单/
	// 前缀簿记行全漏数——双口径漂移假账实案。
	_ = s.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE closed IS NOT TRUE),
		count(*) FILTER (WHERE dispatchable)
		FROM ops_ticket_desk WHERE filed_ts > now()-interval '30 days'`).Scan(&tickets, &claimable)
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
		cn := feedSeatCN[fs.seat]
		if cn == "" {
			cn = fs.seat // 未录席名原样透出（字典非白名单——宁露黑话不漏席）
		}
		switch {
		case fs.ticket > 0:
			sub := fs.subject
			if sub != "" {
				sub = "「" + sub + "」"
			}
			parts = append(parts, fmt.Sprintf("%s在修工单#%d%s(%s)",
				cn, fs.ticket, sub, frameAge(time.Since(fs.claimTs))))
		default:
			parts = append(parts, cn+"在跑(未领件)")
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
