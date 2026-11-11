// gate.go —— feishubridge 窄门纯函数面（动作闭集/口令解析/卡件
// 构建/水位件——无 I/O 可测面，main.go 只留装配+回调）。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// envDSN 从 env 文件取 KEY=VALUE（只读不回显值——凭据零入日志零入 stderr）。
func envDSN(path, key string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("env 读败 %s: %w", path, err)
	}
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, key+"=") {
			return strings.TrimPrefix(ln, key+"="), nil
		}
	}
	return "", fmt.Errorf("env %s 缺 %s", path, key)
}

// actState 钮动作→fin_condition_gap 终态闭集映射（值域外=窄门拒）。
func actState(act string) string {
	switch act {
	case "fulfill":
		return "fulfilled"
	case "reject":
		return "rejected"
	case "dismiss":
		return "dismissed"
	}
	return ""
}

// actFromValue 卡钮 value {act,req}→(终态,req_id)——值域外/号畸=拒。
func actFromValue(v map[string]any) (string, int64, bool) {
	st := actState(fmt.Sprint(v["act"]))
	var req int64
	switch n := v["req"].(type) {
	case float64:
		req = int64(n)
	case string:
		req, _ = strconv.ParseInt(n, 10, 64)
	case json.Number:
		req, _ = n.Int64()
	}
	return st, req, st != "" && req > 0
}

// cmdFromText 文本口令「签/驳/销 N」（兼容 fulfill/reject/dismiss 词）→
// (终态,req_id)——业主域等效钮的文本径。
func cmdFromText(t string) (string, int64, bool) {
	f := strings.Fields(strings.TrimSpace(t))
	if len(f) != 2 {
		return "", 0, false
	}
	verb := map[string]string{"签": "fulfill", "驳": "reject", "销": "dismiss",
		"fulfill": "fulfill", "reject": "reject", "dismiss": "dismiss"}[f[0]]
	n, _ := strconv.ParseInt(strings.TrimPrefix(f[1], "#"), 10, 64)
	if st := actState(verb); st != "" && n > 0 {
		return st, n, true
	}
	return "", 0, false
}

// kindTitle 请求 kind→人话题（T-0081 黑话挪括注尾律——业主面零 kind 名）。
var kindTitle = map[string]string{
	"capital":    "需要你出钱",
	"credential": "需要凭据（密钥/账号）",
	"permission": "需要你授权",
	"constraint": "需要你放开约束",
	"holding":    "持仓请你定",
	"executor":   "需要你操作",
}

// stateLabel 终态→人话（卡面/回执一致口径）。
var stateLabel = map[string]string{
	"fulfilled": "已办",
	"rejected":  "已驳回",
	"dismissed": "已销",
}

// askFromPayload payload→业主 ask 文本（写臂约定 ask 键载人话；
// 缺则取 note/reason，再缺回原始文——黑话挪括注尾不吞证据）。
func askFromPayload(payload string) string {
	var p map[string]any
	if json.Unmarshal([]byte(payload), &p) != nil {
		return payload
	}
	for _, k := range []string{"ask", "note", "reason"} {
		if s, ok := p[k].(string); ok && s != "" {
			return s
		}
	}
	return payload
}

// reqCard 条件请求卡（v1 interactive schema——T-0081 人话面：题=要你做
// 白话、体=ask、黑话挪括注尾小字保追溯）。
func reqCard(id int64, kind, venue, surface, payload string) string {
	title := kindTitle[kind]
	if title == "" {
		title = "需要你处理"
	}
	body := fmt.Sprintf("**要你做**\n%s", askFromPayload(payload))
	foot := fmt.Sprintf("%s · %s · %s · #%d", kind, venue, surface, id)
	btn := func(label, typ, act string) map[string]any {
		return map[string]any{"tag": "button", "type": typ,
			"text":  map[string]any{"tag": "plain_text", "content": label},
			"value": map[string]any{"act": act, "req": id}}
	}
	card := map[string]any{
		"config": map[string]any{"wide_screen_mode": true},
		"header": map[string]any{"template": "orange",
			"title": map[string]any{"tag": "plain_text", "content": title}},
		"elements": []any{
			map[string]any{"tag": "div", "text": map[string]any{"tag": "lark_md", "content": body}},
			map[string]any{"tag": "note", "elements": []any{
				map[string]any{"tag": "plain_text", "content": foot}}},
			map[string]any{"tag": "action", "actions": []any{
				btn("已办", "primary", "fulfill"),
				btn("驳回", "danger", "reject"),
				btn("销", "default", "dismiss")}},
		},
	}
	b, _ := json.Marshal(card)
	return string(b)
}

// doneCard 结案卡（无钮——终态显形）。state→人话。
func doneCard(id int64, state string) string {
	l := stateLabel[state]
	if l == "" {
		l = state
	}
	card := map[string]any{
		"config": map[string]any{"wide_screen_mode": true},
		"header": map[string]any{"template": "grey",
			"title": map[string]any{"tag": "plain_text", "content": fmt.Sprintf("请求 #%d — %s", id, l)}},
	}
	b, _ := json.Marshal(card)
	return string(b)
}

// wmPath 请求推送水位（重启不重复推——relay 系同约）。
func wmPath() string {
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".local/state/o-worker/feishubridge_last_req")
	}
	return filepath.Join(os.TempDir(), "feishubridge_last_req")
}
func readWM() int64 {
	b, _ := os.ReadFile(wmPath())
	n, _ := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	return n
}
func writeWM(id int64) {
	p := wmPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(strconv.FormatInt(id, 10)), 0o644)
}
