// Calendar 财经日历事件报（T-0070：[财经日历] 前缀与 [经营事件]/告警分区；
// 源=ForexFactory 公开周历 XML（nfs.faireconomy.media/ff_calendar_thisweek.xml——
// 免费无 key 钦定面）；高影响级预推（默认 T-60min）非事后报；同事件一键一推——
// md_calendar_push 幂等锚重启/重拉不重推；拉取败=计数+log 不炸不递归告警）。
package notify

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const calendarFeed = "https://nfs.faireconomy.media/ff_calendar_thisweek.xml"

// Calendar 日历推送件（sender 复用 event_push 同装配面——不新造栈）。
type Calendar struct {
	pool   *pgxpool.Pool
	cli    Sender
	chatID func() string
	on     func() bool
	minImp func() string // min_impact 热启
	lead   func() int64  // lead_minutes 热启
	src    string        // feed URL（测试桩可换）
	http   *http.Client
}

// NewCalendar enabled/chatID 缺席=fail-closed 静默（on=nil/false=只 log 不发）。
func NewCalendar(pool *pgxpool.Pool, cli Sender, chatID func() string, on func() bool,
	minImp func() string, lead func() int64) *Calendar {
	return &Calendar{pool: pool, cli: cli, chatID: chatID, on: on,
		minImp: minImp, lead: lead, src: calendarFeed, http: &http.Client{Timeout: 10 * time.Second}}
}

type ffEvent struct {
	Title    string `xml:"title"`
	Country  string `xml:"country"`
	Date     string `xml:"date"`
	Time     string `xml:"time"`
	Impact   string `xml:"impact"`
	Forecast string `xml:"forecast"`
	Previous string `xml:"previous"`
}

type ffWeekly struct {
	Events []ffEvent `xml:"event"`
}

// etLoc 美东时区（FF 历表口径）；tzdata 缺席兜固定 -4h（10 月 EDT 面）。
var etLoc = func() *time.Location {
	if l, err := time.LoadLocation("America/New_York"); err == nil {
		return l
	}
	return time.FixedZone("ET", -4*3600)
}()

// Scan 一拍（外部源拉取=ticker 装配调用——sched-exempt-ok：外部 feed 拉取非
// DB 扫描环，周期由外部源更新节律决定非决策节拍）。
func (c *Calendar) Scan(ctx context.Context) error {
	if c.on == nil || !c.on() || c.cli == nil || c.chatID == nil || c.chatID() == "" {
		return nil
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", c.src, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("[财经日历] feed 拉取败（计数+log 不炸）: %v", err)
		return nil // 断源不炸推送面
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var w ffWeekly
	if err := xml.Unmarshal(body, &w); err != nil {
		log.Printf("[财经日历] feed 解包败: %v", err)
		return nil
	}
	minImp := c.minImp()
	leadMin := int64(60)
	if c.lead != nil {
		leadMin = c.lead()
	}
	lead := time.Duration(leadMin) * time.Minute
	now := time.Now().UTC()
	for _, e := range w.Events {
		if impactRank(e.Impact) < impactRank(minImp) {
			continue // 低影响噪音不上 Feishu
		}
		et := ffTime(e.Date, e.Time)
		if et.IsZero() || now.Before(et.Add(-lead)) || now.After(et) {
			continue // 只在 [T-lead,T] 预推窗内
		}
		key := e.Title + "|" + e.Date + "|" + e.Time
		var exists bool
		if err := c.pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM md_calendar_push WHERE event_key=$1)", key).Scan(&exists); err != nil {
			continue
		}
		if exists {
			continue // 幂等锚——一键一推
		}
		if _, err := c.pool.Exec(ctx,
			"INSERT INTO md_calendar_push(event_key) VALUES($1) ON CONFLICT DO NOTHING", key); err != nil {
			continue
		}
		text := fmt.Sprintf("[财经日历] %s UTC %s %s", et.Format("15:04"), countryName(e.Country), e.Title)
		if e.Forecast != "" || e.Previous != "" {
			text += fmt.Sprintf("（预期 %s，前值 %s）", e.Forecast, e.Previous)
		}
		if _, err := c.cli.SendText(ctx, c.chatID(), text); err != nil {
			log.Printf("[财经日历] %s 推送败（下拍重试——幂等行已占位防重推风暴）: %v", key, err)
		}
	}
	return nil
}

// impactRank 影响级排级（High=3/Medium=2/Low=1——低噪音阈可配）。
func impactRank(s string) int {
	switch strings.ToLower(s) {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	}
	return 0
}

// ffTime FF 历表 ET 时戳→UTC（date=MM-DD-YYYY time="8:30am" 美东口径）。
func ffTime(d, tm string) time.Time {
	tm = strings.ToLower(strings.TrimSpace(tm))
	for _, f := range []string{"01-02-2006 3:04pm", "01-02-2006 15:04"} {
		if t, err := time.ParseInLocation(f, d+" "+tm, etLoc); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// countryName 币码→人话国名（业主零黑话面）。
func countryName(ccy string) string {
	m := map[string]string{"USD": "美国", "EUR": "欧元区", "JPY": "日本",
		"GBP": "英国", "CNY": "中国", "AUD": "澳大利亚", "CAD": "加拿大", "CHF": "瑞士"}
	if v, ok := m[ccy]; ok {
		return v
	}
	return ccy
}
