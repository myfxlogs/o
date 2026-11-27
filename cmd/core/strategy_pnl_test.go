// strategy_pnl_test.go —— O-41386 锚：ListStrategyPnl 策略盈亏榜——
// realized/fees/funding 三分列+net=三者和+positions 计数+net 降序+
// position_ref NULL 行不入围+days 窗外行截断。
package main

import (
	"context"
	"testing"

	"o/internal/gen/dashboardv1"
)

func stratID(t *testing.T, c coreService, ctx context.Context, posID int64) int64 {
	t.Helper()
	var sid int64
	if err := c.pool.QueryRow(ctx,
		`SELECT strategy_ref FROM fin_position WHERE id=$1`, posID).Scan(&sid); err != nil {
		t.Fatalf("strategy_ref: %v", err)
	}
	return sid
}

func pnlRowByStrat(t *testing.T, r *dashboardv1.ListStrategyPnlResponse, sid int64) *dashboardv1.StrategyPnlRow {
	t.Helper()
	for _, row := range r.Rows {
		if row.StrategyRef == sid {
			return row
		}
	}
	t.Fatalf("策略 %d 不在榜: %+v", sid, r.Rows)
	return nil
}

// TestListStrategyPnl O-41386 锚：两策略——A(trade_pnl+10/fee-1/funding+0.5
// →net=9.5)、B(trade_pnl-4/fee-2→net=-6)；榜序 net 降序 A 先；
// position_ref NULL spot 腿+40d 前旧行均不入 30d 窗/不入围；
// days=90 窗旧行入账对拍。
func TestListStrategyPnl(t *testing.T) {
	c := opsSetup(t)
	ctx := context.Background()
	fee := -0.01

	posA, _ := seedPos(t, c, ctx, "long_perp", "ETH-USDT-SWAP", "buy", 3000, 100, &fee)
	posB, _ := seedPos(t, c, ctx, "short_perp", "BTC-USDT-SWAP", "sell", 60000, 200, &fee)
	sidA := stratID(t, c, ctx, posA)
	sidB := stratID(t, c, ctx, posB)

	seedCash(t, c, ctx, posA, "trade_pnl", 10)
	seedCash(t, c, ctx, posA, "fee", -1)
	seedCash(t, c, ctx, posA, "funding_settle", 0.5)
	seedCash(t, c, ctx, posB, "trade_pnl", -4)
	seedCash(t, c, ctx, posB, "fee", -2)

	// 账外腿（position_ref NULL spot_buy）——视图已滤+inner join 双保险不入围。
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('spot_buy','okx','USDT',999,NULL,1,'x',now())`); err != nil {
		t.Fatal(err)
	}
	// 40 天前旧行——30d 窗（days<=0 缺省）截断，90d 窗入账。
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('trade_pnl','okx','USDT',777,$1,1,'fin_order_leg',now()-interval '40 days')`, posA); err != nil {
		t.Fatal(err)
	}

	r, err := c.ListStrategyPnl(ctx, &dashboardv1.ListStrategyPnlRequest{Days: 0})
	if err != nil {
		t.Fatalf("ListStrategyPnl: %v", err)
	}
	if len(r.Rows) != 2 {
		t.Fatalf("榜行数=%d want=2: %+v", len(r.Rows), r.Rows)
	}
	// net 降序：A(9.5) 先于 B(-6)。
	if r.Rows[0].StrategyRef != sidA || r.Rows[1].StrategyRef != sidB {
		t.Fatalf("榜序错: %+v", r.Rows)
	}
	a := pnlRowByStrat(t, r, sidA)
	near(t, "A.realized", a.RealizedUsd, 10, 1e-9)
	near(t, "A.fees", a.FeesUsd, -1, 1e-9)
	near(t, "A.funding", a.FundingUsd, 0.5, 1e-9)
	near(t, "A.net=三者和", a.NetUsd, 9.5, 1e-9)
	if a.Positions != 1 {
		t.Fatalf("A.positions=%d want=1", a.Positions)
	}
	if a.LastTs == nil {
		t.Fatal("A.last_ts 空——末笔账行时缺席")
	}
	b := pnlRowByStrat(t, r, sidB)
	near(t, "B.realized", b.RealizedUsd, -4, 1e-9)
	near(t, "B.fees", b.FeesUsd, -2, 1e-9)
	near(t, "B.funding", b.FundingUsd, 0, 1e-9)
	near(t, "B.net", b.NetUsd, -6, 1e-9)

	// 90d 窗：40d 前旧行入 A 账——realized 10+777=787。
	r90, err := c.ListStrategyPnl(ctx, &dashboardv1.ListStrategyPnlRequest{Days: 90})
	if err != nil {
		t.Fatalf("ListStrategyPnl 90d: %v", err)
	}
	a90 := pnlRowByStrat(t, r90, sidA)
	near(t, "A90.realized 含旧行", a90.RealizedUsd, 787, 1e-9)
	near(t, "A90.net", a90.NetUsd, 786.5, 1e-9)
}
