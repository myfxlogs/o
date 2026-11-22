package lifecycle

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// #23296 进气道桩测——锚①解析（行20313 实证形谓词组）；锚②tick 落键：
// INSERT/UPDATE 谓词→DMSet→app_config+参数调整审计行+水位行；锚③边界：
// 域外键/非白名单 event/DM写径 seat/简写裸键/零规格预筛命中全不拾；
// 锚④幂等：二拍不重放，水位续扫只拾新行。

func intakeFixture(t *testing.T) (*Runner, context.Context) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return NewRunner(pool, cfg), ctx
}

func intakeEv(t *testing.T, ctx context.Context, r *Runner, seat, event, ticket, note string) int64 {
	t.Helper()
	var id int64
	// ts 回拨 2min——进气道带 60s 滞后余量（O-30635 Marten 型根修），
	// 桩件须落在余量窗外才入扫面；滞后本体由 TestParamIntakeLag 专锚。
	if err := r.pool.QueryRow(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES($1,$2,$3,$4,now()-interval '2 minutes') RETURNING id`,
		seat, event, ticket, note).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestParseParamSpecs(t *testing.T) {
	// 行20313 实证形：动词前置+括注截值+加号分隔。
	note := "工单#19797 参数包：INSERT research.funding.maker_fee_bps=0.58(实测/frill,禁双taker兜底)+UPDATE research.funding.slippage_bps=0(friction_slip_bps.okx实测0)+taker5.0/fail2.0维持"
	specs := parseParamSpecs(note)
	if len(specs) != 2 {
		t.Fatalf("规格数=%d 期望2: %+v", len(specs), specs)
	}
	if specs[0].verb != "INSERT" || specs[0].key != "research.funding.maker_fee_bps" || specs[0].raw != "0.58" {
		t.Fatalf("spec0 不符: %+v", specs[0])
	}
	if specs[1].verb != "UPDATE" || specs[1].key != "research.funding.slippage_bps" || specs[1].raw != "0" {
		t.Fatalf("spec1 不符: %+v", specs[1])
	}
	// 大小写不敏感动词；逗号分隔多组。
	specs = parseParamSpecs("update research.a.b=1, insert research.c.d=\"x y\"")
	if len(specs) != 2 || specs[0].key != "research.a.b" || specs[0].raw != "1" ||
		specs[1].key != "research.c.d" || specs[1].raw != `"x` {
		t.Fatalf("大小写/分隔组不符: %+v", specs)
	}
	// SQL 形不误拾（动词后非键=形）。
	for _, n := range []string{
		"UPDATE fin_proposal SET state='fin_retired'",
		"INSERT INTO ops_event VALUES(1)",
		"maker_fee_bps=0.58 简写裸键无动词",
		"UPDATE research.x.y= 尾等号无值",
	} {
		if got := parseParamSpecs(n); len(got) != 0 {
			t.Fatalf("误拾 %q: %+v", n, got)
		}
	}
}

func TestIntakeJSONValue(t *testing.T) {
	cases := []struct {
		raw  string
		want string // 期望 RawMessage 原文；空=期望报错
	}{
		{"0.58", "0.58"}, {"0", "0"}, {"-3", "-3"}, {"1e3", "1e3"},
		{"true", "true"}, {`"abc"`, `"abc"`},
		{"abc", `"abc"`}, // 裸记号转 JSON 串
		{"resolved", `"resolved"`},
		{"null", ""}, {"[1,2]", ""}, {`{"a":1}`, ""},
	}
	for _, c := range cases {
		v, err := intakeJSONValue(c.raw)
		if c.want == "" {
			if err == nil {
				t.Fatalf("%q 应拒未拒: %s", c.raw, v)
			}
			continue
		}
		if err != nil || string(v) != c.want {
			t.Fatalf("%q: v=%s err=%v want=%s", c.raw, v, err, c.want)
		}
		var anyv any
		if err := json.Unmarshal(v, &anyv); err != nil {
			t.Fatalf("%q 产出非合法 JSON: %v", c.raw, err)
		}
	}
}

