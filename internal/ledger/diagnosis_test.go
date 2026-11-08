// diagnosis_test.go —— T-0431 红桩+禁线断言：
//
//	红桩（票面钦定）：pos34 案卷 fixture（attempts=5+bare_bag+假失衡证据锚）
//	→诊断草案命中「单位语义错配」族假设（非必点名同词——族命中即可，
//	判据=matchDiagFamilyText 词谱机读形写进桩）；
//	禁线（票面钦定）：诊断席零 venue/fin 写径——代码面核（Go 本件 import
//	面+写径语句面+裁定席 SKILL 第四分支落面白名单三重机检）。
package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// pos34Facts 案卷 fixture（票面钦定三件：attempts=5+bare_bag+假失衡证据锚
// ——数字=pos34 生产实证形：CORE 袋 5737.95 枚≈$74.89、settled −24.67/50
// 名义=−49%、delta_drift 幻觉杀环）。
func pos34Facts() diagCaseFacts {
	return diagCaseFacts{
		PosID: 34, ExecID: 9001, InstID: "CORE-USDT-SWAP",
		Anomalies:  []string{"fast_close", "settled_over", "bare_bag"},
		BareBagUSD: 74.89, BareBagQty: 5737.95,
		Attempts: 5, Trigger: "delta_drift", SettledPct: -49.34,
	}
}

// 红桩：pos34 案卷重放→单位语义错配族命中（族命中判据写进桩）。
func TestDiagFamilySeed_pos34Replay(t *testing.T) {
	f := pos34Facts()
	if !diagnosisWorthy(f) {
		t.Fatal("pos34 案卷未过诊断闸（三案卷源应齐命中）")
	}
	fams := matchDiagFamilies(f)
	if len(fams) != 1 || fams[0].ID != "unit_mismatch" {
		t.Fatalf("族谱命中=%v（应恰 unit_mismatch）", famIDs(fams))
	}
	nums := buildDiagTaskNumbers(f, json.RawMessage(`{"anomalies":["fast_close","settled_over","bare_bag"]}`))
	seeds, err := json.Marshal(nums["seeds"])
	if err != nil {
		t.Fatal(err)
	}
	var ss []diagSeed
	if err := json.Unmarshal(seeds, &ss); err != nil {
		t.Fatal(err)
	}
	if len(ss) != 1 || ss[0].Family != "unit_mismatch" {
		t.Fatalf("种子族=%+v（应 unit_mismatch）", ss)
	}
	// 族命中判据（票面「非必点名同词——族命中即可」的机读形）：草案文本
	// 过词谱即族命中——种子假设+修法面拼文过 matchDiagFamilyText。
	draft := ss[0].Hypothesis + " " + ss[0].FixSurface
	if got := matchDiagFamilyText(draft); len(got) == 0 || got[0] != "unit_mismatch" {
		t.Fatalf("草案文本族命中=%v（应 unit_mismatch）", got)
	}
	// 非必点名同词实证：换述草案（不含族名/族名词）仍词谱命中。
	if got := matchDiagFamilyText("开腿美元额被当币量发单——sz_unit 两义混用，quote 与 base 不分，平仓只清零头"); len(got) == 0 || got[0] != "unit_mismatch" {
		t.Fatalf("换述草案族命中=%v（应 unit_mismatch——非必点名同词）", got)
	}
	// 证据锚三件齐：腿对拍/守卫环/账实残袋（假失衡证据锚入任务）。
	// 占位符已实化（裁定席会话可直接执行——非模板裸文）。
	for _, w := range []string{"fin_order_leg", "fin_guard_close", "fin_holding"} {
		hit := false
		for _, q := range ss[0].Evidence {
			if strings.Contains(q, w) {
				hit = true
			}
			if strings.Contains(q, "{pos}") || strings.Contains(q, "{exec}") || strings.Contains(q, "{inst}") {
				t.Fatalf("证据锚占位未实化: %s", q)
			}
		}
		if !hit {
			t.Fatalf("证据锚缺 %s", w)
		}
	}
	// 案卷源两件（pos34 触发类 delta_drift 在认知集——trigger_unknown 不
	// 触发；账实差+守卫环双源在案）+案卷原貌直通。
	srcs, _ := nums["sources"].([]string)
	if len(srcs) != 2 {
		t.Fatalf("案卷源=%v（应两件）", srcs)
	}
	for _, w := range []string{"账实差", "守卫环"} {
		if !strings.Contains(strings.Join(srcs, "×"), w) {
			t.Fatalf("案卷源缺 %s: %v", w, srcs)
		}
	}
	if string(nums["case"].(json.RawMessage)) != `{"anomalies":["fast_close","settled_over","bare_bag"]}` {
		t.Fatalf("案卷原貌未直通: %v", nums["case"])
	}
}

