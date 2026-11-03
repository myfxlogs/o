// venue_req.go —— T-0312 第二刀③：venue 注册请求 O 群推（state='recommended'
// 翻牌→【venue-scout】请注册——锚键=(venue,request_at)对 30d 重发各推一次；
// 发败不落锚下拍重试、关态落锚不发——regime 同语义）。
package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
	fall := fmt.Sprintf("【venue-scout】请注册 %s（state→recommended，度量件查 fin_venue_candidate.metrics）——红线件：新平台注册归业主裁决", venue)
	// 返修①甲案：margin 伪阈删（生产 metrics jsonb 无 margin 字段——
	// state='recommended' 即达阈认证推送面不二次裁决阈=双真相源防；
	// 直取 persist 最高 board 组文）。struct 字段=scout board 六件真形。
	var m struct {
		Boards []struct {
			Family         string   `json:"family"`
			NetAprMid      *float64 `json:"net_apr_mid"`
			Persist        *float64 `json:"persist"`
			Baseline       *float64 `json:"baseline"`
			BaselineSource string   `json:"baseline_source"`
			TopN           *int     `json:"top_n"`
		} `json:"boards"`
	}
	if err := json.Unmarshal(mets, &m); err != nil || len(m.Boards) == 0 {
		return fall
	}
	best := -1
	for i, b := range m.Boards {
		if b.NetAprMid == nil || b.Persist == nil || b.Baseline == nil {
			continue
		}
		if best < 0 || *b.Persist > *m.Boards[best].Persist {
			best = i
		}
	}
	if best < 0 {
		return fall
	}
	b := m.Boards[best]
	return fmt.Sprintf("【venue-scout】请注册 %s：%s 30d 实测净年化 %v%% vs 基线 %v%%（持续占比 %v%%）——红线件：新平台注册归业主裁决。凭据到手后落 skills/venue-scout-%s 能力账径。",
		venue, b.Family, fmtPct(*b.NetAprMid), fmtPct(*b.Baseline), fmtPct(*b.Persist), venue)
}

func fmtPct(v float64) string { return strings.TrimSuffix(fmt.Sprintf("%.2f", v*100), ".00") }
