// main_test.go —— feishubridge 窄门面纯函数件（动作闭集/号畸/口令解析）
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestActFromValue(t *testing.T) {
	cases := []struct {
		name  string
		value map[string]any
		state string
		req   int64
		ok    bool
	}{
		{"fulfill-float", map[string]any{"act": "fulfill", "req": float64(42)}, "fulfilled", 42, true},
		{"reject-string", map[string]any{"act": "reject", "req": "7"}, "rejected", 7, true},
		{"dismiss", map[string]any{"act": "dismiss", "req": float64(3)}, "dismissed", 3, true},
		{"act-域外", map[string]any{"act": "hack", "req": float64(1)}, "", 1, false},
		{"req-零", map[string]any{"act": "fulfill", "req": float64(0)}, "fulfilled", 0, false},
		{"req-负", map[string]any{"act": "fulfill", "req": float64(-2)}, "fulfilled", -2, false},
		{"req-缺", map[string]any{"act": "fulfill"}, "fulfilled", 0, false},
		{"req-文本畸", map[string]any{"act": "fulfill", "req": "abc"}, "fulfilled", 0, false},
	}
	for _, c := range cases {
		st, req, ok := actFromValue(c.value)
		if st != c.state || req != c.req || ok != c.ok {
			t.Fatalf("%s: 得 (%q,%d,%v) 望 (%q,%d,%v)", c.name, st, req, ok, c.state, c.req, c.ok)
		}
	}
}

func TestCmdFromText(t *testing.T) {
	cases := []struct {
		in    string
		state string
		req   int64
		ok    bool
	}{
		{"签 42", "fulfilled", 42, true},
		{"驳 7", "rejected", 7, true},
		{"销 #3", "dismissed", 3, true},
		{"fulfill 9", "fulfilled", 9, true},
		{"签", "", 0, false},                 // 缺号
		{"签 abc", "", 0, false},             // 号畸
		{"签 -1", "", 0, false},              // 负号
		{"在吗", "", 0, false},                // 普通对话
		{"签 1 2", "", 0, false},             // 三段超形
		{"  签   5  ", "fulfilled", 5, true}, // 多空宽容
	}
	for _, c := range cases {
		st, req, ok := cmdFromText(c.in)
		if st != c.state || req != c.req || ok != c.ok {
			t.Fatalf("%q: 得 (%q,%d,%v) 望 (%q,%d,%v)", c.in, st, req, ok, c.state, c.req, c.ok)
		}
	}
}

func TestReqCard(t *testing.T) {
	raw := reqCard(9, "credential", "bybit", "", `{"ask":"需要 API key 才能接"}`)
	var card map[string]any
	if err := json.Unmarshal([]byte(raw), &card); err != nil {
		t.Fatalf("卡 JSON 畸: %v", err)
	}
	els := card["elements"].([]any)
	if len(els) != 3 {
		t.Fatalf("elements=%d 望 3（体+注+钮行）", len(els))
	}
	act := els[2].(map[string]any)["actions"].([]any)
	if len(act) != 3 {
		t.Fatalf("钮数=%d 望 3", len(act))
	}
	v := act[0].(map[string]any)["value"].(map[string]any)
	st, req, ok := actFromValue(v)
	if !ok || st != "fulfilled" || req != 9 {
		t.Fatalf("首钮 value 解回畸: %v", v)
	}
	hdr := card["header"].(map[string]any)["title"].(map[string]any)["content"]
	if hdr != "需要凭据（密钥/账号）" {
		t.Fatalf("题=%v（人话律违）", hdr)
	}
	body := els[0].(map[string]any)["text"].(map[string]any)["content"]
	if body != "**要你做**\n需要 API key 才能接" {
		t.Fatalf("体=%v", body)
	}
}