func famIDs(fs []diagFamily) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.ID)
	}
	return out
}

// 市况形不产诊断任务（纯幅超阈/秒级开平——裁定席三分支消化，不烧诊断额度）。
func TestDiagNotWorthy_pureMarket(t *testing.T) {
	market := diagCaseFacts{PosID: 1, ExecID: 2, InstID: "BTC-USDT-SWAP",
		Anomalies: []string{"settled_over"}, SettledPct: -30, Attempts: 0}
	if diagnosisWorthy(market) {
		t.Fatal("纯市况案过诊断闸（应不生成）")
	}
	if fams := matchDiagFamilies(market); len(fams) != 0 {
		t.Fatalf("纯市况案族命中=%v（应空）", famIDs(fams))
	}
}

// 族词谱匹配器单元（红桩判据机读形本体）。
func TestDiagFamilyTextMatcher(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"sz_unit=quote 的 leg.sz 被当 base 币量消费", true},
		{"开仓金额被当成币量发出——quote/base 两义混用", true},
		{"双制量字段消费面不认制", true},
		{"行情暴跌触发止损，市况使然", false},
		{"残袋被 Earn 收息——归置决策待裁", false},
		{"", false},
	}
	for i, c := range cases {
		if got := len(matchDiagFamilyText(c.text)) > 0; got != c.want {
			t.Fatalf("case %d %q: 族命中=%v（应 %v）", i, c.text, got, c.want)
		}
	}
}

// 全链锚：pos34 形仓 settle 同拍——监察案+诊断任务双落账，种子族命中。
func TestPostmortemEmitsDiagTask_pos34Replay(t *testing.T) {
	ctx, r := setup(t)
	now := time.Now()
	pid := pmPos(t, ctx, r, 50, now.Add(-15*time.Second), now)
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_guard_close(position_id,state,trigger_kind,attempts,closed_at)
		VALUES($1,'closed','delta_drift',5,now())`, pid); err != nil {
		t.Fatal(err)
	}
	pmDecisionClose(t, ctx, r, pid, "delta_drift", "delta_drift：notionalNow 漂移超阈——守卫平")
	pmCash(t, ctx, r, pid, -24.67)
	if _, err := r.pool.Exec(ctx, `UPDATE fin_position SET bare_bag_ccy='CORE',
		bare_bag_qty=5737.95,bare_bag_usd=74.89 WHERE id=$1`, pid); err != nil {
		t.Fatal(err)
	}
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := pmLessonCount(t, ctx, r, pid); n != 1 {
		t.Fatalf("监察案行=%d（应 1 pending）", n)
	}
	var nums string
	if err := r.pool.QueryRow(ctx, `SELECT numbers::text FROM fin_econ_lesson
		WHERE kind='rootcause_diag' AND entity_ref=$1 AND status='pending'`,
		fmt.Sprintf("position#%d", pid)).Scan(&nums); err != nil {
		t.Fatalf("诊断任务行缺席: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(nums), &m); err != nil {
		t.Fatal(err)
	}
	seeds, _ := json.Marshal(m["seeds"])
	var ss []diagSeed
	if err := json.Unmarshal(seeds, &ss); err != nil {
		t.Fatal(err)
	}
	if len(ss) != 1 || ss[0].Family != "unit_mismatch" {
		t.Fatalf("诊断任务种子族=%+v（应 unit_mismatch）", ss)
	}
	if srcs, _ := json.Marshal(m["sources"]); !strings.Contains(string(srcs), "守卫环") {
		t.Fatalf("案卷源缺守卫环: %s", srcs)
	}
	// 幂等：settle 重放不重行（UNIQUE(kind,entity_ref)）。
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_econ_lesson
		WHERE kind='rootcause_diag' AND entity_ref=$1`,
		fmt.Sprintf("position#%d", pid)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("诊断任务重放重行（n=%d）", n)
	}
}

