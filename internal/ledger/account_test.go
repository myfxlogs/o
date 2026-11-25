package ledger

import (
	"context"
	"testing"
)

// O-39440 桩测：账户维解析/归户/折叠纯函数案。

// TestAccountResolveTrigger 归户触发器集成案（O-39440② 注册表驱动——
// 硬编码 venue→account 已撤,Post 显式或 NULL 交 fin_cashflow_account_default;
// 断言=落库 account_id 实值,覆盖 okx kind 规则/cfd 注册表/legacy 兜底/
// disabled 仍归户/closed→legacy/多 main 歧义→legacy/显式直通）。
func TestAccountResolveTrigger(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	post := func(venue, kind, ccy string, acct string) string {
		t.Helper()
		if _, err := Post(ctx, pool, Entry{Kind: kind, Venue: venue, Ccy: ccy,
			Amount: 1, SourceRef: 1, SourceTable: "t", AccountID: acct}); err != nil {
			t.Fatalf("Post(%s,%s): %v", venue, kind, err)
		}
		var id string
		if err := pool.QueryRow(ctx, `SELECT account_id FROM fin_cashflow
			WHERE venue=$1 AND kind=$2 ORDER BY id DESC LIMIT 1`, venue, kind).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	// okx kind 规则（触发器 CASE 同源）
	for _, c := range []struct{ kind, want string }{
		{"deposit", "okx:funding"}, {"withdraw", "okx:funding"}, {"owner_in", "okx:funding"},
		{"earn_in", "okx:earn"}, {"interest", "okx:earn"}, {"earn_out", "okx:earn"},
		{"dcd_out", "okx:dcd"}, {"dcd_in", "okx:dcd"},
		{"recon_adjust", "okx:legacy"}, {"unmapped_bill", "okx:legacy"},
		{"fee", "okx:trading"}, {"trade_pnl", "okx:trading"},
	} {
		if got := post("okx", c.kind, "USDT", ""); got != c.want {
			t.Fatalf("okx %s 归户=%s want %s", c.kind, got, c.want)
		}
	}
	// cfd 注册表驱动（venue 唯一活跃 main 户——frontend 插行同径）
	if got := post("exness", "fee", "USD", ""); got != "exness:3001157" {
		t.Fatalf("exness 归户=%s", got)
	}
	if got := post("xmglobal", "deposit", "USD", ""); got != "xmglobal:110660304" {
		t.Fatalf("xmglobal 归户=%s", got)
	}
	// 新经纪商=插行即归户（零码——fxpro 桩行实证通用性）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_account(account_id,venue,account_no,alias,ccy,sub_type,state)
		VALUES('fxpro:88001','fxpro','88001','FXPro','USD','main','active')`); err != nil {
		t.Fatal(err)
	}
	if got := post("fxpro", "fee", "USD", ""); got != "fxpro:88001" {
		t.Fatalf("fxpro 注册表归户=%s want fxpro:88001", got)
	}
	// disabled 仍归户（钱照样在本户——停用=运营面非簿记面）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_account(account_id,venue,account_no,alias,ccy,sub_type,state)
		VALUES('tstdis:1','tstdis','1','t','USDT','main','disabled')`); err != nil {
		t.Fatal(err)
	}
	if got := post("tstdis", "fee", "USDT", ""); got != "tstdis:1" {
		t.Fatalf("disabled 户归属=%s want tstdis:1", got)
	}
	// closed→legacy（退役户不再收新归属）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_account(account_id,venue,account_no,alias,ccy,sub_type,state)
		VALUES('tstcls:1','tstcls','1','t','USDT','main','closed')`); err != nil {
		t.Fatal(err)
	}
	if got := post("tstcls", "fee", "USDT", ""); got != "tstcls:legacy" {
		t.Fatalf("closed 户归属=%s want tstcls:legacy", got)
	}
	// 双活跃 main=歧义→legacy（禁猜）
	for _, id := range []string{"tstmulti:a", "tstmulti:b"} {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_account(account_id,venue,account_no,alias,ccy,sub_type,state)
			VALUES($1,'tstmulti','1','t','USDT','main','active')`, id); err != nil {
			t.Fatal(err)
		}
	}
	if got := post("tstmulti", "fee", "USDT", ""); got != "tstmulti:legacy" {
		t.Fatalf("歧义归属=%s want tstmulti:legacy", got)
	}
	// 无册 venue→legacy 兜底自动开户（sim 域有演习闸禁簿记——用中性 token）
	if got := post("tstnone", "fee", "USDT", ""); got != "tstnone:legacy" {
		t.Fatalf("tstnone 归属=%s want tstnone:legacy", got)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_account WHERE account_id='tstnone:legacy'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("legacy 自动开户缺失 n=%d err=%v", n, err)
	}
	// 显式 account_id 直通（触发器不覆写——含 okx kind 规则域外显式）
	if got := post("okx", "fee", "USDT", "okx:earn"); got != "okx:earn" {
		t.Fatalf("显式归属=%s want okx:earn", got)
	}
}