func TestOwnerFacing(t *testing.T) {
	cases := []struct {
		kind, payload string
		want          bool
	}{
		{"capital", `{}`, true},                           // 出钱=业主域
		{"credential", `{}`, true},                        // 出凭据=业主域
		{"permission", `{}`, true},                        // 放行权限=业主域
		{"constraint", `{}`, true},                        // 红线放行=业主域
		{"executor", `{}`, false},                         // executor 默认会话办
		{"holding", `{}`, false},                          // holding 默认会话办
		{"executor", `{"owner_required":true}`, true},     // 显式标业主件→推
		{"permission", `{"owner_required":false}`, false}, // 显式豁免→不推
		{"executor", `{bad json`, false},                  // 畸 payload 按类默认
	}
	for _, c := range cases {
		if got := ownerFacing(c.kind, c.payload); got != c.want {
			t.Fatalf("ownerFacing(%q,%q)=%v 望 %v", c.kind, c.payload, got, c.want)
		}
	}
}

func TestAskFromPayload(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{"ask":"人话文本"}`, "人话文本"},
		{`{"note":"note文"}`, "note文"},
		{`{"ask":"","reason":"r文"}`, "r文"},
		{`{"verdict":"判词文"}`, "判词文"},    // verdict 降档取读（executor 系写臂惯例）
		{`{"n":3}`, `{"n":3}`},          // 无 ask 键回原始文
		{`非JSON`, "非JSON"},              // 畸 payload 直通
		{`{"ask":"a","note":"n"}`, "a"}, // ask 优先
	}
	for _, c := range cases {
		if got := askFromPayload(c.in); got != c.want {
			t.Fatalf("%q: 得 %q 望 %q", c.in, got, c.want)
		}
	}
}

// TestPushPendingContigWM O-30635 水位连续性根修锚：发败行不再被后位成功行
// 跳过——水位钉死于连续成功末行，败行下醒追偿；后位已发行凭 b.sent 免重发。
// 另锚 requested_at 60s 滞后余量（Marten 型跳读防护）。
func TestPushPendingContigWM(t *testing.T) {
	ctx := context.Background()
	t.Setenv("HOME", t.TempDir()) // wmPath 落临时家目录——不脏真水位件
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 三请求行：r1 发败、r2 发成、r3 域外（kind=executor 非 ownerFacing）。
	var ids []int64
	for i, k := range []string{"capital", "capital", "executor"} {
		var id int64
		// surface 逐行异——pending_uq(venue,surface,kind) 唯一闸防同面同类双开。
		if err := pool.QueryRow(ctx, `INSERT INTO fin_condition_gap(kind,venue,surface,payload,requested_at)
			VALUES($1,'okx',$2,'{}',now()-interval '2 minutes') RETURNING id`,
			k, fmt.Sprintf("s%d", i)).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	b := &bridge{pool: pool, chatID: "c", sent: map[int64]string{}}
	calls := 0 // 首卡发败、其后发成——败行=r1
	b.sendCard = func(ctx context.Context, chatID, content string) (string, error) {
		calls++
		if calls == 1 {
			return "", fmt.Errorf("boom")
		}
		return "mid", nil
	}
	b.pushPending(ctx)
	// 断言：r1 未推进水位（readWM<r1.id 或=前值0）；r2 已发（b.sent 有）但
	// 水位不得越过 r1——wm 应仍=0（首行即败，后位行水位全不进）。
	if wm := readWM(); wm != 0 {
		t.Fatalf("发败首行后水位不应推进: wm=%d", wm)
	}
	if _, ok := b.sent[ids[1]]; !ok {
		t.Fatal("败行后位 ownerFacing 行应照发（不阻塞）")
	}
	// 二拍：r1 复发成→水位连进 r1,r2；r3 域外随水位；b.sent[r2] 免重发。
	resends := 0
	b.sendCard = func(ctx context.Context, chatID, content string) (string, error) {
		resends++
		return "mid", nil
	}
	b.pushPending(ctx)
	if wm := readWM(); wm != ids[2] {
		t.Fatalf("追偿拍水位应连进至域外末行: wm=%d want=%d", wm, ids[2])
	}
	if resends != 1 { // 仅 r1 重发——r2/r3 凭 sent/域外免发
		t.Fatalf("追偿拍应仅重发败行: send calls=%d", resends)
	}
}
