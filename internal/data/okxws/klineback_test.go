// klineback_test.go —— 业主令 2026-11-15 观察名单并入桩测：活跃集闭集
// 外标的经名单入扫集（失明病灶根修钉死）；空名单/重复键幂等零副作用。
// 工单#18314：流动性序动扩排序件桩测（rankSwapTickers 纯函数面）。
package okxws

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestWatchUnion 名单并入语义三断言：集外键补入、集内键幂等、空串滤除。
func TestWatchUnion(t *testing.T) {
	set := map[string]bool{"AAA-USDT-SWAP": true}
	watchUnion(set, []string{"BTC-USDT-SWAP", "AAA-USDT-SWAP", ""})
	if !set["BTC-USDT-SWAP"] {
		t.Fatal("名单标的未入扫集——失明病灶在")
	}
	if len(set) != 2 {
		t.Fatalf("重复键/空串污染扫集: got %d want 2", len(set))
	}
}

// TestWatchUnionEmpty 空名单零副作用（缺键=纯活跃集行为不变）。
func TestWatchUnionEmpty(t *testing.T) {
	set := map[string]bool{"AAA-USDT-SWAP": true}
	watchUnion(set, nil)
	if len(set) != 1 {
		t.Fatalf("空名单改集: got %d want 1", len(set))
	}
}

// TestRankSwapTickers 排序三断言：volCcy24h 降序、币本位/现货滤除
// （USDT 计价闸）、零量/坏载荷不崩。
func TestRankSwapTickers(t *testing.T) {
	raw := json.RawMessage(`[
		{"instId":"DOGE-USDT-SWAP","volCcy24h":"900.5"},
		{"instId":"BTC-USDT-SWAP","volCcy24h":"50000"},
		{"instId":"BTC-USD-SWAP","volCcy24h":"999999"},
		{"instId":"ETH-USDT-SWAP","volCcy24h":"30000"},
		{"instId":"BTC-USDT","volCcy24h":"80000"},
		{"instId":"NEW-USDT-SWAP","volCcy24h":""}
	]`)
	r := rankSwapTickers(raw)
	want := []string{"BTC-USDT-SWAP", "ETH-USDT-SWAP", "DOGE-USDT-SWAP", "NEW-USDT-SWAP"}
	if len(r) != len(want) {
		t.Fatalf("滤面行数: got %d want %d", len(r), len(want))
	}
	for i, inst := range want {
		if r[i].inst != inst {
			t.Fatalf("序位 %d: got %s want %s", i, r[i].inst, inst)
		}
	}
	if rankSwapTickers(json.RawMessage(`not-json`)) != nil {
		t.Fatal("坏载荷应返 nil")
	}
}

// TestRankSpotTickers O-41366 现货排序滤形三断言：-USDT 后缀收、
// -USDT-SWAP/-USDC/币本位/-USD 形全拒（USDT 计价闸域内钉死）。
func TestRankSpotTickers(t *testing.T) {
	raw := json.RawMessage(`[
		{"instId":"ETH-USDT","volCcy24h":"40000"},
		{"instId":"BTC-USDT","volCcy24h":"90000"},
		{"instId":"SOL-USDT","volCcy24h":"20000"},
		{"instId":"BTC-USDT-SWAP","volCcy24h":"999999"},
		{"instId":"ETH-USDC","volCcy24h":"88888"},
		{"instId":"BTC-USD","volCcy24h":"77777"},
		{"instId":"DOGE-USDT","volCcy24h":""}
	]`)
	r := rankSpotTickers(raw)
	want := []string{"BTC-USDT", "ETH-USDT", "SOL-USDT", "DOGE-USDT"}
	if len(r) != len(want) {
		t.Fatalf("滤面行数: got %d want %d (%v)", len(r), len(want), r)
	}
	for i, inst := range want {
		if r[i].inst != inst {
			t.Fatalf("序位 %d: got %s want %s", i, r[i].inst, inst)
		}
	}
	if rankSpotTickers(json.RawMessage(`not-json`)) != nil {
		t.Fatal("坏载荷应返 nil")
	}
}

// univExpandCfgKeys okxws.kline_* 键播种面（fin 域迁移+键 upsert——
// univExpand 读拍驱动不动 app_config，测试面直插）。
func univExpandCfg(t *testing.T, keys map[string]string) *Collector {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for k, v := range keys {
		if _, err := pool.Exec(ctx,
			`INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text))
			 ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return NewCollector(pool, cfg)
}

// TestUnivExpandSpotMerge O-41366 验收③：SPOT tickers 单发→-USDT 现货
// 并入扫集（BTC-USDT 真键入集；-USDC/-SWAP 形滤出）；SWAP 臂照旧并入。
func TestUnivExpandSpotMerge(t *testing.T) {
	c := univExpandCfg(t, map[string]string{
		"okxws.kline_topn":              "0", // SWAP 主径关——隔离现货臂断言
		"okxws.kline_census_hours":      "0",
		"okxws.kline_spot_topn":         "2",
		"okxws.kline_spot_census_hours": "0",
	})
	var spotSeen, swapSeen bool
	c.AttachRestGet(func(_ context.Context, pq string) (json.RawMessage, error) {
		switch {
		case strings.Contains(pq, "instType=SPOT"):
			spotSeen = true
			return json.RawMessage(`[
				{"instId":"BTC-USDT","volCcy24h":"90000"},
				{"instId":"ETH-USDT","volCcy24h":"40000"},
				{"instId":"SOL-USDT","volCcy24h":"20000"},
				{"instId":"XRP-USDC","volCcy24h":"99999"},
				{"instId":"DOGE-USDT-SWAP","volCcy24h":"88888"}]`), nil
		case strings.Contains(pq, "instType=SWAP"):
			swapSeen = true
			return json.RawMessage(`[]`), nil
		}
		return nil, nil
	})
	set := map[string]bool{}
	c.univExpand(context.Background(), set)
	if !spotSeen {
		t.Fatal("instType=SPOT 拉取未发——现货臂未接线")
	}
	if !set["BTC-USDT"] || !set["ETH-USDT"] {
		t.Fatalf("现货 top2 未并入集: %v", set)
	}
	if set["SOL-USDT"] {
		t.Fatal("topn=2 外 SOL-USDT 不应并入")
	}
	if set["XRP-USDC"] || set["DOGE-USDT-SWAP"] {
		t.Fatalf("非 -USDT 后缀形混入集: %v", set)
	}
	if swapSeen {
		t.Fatal("SWAP 双径全关不应发 SWAP 拉取")
	}
}

// TestUnivExpandSpotTopNZero ②okxws.kline_spot_topn=0→现货臂零副作用
// （零拉取零并入——回退闸实证；SWAP 臂独立键不受影响）。
func TestUnivExpandSpotTopNZero(t *testing.T) {
	c := univExpandCfg(t, map[string]string{
		"okxws.kline_topn":              "0",
		"okxws.kline_census_hours":      "0",
		"okxws.kline_spot_topn":         "0",
		"okxws.kline_spot_census_hours": "0",
	})
	var anyFetch bool
	c.AttachRestGet(func(_ context.Context, pq string) (json.RawMessage, error) {
		if strings.Contains(pq, "market/tickers") {
			anyFetch = true
		}
		return json.RawMessage(`[]`), nil
	})
	set := map[string]bool{"AAA-USDT-SWAP": true}
	c.univExpand(context.Background(), set)
	if anyFetch {
		t.Fatal("四径全关不应发任何 tickers 拉取")
	}
	if len(set) != 1 {
		t.Fatalf("关闸后集被污染: %v", set)
	}
}
