// bridge_test.go —— 桥腿桩测（T-0141）：凭据 env 契约 fail-closed（缺件=装配
// 错不拖 OKX 面）；symbols 键域 bridge.mt5.symbols 缺/空=不虚构标的集。
package mt5

import (
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
)

// 锚：env 零注入→NewBridgeCollector 报错（fail-closed 装配层 WARN 放行余桥）。
func TestBridgeCredsFailClosed(t *testing.T) {
	for _, k := range []string{"COLLECTO_MT5_LOGIN", "COLLECTO_MT5_PASSWORD",
		"COLLECTO_MT5_SERVER", "COLLECTO_MT5_GATEWAY_ADDR"} {
		t.Setenv(k, "")
	}
	pool := testutil.Pool(t)
	if _, err := NewBridgeCollector(pool, config.New(pool)); err == nil {
		t.Fatal("凭据零注入应装配败（桥腿缺席不拖 OKX 面）")
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
