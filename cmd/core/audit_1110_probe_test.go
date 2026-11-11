// audit_1110_probe_test.go —— 审计/探测面 2026-11-10 RED 锚（断言正确语义——
// 修复落地即转 GREEN，不红=命题假）：
//
//	①audit_stall 装臂证在案（审计机唤起行）而产出锚零行=死——NULL 只在
//	  「臂未装」时是语义空态；现有 TestAuditStallEsc 把「零行=未装臂」钉成
//	  锚，但装上唤器后唤起行=装臂机读证，其后零产出是死不是合法态。
//	②唤知面死活性锚=认知环自身产出——他席（复审/CEO/交互会话）处置行
//	  不遮蔽认知环死（KB cognition-spawn-env-drift-lockout 坑③实证：
//	  「探针看产出不看自主产出」）。
//	③esc_class 写者全集 ⊆ runbook 判词档——机检面=全仓写者非 cmd/core
//	  一隅（esc_runbook_align_test 的钦定不变量同款，glob 面补齐）。
//	④deployEscCheck 净值读件陈值=读件缺——陈 NAV 对拍=假证据升件
//	  （「禁假证据升件」钦定的自然延伸；reconcile「快照陈>5min 不判」先例）。
//	⑤销案引证区间形（升件#a-b）=范围内逐 id 已销——判据谓词须机读
//	  处置行引证面，区间销在产线实证把 12 件滞留未决（#9275→#9285 补销）。
//	⑥探针册注释枚举=实现枚举——meta_esc.go 头注「四类」而文件内 escEdgeU
//	  类已六类（商业产出/能力债后增未补档）。
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// ①audit_stall：装臂证（seat=审计机 event=唤起 行=唤器 spawn 成功落账）
// 在案而产出锚 NULL——臂被唤过但从未落审计行=死（session 起后零产出
// 静默挂死=本探唯一可见面）。距唤起 <48h 的首飞会话=在飞非死（不误报）。
func TestAuditStallInstalledArmBlind(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='audit_stall' OR note LIKE '%t-audstall%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	var n int

	// 装臂证在案且陈 49h+零审计产出→死——当前实现 NULL 一概非死=盲
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,ts)
		VALUES('审计机','唤起','','unit=t-audstall scope=t_x',now()-interval '49 hours')`); err != nil {
		t.Fatalf("唤起: %v", err)
	}
	svc.auditStallEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_class='唤知' AND esc_scope='audit_stall'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("装臂后零产出未火 n=%d err=%v——audit_stall 对「已装臂从未产出」永绿", n, err)
	}

	// 恢复语义：审计行落地→恢复自闭（同既有契约）
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('审计席','审计','','t-audstall 首产')`); err != nil {
		t.Fatalf("新行: %v", err)
	}
	svc.auditStallEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE seat='探测席' AND event='恢复' AND note LIKE '%audit_stall%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("恢复 n=%d err=%v", n, err)
	}
}

// ②cognitionEscCheck：活性锚「24h 无处置行」的「处置行」必须=处置席
// （认知环处置座）自主产出。他席处置行证明的是他环活非本环活——
// 非处置席处置不重置死钟；处置席处置才复绿。
func TestCognitionDeadForeignSeatMask(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope IN ('t_cogmask','cognition_dead') OR note LIKE '%t-cogmask%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','','t-cogmask 未决','断流','t_cogmask')`); err != nil {
		t.Fatalf("esc: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	var n int

	// 他席（复审席）处置行新鲜——认知环照样死：本拍应火 cognition_dead
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('复审席','处置','','t-cogmask 他席处置行')`); err != nil {
		t.Fatalf("他席处置: %v", err)
	}
	svc.cognitionEscCheck(ctx, deadSince, nil)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='唤知' AND esc_scope='cognition_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("他席处置遮蔽唤知死 n=%d err=%v——活性锚混入非自主产出", n, err)
	}

	// 处置席处置行落地→认知环活证据=恢复自闭
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席','处置','','t-cogmask 本席处置行')`); err != nil {
		t.Fatalf("本席处置: %v", err)
	}
	svc.cognitionEscCheck(ctx, deadSince, nil)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE seat='探测席' AND event='恢复' AND note LIKE '%cognition_dead%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("本席处置未复绿 n=%d err=%v", n, err)
	}
}

