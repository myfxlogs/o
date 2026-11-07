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

// stubGet 计数读件（path 分派 instruments SWAP/SPOT/FUTURES 载荷——
// T-0427 三族目录）。
func stubGet(t *testing.T, calls *int, swapRaw, spotRaw, futRaw json.RawMessage, err error) ScanGet {
	t.Helper()
	return func(ctx context.Context, path string) (json.RawMessage, error) {
		*calls++
		if err != nil {
			return nil, err
		}
		switch path {
		case "/api/v5/public/instruments?instType=SPOT":
			return spotRaw, nil
		case "/api/v5/public/instruments?instType=FUTURES":
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
	spotCatRaw = json.RawMessage(`[
		{"instId":"AAA-USDT","state":"live"},
		{"instId":"ZZZ-USDT","state":"live"},
		{"instId":"EEE-USDC","state":"live"},
		{"instId":"OLD-USDT","state":"suspend"}]`)
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
// 非 live/币本位 SWAP 滤除；SPOT 族 live -USDT 对入 spot∪all（T-0427——
// 非 SWAP 基币现货对 ZZZ-USDT 经族并集入宇宙，USDC/挂起滤除）；基币现货
// 派生兜底入 all 不入 spot；quarter 期货+uly 指数键入 all；非 quarter/
// preopen 期货滤除。
func TestUniverseCatalogParse(t *testing.T) {
	var calls int
	cat := NewUniverseCatalog(stubGet(t, &calls, swapCatRaw, spotCatRaw, futCatRaw, nil), 0)
	ctx := context.Background()
	all, err := cat.Symbols(ctx)
	if err != nil {
		t.Fatal(err)
	}
	hasAll(t, all, "AAA-USDT-SWAP", "BBB-USDT-SWAP", "AAA-USDT", "ZZZ-USDT",
		"BBB-USDT", "AAA-USD-261225", "AAA-USD")
	for _, bad := range []string{"OLD-USDT-SWAP", "CCC-USD-SWAP", "EEE-USDC",
		"OLD-USDT", "BBB-USD-261226", "DDD-USD-260627"} {
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
	if !set["AAA-USDT"] || !set["ZZZ-USDT"] || set["OLD-USDT-SWAP"] {
		t.Fatalf("Set 映射错: %v", set)
	}
}

// TestUniverseCatalogFamSets T-0427 族谱域闸：FamSet 按族取子集（行携
// inst_type 归位的供给面）；族读败=该族 err 上抛他族照常（fail-closed
// 粒度到族）；扫册外族拒（OPTION/MARGIN 票面不收）。
func TestUniverseCatalogFamSets(t *testing.T) {
	ctx := context.Background()
	cat := NewUniverseCatalog(stubGet(t, new(int), swapCatRaw, spotCatRaw, futCatRaw, nil), 0)
	swap, err := cat.FamSet("SWAP")(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(swap) != 2 || !swap["AAA-USDT-SWAP"] {
		t.Fatalf("SWAP 族集错: %v", swap)
	}
	spot, err := cat.FamSet("SPOT")(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(spot) != 2 || !spot["ZZZ-USDT"] || spot["EEE-USDC"] {
		t.Fatalf("SPOT 族集错（USDC 应键域滤除）: %v", spot)
	}
	fut, err := cat.FamSet("FUTURES")(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(fut) != 1 || !fut["AAA-USD-261225"] {
		t.Fatalf("FUTURES 族集错（quarter 形沿用）: %v", fut)
	}
	if _, err := cat.FamSet("OPTION")(ctx); err == nil {
		t.Fatal("扫册外族应 fail-closed 拒")
	}
}

// TestUniverseCatalogFamFail 族级 fail-closed：SPOT 族读败=FamSet("SPOT")
// 上抛让位、Symbols 面降级照供（基币派生兜底不连带）——他族照常。
func TestUniverseCatalogFamFail(t *testing.T) {
	ctx := context.Background()
	cat := NewUniverseCatalog(func(ctx context.Context, path string) (json.RawMessage, error) {
		switch path {
		case "/api/v5/public/instruments?instType=SPOT":
			return nil, errors.New("spot 面断")
		case "/api/v5/public/instruments?instType=FUTURES":
			return futCatRaw, nil
		}
		return swapCatRaw, nil
	}, 0)
	if _, err := cat.FamSet("SPOT")(ctx); err == nil {
		t.Fatal("SPOT 族读败应上抛让位")
	}
	swap, err := cat.FamSet("SWAP")(ctx)
	if err != nil {
		t.Fatalf("他族应照常: %v", err)
	}
	if len(swap) != 2 {
		t.Fatalf("SWAP 族集错: %v", swap)
	}
	all, err := cat.Symbols(ctx)
	if err != nil {
		t.Fatalf("Symbols 应降级不连带: %v", err)
	}
	hasAll(t, all, "AAA-USDT-SWAP", "AAA-USDT") // 基币派生兜底在
	for _, s := range all {
		if s == "ZZZ-USDT" {
			t.Fatal("SPOT 族败日族符不应虚构入宇宙")
		}
	}
}

// TestUniverseCatalogTTL 常驻缓存：TTL 内多消费方零重读（一处供给多处
// 消费——venue 不打爆）；缓存败不污旧账待复用。
func TestUniverseCatalogTTL(t *testing.T) {
	var calls int
	cat := NewUniverseCatalog(stubGet(t, &calls, swapCatRaw, spotCatRaw, futCatRaw, nil), 0)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := cat.Symbols(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := cat.SwapSet(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 3 { // SWAP+SPOT+FUTURES 各一——余拍全命中缓存
		t.Fatalf("TTL 内应零重读: calls=%d", calls)
	}
}

// TestUniverseCatalogFailClosed 读败/SWAP 零行/解析败=上抛不虚构；
// FUTURES 面败=降级不连带（SWAP 全集已得照供）；SPOT 族零 live=族病灶
// 记账（FamSet 上抛让位——Symbols 降级照供）。
func TestUniverseCatalogFailClosed(t *testing.T) {
	ctx := context.Background()
	var calls int
	// ① 读败=上抛零集。
	cat := NewUniverseCatalog(stubGet(t, &calls, nil, nil, nil, errors.New("venue 断")), 0)
	if _, err := cat.Symbols(ctx); err == nil {
		t.Fatal("读败应上抛 fail-closed")
	}
	// ② SWAP live 零行=目录面病灶上抛。
	calls = 0
	cat = NewUniverseCatalog(stubGet(t, &calls,
		json.RawMessage(`[{"instId":"OLD-USDT-SWAP","state":"preopen"}]`), spotCatRaw, futCatRaw, nil), 0)
	if _, err := cat.SwapSet(ctx); err == nil {
		t.Fatal("SWAP 零行应上抛 fail-closed")
	}
	// ③ FUTURES 败=降级照供 SWAP 全集+SPOT 族。
	calls = 0
	cat = NewUniverseCatalog(func(ctx context.Context, path string) (json.RawMessage, error) {
		calls++
		if path == "/api/v5/public/instruments?instType=FUTURES" {
			return nil, errors.New("futures 面断")
		}
		if path == "/api/v5/public/instruments?instType=SPOT" {
			return spotCatRaw, nil
		}
		return swapCatRaw, nil
	}, 0)
	all, err := cat.Symbols(ctx)
	if err != nil {
		t.Fatalf("FUTURES 败应降级不连带: %v", err)
	}
	hasAll(t, all, "AAA-USDT-SWAP", "BBB-USDT-SWAP", "AAA-USDT", "ZZZ-USDT")
	if _, err := cat.FamSet("FUTURES")(ctx); err == nil {
		t.Fatal("FUTURES 族读败应 FamSet 上抛（族让位非静默虚构空集）")
	}
	// ④ SPOT 族 live 零行=族病灶——FamSet("SPOT") 上抛，他族照常。
	calls = 0
	cat = NewUniverseCatalog(stubGet(t, &calls, swapCatRaw,
		json.RawMessage(`[{"instId":"OLD-USDT","state":"suspend"}]`), futCatRaw, nil), 0)
	if _, err := cat.FamSet("SPOT")(ctx); err == nil {
		t.Fatal("SPOT live 零行应族病灶上抛")
	}
	if _, err := cat.FamSet("SWAP")(ctx); err != nil {
		t.Fatalf("他族应照常: %v", err)
	}
}