// 市况形全链反锚：幅超阈单发（无残袋/无守卫环/认知集内）——监察案照立、
// 诊断任务零产。
func TestPostmortemNoDiagTask_marketShape(t *testing.T) {
	ctx, r := setup(t)
	now := time.Now()
	pid := pmPos(t, ctx, r, 100, now.Add(-2*time.Hour), now)
	pmGuardClose(t, ctx, r, pid, "funding_flip")
	pmCash(t, ctx, r, pid, -30)
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := pmLessonCount(t, ctx, r, pid); n != 1 {
		t.Fatalf("监察案行=%d（市况异常案应立）", n)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_econ_lesson
		WHERE kind='rootcause_diag' AND entity_ref=$1`,
		fmt.Sprintf("position#%d", pid)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("市况形产诊断任务（n=%d，应 0）", n)
	}
}

// 禁线断言（票面钦定——诊断席零 venue/fin 写径，代码面核）：
// A. Go 本件 import 面=安全闭集+写径语句面（仅 INSERT fin_econ_lesson）；
// B. 裁定席 SKILL 第四分支落面白名单（草案落位+归流两窄面，禁线句在案，
//
//	钱径表 UPDATE 零）。
func TestDiagSeatZeroMoneyPathWrites(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("caller 路径不可知——代码面核跳")
	}
	dir := filepath.Dir(file)
	root := filepath.Dir(filepath.Dir(dir)) // internal/ledger → 仓根

	// A. Go 本件。
	src, err := os.ReadFile(filepath.Join(dir, "diagnosis.go"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	if strings.Contains(s, "UPDATE ") || strings.Contains(s, "DELETE ") {
		t.Fatal("diagnosis.go 含 UPDATE/DELETE 写径（诊断面禁）")
	}
	if n := strings.Count(s, "INSERT INTO"); n != 1 || !strings.Contains(s, "INSERT INTO fin_econ_lesson") {
		t.Fatalf("diagnosis.go INSERT 语句面=%d（应恰 1 且靶=fin_econ_lesson 案卷锚表）", n)
	}
	impStart := strings.Index(s, "import (")
	impEnd := strings.Index(s, ")")
	if impStart < 0 || impEnd < impStart {
		t.Fatal("import 块定位败")
	}
	imp := s[impStart:impEnd]
	for _, banned := range []string{"o/internal/trading", "o/internal/guard", "o/internal/data", "o/internal/portfolio", "net/http"} {
		if strings.Contains(imp, banned) {
			t.Fatalf("diagnosis.go import 含禁面 %s（venue/交易/守卫写径载体）", banned)
		}
	}

	// B. 裁定席 SKILL 第四分支。
	skill, err := os.ReadFile(filepath.Join(root, "skills", "裁定席", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	ss := string(skill)
	start := strings.Index(ss, "## 根因诊断案受理节")
	if start < 0 {
		t.Fatal("裁定席 SKILL 缺根因诊断案受理节")
	}
	sec := ss[start:]
	if next := strings.Index(sec[len("## 根因诊断案受理节"):], "\n## "); next >= 0 {
		sec = sec[:next+len("## 根因诊断案受理节")]
	}
	for _, must := range []string{
		"禁 venue 写禁 fin_* 写", "禁自批", "status='drafted'", "fin_econ_lesson",
		"fin_position", "fin_order", "fin_holding", "fin_guard_close", // 禁线句点名钱径表
		"归流执行", // DM 既裁执行段在案
	} {
		if !strings.Contains(sec, must) {
			t.Fatalf("受理节缺禁线/白名单件 %q", must)
		}
	}
	for _, banned := range []string{
		"UPDATE fin_position", "UPDATE fin_order", "UPDATE fin_holding",
		"UPDATE fin_guard_close", "UPDATE fin_cashflow", "UPDATE fin_ledger",
		"INSERT INTO fin_position", "INSERT INTO fin_order", "INSERT INTO fin_holding",
		"INSERT INTO app_config",
	} {
		if strings.Contains(sec, banned) {
			t.Fatalf("受理节含钱径写径形 %q（诊断席禁）", banned)
		}
	}
}

// T-0464 红桩：四值机判序（code→ops→market→unattributed 显式位——零锚
// 不沉默；锚命中序钉死：code 优先于 ops 优先于 market）。
func TestAttributePnlFourValues(t *testing.T) {
	cases := []struct {
		name string
		f    attrFacts
		want string
	}{
		{"零锚=unattributed 显式判词", attrFacts{}, "unattributed"},
		{"病形码件=code", attrFacts{BugShaped: true}, "code"},
		{"腿错因=code", attrFacts{LegErrs: 2}, "code"},
		{"code 压 ops（判序钉死）", attrFacts{BugShaped: true, OpsDisrupt: 3, MktTrigger: true}, "code"},
		{"ops 断供锚=ops", attrFacts{OpsDisrupt: 1}, "ops"},
		{"守卫 venue 期败=ops", attrFacts{GuardVenueEr: true}, "ops"},
		{"ops 压 market（判序钉死）", attrFacts{OpsDisrupt: 1, MktTrigger: true}, "ops"},
		{"市场触发=market", attrFacts{MktTrigger: true}, "market"},
		{"费率不利主因=market", attrFacts{FundingNet: -6, Settled: -10}, "market"},
		{"费率有利主因=market", attrFacts{FundingNet: 8, Settled: 10}, "market"},
		{"费率反向不锚=unattributed", attrFacts{FundingNet: -6, Settled: 10}, "unattributed"},
		{"费率微量不锚=unattributed", attrFacts{FundingNet: -1, Settled: -10}, "unattributed"},
		{"settled=0 无符号对齐=unattributed", attrFacts{FundingNet: -6, Settled: 0}, "unattributed"},
	}
	for _, c := range cases {
		if got := attributePnl(c.f); got != c.want {
			t.Fatalf("%s: attributePnl=%q want %q", c.name, got, c.want)
		}
	}
}

// GW-001 段C2 族↔册机械对账桩：diagFamilies 每族 ID 必有同拍机批册键
// rootcause:<ID>（fin_0148 按族入册——新族扩谱必同拍补册，漏补=本桩红，
// 族误映射/册漂防在码面非散文）。负目录面同锚：族册值径永不入钱径
// （scripts/meta/autoapprove_negative_catalog.tsv——dm_autoapprove 扫描闸
// 同源表，此处静态复证表面在+族册值不含执行/ledger/venue 面）。
func TestDiagFamiliesRegisteredInAutoapproveRegistry(t *testing.T) {
	// 族册键断言（cwd=包目录 internal/ledger——仓根=../..，readTemplate 同径式）：
	// 迁移件读不到=测试环境形态缺（Skip 非绿——按族入册锚在 fin_0148 实存件）。
	b, err := os.ReadFile(filepath.Join("..", "..", "migrations", "fin_0148_autoapprove_family.sql"))
	if err != nil {
		t.Skipf("fin_0148 迁移件相对径不可达（非仓根跑）: %v", err)
	}
	for _, fam := range diagFamilies {
		key := "rootcause:" + fam.ID
		if !strings.Contains(string(b), `"`+key+`"`) {
			t.Fatalf("族 %s 无机批册键 %s——按族入册缺（fin_0148 补册后本桩绿）", fam.ID, key)
		}
	}
	ngb, err := os.ReadFile(filepath.Join("..", "..", "scripts", "meta", "autoapprove_negative_catalog.tsv"))
	if err != nil || len(strings.TrimSpace(string(ngb))) == 0 {
		t.Fatalf("负目录表缺/空——机批断言面不可判（fail-closed 源头在册）")
	}
}

// —— GW-001 段D1 fast_close 复发闸 ——

// 纯函数三形：单次秒平盈利不入闸（不烧诊断额度）；复发实断/秒平亏损
// 二径入闸；源标签随入径落（复用 diagSources 载荷面同源）。
func TestDiagWorthy_fastCloseRepeatGate(t *testing.T) {
	single := diagCaseFacts{PosID: 1, ExecID: 2, InstID: "BTC-USDT-SWAP",
		Anomalies: []string{"fast_close"}, SettledPct: -0.4, SettledUSD: 0.5}
	if diagnosisWorthy(single) {
		t.Fatal("单次秒平盈利过诊断闸（应不烧额度）")
	}
	if s := diagSources(single); len(s) != 0 {
		t.Fatalf("单次秒平盈利源标签=%v（应空——未入闸无源）", s)
	}
	repeat := single
	repeat.FastCloseRepeat = true
	if !diagnosisWorthy(repeat) {
		t.Fatal("复发实断秒平未入闸（应生成）")
	}
	loss := single
	loss.SettledUSD = -3.5
	if !diagnosisWorthy(loss) {
		t.Fatal("秒平亏损未入闸（应生成）")
	}
	if s := diagSources(loss); len(s) != 1 || !strings.Contains(s[0], "秒平亏损") {
		t.Fatalf("秒平亏损源标签=%v（应含秒平亏损径）", s)
	}
}

// pmPosFC 段D1 链测仓 fixture（pmPos 变体——inst 可指定：复发键域=
// strategy_ref∨inst_id，同 inst 即同复发集，跨用例隔离靠独占 inst）。
func pmPosFC(t *testing.T, ctx context.Context, r *Runner, inst string,
	notional float64, opened, closed time.Time) int64 {
	t.Helper()
	var xid, pid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES($1,$2,'closed') RETURNING id`, inst, notional).Scan(&xid); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,opened_at,closed_at)
		VALUES($1,$2,$3,$4,$5) RETURNING id`, xid, inst, notional, opened, closed).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid
}

// fcInstCleanup 段D1 链测隔离面：独占 inst 域清场（lesson→position 序——
// 引用向；跨用例残留防串扰，inst 域即复发键域）。
func fcInstCleanup(t *testing.T, ctx context.Context, r *Runner, inst string) {
	t.Helper()
	if _, err := r.pool.Exec(ctx, `DELETE FROM fin_econ_lesson WHERE entity_ref IN (
		SELECT 'position#'||id FROM fin_position WHERE inst_id=$1)`, inst); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `DELETE FROM fin_position WHERE inst_id=$1`, inst); err != nil {
		t.Fatal(err)
	}
}

// fcPinRepeatN 段D1 链测键钉：N 键显式重钉+快照重载（迁移种子默认 2
// ——显式钉防漂）；t.Cleanup 还原种子值+重载（键域共享防漏）。
func fcPinRepeatN(t *testing.T, ctx context.Context, r *Runner, n string) {
	t.Helper()
	if _, err := r.pool.Exec(ctx, `INSERT INTO app_config(key,value)
		VALUES('guard.postmortem.fast_close_repeat_n',$1::jsonb)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, n); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = r.pool.Exec(ctx, `INSERT INTO app_config(key,value)
			VALUES('guard.postmortem.fast_close_repeat_n','2')
			ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`)
		_ = r.cfg.Load(ctx)
	})
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
}

