package hunger

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T-0387 锚①：skill 签名聚类→skill_patch pending 落（target/target_skill/
// lesion_anchor/patch_draft+evidence_ids 齐）；锚③：skill_patch 日额 ≤2
// （越额簇降级 lesson 不丢信号）；锚④：非 skill 签名簇照旧 lesson 面回归；
// 名册机钉（册键⊆skills/ 实存目录+坑引⊆条目 skills: 标签双向对拍）。

// TestMinerSkillSigProducesPatch 锚①前半：skill 签名簇→skill_patch pending
// 载荷钦件齐（目标/病灶锚/补丁草案/证据锚列）。
func TestMinerSkillSigProducesPatch(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	note := "OrderSend 返回体 exec_id 字段缺——canon 载字段名与实况漂移"
	seedOps(t, pool, "决策席", "升件", "T-0400", note, "协议面", "mt5-trading-mutating")
	seedOps(t, pool, "决策席", "升件", "T-0400", note+"（再犯）", "协议面", "mt5-trading-mutating")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 1 || pendingCount(t, pool) != 1 {
		t.Fatalf("skill 簇未产: n=%d pending=%d", n, pendingCount(t, pool))
	}
	var num []byte
	if err := pool.QueryRow(ctx, `SELECT numbers FROM fin_eng_lesson
		WHERE signature='升件:协议面:mt5-trading-mutating'`).Scan(&num); err != nil {
		t.Fatalf("pending 缺: %v", err)
	}
	var p minerPayload
	if err := json.Unmarshal(num, &p); err != nil {
		t.Fatalf("numbers 非机读: %v", err)
	}
	if p.Target != "skill_patch" || p.TargetSkill != "mt5-trading-mutating" ||
		p.LesionAnchor == "" || p.PatchDraft == "" || len(p.EvidenceIDs) != 2 {
		t.Fatalf("skill_patch 载荷不齐: %+v", p)
	}
	if !strings.Contains(p.LesionAnchor, "mt5-trading-mutating@esc_scope") {
		t.Fatalf("病灶锚缺命中面: %s", p.LesionAnchor)
	}
	if !strings.Contains(p.PatchDraft, "skills/mt5-trading-mutating/SKILL.md") {
		t.Fatalf("补丁草案缺目标: %s", p.PatchDraft)
	}
	// 留账行携 target 标（canon 产线可视面）。
	var m int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='教训矿工' AND note LIKE '%target=skill_patch skill=mt5-trading-mutating%'`).Scan(&m); err != nil || m != 1 {
		t.Fatalf("留账行 target 标缺: m=%d err=%v", m, err)
	}
}

// TestMinerSkillPatchCapTwo 锚③：skill_patch 日额 ≤2——三 skill 簇同窗，
// 两额产 patch 型，第三簇降级 lesson（信号不丢、防灌闸守）。
func TestMinerSkillPatchCapTwo(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	for _, scope := range []string{"ctx-sources", "mt5-quotes-history", "okx-exec"} {
		seedOps(t, pool, "决策席", "升件", "T-0401", scope+" 面端点漂移", "协议面", scope)
		seedOps(t, pool, "决策席", "升件", "T-0401", scope+" 面端点漂移再犯", "协议面", scope)
	}
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 3 || pendingCount(t, pool) != 3 {
		t.Fatalf("产数坏: n=%d pending=%d", n, pendingCount(t, pool))
	}
	var patch, lesson int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_eng_lesson
		WHERE numbers->>'target'='skill_patch'`).Scan(&patch); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_eng_lesson
		WHERE numbers->>'target'='lesson'`).Scan(&lesson); err != nil {
		t.Fatal(err)
	}
	if patch != minerPatchDailyCap || lesson != 1 {
		t.Fatalf("日额闸失效: patch=%d lesson=%d", patch, lesson)
	}
	// 降级件=字典序末簇 okx-exec（定排可复现——额尽走 lesson 面）。
	var tgt string
	if err := pool.QueryRow(ctx, `SELECT numbers->>'target' FROM fin_eng_lesson
		WHERE signature='升件:协议面:okx-exec'`).Scan(&tgt); err != nil || tgt != "lesson" {
		t.Fatalf("降级面坏: tgt=%s err=%v", tgt, err)
	}
}

// TestMinerNonSkillClusterLessonTarget 锚④回归：非 skill 签名簇照旧
// target=lesson 且 skill_patch 三件空——T-0385 面零回归。
func TestMinerNonSkillClusterLessonTarget(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	seedOps(t, pool, "决策席", "升件", "T-0376", "hungertick detect 败", "写败", "LIFECYCLE-HUNGER")
	seedOps(t, pool, "决策席", "升件", "T-0376", "hungertick respond 败", "写败", "LIFECYCLE-HUNGER")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 1 {
		t.Fatalf("lesson 簇未产: n=%d", n)
	}
	var p minerPayload
	var num []byte
	if err := pool.QueryRow(ctx, `SELECT numbers FROM fin_eng_lesson`).Scan(&num); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(num, &p); err != nil {
		t.Fatal(err)
	}
	if p.Target != "lesson" || p.TargetSkill != "" || p.PatchDraft != "" {
		t.Fatalf("lesson 面污染: %+v", p)
	}
}

// TestSkillRosterPinned 名册机钉（canon 写域名册对拍）：①册键全为
// skills/<name>/SKILL.md 实存目录；②坑引 id 全为工程经验库实存条目且
// frontmatter skills: 标签含名册 skill（双向对拍——册错位/条目删=红）。
func TestSkillRosterPinned(t *testing.T) {
	root := repoRoot(t)
	for skill, sig := range skillSigRoster {
		if _, err := os.Stat(filepath.Join(root, "skills", skill, "SKILL.md")); err != nil {
			t.Fatalf("名册 skill 非实存 canon 目录: %s", skill)
		}
		for _, pit := range sig.pits {
			b, err := os.ReadFile(filepath.Join(root, "docs", "工程经验库", "条目", pit+".md"))
			if err != nil {
				t.Fatalf("坑引条目缺: %s（%s）", pit, skill)
			}
			head := string(b)
			if i := strings.Index(head, "\n---\n"); i > 0 {
				head = head[:i]
			}
			if !strings.Contains(head, skill) {
				t.Fatalf("坑引错位: %s frontmatter skills 不含 %s", pit, skill)
			}
		}
	}
}

// repoRoot 测试件上溯 go.mod 锚定（worktree/模块内定位——cwd 无关）。
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		p := filepath.Dir(dir)
		if p == dir {
			t.Fatal("repo root 缺（go.mod 上溯尽）")
		}
		dir = p
	}
}

// TestHasTokenBoundaries 词界匹配单测（横线当界——连缀中/字母粘不中）。
func TestHasTokenBoundaries(t *testing.T) {
	cases := []struct {
		text, tok string
		want      bool
	}{
		{"OrderSend 返回字段漂移", "OrderSend", true},
		{"支t/0400-okx-exec-drift 收口", "okx-exec", true},
		{"xokx-exec", "okx-exec", false},
		{"OrderSendV2 新增", "OrderSend", false},
		{"pre_OrderSend", "OrderSend", false},
		{"（OrderSend)", "OrderSend", true},
		{"mt5-xvenue-carry-paper", "mt5-xvenue-carry", true},
		{"clOrdId 定态三分支", "clOrdId", true},
		{"myclOrdId", "clOrdId", false},
	}
	for _, c := range cases {
		if got := hasToken(c.text, c.tok); got != c.want {
			t.Fatalf("hasToken(%q,%q)=%v want %v", c.text, c.tok, got, c.want)
		}
	}
}

// TestMatchSkillSigScoring 目标择=特异 token 命中数最多者（多 token 同
// skill 胜单 token 他 skill；平手字典序定排）。
func TestMatchSkillSigScoring(t *testing.T) {
	c := &mineCluster{sig: "升件:协议面:x", rows: []minerRow{
		{note: "OrderSend 与 OrderModify 双字段漂移 okx-exec 亦提及"},
	}}
	skill, hits := matchSkillSig(c)
	if skill != "mt5-trading-mutating" {
		t.Fatalf("目标择错: skill=%s hits=%+v", skill, hits)
	}
	var toks []string
	for _, h := range hits {
		toks = append(toks, h.token)
	}
	if strings.Join(toks, ",") != "OrderModify,OrderSend" {
		t.Fatalf("命中定排坏: %v", toks)
	}
}