func TestVenueOfAndCompAccount(t *testing.T) {
	if VenueOf("okx:trading") != "okx" || VenueOf("exness:3001157") != "exness" || VenueOf("okx") != "okx" {
		t.Fatal("VenueOf 折叠错")
	}
	accts := []AccountRow{
		{ID: "okx:trading", Venue: "okx", SubType: "trading", State: "active"},
		{ID: "okx:funding", Venue: "okx", SubType: "funding", State: "active"},
		{ID: "okx:earn", Venue: "okx", SubType: "earn", State: "active"},
		{ID: "okx:dcd", Venue: "okx", SubType: "dcd", State: "active"},
		{ID: "exness:3001157", Venue: "exness", SubType: "main", State: "active"},
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
	if compAccount(accts, "exness", "trading") != "exness:3001157" {
		t.Fatal("exness 单 main 应兜底归属")
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
		{"okx:earn", "BTC"}: 1, {"exness:3001157", "USD"}: 2000,
	}
	wl := map[[2]string]bool{{"okx:trading", "USDT"}: true, {"okx:funding", "USDT"}: true}
	venV, wlV := foldAcctKeys(ven, wl)
	if venV[[2]string{"okx", "USDT"}] != 150 || venV[[2]string{"okx", "BTC"}] != 1 ||
		venV[[2]string{"exness", "USD"}] != 2000 {
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
	// 台账组件桩：parked {okx,TSTP}、dcd {okx,TSTD}、native {okx,TSTN}
	//   （异域 ccy 防共库他测/迁移种子行求和互踩——intent ccy 自由文本）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind)
		VALUES ('{"op":"residual_park","venue":"okx","ccy":"TSTP","amt":10}','allow','allow',1,'open'),
		       ('{"op":"dcd_baseline","venue":"okx","ccy":"TSTD","amt":0.5}','allow','allow',1,'open'),
		       ('{"op":"native_baseline","venue":"okx","ccy":"TSTN","amt":0.1}','allow','allow',1,'open')`); err != nil {
		t.Fatal(err)
	}
	wl, parked, dcdB, natB, err := ReconSets(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	// parked→okx:earn、dcd→okx:dcd、native→okx:trading 账户键
	for _, k := range [][2]string{{"okx:earn", "TSTP"}, {"okx:dcd", "TSTD"}, {"okx:trading", "TSTN"}} {
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
	if parked[[2]string{"okx", "TSTP"}] != 10 || dcdB[[2]string{"okx", "TSTD"}] != 0.5 ||
		natB[[2]string{"okx", "TSTN"}] != 0.1 {
		t.Fatalf("台账组件 venue 级量错 parked=%v dcdB=%v natB=%v", parked, dcdB, natB)
	}
}
