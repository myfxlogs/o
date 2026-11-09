// bridge_test.go —— 桥腿桩测（T-0141）：凭据/址链归 app_config fail-closed
// （缺键=会话返错不虚构）；symbols 键域 bridge.mt5.symbols 缺/空=不虚构标的集。
package mt5

import (
	"context"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 锚：凭据/址链全归 app_config（D-0005 env 只留 DSN 钦定收口）——
// bridge.mt5.* 四键齐→bridgeAuth 出凭据；缺键→fail-closed error。
func TestBridgeAuthFromConfig(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if _, err := bridgeAuth(cfg); err == nil {
		t.Fatal("快照未就绪应 err（fail-closed）")
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := bridgeAuth(cfg); err == nil {
		t.Fatal("bridge.mt5.* 全缺应 err（不虚构凭据）")
	}
	for k, v := range map[string]string{
		"bridge.mt5.password": "pw",
		"bridge.mt5.server":   "srv",
	} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text))", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, "INSERT INTO app_config(key,value) VALUES('bridge.mt5.login',to_jsonb(3001157))"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	cr, err := bridgeAuth(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cr.Login != 3001157 || cr.Password != "pw" || cr.Server != "srv" || cr.Host != "" {
		t.Fatalf("cr=%+v 对不上（host 缺省空串合法）", cr)
	}
}

// 锚：symbols 键域热启——缺=空集（bridgeSession fail-closed 返错不订阅）。
func TestBridgeSymbolsFailClosed(t *testing.T) {
	pool := testutil.Pool(t)
	c := &BridgeCollector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5bridge", "test")}
	if syms := c.symbols(); len(syms) != 0 {
		t.Fatalf("键缺应空集 got %v", syms)
	}
}
