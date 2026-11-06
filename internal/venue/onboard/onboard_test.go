// onboard_test.go —— T-0379 桩测四锚：①假凭据全链（探测→草案→map闸→paper——
// 四拍 ops_event 落行+态迁移+查询面断言）；④三败径（凭据坏/网关不可达/
// 符号表空）fail-closed+自报。Session seam 注假协议面。
package onboard

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// fakeSession 桩会话（注入败径/符号表面）。
type fakeSession struct {
	connectErr  error
	syms        []string
	eq, fm, bal float64
}

func (f *fakeSession) Connect(context.Context, mt5.Credentials) error { return f.connectErr }
func (f *fakeSession) AccountSummary(context.Context) (float64, float64, float64, error) {
	return f.eq, f.fm, f.bal, nil
}
func (f *fakeSession) SymbolList(context.Context) ([]string, error) { return f.syms, nil }
func (f *fakeSession) SymbolFacts(_ context.Context, s string) (map[string]any, error) {
	return map[string]any{"contractSize": 100.0, "minLots": 0.01, "tradeMode": int32(4)}, nil
}
func (f *fakeSession) Disconnect(context.Context) {}
func (f *fakeSession) Close()                     {}

var testMK = []byte("0123456789abcdef0123456789abcdef")

// fixture 两链迁移+装配（md=fin=同测试库——生产双池形态桩内合一）。
func fixture(t *testing.T) (*Runner, *pgxpool.Pool, *config.Client) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	for _, chain := range []string{"md", "fin"} {
		if err := pgstore.Migrate(ctx, pool, chain); err != nil {
			t.Fatalf("migrate %s: %v", chain, err)
		}
	}
	cfg := config.New(pool)
	seedCfg(t, ctx, pool, map[string]string{
		"mt5.gateway_addr": `"mt5grpc3.mtapi.io:443"`,
		"okx.symbols":      `["BTC-USDT-SWAP","ETH-USDT-SWAP"]`,
	})
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := New(pool, pool, cfg)
	r.mk = func() ([]byte, error) { return testMK, nil }
	return r, pool, cfg
}

func seedCfg(t *testing.T, ctx context.Context, pool *pgxpool.Pool, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		if _, err := pool.Exec(ctx,
			`INSERT INTO app_config(key,value,updated_by) VALUES($1,$2::jsonb,'t0379')
			 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, k, v); err != nil {
			t.Fatal(err)
		}
	}
}

// seedCreds 假凭据 Seal 入 app_secrets（<broker>.mt5.* 键形票面钉死）。
func seedCreds(t *testing.T, ctx context.Context, pool *pgxpool.Pool, broker string, fields map[string]string) {
	t.Helper()
	for f, v := range fields {
		ct, err := cryptoutil.Seal(testMK, v)
		if err != nil {
			t.Fatal(err)
		}
		key := fmt.Sprintf("%s.mt5.%s", broker, f)
		if broker == "mt5" {
			key = "mt5." + f
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)
			 ON CONFLICT (key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext`, key, ct); err != nil {
			t.Fatal(err)
		}
	}
}

func stateOf(t *testing.T, ctx context.Context, pool *pgxpool.Pool, venue string) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(ctx,
		`SELECT state FROM fin_venue_candidate WHERE venue=$1`, venue).Scan(&s); err != nil {
		t.Fatalf("state read: %v", err)
	}
	return s
}

