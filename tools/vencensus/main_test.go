package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTSV(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "u.tsv")
	os.WriteFile(p, []byte("/api/v5/trade/order\ttrade\n/api/v5/tradingBot/grid/order-algo\ttradingBot\n\n"), 0o644)
	items, err := loadTSV(p)
	if err != nil || len(items) != 2 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if items[1].Fam != "tradingBot" {
		t.Fatalf("fam=%q", items[1].Fam)
	}
}

func TestRPCItems(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.proto")
	os.WriteFile(p, []byte("service S {\n  rpc ConnectThread (A) returns (B);\n\trpc OrderSend (C) returns (D); // x\n}\n// rpc Fake (\n"), 0o644)
	items, err := rpcItems(p)
	if err != nil || len(items) != 2 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if items[0].Face != "ConnectThread" {
		t.Fatalf("face=%q", items[0].Face)
	}
}

func TestGapDiff(t *testing.T) {
	items := []item{
		{Face: "/api/v5/trade/order", Fam: "trade"},
		{Face: "/api/v5/tradingBot/grid/order-algo", Fam: "tradingBot"},
		{Face: "/api/v5/finance/savings/balance", Fam: "finance"},
	}
	canon := []byte("/api/v5/trade/order 已测")                     // canon 登记
	code := []byte(`const p = "/api/v5/finance/savings/balance"`) // 码面调用
	g := gap("okx", items, canon, code)
	if len(g) != 1 || len(g["tradingBot"]) != 1 {
		t.Fatalf("gap=%v（tradingBot 族须入差集——实证锚）", g)
	}
}

func TestBinanceItems(t *testing.T) {
	code := []byte(`u := "https://fapi.binance.com/fapi/v1/fundingInfo" + "https://fapi.binance.com/fapi/v1/premiumIndex"`)
	items := binanceItems(code)
	if len(items) != 2 {
		t.Fatalf("items=%v", items)
	}
}

func TestSnapshotStable(t *testing.T) {
	dir := t.TempDir()
	g := map[string]map[string][]string{
		"okx": {"tradingBot": {"/api/v5/tradingBot/b", "/api/v5/tradingBot/a"}},
	}
	p := filepath.Join(dir, "s.md")
	if err := writeSnapshot(p, "test", g); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !contains(string(b), "tradingBot (2)") {
		t.Fatalf("snap 缺族头:\n%s", b)
	}
	// 排序稳定——diff 基线钦定
	ai, bi := indexOf(string(b), "tradingBot/a"), indexOf(string(b), "tradingBot/b")
	if ai < 0 || ai > bi {
		t.Fatal("端点未排序")
	}
}

func contains(s, sub string) bool { return indexOf(s, sub) >= 0 }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
