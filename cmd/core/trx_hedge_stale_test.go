package main

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// trx_hedge_stale_test.go —— CMC 空仓不落痕票②：DB 回退鲜度闸——
// fin_cmc_positions/summary 行龄>15min→域不填（陈行不显活仓）；鲜行照填。

func TestFreshLiveDBStaleGate(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	st := pgstore.NewStore(pool)
	w := &trxWiring{st: st} // cmcLive=nil——纯 DB 兜底径

	// 陈行（1h 前 qty=89300——生产实案形）：闸应拒填，ls.Qty 恒 0。
	if err := st.InsertCMCPositions(ctx, 1, []pgstore.CMCPosition{{
		InstrumentID: "X-AFQUL", Direction: "LONG", Quantity: 89300, EntryPrice: 100,
		EvaluationBidPrice: 99, EvaluationAskPrice: 101, UnrealisedPnL: 5,
		FetchedAt: time.Now().Add(-time.Hour),
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertCMCAccountSummary(ctx, pgstore.CMCAccountSummary{
		AccountID: 1, TotalAccountValue: 1000, TotalCashBalance: 900,
		FetchedAt: time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	ls := w.freshLive(ctx)
	if ls.Qty != 0 {
		t.Fatalf("陈持仓行（1h）应闸不填——qty 应 0 非 89300: %v", ls.Qty)
	}
	if ls.Equity != 0 || ls.Cash != 0 {
		t.Fatalf("陈摘要行应闸不填: eq=%v cash=%v", ls.Equity, ls.Cash)
	}

	// 鲜行（墓碑 qty=0 新鲜 fetched_at——天然过闸）：照填。
	if err := st.InsertCMCPositions(ctx, 1, []pgstore.CMCPosition{{
		InstrumentID: "X-AFQUL", Direction: "LONG", Quantity: 0, EntryPrice: 100,
		EvaluationBidPrice: 99, EvaluationAskPrice: 101, FetchedAt: time.Now(),
	}}); err != nil {
		t.Fatal(err)
	}
	ls = w.freshLive(ctx)
	if ls.Ask != 101 {
		t.Fatalf("鲜墓碑行评估价应填（qty=0 语义通读面）: ask=%v", ls.Ask)
	}
}
