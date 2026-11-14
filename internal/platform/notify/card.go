// card.go —— 飞书 interactive 卡 JSON 微装配（系统频富卡径——业主令 2026-11-15：
// system 流走富卡片）。只出 JSON 字符串，发送走 FeishuClient.CreateInteractive。
package notify

import (
	"context"
	"encoding/json"
	"strings"
)

// CardSender 富卡发送口（FeishuClient 满足——CreateInteractive 在册）；
// Alerter/系统面断言升级用，不满足者回落 SendText。
type CardSender interface {
	CreateInteractive(ctx context.Context, chatID, content string) (string, error)
}

// sysCard 装 interactive 卡（header 色+标题+markdown 体）。色板=飞书 template
// 枚举（blue/green/red/orange/grey）。
func sysCard(color, title, bodyMD string) string {
	card := map[string]any{
		"config": map[string]any{"wide_screen_mode": true},
		"header": map[string]any{
			"template": color,
			"title":    map[string]string{"tag": "plain_text", "content": title},
		},
		"elements": []any{
			map[string]any{"tag": "div",
				"text": map[string]string{"tag": "lark_md", "content": bodyMD}},
		},
	}
	b, _ := json.Marshal(card) // 全字面量 map 装配——marshal 无失败路径
	return string(b)
}

// AlertCard 告警级系统卡（actable=red 业主可处置，否则 orange 观测级）。
func AlertCard(title, body string, actable bool) string {
	color := "orange"
	if actable {
		color = "red"
	}
	return sysCard(color, title, body)
}

// StatusCard 在办面板卡（blue=有活在干，grey=无活/下班）。
func StatusCard(lines []string, active bool) string {
	color := "grey"
	if active {
		color = "blue"
	}
	return sysCard(color, "O·在办", strings.Join(lines, "\n"))
}
