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
// T-0504：推送实发=研究员触发①（达阈）事件落账——锚与触发行同事务
// 双 INSERT（发败/锚败=整单回滚下拍重推不双写；推送关态只锚不发=
// 不触发——钦定语义「已向业主发请注册条件请求时」）。
func (v *VenueReq) pushOne(ctx context.Context, r struct {
	venue string
	at    time.Time
	mets  []byte
}) error {
	msg := venueMsg(r.venue, r.mets)
	sent := false
	if v.on == nil || !v.on() {
		log.Printf("[venue_req] 推送关——%s 待推（不发只锚）", r.venue)
	} else if _, err := v.cli.SendText(ctx, v.chatID(), msg); err != nil {
		return err // 发败=上抛不落锚（下拍重试）
	} else {
		sent = true
	}
	tx, err := v.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // tolerate: 回滚尽力而为——提交成败已定，败=连接断合法径
	if _, err := tx.Exec(ctx, `INSERT INTO md_venue_req_push(venue,request_at)
		VALUES($1,$2) ON CONFLICT DO NOTHING`, r.venue, r.at); err != nil {
		return err
	}
	if sent {
		if _, err := tx.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('venue哨兵','达阈','META-VENUEWATCH',$1)`, venueTriggerNote(r.venue, r.at, r.mets)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// venueTriggerNote 达阈触发行 note（研究员会话 payload——venue/family/
// 实测净年化/persist/基线/request_at；metrics 解析败=venue+request_at
// 保底形不丢事件）。
func venueTriggerNote(venue string, at time.Time, mets []byte) string {
	b := bestBoard(mets)
	if b == nil {
		return fmt.Sprintf("venue=%s request_at=%s", venue, at.UTC().Format(time.RFC3339))
	}
	return fmt.Sprintf("venue=%s family=%s net_apr_mid=%s persist=%s baseline=%s request_at=%s",
		venue, b.Family, fmtPct(b.NetAprMid), fmtPct(b.Persist),
		fmtPct(b.Baseline), at.UTC().Format(time.RFC3339))
}

// bestBoard metrics.boards 取 persist 最高者（venueMsg 组文同判据——
// 解析败/空 boards=nil）。
func bestBoard(mets []byte) *scoutv1.MetricBoard {
	var m scoutv1.VenueMetrics
	if err := protojson.Unmarshal(mets, &m); err != nil || len(m.Boards) == 0 {
		return nil
	}
	best := 0
	for i := range m.Boards {
		if m.Boards[i].Persist > m.Boards[best].Persist {
			best = i
		}
	}
	return m.Boards[best]
}

// venueMsg 组文（metrics.boards 取达阈族 persist 最高者；解析败/空→保底文
// 不丢推送——票面禁线）。红线句钉死。
func venueMsg(venue string, mets []byte) string {
	fall := fmt.Sprintf("【新交易所申请】请注册 %s——度量达标件在系统台账；注册属重大事项归业主裁决", venue)
	// 返修①甲案：margin 伪阈删（生产 metrics jsonb 无 margin 字段——
	// state='recommended' 即达阈认证推送面不二次裁决阈=双真相源防；
	// 直取 persist 最高 board 组文）。struct 字段=scout board 六件真形。
	b := bestBoard(mets)
	if b == nil {
		return fall
	}
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
