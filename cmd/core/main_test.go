package main

import (
	"context"
	"o/internal/gen/dashboardv1"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"testing"

	"o/internal/guard"
)

// T-0043 锚①：Warden 平仓径双腿必置 Close=true（漏置=按开仓腿装配→
// short 仓上 buy 反向开 long 非平仓——阻断级雷）。
func TestCloseLegOfSetsClose(t *testing.T) {
	spec := guard.CloseSpec{NotionalUSD: 100, Legs: [2]guard.CloseLeg{
		{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: 1},
		{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: 0.0012},
	}}
	for i, l := range spec.Legs {
		got := closeLegOf(spec, l)
		if !got.Close {
			t.Fatalf("leg%d Close 漏置（反向开仓雷）: %+v", i+1, got)
		}
		if got.Venue != l.Venue || got.InstID != l.InstID || got.Side != l.Side || got.Notional != 50 {
			t.Fatalf("leg%d 装配串味: %+v", i+1, got)
		}
	}
}

// T-0043 欠件锚（T-0046 补）：closeIntent Kind 必为 "close"（live_switch 翻 false
// 时平仓被拦=安全律反向——Kind 错=平仓被开仓闸吞）。
func TestCloseIntentKindIsClose(t *testing.T) {
	in := closeIntent(guard.CloseSpec{DecisionID: 1, StrategyRef: 1, InstID: "BTC-USDT-SWAP", NotionalUSD: 100,
		Legs: [2]guard.CloseLeg{{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: 1},
			{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: 0.001}}})
	if in.Kind != "close" {
		t.Fatalf("close Intent Kind=%q（应 close——免闸免赎语义位）", in.Kind)
	}
}

// T-0086 ①②④锚：NAV 三户合计（交易户帧+资金户+Earn 活读，缺户=0）+钱在哪
// 三分卡逐币行+非在册币种遗留资产标。
func TestOverviewThreeAccount(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM md_account_events WHERE channel IN ('account','earn_parking')"); err != nil {
		t.Fatal(err)
	}
	// 交易户帧：USDT eqUsd=2100 + AVAX 尘 3（非在册）
	if _, err := pool.Exec(ctx, `INSERT INTO md_account_events(ts,channel,ccy,data)
		VALUES(now(),'account','USDT',$1)`,
		[]byte(`{"details":[{"ccy":"USDT","eqUsd":"2100"},{"ccy":"AVAX","eqUsd":"3"}]}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at) VALUES('deposit','okx','USDT',100,1,'t',now())"); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool,
		fundingLive: func(context.Context) (map[string]float64, error) {
			return map[string]float64{"USDT": 0.2, "COMP": 0.05}, nil
		},
		earnLive: func(context.Context) (map[string]float64, error) {
			return map[string]float64{"USDT": 2057.37, "DOGE": 295.0}, nil
		}}
	o, err := svc.Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	// NAV=2100+3+0.2+0.05+2057.37（amt估值：DOGE 无 tick=0 不入 NAV 仍上卡）
	if o.NavTotal < 4160 || o.NavTotal > 4161 {
		t.Fatalf("NAV 三户合计: %v", o.NavTotal)
	}
	sec := map[string][]*dashboardv1.MoneyRow{}
	for _, m := range o.Money {
		sec[m.Section] = append(sec[m.Section], m)
	}
	if len(sec["交易户"]) != 2 || len(sec["资金户"]) != 2 || len(sec["Earn"]) != 2 {
		t.Fatalf("三分卡行数: %+v", o.Money)
	}
	for _, m := range o.Money {
		if m.Ccy == "USDT" && m.Legacy {
			t.Fatalf("USDT 在册不应标遗留: %+v", m)
		}
		if (m.Ccy == "AVAX" || m.Ccy == "COMP" || m.Ccy == "DOGE") && !m.Legacy {
			t.Fatalf("账外币应标遗留资产: %+v", m)
		}
	}
	// ④：Earn 活读（无 earn_parking 帧仍出 Earn 行——venue 唯一事实源）
	if len(sec["Earn"]) == 0 {
		t.Fatal("Earn 活读行缺席")
	}
}

// T-0086 ①：缺户=0 不误报——两读件 nil+无帧 → NAV=0 零炸。
func TestOverviewMissingAccountsZero(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM md_account_events WHERE channel IN ('account','earn_parking')"); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	o, err := svc.Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if o.NavTotal != 0 {
		t.Fatalf("缺户 NAV 应 0: %v", o.NavTotal)
	}
}
