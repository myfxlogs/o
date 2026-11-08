package seatcard

// 卡面渲染——纯函数（CardState→经典卡 JSON 字符串）。卡 JSON 止于本包
// （lark API 边界，meta card.go R3 豁免面同判例）。全量重渲无 diff——
// 哈希闸在外层（cards.go）挡 PATCH。
//
// 时间显示截断至分钟（CTO 裁定③去抖闸之一）：渲染墙钟若逐秒入卡面，
// 内容永变→哈希闸失效→每拍 PATCH；截分后同分窗内容逐字节同。
// 末动 note 截 40 字（CTO 裁定①末动段钦定）。

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// tsMin 分钟截断显示（01-02 15:04）。
func tsMin(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("01-02 15:04")
}

// clipNote 末动 note 截 40 rune（rune 截防中文半字）。
func clipNote(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > 40 {
		return string(r[:40]) + "…"
	}
	if len(r) == 0 {
		return "—"
	}
	return string(r)
}

// statusTemplate 班态→卡头模板色（五标五色——异常红/挂起橙/在岗绿/
// 作业中蓝/退班灰；表外回 default）。
func statusTemplate(status string) string {
	switch status {
	case StatusOnDuty:
		return "green"
	case StatusWorking:
		return "blue"
	case StatusBlocked:
		return "orange"
	case StatusError:
		return "red"
	case StatusOff:
		return "grey"
	}
	return "blue"
}

// elapsed 已运行时长（worker 行——截分显示，同哈希闸截断律）。
func elapsed(since, now time.Time) string {
	if since.IsZero() {
		return ""
	}
	d := now.Sub(since)
	if d < 0 {
		d = 0
	}
	if d < time.Hour {
		return fmt.Sprintf("已 %d 分", int64(d.Minutes()))
	}
	return fmt.Sprintf("已 %.1f 时", d.Hours())
}

// Render CardState→卡 JSON。四段钦定（CTO 裁定①）：头段席名+班态字标
// （模板色随班态）；在办段=最近领/收/移/裁/派行（无活件=「待派」）；
// 队列段=待裁/待派深；末动段=最新行 ts+event+note 截 40 字。
// 施工席批量班不糊律：并行活件列 N 行（tid+model+elapsed），不开票级卡。
func Render(cs CardState) string {
	working := "待派"
	if cs.Working != nil {
		working = fmt.Sprintf("%s %s（%s）", cs.Working.Ticket, cs.Working.Event, tsMin(cs.Working.Ts))
	}
	els := []any{
		mdLine(fmt.Sprintf("**班态**：%s", cs.Status)),
		mdLine(fmt.Sprintf("**在办**：%s", working)),
	}
	for _, w := range cs.Workers {
		row := w.TID
		if w.Model != "" {
			row += " · " + w.Model
		}
		if e := elapsed(w.Since, cs.Now); e != "" {
			row += " · " + e
		}
		els = append(els, mdLine("**并行活件**："+row))
	}
	els = append(els,
		mdLine(fmt.Sprintf("**队列**：待裁 %d · 待派 %d", cs.QueueReview, cs.QueueApproved)),
	)
	if cs.Latest != nil {
		els = append(els, mdLine(fmt.Sprintf("**末动**：%s %s %s——%s",
			tsMin(cs.Latest.Ts), cs.Latest.Event, cs.Latest.Ticket, clipNote(cs.Latest.Note))))
	} else {
		els = append(els, mdLine("**末动**：—"))
	}
	card := map[string]any{
		"config": map[string]any{"wide_screen_mode": true},
		"header": map[string]any{
			"template": statusTemplate(cs.Status),
			"title": map[string]any{
				"tag":     "plain_text",
				"content": "席位活卡 · " + cs.Seat,
			},
		},
		"elements": els,
	}
	b, err := json.Marshal(card)
	if err != nil {
		return `{"elements":[{"tag":"div","text":{"tag":"plain_text","content":"席位卡渲染失败"}}]}`
	}
	return string(b)
}

// mdLine 一行 lark_md div（meta card.go 同构——note 用户源文本由 lark_md
// 渲染，JSON 层已转义）。
func mdLine(text string) map[string]any {
	return map[string]any{
		"tag":  "div",
		"text": map[string]any{"tag": "lark_md", "content": text},
	}
}