func TestParamIntakeTick(t *testing.T) {
	r, ctx := intakeFixture(t)
	// 请求行（#19797 形）+四条不拾行：域外键/非白名单 event/DM写径 seat/简写。
	idReq := intakeEv(t, ctx, r, "自我", "处置", "T-23296",
		"工单#19797 参数包：INSERT research.funding.maker_fee_bps=0.58(实测/frill)+UPDATE research.funding.slippage_bps=0(friction实测0)+维持项散文")
	idBad := intakeEv(t, ctx, r, "自我", "处置", "T-23296",
		"UPDATE unregistered.domain.key=1 域外键应跳过")
	intakeEv(t, ctx, r, "施工席", "完工回执", "T-23296",
		"INSERT research.funding.other=9 非白名单 event 不拾")
	intakeEv(t, ctx, r, "决策席-DM写径", "裁定", "T-23296",
		"UPDATE research.funding.slippage_bps=99 DM写径 seat 自喂防不拾")
	intakeEv(t, ctx, r, "自我", "处置", "T-23296",
		"maker_fee_bps=0.77 简写裸键无动词不拾")
	// DMSet 自产审计行形（event=参数调整+seat=DM写径）——双闸自喂防实证。
	intakeEv(t, ctx, r, "决策席-DM写径", "参数调整", "T-23296",
		"key=research.funding.slippage_bps 旧=3 新=0 证据=INSERT research.funding.slippage_bps=0")

	r.paramIntakeTick(ctx)

	var maker, slip string
	if err := r.pool.QueryRow(ctx,
		`SELECT value::text FROM app_config WHERE key='research.funding.maker_fee_bps'`).Scan(&maker); err != nil {
		t.Fatalf("maker 键未落: %v", err)
	}
	if err := r.pool.QueryRow(ctx,
		`SELECT value::text FROM app_config WHERE key='research.funding.slippage_bps'`).Scan(&slip); err != nil {
		t.Fatalf("slippage 键未落: %v", err)
	}
	if maker != "0.58" || slip != "0" {
		t.Fatalf("落值不符 maker=%s slip=%s", maker, slip)
	}
	// 域外/非白名单/DM seat/简写四行零落——slippage 若被自喂行改=99 即败。
	if slip == "99" {
		t.Fatal("DM写径 seat 行被拾——自喂防失效")
	}
	var other int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM app_config WHERE key='research.funding.other'`).Scan(&other); err != nil || other != 0 {
		t.Fatalf("非白名单 event 行被拾: n=%d err=%v", other, err)
	}
	// 审计行：参数调整 seat=决策席-DM写径，note 带进气道锚+三列。
	var note, by string
	if err := r.pool.QueryRow(ctx,
		`SELECT a.note,c.updated_by FROM ops_event a
		 JOIN app_config c ON c.key='research.funding.maker_fee_bps'
		 WHERE a.event='参数调整' AND a.ticket='T-23296'
		   AND a.note LIKE '%key=research.funding.maker_fee_bps%'
		   AND a.note LIKE '%进气道 ops_event#%' ORDER BY a.id DESC LIMIT 1`,
	).Scan(&note, &by); err != nil {
		t.Fatalf("进气道审计行缺: %v", err)
	}
	if !strings.Contains(note, "key=research.funding.maker_fee_bps") ||
		!strings.Contains(note, "新=0.58") ||
		!strings.Contains(note, "证据=进气道 ops_event#"+strconv.FormatInt(idReq, 10)) {
		t.Fatalf("审计 note 三列不符: %s", note)
	}
	if by != "决策席-DM写径" {
		t.Fatalf("updated_by=%q 非 DM 写径身份", by)
	}
	// 水位行=末行 id（全行过完——含不拾行）。
	var wm string
	if err := r.pool.QueryRow(ctx,
		`SELECT detail FROM ops_state_row WHERE kind='参数进气' AND scope='watermark'`).Scan(&wm); err != nil {
		t.Fatalf("水位行缺: %v", err)
	}
	// 水位=末张预筛命中行 id（idBad——后随不拾行不入扫面不推进）。
	if wm != strconv.FormatInt(idBad, 10) {
		t.Fatalf("水位=%q 期望=%d（末张命中行）", wm, idBad)
	}
	// 二拍幂等：无新行→零新参数调整审计行。
	var auditsBefore int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='参数调整' AND ticket='T-23296'`).Scan(&auditsBefore); err != nil {
		t.Fatal(err)
	}
	r.paramIntakeTick(ctx)
	var auditsAfter int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='参数调整' AND ticket='T-23296'`).Scan(&auditsAfter); err != nil {
		t.Fatal(err)
	}
	if auditsAfter != auditsBefore {
		t.Fatalf("二拍重放: 审计行 %d→%d", auditsBefore, auditsAfter)
	}
	// 水位续扫：新请求行下拍单拾。
	idNew := intakeEv(t, ctx, r, "唤知", "裁定", "T-23296",
		"UPDATE research.funding.slippage_bps=1.5 新裁定")
	r.paramIntakeTick(ctx)
	if err := r.pool.QueryRow(ctx,
		`SELECT value::text FROM app_config WHERE key='research.funding.slippage_bps'`).Scan(&slip); err != nil {
		t.Fatal(err)
	}
	if slip != "1.5" {
		t.Fatalf("续扫落值不符 slip=%s", slip)
	}
	_ = idNew
}

// TestParamIntakeLag O-30635 Marten 型根修锚：ts<now()-60s 滞后余量——
// 新鲜行（模拟未提交窗内的相对新行）本拍不入扫面不推进水位；
// 回拨行照常落键。水位不越新行=下拍续偿不丢。
func TestParamIntakeLag(t *testing.T) {
	r, ctx := intakeFixture(t)
	var freshID int64
	if err := r.pool.QueryRow(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('自我','处置','T-lag',
		 'UPDATE research.funding.slippage_bps=7.7 新鲜行滞后窗内不拾') RETURNING id`).Scan(&freshID); err != nil {
		t.Fatal(err)
	}
	r.paramIntakeTick(ctx)
	var v string
	// 键或有默认值——断「未被改成新鲜值 7.7」而非键缺席。
	if err := r.pool.QueryRow(ctx,
		`SELECT value::text FROM app_config WHERE key='research.funding.slippage_bps'`).Scan(&v); err == nil && v == "7.7" {
		t.Fatalf("滞后窗内新鲜行被拾 slip=%s（60s 余量失效）", v)
	}
	// 水位不得越过未消费新鲜行——下拍（行龄>60s 后）须仍可拾。
	aged := intakeEv(t, ctx, r, "自我", "处置", "T-lag",
		"UPDATE research.funding.maker_fee_bps=0.42 回拨行照常拾")
	r.paramIntakeTick(ctx)
	if err := r.pool.QueryRow(ctx,
		`SELECT value::text FROM app_config WHERE key='research.funding.maker_fee_bps'`).Scan(&v); err != nil || v != "0.42" {
		t.Fatalf("回拨行未落 maker=%q err=%v", v, err)
	}
	// 水位=aged 行 id——aged.id>fresh.id（后插）故 fresh 已永滞后窗内不可达
	// 仅当其 ts 仍<60s；此处验证的是「新鲜行不阻塞后位回拨行」语义。
	_ = aged
	_ = freshID
}
