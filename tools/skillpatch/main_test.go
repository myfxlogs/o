// main_test.go —— skillpatch 五闸+落盘桩测（T-0387 锚①后半「verdict→
// skills 落盘 diff 断言」+锚②「无锚草案拒入」+闸面矩阵）。
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

const fixtureSkill = "okx-exec"

// fixture repo 根替身（skills/<name>/SKILL.md 最小 canon 件）。
func skillRoot(t *testing.T) (root, path string) {
	t.Helper()
	root = t.TempDir()
	path = filepath.Join(root, "skills", fixtureSkill, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	doc := "---\nname: " + fixtureSkill + "\n---\n# " + fixtureSkill + "\n\n## 证据\n\n旧文。\n\n## 禁线\n\n守。\n"
	if err := os.WriteFile(path, []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}
	return root, path
}

func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	p := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), p, "fin"); err != nil {
		t.Fatal(err)
	}
	return p
}

// seedPending 插 fin_eng_lesson 行（target/skill/锚列可塑——闸面矩阵素材）。
func seedPending(t *testing.T, p *pgxpool.Pool, status, numbers string) int64 {
	t.Helper()
	var id int64
	err := p.QueryRow(context.Background(),
		`INSERT INTO fin_eng_lesson(signature,cluster_n,verify_state,numbers,status)
		 VALUES('升件:协议面:okx-exec',2,'ai_evaluated',$1::jsonb,$2) RETURNING id`,
		numbers, status).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// okNums 齐件 skill_patch 载荷（五闸全过的素材）。
func okNums() string {
	return fmt.Sprintf(`{"event":"升件","target":"skill_patch","target_skill":"%s",
		"lesion_anchor":"clOrdId@note","patch_draft":"节替换草案","evidence_ids":[11,22]}`,
		fixtureSkill)
}

func read(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestApplyAppendLandsDiff 锚①后半：approved 径桩——pending→apply→
// skills 落盘 diff（新节+证据锚行+补丁文三件断言）。
func TestApplyAppendLandsDiff(t *testing.T) {
	p := pool(t)
	root, path := skillRoot(t)
	before := read(t, path)
	id := seedPending(t, p, "pending", okNums())
	out, err := apply(context.Background(), p, root, id, fixtureSkill,
		"append", "漂移勘正", "OrderSend 响应 exec_id 字段实为 execution_id——表列改。")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	after := read(t, path)
	if after == before {
		t.Fatal("落盘零 diff")
	}
	for _, want := range []string{
		"## 漂移勘正", "漂移证据锚=ops_event#11,22", "病灶锚=clOrdId@note",
		fmt.Sprintf("pending#%d", id), "execution_id", "## 禁线",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("落盘缺 %q:\n%s", want, after)
		}
	}
	if !strings.Contains(out, "kbgen skills") {
		t.Fatalf("摘要缺索引重建指引: %s", out)
	}
	// 原文段不动（append 只追加——漂移勘正须文末、居禁线后）。
	if strings.Index(after, "## 漂移勘正") < strings.Index(after, "## 禁线") {
		t.Fatal("节序坏——漂移勘正应文末")
	}
}

// TestApplyReplaceSection replace 模：既有 ## 节正文整换+证据锚行注入，
// 邻节不动。
func TestApplyReplaceSection(t *testing.T) {
	p := pool(t)
	root, path := skillRoot(t)
	id := seedPending(t, p, "pending", okNums())
	if _, err := apply(context.Background(), p, root, id, fixtureSkill,
		"replace", "证据", "新文替换——clOrdId 定态改。"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	after := read(t, path)
	for _, want := range []string{"## 证据", "漂移证据锚=ops_event#11,22",
		"clOrdId 定态改", "## 禁线\n\n守。"} {
		if !strings.Contains(after, want) {
			t.Fatalf("replace 落盘缺 %q:\n%s", want, after)
		}
	}
	if strings.Contains(after, "旧文") {
		t.Fatal("replace 未换正文")
	}
}

// TestGateMatrix 闸面矩阵（锚②无锚拒入在列）：逐闸断言拒+文件零触。
func TestGateMatrix(t *testing.T) {
	p := pool(t)
	cases := []struct {
		name          string
		status        string
		numbers       string
		skill         string
		mode, section string
		missSkill     bool // 不建 skills 件（闸③名册实存拒）
		wantErrSubstr string
	}{
		{"闸①非pending", "verdicted", okNums(), fixtureSkill, "append", "漂移勘正", false, "非 pending"},
		{"闸②lesson型", "pending", `{"event":"升件","target":"lesson","evidence_ids":[1]}`,
			fixtureSkill, "append", "漂移勘正", false, "非 skill_patch"},
		{"闸③目标错位", "pending", okNums(), "okx-account", "append", "漂移勘正", false, "目标错位"},
		{"闸③名册缺席", "pending", okNums(), fixtureSkill, "append", "漂移勘正", true, "名册实存拒"},
		{"闸④无锚拒入", "pending",
			fmt.Sprintf(`{"target":"skill_patch","target_skill":"%s","lesion_anchor":"a@note","patch_draft":"d","evidence_ids":[]}`, fixtureSkill),
			fixtureSkill, "append", "漂移勘正", false, "无证据锚补丁拒入"},
		{"闸⑤缺病灶锚", "pending",
			fmt.Sprintf(`{"target":"skill_patch","target_skill":"%s","patch_draft":"d","evidence_ids":[1]}`, fixtureSkill),
			fixtureSkill, "append", "漂移勘正", false, "缺件"},
		{"replace节缺席", "pending", okNums(), fixtureSkill, "replace", "无此节", false, "缺席"},
		{"mode非法", "pending", okNums(), fixtureSkill, "truncate", "漂移勘正", false, "mode"},
	}
	for _, c := range cases {
		root, path := skillRoot(t)
		if c.missSkill {
			root = t.TempDir()
		}
		before := ""
		if !c.missSkill {
			before = read(t, path)
		}
		id := seedPending(t, p, c.status, c.numbers)
		if _, err := apply(context.Background(), p, root, id, c.skill,
			c.mode, c.section, "补丁文"); err == nil ||
			!strings.Contains(err.Error(), c.wantErrSubstr) {
			t.Fatalf("%s: 拒因不符 want %q got %v", c.name, c.wantErrSubstr, err)
		}
		if !c.missSkill && read(t, path) != before {
			t.Fatalf("%s: 拒写径动了文件", c.name)
		}
	}
}
