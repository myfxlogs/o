// survey_test.go —— O-41375 桩测锚：①probed+空 symbol_params+ts 陈行
// →臂跑→evidence.survey 实载+onboard.probe.symbol_params 补缺+升件行；
// ②证据齐行（symbol_params 非空）scan 层不重跑；③roster 无户=skip 标；
// ④承接面判词三档（能/部分/不能结构）机算断言。
package survey

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// fakeSession 桩会话（注入符号表/参数/报价/保证金面）。
type fakeSession struct {
	syms   []string
	params map[string]map[string]any
	margin map[string]float64
	closed bool
}

func (f *fakeSession) SymbolList(context.Context) ([]string, error) { return f.syms, nil }
func (f *fakeSession) AccountFacts(context.Context) (int64, int64, string, error) {
	return 110660304, 1000, "USC", nil
}
func (f *fakeSession) AccountSummary(context.Context) (float64, float64, float64, error) {
	return 1000, 1000, 1000, nil
}
func (f *fakeSession) SymbolParams(_ context.Context, s string) (map[string]any, error) {
	if p, ok := f.params[s]; ok {
		return p, nil
	}
	return map[string]any{"digits": 5, "spread_pts": 12, "min_lots": 0.01,
		"lots_step": 0.01, "trade_mode": "FullAccess", "trade_mode_id": int32(0),
		"contract_size": 100000.0, "swap_type": "NoSwap", "swap_long": 0.0,
		"swap_short": 0.0, "triple_swap_day": "Disabled"}, nil
}
func (f *fakeSession) Quote(_ context.Context, s string) (float64, float64, error) {
	if s == "DEADUSD" {
		return 0, 0, nil
	}
	return 1.1, 1.1002, nil
}
func (f *fakeSession) Margin1Lot(_ context.Context, s string) (float64, error) {
	return f.margin[s], nil
}
func (f *fakeSession) Close() { f.closed = true }

// fixture 双链迁移+装配（md=fin=同测试库——生产双池形态桩内合一）。
func fixture(t *testing.T) (*Runner, *pgxpool.Pool, context.Context) {
	t.Helper()
	t.Setenv("O_VENUE_SURVEY_DIR", t.TempDir())
	pool := testutil.Pool(t)
	ctx := context.Background()
	for _, chain := range []string{"md", "fin"} {
		if err := pgstore.Migrate(ctx, pool, chain); err != nil {
			t.Fatalf("migrate %s: %v", chain, err)
		}
	}
	r := New(pool, pool, config.New(pool))
	return r, pool, ctx
}

func seedVenue(t *testing.T, ctx context.Context, pool *pgxpool.Pool, venue, state string, ev string, age time.Duration) {
	t.Helper()
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state,evidence,ts)
		VALUES($1,'cfd-broker',$2,$3::jsonb,now()-$4::interval)
		ON CONFLICT (venue) DO UPDATE SET state=$2,evidence=$3::jsonb,ts=now()-$4::interval`,
		venue, state, ev, fmt.Sprintf("%d hours", int(age.Hours()))); err != nil {
		t.Fatal(err)
	}
}

func seedRoster(t *testing.T, ctx context.Context, pool *pgxpool.Pool, venue string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_roster(venue,proto,cred_ns,state,login)
		VALUES($1,'mt5',$1||'.mt5','active','1') ON CONFLICT (venue) DO NOTHING`, venue); err != nil {
		t.Fatal(err)
	}
}

func surveyDueNow(t *testing.T) {
	old := surveyStaleAfter
	surveyStaleAfter = time.Millisecond
	t.Cleanup(func() { surveyStaleAfter = old })
}

