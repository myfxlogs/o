// klineback_test.go —— 业主令 2026-11-15 观察名单并入桩测：活跃集闭集
// 外标的经名单入扫集（失明病灶根修钉死）；空名单/重复键幂等零副作用。
package okxws

import "testing"

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
