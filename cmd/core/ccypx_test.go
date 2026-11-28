package main

// ccypx_test.go —— O-41395 件①对抗锚测：鲜度优先判序实证
// （陈 tick 被鲜 index 压过=spec 钦定验收式；REST 败回退 DB 现语义保；
// 全无=0 不虚计）。ccyIdxPx 全局注入件——测后复位恒败桩防跨测污染。

import (
	"context"
	"errors"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// idxStub 价件桩——at 直喂鲜度戳（缓存语义在 indexPxFor 闭包内，
// ccyPxUSD 只消费 (px,at,err) 三元组）。
func idxStub(px float64, at time.Time, err error) func(context.Context, string) (float64, time.Time, error) {
	return func(context.Context, string) (float64, time.Time, error) { return px, at, err }
}

func idxAbsent(context.Context, string) (float64, time.Time, error) {
	return 0, time.Time{}, errors.New("index-tickers 断供桩")
}

func resetIdxPx(t *testing.T) {
	t.Helper()
	ccyIdxPx.Store(idxAbsent)
	t.Cleanup(func() { ccyIdxPx.Store(idxAbsent) })
}

func TestCcyPxUSDFreshnessPriority(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	const ccy = "TSTCCY"
	const inst = "TSTCCY-USDT"
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM md_ticks WHERE inst_id=$1`, inst)
		_, _ = pool.Exec(ctx, `DELETE FROM md_mark_index WHERE inst_id=$1`, inst)
	}()
	resetIdxPx(t)
	now := time.Now()

	// 案①钦定对抗：陈 tick(25h 前) vs 鲜 index(现拉)——index 压过。
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,inst_id,last) VALUES($1,$2,$3)
		 ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`,
		now.Add(-25*time.Hour), inst, 60000); err != nil {
		t.Fatal(err)
	}
	ccyIdxPx.Store(idxStub(65000, now, nil))
	px, ok := ccyPxUSD(ctx, pool, ccy)
	if !ok || px != 65000 {
		t.Fatalf("陈tick+鲜index 应取 index 65000，得 %v ok=%v", px, ok)
	}

	// 案②鲜度真比较：鲜 tick(1min) vs 陈 index 缓存(2min)——tick 胜。
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,inst_id,last) VALUES($1,$2,$3)
		 ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`,
		now.Add(-time.Minute), inst, 61000); err != nil {
		t.Fatal(err)
	}
	ccyIdxPx.Store(idxStub(65000, now.Add(-2*time.Minute), nil))
	px, ok = ccyPxUSD(ctx, pool, ccy)
	if !ok || px != 61000 {
		t.Fatalf("鲜tick应压陈index缓存，得 %v ok=%v", px, ok)
	}

	// 案③REST 断供：回退 DB 最鲜行（61000 tick）——现语义保。
	ccyIdxPx.Store(idxAbsent)
	px, ok = ccyPxUSD(ctx, pool, ccy)
	if !ok || px != 61000 {
		t.Fatalf("REST败应回退DB最鲜行61000，得 %v ok=%v", px, ok)
	}

	// 案④mark_index 行入 UNION：清 tick 后 index-tickers channel 行独存
	// 应取 mark px（现 UNION 语义保——index-tickers 行本在 UNION 内）。
	if _, err := pool.Exec(ctx, `DELETE FROM md_ticks WHERE inst_id=$1`, inst); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES($1,$2,'index-tickers',$3)`,
		now.Add(-59*time.Minute), inst, 62000); err != nil {
		t.Fatal(err)
	}
	ccyIdxPx.Store(idxAbsent)
	px, ok = ccyPxUSD(ctx, pool, ccy)
	if !ok || px != 62000 {
		t.Fatalf("mark_index index-tickers 行应入 UNION 取 62000，得 %v ok=%v", px, ok)
	}

	// 案⑤全无+REST断=不虚计；USDT/USD 直入。
	if _, err := pool.Exec(ctx, `DELETE FROM md_mark_index WHERE inst_id=$1`, inst); err != nil {
		t.Fatal(err)
	}
	if _, ok := ccyPxUSD(ctx, pool, ccy); ok {
		t.Fatal("无 tick 无 index 应 ok=false 不虚计")
	}
	if v := valCcy(ctx, pool, ccy, 2.5); v != 0 {
		t.Fatalf("无价 valCcy 应=0，得 %v", v)
	}
	if px, ok := ccyPxUSD(ctx, pool, "USDT"); !ok || px != 1 {
		t.Fatalf("USDT 应直返(1,true)，得 %v ok=%v", px, ok)
	}
	if v := valCcy(ctx, pool, "USD", 7); v != 7 {
		t.Fatalf("USD valCcy 应直入 7，得 %v", v)
	}
	// 案⑥鲜 index 压 mark_index 陈行（REST at=now > 落库 ts）。
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES($1,$2,'mark-price',$3)`,
		now.Add(-time.Hour), inst, 59000); err != nil {
		t.Fatal(err)
	}
	ccyIdxPx.Store(idxStub(66000, now, nil))
	px, ok = ccyPxUSD(ctx, pool, ccy)
	if !ok || px != 66000 {
		t.Fatalf("鲜index应压落库mark行，得 %v ok=%v", px, ok)
	}
}