// 全链锚（复发径）：单次秒平盈利——监察案立（案不丢）、诊断任务零产
// （不烧额度）、diag_gate 去向字段在案（半成品窗去向写明）；同 inst
// 7d 窗第二例——含本仓共 2≥N=2=复发，诊断任务产+源标签带复发径。
func TestPostmortemFastCloseRepeatGate(t *testing.T) {
	ctx, r := setup(t)
	inst := "FC-REPEAT-GATE-TEST"
	fcInstCleanup(t, ctx, r, inst)
	fcPinRepeatN(t, ctx, r, "2")
	now := time.Now()
	pid1 := pmPosFC(t, ctx, r, inst, 50, now.Add(-15*time.Second), now)
	pmCash(t, ctx, r, pid1, 1.23)
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := pmLessonCount(t, ctx, r, pid1); n != 1 {
		t.Fatalf("单次秒平监察案行=%d（应 1——案不丢）", n)
	}
	var gate string
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(numbers->>'diag_gate','')
		FROM fin_econ_lesson WHERE kind='trade_postmortem' AND entity_ref=$1`,
		fmt.Sprintf("position#%d", pid1)).Scan(&gate); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gate, "复发未达") || !strings.Contains(gate, "去向=监察案") {
		t.Fatalf("单次秒平 diag_gate 去向字段缺/形不对: %q", gate)
	}
	var nd int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_econ_lesson
		WHERE kind='rootcause_diag' AND entity_ref=$1`,
		fmt.Sprintf("position#%d", pid1)).Scan(&nd); err != nil {
		t.Fatal(err)
	}
	if nd != 0 {
		t.Fatalf("单次秒平产诊断任务（n=%d，应 0——不烧额度）", nd)
	}
	pid2 := pmPosFC(t, ctx, r, inst, 50, now.Add(-15*time.Second), now)
	pmCash(t, ctx, r, pid2, 0.88)
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := pmLessonCount(t, ctx, r, pid2); n != 1 {
		t.Fatalf("复发秒平监察案行=%d（应 1）", n)
	}
	var nums2 string
	if err := r.pool.QueryRow(ctx, `SELECT numbers::text FROM fin_econ_lesson
		WHERE kind='rootcause_diag' AND entity_ref=$1`,
		fmt.Sprintf("position#%d", pid2)).Scan(&nums2); err != nil {
		t.Fatalf("复发秒平诊断任务缺席: %v", err)
	}
	if !strings.Contains(nums2, "秒平复发") {
		t.Fatalf("复发诊断任务源标签缺秒平复发径: %s", nums2)
	}
}

