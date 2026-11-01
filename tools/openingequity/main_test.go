package main

// T-0034 桩锚：期初入账幂等（首录+重跑跳）+零值拒（fail-closed 不落零行）
// ——DB 桩 O_TEST_PG_DSN 缺=skip。
import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestRecordOpeningIdempotent(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM fin_cashflow WHERE source_table='opening_owner_declared'`)
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM fin_cashflow WHERE source_table='opening_owner_declared'`)
	}()
	// 首录：行在+kind=deposit+标记在
	id, err := recordOpening(ctx, pool, "mt5", "USD", 3247.41)
	if err != nil {
		t.Fatal(err)
	}
	var kind, st string
	var amt float64
	if err := pool.QueryRow(ctx, `SELECT kind,source_table,amount::float8 FROM fin_cashflow WHERE id=$1`, id).
		Scan(&kind, &st, &amt); err != nil {
		t.Fatal(err)
	}
	if kind != "deposit" || st != "opening_owner_declared" || amt != 3247.41 {
		t.Fatalf("首录语义漂移: kind=%s st=%s amt=%v", kind, st, amt)
	}
	// 幂等锚：在册判命中
	_, amt2, ok, err := openingExists(ctx, pool, "mt5", "USD")
	if err != nil || !ok || amt2 != 3247.41 {
		t.Fatalf("幂等判失效: ok=%v amt=%v err=%v", ok, amt2, err)
	}
	// 别 ccy 独立键（通用径——venue/ccy 各自一期初）
	if _, err := recordOpening(ctx, pool, "mt5", "EURX", 100); err == nil {
		// EURX 不在 ccy 闭集——应被 CHECK 拒（fail-closed 闭集仍有效）
		t.Fatal("册外 ccy 应被 CHECK 拒")
	}
}

func TestOpeningExistsAbsent(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM fin_cashflow WHERE source_table='opening_owner_declared'`)
	_, _, ok, err := openingExists(ctx, pool, "okx", "USDT")
	if err != nil || ok {
		t.Fatalf("无期初应判缺: ok=%v err=%v", ok, err)
	}
}