func evMap(t *testing.T, ctx context.Context, pool *pgxpool.Pool, venue string) map[string]any {
	t.Helper()
	var ev []byte
	if err := pool.QueryRow(ctx,
		`SELECT evidence FROM fin_venue_candidate WHERE venue=$1`, venue).Scan(&ev); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(ev, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestSurveyFullPathEvidenceAndEscalation(t *testing.T) {
	surveyDueNow(t)
	r, pool, ctx := fixture(t)
	venue := "tstsym"
	seedVenue(t, ctx, pool, venue, "probed",
		`{"onboard":{"probe":{"symbol_params":{},"symbols_total":4}},"map_draft":{"pairs":{}}}`, 48*time.Hour)
	seedRoster(t, ctx, pool, venue)
	// 族宇宙桩：mt5_dir_mom 实评三 CFD 符（canon BTCUSD/XAUUSD/USDJPY——venue 命中前二=部分）
	for i, inst := range []string{"BTCUSDm", "XAUUSDm", "USDJPYm"} {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval
			(inst_id,template,direction,verdict,legs,evidence,net_apr,ts)
			VALUES($1,'mt5_dir_mom','short_perp_long_spot','proposed','{}','{}',0.1,now()-$2::interval)`,
			inst, fmt.Sprintf("%d hours", i+1)); err != nil {
			t.Fatal(err)
		}
	}
	fs := &fakeSession{syms: []string{"BTCUSD", "XAUUSD", "EURUSD", "DEADUSD"},
		margin: map[string]float64{"BTCUSD": 50, "XAUUSD": 800}}
	r.dial = func(context.Context, string) (Session, error) { return fs, nil }
	if err := r.surveyVenue(ctx, venue); err != nil {
		t.Fatalf("surveyVenue: %v", err)
	}
	m := evMap(t, ctx, pool, venue)
	// ①evidence.survey 实载+onboard.probe.symbol_params 补缺列断言
	sv, _ := m["survey"].(map[string]any)
	if sv == nil || sv["symbols_total"].(float64) != 4 {
		t.Fatalf("survey 实载缺: %v", sv)
	}
	ob, _ := m["onboard"].(map[string]any)
	probe, _ := ob["probe"].(map[string]any)
	sp, _ := probe["symbol_params"].(map[string]any)
	if len(sp) != 4 {
		t.Fatalf("symbol_params 补缺列应=4 符: %d", len(sp))
	}
	btc, _ := sp["BTCUSD"].(map[string]any)
	if btc["contractSize"].(float64) != 100000 || btc["spread"].(float64) != 12 {
		t.Fatalf("exness 同形字段错位: %v", btc)
	}
	// ②margin 分层：BTCUSD=50→le_100；XAUUSD=800>eq*0.5→gt；余 margin=0→no_data
	mt, _ := sv["margin_tiers"].(map[string]any)
	if mt["le_100"].(float64) != 1 || mt["gt"].(float64) != 1 || mt["no_data"].(float64) != 2 {
		t.Fatalf("margin 分层错: %v", mt)
	}
	// ③判词：mt5_dir_mom 宇宙 canon{BTCUSD,XAUUSD,USDJPY} venue canon 命中 2/3→部分
	fv, _ := sv["family_verdicts"].(map[string]any)
	dm, _ := fv["mt5_dir_mom"].(map[string]any)
	if dm["verdict"] != "部分" || dm["covered"].(float64) != 2 || dm["total"].(float64) != 3 {
		t.Fatalf("承接面判词错: %v", dm)
	}
	// ④map_draft.pairs：canon BTC/XAU 命中（USDJPY 宇宙在 venue 无→不配）
	mdraft, _ := sv["map_draft"].(map[string]any)
	pairs, _ := mdraft["pairs"].(map[string]any)
	if pairs["BTC"] != "BTCUSD" || pairs["XAU"] != "XAUUSD" || len(pairs) != 2 {
		t.Fatalf("map_draft pairs 错: %v", pairs)
	}
	// ⑤升件行+拍行断言
	var esc int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='venue上岗' AND esc_scope=$1`, venue).Scan(&esc); err != nil || esc != 1 {
		t.Fatalf("升件行缺: n=%d err=%v", esc, err)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_scope=$1`, venue).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "证件齐") || !strings.Contains(note, "4符") {
		t.Fatalf("升件词形错: %s", note)
	}
	if !fs.closed {
		t.Fatal("会话未 Close")
	}
}

func TestSurveyScanSkipsCompleteEvidence(t *testing.T) {
	surveyDueNow(t)
	r, pool, ctx := fixture(t)
	venue := "donev"
	// 证据列齐行（symbol_params 非空）——exness mapped 同构态，scan 层 due=false
	seedVenue(t, ctx, pool, venue, "mapped",
		`{"onboard":{"probe":{"symbol_params":{"AAAUSDm":{"digits":5}}}},"survey":{"at":"2020-01-01T00:00:00Z"}}`, 72*time.Hour)
	seedRoster(t, ctx, pool, venue)
	called := false
	r.dial = func(context.Context, string) (Session, error) {
		called = true
		return &fakeSession{}, nil
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("证据齐行不应拨号（幂等闸失效）")
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_scope=$1`, venue).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("齐证 venue 不应产事件行: %d", n)
	}
}

func TestSurveyDueFilter(t *testing.T) {
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	cases := []struct {
		name string
		ev   map[string]any
		ts   time.Time
		want bool
	}{
		{"空载+ts陈=due", map[string]any{"onboard": map[string]any{"probe": map[string]any{"symbol_params": map[string]any{}}}}, old, true},
		{"齐证不due", map[string]any{"onboard": map[string]any{"probe": map[string]any{"symbol_params": map[string]any{"A": 1}}}}, old, false},
		{"ts鲜不due", map[string]any{"onboard": map[string]any{"probe": map[string]any{"symbol_params": map[string]any{}}}}, now, false},
		{"退避内不due", map[string]any{"onboard": map[string]any{"probe": map[string]any{"symbol_params": map[string]any{}}}, "survey": map[string]any{"at": now.UTC().Format(time.RFC3339)}}, old, false},
		{"退避过due", map[string]any{"onboard": map[string]any{"probe": map[string]any{"symbol_params": map[string]any{}}}, "survey": map[string]any{"at": old.Add(-time.Hour).UTC().Format(time.RFC3339)}}, old, true},
	}
	for _, c := range cases {
		if got := surveyDue(c.ev, c.ts, now); got != c.want {
			t.Errorf("%s: got=%v want=%v", c.name, got, c.want)
		}
	}
}

func TestSurveyNoRosterSkips(t *testing.T) {
	surveyDueNow(t)
	r, pool, ctx := fixture(t)
	venue := "noroster"
	seedVenue(t, ctx, pool, venue, "candidate", `{"onboard":{"probe":{"symbol_params":{}}}}`, 48*time.Hour)
	called := false
	r.dial = func(context.Context, string) (Session, error) {
		called = true
		return &fakeSession{}, nil
	}
	if err := r.surveyVenue(ctx, venue); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("roster 无户不应拨号")
	}
	m := evMap(t, ctx, pool, venue)
	sv, _ := m["survey"].(map[string]any)
	if sv["skipped"] != "no_roster" {
		t.Fatalf("skip 标缺: %v", sv)
	}
}
