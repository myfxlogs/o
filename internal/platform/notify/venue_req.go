// venue_req.go —— T-0312 第二刀③：venue 注册请求 O 群推（state='recommended'
// 翻牌→【venue-scout】请注册——锚键=(venue,request_at)对 30d 重发各推一次；
// 发败不落锚下拍重试、关态落锚不发——regime 同语义）。
package notify

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/gen/scoutv1" // 返修①丙案：scoutv1.VenueMetrics 生成型——
	// protojson 同型出入 metrics jsonb（字段漂移编译期炸）。
	"google.golang.org/protobuf/encoding/protojson"
)

// VenueReq venue 注册请求推送件（sender 复用同装配面——不新造栈）。
type VenueReq struct {
	pool   *pgxpool.Pool
	cli    Sender
	chatID func() string
	on     func() bool
}

// NewVenueReq enabled/chatID 缺席=fail-closed 静默（on=nil/false=只 log 不发）。
func NewVenueReq(pool *pgxpool.Pool, cli Sender, chatID func() string, on func() bool) *VenueReq {
	return &VenueReq{pool: pool, cli: cli, chatID: chatID, on: on}
}

// Scan 一拍（ev_venue_req 帧调用点）：查 recommended 未推请求——零新行零推送。
func (v *VenueReq) Scan(ctx context.Context) error {
	rows, err := v.pool.Query(ctx, `SELECT c.venue, c.last_request_at, c.metrics
		FROM fin_venue_candidate c
		LEFT JOIN md_venue_req_push p
			ON p.venue=c.venue AND p.request_at=c.last_request_at
		WHERE c.state='recommended' AND c.last_request_at IS NOT NULL
		  AND p.venue IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type req struct {
		venue string
		at    time.Time
		mets  []byte
	}
	var out []req
	for rows.Next() {
		var r req
		if err := rows.Scan(&r.venue, &r.at, &r.mets); err != nil {
			return err
		}
		out = append(out, r)
	}
	for _, r := range out {
		if err := v.pushOne(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// pushOne 单件组文+推+锚（组文败=保底文不丢推送——票面禁线）。
func (v *VenueReq) pushOne(ctx context.Context, r struct {
	venue string
	at    time.Time
	mets  []byte
}) error {
	msg := venueMsg(r.venue, r.mets)
	if v.on == nil || !v.on() {
		log.Printf("[venue_req] 推送关——%s 待推（不发只锚）", r.venue)
	} else if _, err := v.cli.SendText(ctx, v.chatID(), msg); err != nil {
		return err // 发败=上抛不落锚（下拍重试）
	}
	_, err := v.pool.Exec(ctx, `INSERT INTO md_venue_req_push(venue,request_at)
		VALUES($1,$2) ON CONFLICT DO NOTHING`, r.venue, r.at)
	return err
}

// venueMsg 组文（metrics.boards 取达阈族 persist 最高者；解析败/空→保底文
// 不丢推送——票面禁线）。红线句钉死。
func venueMsg(venue string, mets []byte) string {
	fall := fmt.Sprintf("【新交易所申请】请注册 %s——度量达标件在系统台账；注册属重大事项归业主裁决", venue)
	// 返修①甲案：margin 伪阈删（生产 metrics jsonb 无 margin 字段——
	// state='recommended' 即达阈认证推送面不二次裁决阈=双真相源防；
	// 直取 persist 最高 board 组文）。struct 字段=scout board 六件真形。
	var m scoutv1.VenueMetrics
	if err := protojson.Unmarshal(mets, &m); err != nil || len(m.Boards) == 0 {
		return fall
	}
	best := 0 // persist 最高 board 直取组文（margin 伪阈已删——state 达阈认证）
	for i := range m.Boards {
		if m.Boards[i].Persist > m.Boards[best].Persist {
			best = i
		}
	}
	b := m.Boards[best]
	return fmt.Sprintf("【新交易所申请】请注册 %s：%s家族近30天实测净年化 %s%%，对照基线 %s%%（持续达标占比 %s%%）——注册属重大事项归业主裁决；凭据到位后按交易所接入流程办理。",
		venue, scoutFamily(b.Family), fmtPct(b.NetAprMid), fmtPct(b.Baseline), fmtPct(b.Persist))
}

func fmtPct(v float64) string { return strings.TrimSuffix(fmt.Sprintf("%.2f", v*100), ".00") }

// scoutFamily 侦察家族机读键人话（carry/xvenue_spread/basis——T-0441 白话令；
// 未知族原样透出不吞）。
func scoutFamily(f string) string {
	switch f {
	case "carry":
		return "单所息差"
	case "xvenue_spread":
		return "跨所价差"
	case "basis":
		return "基差"
	}
	return f
}
