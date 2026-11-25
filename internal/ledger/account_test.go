package ledger

import (
	"context"
	"testing"
)

// O-39440 桩测：账户维解析/归户/折叠纯函数案。

func TestAccountForRules(t *testing.T) {
	cases := []struct{ venue, kind, expl, want string }{
		{"okx", "deposit", "", "okx:funding"},
		{"okx", "withdraw", "", "okx:funding"},
		{"okx", "owner_in", "", "okx:funding"},
		{"okx", "earn_in", "", "okx:earn"},
		{"okx", "interest", "", "okx:earn"},
		{"okx", "dcd_settle", "", "okx:dcd"},
		{"okx", "dcd_owner", "", "okx:dcd"},
		{"okx", "recon_adjust", "", "okx:legacy"},
		{"okx", "spot_buy", "", "okx:trading"},
		{"okx", "fee", "", "okx:trading"},
		{"mt5", "deposit", "", "mt5:3001157"}, // 回填同规则——Exness 户实证
		{"mt5", "deposit", "mt5:110660304", "mt5:110660304"},
		{"xmglobal", "deposit", "", "xmglobal:110660304"},
		{"sim", "fee", "", "sim:legacy"},
		{"tron", "prop_credit", "", "tron:legacy"},
	}
	for _, c := range cases {
		if got := AccountFor(c.venue, c.kind, c.expl); got != c.want {
			t.Fatalf("AccountFor(%s,%s,%s)=%s want %s", c.venue, c.kind, c.expl, got, c.want)
		}
	}
}

func TestVenueOfAndCompAccount(t *testing.T) {
	if VenueOf("okx:trading") != "okx" || VenueOf("mt5:3001157") != "mt5" || VenueOf("okx") != "okx" {
		t.Fatal("VenueOf 折叠错")
	}
	accts := []AccountRow{
		{ID: "okx:trading", Venue: "okx", SubType: "trading", State: "active"},
		{ID: "okx:funding", Venue: "okx", SubType: "funding", State: "active"},
		{ID: "okx:earn", Venue: "okx", SubType: "earn", State: "active"},
		{ID: "okx:dcd", Venue: "okx", SubType: "dcd", State: "active"},
		{ID: "mt5:3001157", Venue: "mt5", SubType: "main", State: "active"},
		{ID: "xmglobal:110660304", Venue: "xmglobal", SubType: "main", State: "active"},
		{ID: "okx:legacy", Venue: "okx", SubType: "main", State: "frozen"}, // frozen 不计
	}
	if compAccount(accts, "okx", "earn") != "okx:earn" {
		t.Fatal("okx earn 归户错")
	}
	if compAccount(accts, "okx", "trading") != "okx:trading" {
		t.Fatal("okx trading 归户错")
	}
	// O-39440 通用模型：Exness/XM 分属 venue token——各 venue 单 main
	// →trading 组件兜底归属成立（双 main 歧义情形不复存在）。
	if compAccount(accts, "mt5", "trading") != "mt5:3001157" {
		t.Fatal("mt5 单 main 应兜底归属")
	}
	if compAccount(accts, "xmglobal", "trading") != "xmglobal:110660304" {
		t.Fatal("xmglobal 单 main 应兜底归属")
	}
	if compAccount(accts, "sim", "trading") != "" {
		t.Fatal("无账户 venue 不归属")
	}
}

func TestFoldAcctKeysAndParts(t *testing.T) {
	ven := map[[2]string]float64{
		{"okx:trading", "USDT"}: 100, {"okx:funding", "USDT"}: 50,
		{"okx:earn", "BTC"}: 1, {"mt5:3001157", "USD"}: 2000,
	}
	wl := map[[2]string]bool{{"okx:trading", "USDT"}: true, {"okx:funding", "USDT"}: true}
	venV, wlV := foldAcctKeys(ven, wl)
	if venV[[2]string{"okx", "USDT"}] != 150 || venV[[2]string{"okx", "BTC"}] != 1 ||
		venV[[2]string{"mt5", "USD"}] != 2000 {
		t.Fatalf("venue 折叠错: %v", venV)
	}
	if !wlV[[2]string{"okx", "USDT"}] || wlV[[2]string{"okx", "BTC"}] {
		t.Fatalf("wl 折叠错: %v", wlV)
	}
	parts := VenueParts{{"okx:trading", "USDT"}: {{Name: "trading", Amount: 100}}}
	if got := acctParts(parts, "okx:trading", "USDT"); len(got) != 1 || got[0].Name != "trading" {
		t.Fatal("acctParts 账户键未中")
	}
	if got := acctParts(parts, "okx:funding", "USDT"); len(got) != 0 {
		t.Fatal("acctParts venue 兜底不应误配他账户键")
	}
}

// ReconSets 账户域白名单集成案——fin_account active 行×台账组件
// 归户键（parked→earn 等）落 wl；venue 桶键不再生成。
func TestReconSetsAccountKeys(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	// 台账组件桩：parked {okx,DOGE}、dcd {okx,BTC}、native {okx,ETH}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind)
		VALUES ('{"op":"residual_park","venue":"okx","ccy":"DOGE","amt":10}','allow','t',1,'open'),
		       ('{"op":"dcd_baseline","venue":"okx","ccy":"BTC","amt":0.5}','allow','t',1,'open'),
		       ('{"op":"native_baseline","venue":"okx","ccy":"ETH","amt":0.1}','allow','t',1,'open')`); err != nil {
		t.Fatal(err)
	}
	wl, parked, dcdB, natB, err := ReconSets(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	// parked→okx:earn、dcd→okx:dcd、native→okx:trading 账户键
	for _, k := range [][2]string{{"okx:earn", "DOGE"}, {"okx:dcd", "BTC"}, {"okx:trading", "ETH"}} {
		if !wl[k] {
			t.Fatalf("wl 缺账户键 %v——全键: %v", k, wl)
		}
	}
	// venue 桶键不得再作对拍键（{"okx",ccy} 形）
	for k := range wl {
		if k[0] == "okx" {
			t.Fatalf("venue 桶键残留对拍面: %v", k)
		}
	}
	if parked[[2]string{"okx", "DOGE"}] != 10 || dcdB[[2]string{"okx", "BTC"}] != 0.5 ||
		natB[[2]string{"okx", "ETH"}] != 0.1 {
		t.Fatal("台账组件 venue 级量错")
	}
}
