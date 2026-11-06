// universe_test.go —— T-0416 宇宙扩件供给件桩测：目录解析（SWAP live
// 全集+基币现货派生+季度期货+uly）/TTL 常驻缓存（内命中零重读）/
// fail-closed 三面（读败/SWAP 零行/解析败——不虚构标的集）。
package okxws

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// stubGet 计数读件（path 分派 instruments SWAP/FUTURES 载荷）。
func stubGet(t *testing.T, calls *int, swapRaw, futRaw json.RawMessage, err error) ScanGet {
	t.Helper()
	return func(ctx context.Context, path string) (json.RawMessage, error) {
		*calls++
		if err != nil {
			return nil, err
		}
		if path == "/api/v5/public/instruments?instType=FUTURES" {
			return futRaw, nil
		}
		return swapRaw, nil
	}
}

var (
	swapCatRaw = json.RawMessage(`[
		{"instId":"AAA-USDT-SWAP","state":"live"},
		{"instId":"BBB-USDT-SWAP","state":"live"},
		{"instId":"OLD-USDT-SWAP","state":"preopen"},
		{"instId":"CCC-USD-SWAP","state":"live"}]`)
	futCatRaw = json.RawMessage(`[
		{"instId":"AAA-USD-261225","alias":"quarter","uly":"AAA-USD","state":"live"},
		{"instId":"BBB-USD-261226","alias":"this_week","uly":"BBB-USD","state":"live"},
		{"instId":"DDD-USD-260627","alias":"quarter","uly":"DDD-USD","state":"preopen"}]`)
)

func hasAll(t *testing.T, got []string, want ...string) {
	t.Helper()
	m := map[string]bool{}
	for _, s := range got {
		m[s] = true
	}
	for _, w := range want {
		if !m[w] {
			t.Fatalf("宇宙缺 %s: %v", w, got)
		}
	}
}

// TestUniverseCatalogParse 目录枚举语义：live -USDT-SWAP 入 swap∪all；
// 非 live/币本位 SWAP 滤除；基币现货派生入 all 不入 swap；quarter 期货+
// uly 指数键入 all；非 quarter/preopen 期货滤除。
func TestUniverseCatalogParse(t *testing.T) {
	var calls int
	cat := NewUniverseCatalog(stubGet(t, &calls, swapCatRaw, futCatRaw, nil), 0)
	ctx := context.Background()
	all, err := cat.Symbols(ctx)
	if err != nil {
		t.Fatal(err)
	}
	hasAll(t, all, "AAA-USDT-SWAP", "BBB-USDT-SWAP", "AAA-USDT", "BBB-USDT",
		"AAA-USD-261225", "AAA-USD")
	for _, bad := range []string{"OLD-USDT-SWAP", "CCC-USD-SWAP", "BBB-USD-261226", "DDD-USD-260627"} {
		for _, s := range all {
			if s == bad {
				t.Fatalf("滤除键混入宇宙: %s", bad)
			}
		}
	}
	// SwapSet=SWAP 目录全集（现货派生/期货/uly 不入谱域）。
	swap, err := cat.SwapSet(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(swap) != 2 || !swap["AAA-USDT-SWAP"] || !swap["BBB-USDT-SWAP"] {
		t.Fatalf("SwapSet 谱域错: %v", swap)
	}
	// Set=订阅宇宙 map 形同 Symbols。
	set, err := cat.Set(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !set["AAA-USDT"] || set["OLD-USDT-SWAP"] {
		t.Fatalf("Set 映射错: %v", set)
	}
}

// TestUniverseCatalogTTL 常驻缓存：TTL 内多消费方零重读（一处供给多处
// 消费——venue 不打爆）；缓存败不污旧账待复用。
func TestUniverseCatalogTTL(t *testing.T) {
	var calls int
	cat := NewUniverseCatalog(stubGet(t, &calls, swapCatRaw, futCatRaw, nil), 0)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := cat.Symbols(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := cat.SwapSet(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 2 { // SWAP+FUTURES 各一——余拍全命中缓存
		t.Fatalf("TTL 内应零重读: calls=%d", calls)
	}
}

// TestUniverseCatalogFailClosed 读败/SWAP 零行/解析败=上抛不虚构；
// FUTURES 面败=降级不连带（SWAP 全集已得照供）。
func TestUniverseCatalogFailClosed(t *testing.T) {
	ctx := context.Background()
	var calls int
	// ① 读败=上抛零集。
	cat := NewUniverseCatalog(stubGet(t, &calls, nil, nil, errors.New("venue 断")), 0)
	if _, err := cat.Symbols(ctx); err == nil {
		t.Fatal("读败应上抛 fail-closed")
	}
	// ② SWAP live 零行=目录面病灶上抛。
	calls = 0
	cat = NewUniverseCatalog(stubGet(t, &calls,
		json.RawMessage(`[{"instId":"OLD-USDT-SWAP","state":"preopen"}]`), futCatRaw, nil), 0)
	if _, err := cat.SwapSet(ctx); err == nil {
		t.Fatal("SWAP 零行应上抛 fail-closed")
	}
	// ③ FUTURES 败=降级照供 SWAP 全集。
	calls = 0
	cat = NewUniverseCatalog(stubGet(t, &calls, swapCatRaw, nil, nil), 0)
	cat.get = func(ctx context.Context, path string) (json.RawMessage, error) {
		calls++
		if path == "/api/v5/public/instruments?instType=FUTURES" {
			return nil, errors.New("futures 面断")
		}
		return swapCatRaw, nil
	}
	all, err := cat.Symbols(ctx)
	if err != nil {
		t.Fatalf("FUTURES 败应降级不连带: %v", err)
	}
	hasAll(t, all, "AAA-USDT-SWAP", "BBB-USDT-SWAP", "AAA-USDT", "BBB-USDT")
}
