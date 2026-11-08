package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRender 模板槽填充断言（确定性——{{KEY}} 字面全替换零残留；
// 候选态/证据引/红线三件票面钦定形态在件）。
func TestRender(t *testing.T) {
	tpl := "# {{TID}} {{TITLE}}\n签名={{SIGNATURE}} 键={{LESION_KEY}}\n证={{EVIDENCE_IDS}}\n" +
		"> {{EVIDENCE_NOTE}}\n{{EVIDENCE_QUERY}}\n域={{DOMAIN_HINT}} 类={{KIND}} 案={{DRAFT_ID}}"
	p := payload{
		Slug: "guance-xvenue-carry", Title: "探针升件自动起草：观测盲@xvenue-carry",
		Signature: "观测盲:xvenue-carry", Kind: "观测盲", LesionKey: "xvenue-carry",
		EvidenceIDs:   []int64{7, 9},
		EvidenceNote:  "族沉默 xvenue-carry：信号活而评估零",
		EvidenceQuery: "SELECT id FROM ops_event WHERE 1=1",
		DomainHint:    "评估×信号链面",
	}
	out := render(tpl, p, "T-9001", "3")
	for _, frag := range []string{"# T-9001 探针升件自动起草：观测盲@xvenue-carry",
		"签名=观测盲:xvenue-carry", "键=xvenue-carry", "证=#7,#9",
		"SELECT id FROM ops_event", "域=评估×信号链面", "类=观测盲", "案=3"} {
		if !strings.Contains(out, frag) {
			t.Fatalf("渲染缺 %q:\n%s", frag, out)
		}
	}
	if strings.Contains(out, "{{") {
		t.Fatalf("槽位残留未填:\n%s", out)
	}
}

// TestRenderFullTemplate 真模板件全槽填充（docs/templates/ticket-draft.md
// 契面——候选态标+证据引节+红线继承模板齐=票面钦②形态）。
func TestRenderFullTemplate(t *testing.T) {
	tplb, err := readTemplate()
	if err != nil {
		t.Skipf("模板件相对径不可达（非仓根跑）: %v", err)
	}
	tpl := string(tplb)
	p := payload{
		Slug: "x", Title: "探针升件自动起草：写败@LIFECYCLE-HUNGER",
		Signature: "写败:LIFECYCLE-HUNGER", Kind: "写败", LesionKey: "LIFECYCLE-HUNGER",
		EvidenceIDs: []int64{42}, EvidenceNote: "hungertick respond 败: 桩",
		EvidenceQuery: "SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='写败' AND esc_scope='LIFECYCLE-HUNGER' ORDER BY id DESC LIMIT 20",
		DomainHint:    "lifecycle 饥饿拍写径",
	}
	out := render(tpl, p, "T-9002", "5")
	for _, frag := range []string{"状态：候选", "ops_event 行 id）：#42",
		"疑似根域", "验收锚", "红线", "结果回填", "T-9002"} {
		if !strings.Contains(out, frag) {
			t.Fatalf("票件形态缺 %q", frag)
		}
	}
	if strings.Contains(out, "{{") {
		t.Fatalf("槽位残留未填:\n%s", out)
	}
}

// GW-001 段C3 水位闸纯函数桩：判据④「WIP 闸到 5 停出票」机检形——
// 已批未领+候选 ≥5 → 草案留 pending 不占号（物料化不发生=取号不发生）。
func TestWipBlocked(t *testing.T) {
	cases := []struct {
		wip, max int
		want     bool
	}{
		{0, 5, false}, {4, 5, false}, {5, 5, true}, {6, 5, true}, {3, 3, true},
	}
	for _, c := range cases {
		if got := wipBlocked(c.wip, c.max); got != c.want {
			t.Fatalf("wipBlocked(%d,%d)=%v want %v", c.wip, c.max, got, c.want)
		}
	}
}

// 水位闸上限：缺省 5（collab 调度水位律）+env 覆写解析（非数/非正=缺省）。
func TestWipGateMax(t *testing.T) {
	t.Setenv("O_TICKETDRAFT_WIP_MAX", "")
	if got := wipGateMax(); got != 5 {
		t.Fatalf("缺省上限=%d want 5", got)
	}
	t.Setenv("O_TICKETDRAFT_WIP_MAX", "3")
	if got := wipGateMax(); got != 3 {
		t.Fatalf("覆写上限=%d want 3", got)
	}
	for _, bad := range []string{"abc", "0", "-2"} {
		t.Setenv("O_TICKETDRAFT_WIP_MAX", bad)
		if got := wipGateMax(); got != 5 {
			t.Fatalf("坏覆写 %q 回落=%d want 5", bad, got)
		}
	}
}

// 水位计数行解析：空行/空白行不计。
func TestCountLines(t *testing.T) {
	if got := countLines([]byte("T-9001\n\nT-9002\n  \n")); got != 2 {
		t.Fatalf("countLines=%d want 2", got)
	}
	if got := countLines(nil); got != 0 {
		t.Fatalf("countLines(nil)=%d want 0", got)
	}
}

// wipCount 桩测：dm_queue.sh 双面行数和（approved+candidate）——计数单源
// 不复制队列判据（假 dm_queue.sh 喂面；DM_ENV_FILE 假 env 走壳）。
func TestWipCount(t *testing.T) {
	dir := t.TempDir()
	qdir := filepath.Join(dir, "scripts", "meta")
	if err := os.MkdirAll(qdir, 0o755); err != nil {
		t.Fatal(err)
	}
	fake := "#!/usr/bin/env bash\ncase \"${1:-}\" in\n  approved) printf 'T-9001\\nT-9002\\n' ;;\n  candidate) printf 'T-9003\\n' ;;\n  *) exit 2 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(qdir, "dm_queue.sh"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DM_ENV_FILE", filepath.Join(dir, "no-env"))
	n, err := wipCount(dir)
	if err != nil {
		t.Fatalf("wipCount: %v", err)
	}
	if n != 3 {
		t.Fatalf("wipCount=%d want 3（approved 2+candidate 1）", n)
	}
	// 查败→err（fail-closed 留 pending 面）
	if err := os.WriteFile(filepath.Join(qdir, "dm_queue.sh"), []byte("#!/usr/bin/env bash\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := wipCount(dir); err == nil {
		t.Fatal("dm_queue 败应 err（fail-closed）")
	}
}