// ③esc_class 覆盖机检——全仓写者的类字面量（escEdge 三参/SQL 字面量/
// XxxClass 常量/升件行末引号参）逐一必须在 runbook 文本有判词档。
// 探针升件而 runbook 无档=升件进认知环无人会裁（esc_runbook_align_test
// 钦定不变量同款——但本测试 glob=全仓非 cmd/core 一隅）。
func TestEscClassCoverageRunbook(t *testing.T) {
	pats := []*regexp.Regexp{
		regexp.MustCompile(`escEdgeU?\(ctx,\s*deadSince,\s*"([^"]+)"`),          // escEdge 类
		regexp.MustCompile(`\w*[Cc]lass\w*\s*=\s*"([^"]{1,40})"`),               // *EscClass/Class* 常量
		regexp.MustCompile(`esc_class\s*=\s*'([^']+)'`),                         // SQL 字面量
		regexp.MustCompile(`,\s*'([^'\s][^']{0,30})'\s*,\s*'[^'\s][^']{0,60}'`), // 引号对首参（class,scope 形）
	}
	// 升件行内的非类引号参（seat/event/state/事件名）——非 esc_class 语义。
	block := map[string]bool{
		"升件": true, "处置": true, "恢复": true, "探测席": true, "唤知机": true,
		"审计机": true, "审计席": true, "唤知席": true, "改进矿工": true, "决策席": true,
		"裁定席": true, "复审席": true, "施工席": true, "仓巡": true, "演习": true,
		"研究员": true, "审计": true, "审计疑点": true, "审计复证": true, "裁决": true,
		"通过": true, "驳回": true, "异常": true, "收口": true, "返修": true, "饥饿": true,
		"覆盖失明": true, "族沉默": true, "面清单新面": true, "依赖对失明": true,
		"依赖对豁免": true, "开仓挂起": true, "开仓解冻": true,
	}
	hasHan := regexp.MustCompile(`\p{Han}`).MatchString
	classes := map[string]string{} // class → 首个出处
	err := filepath.Walk(filepath.Join("..", ".."), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", "gen", "vendor", "web", "arch":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if !strings.Contains(string(b), "esc_class") && !strings.Contains(string(b), "escEdge") {
			return nil
		}
		for i, pat := range pats {
			for _, m := range pat.FindAllSubmatch(b, -1) {
				var cand string
				if i == 3 {
					cand = string(m[1]) // 引号对首参=class
				} else {
					cand = string(m[1])
				}
				if cand == "" || block[cand] || !hasHan(cand) {
					continue
				}
				if _, ok := classes[cand]; !ok {
					classes[cand] = p
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	rb, err := os.ReadFile(filepath.Join("..", "..", "scripts", "cognition", "runbook.md"))
	if err != nil {
		t.Fatalf("runbook: %v", err)
	}
	text := string(rb)
	var missing []string
	for c, src := range classes {
		if !strings.Contains(text, c) {
			missing = append(missing, fmt.Sprintf("%s(%s)", c, src))
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("esc_class 写者类缺 runbook 判词档（%d）: %v", len(missing), missing)
	}
}

// ④deployEscCheck：净值行陈 >48h=读件缺——陈 NAV 对拍出的 cap_hit 是假
// 证据升件（md_equity_curve NAV=日拍件，陈 30d 行与零行同语义；禁假证据
// 升件钦定的自然延伸）。
func TestDeployEscStaleNav(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM md_equity_curve WHERE ccy='NAV';
		DELETE FROM fin_strategy WHERE inst_id='T-DEP-USDT';
		DELETE FROM ops_event WHERE esc_scope IN ('cap_hit','idle_capital');
		DELETE FROM app_config WHERE key='risk.max_exposure_usd'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_config(key,value)
		VALUES('risk.max_exposure_usd',to_jsonb('10000'::text))`); err != nil {
		t.Fatalf("cfg: %v", err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatalf("load: %v", err)
	}
	// 陈 30d NAV——与零行同语义的读件缺
	if _, err := pool.Exec(ctx, `INSERT INTO md_equity_curve(ts,venue,ccy,equity,src)
		VALUES(now()-interval '30 days','ledger','NAV',10000,'t-dep')`); err != nil {
		t.Fatalf("nav: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,entered_at,criteria,created_at,venue)
		VALUES('T-DEP-USDT','funding-hedge','live',9500,now(),'{}',now(),'okx')`); err != nil {
		t.Fatalf("strategy: %v", err)
	}
	svc := coreService{pool: pool, cfg: cfg}
	deadSince := map[string]time.Time{}
	svc.deployEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='cap_hit'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("陈 NAV 假证据升件 n=%d err=%v——陈值读件缺应让位不判", n, err)
	}
}

// ⑤销案引证机读面：处置行「升件#a-b」区间引证=范围内逐 id 已销——
// open-scan/dedup 谓词所见未决数应为 0。逐 id 引证形为对照（现行可机读）。
func TestEscCitationRangeResolves(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope LIKE 't_range%' OR note LIKE '%t-range%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	unresolved := func(id int64) int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event e
			WHERE e.event='升件' AND e.id=$1
			AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
				AND (x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')
					OR EXISTS (SELECT 1 FROM regexp_matches(x.note,'升件#(\d+)-(\d+)','g') AS m(mm)
						WHERE e.id BETWEEN mm[1]::bigint AND mm[2]::bigint)))`, id).Scan(&n); err != nil {
			t.Fatalf("unresolved: %v", err)
		}
		return n
	}
	ins := func(scope string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
			VALUES('探测席','升件','','t-range 件','断流',$1) RETURNING id`, scope).Scan(&id); err != nil {
			t.Fatalf("esc: %v", err)
		}
		return id
	}
	// 对照：逐 id 引证形（现行可机读）
	a := ins("t_range_a")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席','处置','',$1)`, fmt.Sprintf("升件#%d 销——t-range 对照", a)); err != nil {
		t.Fatalf("销: %v", err)
	}
	if n := unresolved(a); n != 0 {
		t.Fatalf("对照逐 id 销未生效 n=%d", n)
	}
	// 区间引证形——同一销案语义的机器可读等价面
	b := ins("t_range_b")
	c := ins("t_range_c")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席','处置','',$1)`,
		fmt.Sprintf("升件#%d-%d 销——t-range 区间销（同批并案）", b, c+8)); err != nil {
		t.Fatalf("区间销: %v", err)
	}
	if n := unresolved(b); n != 0 {
		t.Fatalf("区间引证滞留未决 n=%d——销案格式机器不可读（升件#%d 已被引证仍计未决）", n, b)
	}
	if n := unresolved(c); n != 0 {
		t.Fatalf("区间引证滞留未决 n=%d（升件#%d）", n, c)
	}
}

// ⑥探针册注释枚举=实现枚举：cmd/core 内 escEdgeU 的每个类字面量必须在
// meta_esc.go 头注枚举面在列（「对账/孤儿账/唤知/数据缺四类」钦定枚举——
// 后增探件须同步补注，否则 dedup 类册对读档者说谎）。
func TestMetaEscHeaderEnumerates(t *testing.T) {
	b, err := os.ReadFile("meta_esc.go")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	src := string(b)
	head := src[:strings.Index(src, "package main")]
	re := regexp.MustCompile(`escEdgeU\(ctx,\s*deadSince,\s*"([^"]+)"`)
	used := map[string]bool{}
	for _, f := range []string{"meta_esc.go", "audit_esc.go", "feed_esc.go", "expect_esc.go"} {
		fb, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, m := range re.FindAllSubmatch(fb, -1) {
			used[string(m[1])] = true
		}
	}
	var missing []string
	for c := range used {
		if !strings.Contains(head, c) {
			missing = append(missing, c)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("escEdgeU 类未入 meta_esc 头注枚举（头注仍写「四类」）: %v", missing)
	}
}
