// klineback_test.go —— 业主令 2026-11-15 观察名单并入桩测：活跃集闭集
// 外标的经名单入扫集（失明病灶根修钉死）；空名单/重复键幂等零副作用。
// 工单#18314：流动性序动扩排序件桩测（rankSwapTickers 纯函数面）。
package okxws

import (
	"encoding/json"
	"testing"
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