// 全链锚（键缺席形——半成品窗反向）：复发键未设=复发支不激活（单次
// 秒平不入闸现制），监察案照立+diag_gate 写明键未设去向（案不丢；
// N 键缺不 fail-closed 整链——判据三键缺才跳全链）。
func TestPostmortemFastCloseKeyMissing(t *testing.T) {
	ctx, r := setup(t)
	inst := "FC-NOKEY-TEST"
	fcInstCleanup(t, ctx, r, inst)
	if _, err := r.pool.Exec(ctx, `DELETE FROM app_config
		WHERE key='guard.postmortem.fast_close_repeat_n'`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = r.pool.Exec(ctx, `INSERT INTO app_config(key,value)
			VALUES('guard.postmortem.fast_close_repeat_n','2')
			ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`)
		_ = r.cfg.Load(ctx)
	})
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	pid := pmPosFC(t, ctx, r, inst, 50, now.Add(-15*time.Second), now)
	pmCash(t, ctx, r, pid, 1.0)
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := pmLessonCount(t, ctx, r, pid); n != 1 {
		t.Fatalf("键缺席监察案行=%d（应 1——案不丢）", n)
	}
	var gate string
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(numbers->>'diag_gate','')
		FROM fin_econ_lesson WHERE kind='trade_postmortem' AND entity_ref=$1`,
		fmt.Sprintf("position#%d", pid)).Scan(&gate); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gate, "未设") || !strings.Contains(gate, "去向=监察案") {
		t.Fatalf("键缺席 diag_gate 去向字段缺/形不对: %q", gate)
	}
	var nd int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_econ_lesson
		WHERE kind='rootcause_diag' AND entity_ref=$1`,
		fmt.Sprintf("position#%d", pid)).Scan(&nd); err != nil {
		t.Fatal(err)
	}
	if nd != 0 {
		t.Fatalf("键缺席产诊断任务（n=%d，应 0——复发支不激活）", nd)
	}
}