func opsBeats(t *testing.T, ctx context.Context, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(ctx,
		`SELECT note FROM ops_event WHERE event='venue接入' ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		out = append(out, n)
	}
	return out
}

// 锚① 桩测全链：假凭证→探测→草案→map闸→paper——四拍每拍 ops_event 落行；
// 状态机迁移+查询面断言（锚③——registered/probed/mapped/paper 态序过 CHECK）。
func TestOnboardFullChain(t *testing.T) {
	r, pool, cfg := fixture(t)
	ctx := context.Background()
	sess := &fakeSession{syms: []string{"BTCUSDm", "ETHUSDm", "XAUUSDm"}, eq: 1000.5, fm: 900.2, bal: 1000.5}
	r.dial = func(string) (Session, error) { return sess, nil }
	seedCreds(t, ctx, pool, "fake1", map[string]string{
		"login": "3001157", "password": "pw", "server": "Fake-MT5"})
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	// 一拍全链推进：registered→probed→mapped（map 未落止）。
	if got := stateOf(t, ctx, pool, "fake1"); got != "mapped" {
		t.Fatalf("探测+草案后态=%s 应 mapped", got)
	}
	var draft string
	if err := pool.QueryRow(ctx,
		`SELECT evidence->'onboard'->'map_draft'->'pairs'->>'BTC-USDT-SWAP'
		 FROM fin_venue_candidate WHERE venue='fake1'`).Scan(&draft); err != nil || draft != "BTCUSDm" {
		t.Fatalf("草案对 BTC-USDT-SWAP→%q err=%v（应 BTCUSDm）", draft, err)
	}
	// map 闸请求盒行（kind=permission venue=fake1 surface=bridge.xvenue.map）。
	var cnt int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_condition_gap
		WHERE kind='permission' AND venue='fake1' AND surface='bridge.xvenue.map' AND state='pending'`).Scan(&cnt); err != nil || cnt != 1 {
		t.Fatalf("condreq pending=%d err=%v（应 1）", cnt, err)
	}
	// owner 单次批落闸（app_config 写——bridge.xvenue.map owner 键）。
	seedCfg(t, ctx, pool, map[string]string{
		"bridge.xvenue.map": `{"BTC-USDT-SWAP":"BTCUSDm","ETH-USDT-SWAP":"ETHUSDm"}`})
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stateOf(t, ctx, pool, "fake1"); got != "paper" {
		t.Fatalf("map 落闸后态=%s 应 paper", got)
	}
	beats := opsBeats(t, ctx, pool)
	if len(beats) != 4 {
		t.Fatalf("四拍 ops_event 行数=%d 应 4: %v", len(beats), beats)
	}
	for i, want := range []string{"beat=探测", "beat=草案", "beat=map闸", "beat=paper激活"} {
		if beats[i][:len(want)] != want {
			t.Fatalf("第%d拍行=%q 应前缀 %q", i+1, beats[i], want)
		}
	}
	// 查询面断言：esc_scope=<venue>:<login> 粒度钉。
	var scope string
	if err := pool.QueryRow(ctx, `SELECT esc_scope FROM ops_event
		WHERE event='venue接入' ORDER BY id LIMIT 1`).Scan(&scope); err != nil || scope != "fake1:3001157" {
		t.Fatalf("esc_scope=%q err=%v（应 fake1:3001157——<venue>:<login> 粒度）", scope, err)
	}
	// 态机非法值拒（CHECK 面断言）。
	if _, err := pool.Exec(ctx, `UPDATE fin_venue_candidate SET state='bogus' WHERE venue='fake1'`); err == nil {
		t.Fatal("非法态应被 CHECK 拒")
	}
}

// 锚④a 凭据坏=fail-closed 自报（stay registered+ops_event 败行）。
func TestOnboardBadCreds(t *testing.T) {
	r, pool, _ := fixture(t)
	ctx := context.Background()
	r.dial = func(string) (Session, error) { return &fakeSession{syms: []string{"BTCUSDm"}}, nil }
	seedCreds(t, ctx, pool, "bad1", map[string]string{
		"login": "not-a-number", "password": "pw", "server": "Fake-MT5"})
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stateOf(t, ctx, pool, "bad1"); got != "registered" {
		t.Fatalf("凭据坏应 stay registered got=%s", got)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='venue接入' AND esc_class='接入败'`).Scan(&n); err != nil || n == 0 {
		t.Fatalf("凭据坏败行 n=%d err=%v（应自报）", n, err)
	}
}

// 锚④b 网关不可达=fail-closed 自报。
func TestOnboardGatewayDown(t *testing.T) {
	r, pool, _ := fixture(t)
	ctx := context.Background()
	r.dial = func(string) (Session, error) { return nil, fmt.Errorf("gateway unreachable") }
	seedCreds(t, ctx, pool, "down1", map[string]string{
		"login": "99", "password": "pw", "server": "Fake-MT5"})
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stateOf(t, ctx, pool, "down1"); got != "registered" {
		t.Fatalf("网关不可达应 stay registered got=%s", got)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='venue接入' AND esc_class='接入败'`).Scan(&n); err != nil || n == 0 {
		t.Fatalf("网关败行 n=%d（应自报）", n)
	}
}

// 锚④c 符号表空=fail-closed 自报。
func TestOnboardEmptySymbols(t *testing.T) {
	r, pool, _ := fixture(t)
	ctx := context.Background()
	r.dial = func(string) (Session, error) { return &fakeSession{syms: nil}, nil }
	seedCreds(t, ctx, pool, "empty1", map[string]string{
		"login": "77", "password": "pw", "server": "Fake-MT5"})
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stateOf(t, ctx, pool, "empty1"); got != "registered" {
		t.Fatalf("符号表空应 stay registered got=%s", got)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='venue接入' AND esc_class='接入败'`).Scan(&n); err != nil || n == 0 {
		t.Fatalf("符号表空败行 n=%d（应自报）", n)
	}
}

// 键形拆分锚：单例 mt5.*→'mt5'；<broker>.mt5.*→broker；冒号/杂形拒。
func TestCredKeySplit(t *testing.T) {
	for _, c := range []struct {
		k, v, f string
		ok      bool
	}{
		{"mt5.login", "mt5", "login", true},
		{"exness.mt5.password", "exness", "password", true},
		{"xm-01.mt5.host", "xm-01", "host", true},
		{"mt5.login", "mt5", "login", true},
		{"okx.api_key", "", "", false},
		{"bad venue.mt5.login", "", "", false},
	} {
		v, f, ok := credKeySplit(c.k)
		if ok != c.ok || v != c.v || f != c.f {
			t.Fatalf("%s → %q %q %v", c.k, v, f, ok)
		}
	}
}
