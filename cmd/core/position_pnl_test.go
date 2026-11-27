// position_pnl_test.go —— O-41385 锚：在持仓盈亏卡字段——
// upl 双向符号（多/空）+mark_index 兜底+陈价 mark_px=0 如实空+
// fees/funding 直加+total 合计含 trade_pnl 不含进 net_pnl_usd。
package main

import (
	"context"
	"math"
	"testing"

	"o/internal/gen/dashboardv1"
)

// seedPos 种 open 仓+execution+单腿 filled（buy=多/sell=空）。
func seedPos(t *testing.T, c coreService, ctx context.Context,
	dir, inst, side string, entryPx, notional float64, fee *float64) (posID, execID int64) {
	t.Helper()
	var sid int64
	if err := c.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction)
		VALUES($1,'funding-hedge','live',$2,$3) RETURNING id`, inst, notional, dir).Scan(&sid); err != nil {
		t.Fatalf("strategy seed: %v", err)
	}
	if err := c.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES($1,$2,'open') RETURNING id`, inst, notional).Scan(&execID); err != nil {
		t.Fatalf("execution seed: %v", err)
	}
	if err := c.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES($1,$2,$3,$4) RETURNING id`, execID, sid, inst, notional).Scan(&posID); err != nil {
		t.Fatalf("position seed: %v", err)
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,px,state,fee)
		VALUES($1,1,$2,'okx',$3,$4,1,$5,'filled',$6)`,
		execID, "cl-"+inst, side, inst, entryPx, fee); err != nil {
		t.Fatalf("leg seed: %v", err)
	}
	return posID, execID
}

func seedCash(t *testing.T, c coreService, ctx context.Context,
	posID int64, kind string, amt float64) {
	t.Helper()
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES($1,'okx','USDT',$2,$3,1,'fin_order_leg',now())`, kind, amt, posID); err != nil {
		t.Fatalf("cashflow seed %s: %v", kind, err)
	}
}

func posByInst(t *testing.T, r *dashboardv1.ListOpenPositionsResponse, inst string) *dashboardv1.OpenPosition {
	t.Helper()
	for _, p := range r.Positions {
		if p.InstId == inst {
			return p
		}
	}
	t.Fatalf("仓 %s 不在册: %+v", inst, r.Positions)
	return nil
}

func near(t *testing.T, name string, got, want float64, eps float64) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Fatalf("%s: got=%.6f want=%.6f", name, got, want)
	}
}

// TestListOpenPositionsPnl O-41385 锚：三仓——多腿 fresh ticks 中价/
// 空腿 mark_index 兜底/陈价腿 mark_px=0；fees/funding/net 逐仓对拍+
// total_upl/total_net 合计（total_net 含 trade_pnl 实现、net_pnl 不含）。
func TestListOpenPositionsPnl(t *testing.T) {
	c := opsSetup(t)
	ctx := context.Background()
	feeA, feeC := -0.02, -0.03

	// 仓A：多 ETH entry=3100，fresh ticks last=3200（last>0 优先径）→
	// upl_frac=+(3200/3100-1)≈+3.2258%；fee=-0.02 funding=+0.5。
	posA, _ := seedPos(t, c, ctx, "long_perp", "ETH-USDT-SWAP", "buy", 3100, 50, &feeA)
	if _, err := c.pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask)
		VALUES(now(),'okx','ETH-USDT-SWAP',3200,3199,3201)`); err != nil {
		t.Fatal(err)
	}
	seedCash(t, c, ctx, posA, "funding_settle", 0.5)
	seedCash(t, c, ctx, posA, "trade_pnl", 7) // 实现盈亏——只进 total_net 不进 net_pnl_usd

	// 仓B：空 BTC entry=80000，无 ticks 行→md_mark_index mark-price 兜底
	// px=70000 → upl_frac=-1×(70000/80000-1)=+12.5%（空赚）。
	posB, _ := seedPos(t, c, ctx, "short_perp", "BTC-USDT-SWAP", "sell", 80000, 100, nil)
	if _, err := c.pool.Exec(ctx, `INSERT INTO md_mark_index(ts,venue,inst_id,channel,px)
		VALUES(now(),'okx','BTC-USDT-SWAP','mark-price',70000)`); err != nil {
		t.Fatal(err)
	}
	seedCash(t, c, ctx, posB, "funding_settle", -0.25)

	// 仓C：多 SOL entry=100，ticks 陈 2h（>exec.tick_max_age_s 缺省 120s）
	// → mark_px=0/upl 全 0 如实空；fee=-0.03 仍显（费是事实非 mark 件）。
	_, _ = seedPos(t, c, ctx, "long_perp", "SOL-USDT-SWAP", "buy", 100, 20, &feeC)
	if _, err := c.pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now()-interval '2 hours','okx','SOL-USDT-SWAP',90)`); err != nil {
		t.Fatal(err)
	}

	r, err := c.ListOpenPositions(ctx, &dashboardv1.ListOpenPositionsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Positions) != 3 {
		t.Fatalf("应 3 仓: %+v", r.Positions)
	}

	// 仓A：多向 mark=3200 frac=+3.2258% upl=50×frac≈+1.6129；
	// net=upl+fee+funding=1.6129-0.02+0.5≈+2.0929（trade_pnl=7 不入）。
	a := posByInst(t, r, "ETH-USDT-SWAP")
	near(t, "A.markPx", a.MarkPx, 3200, 1e-6)
	wantFracA := 3200.0/3100.0 - 1
	near(t, "A.uplFrac", a.UplFrac, wantFracA, 1e-9)
	near(t, "A.uplUsd", a.UplUsd, 50*wantFracA, 1e-6)
	near(t, "A.feesUsd", a.FeesUsd, -0.02, 1e-9)
	near(t, "A.fundingUsd", a.FundingUsd, 0.5, 1e-9)
	near(t, "A.netPnlUsd", a.NetPnlUsd, 50*wantFracA-0.02+0.5, 1e-6)

	// 仓B：空向 mark_index 兜底 mark=70000 frac=+12.5%（空赚正号）；
	// funding=-0.25 fee=0 → net=12.5-0.25=12.25。
	b := posByInst(t, r, "BTC-USDT-SWAP")
	near(t, "B.markPx", b.MarkPx, 70000, 1e-6)
	near(t, "B.uplFrac", b.UplFrac, 0.125, 1e-9)
	near(t, "B.uplUsd", b.UplUsd, 12.5, 1e-6)
	near(t, "B.fundingUsd", b.FundingUsd, -0.25, 1e-9)
	near(t, "B.netPnlUsd", b.NetPnlUsd, 12.25, 1e-6)

	// 仓C：陈价 → mark/upl 全 0；fees=-0.03 funding=0 net=-0.03。
	cc := posByInst(t, r, "SOL-USDT-SWAP")
	if cc.MarkPx != 0 || cc.UplFrac != 0 || cc.UplUsd != 0 {
		t.Fatalf("陈价仓 mark/upl 应全 0: %+v", cc)
	}
	near(t, "C.feesUsd", cc.FeesUsd, -0.03, 1e-9)
	near(t, "C.netPnlUsd", cc.NetPnlUsd, -0.03, 1e-9)

	// 合计：total_upl=Σupl；total_net=Σnet+Σtrade_pnl（A 的 +7 只进此）。
	wantUpl := 50*wantFracA + 12.5
	wantNet := (50*wantFracA - 0.02 + 0.5) + 12.25 + (-0.03) + 7
	near(t, "total_upl", r.TotalUplUsd, wantUpl, 1e-6)
	near(t, "total_net", r.TotalNetUsd, wantNet, 1e-6)
}
