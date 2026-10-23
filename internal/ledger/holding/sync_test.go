package holding

import (
	"context"
	"fmt"
	"testing"

	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

func TestSyncAndStale(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	ok := true
	s := NewSyncer(pool, func(ctx context.Context) ([]AssetRow, error) {
		if !ok {
			return nil, fmt.Errorf("venue down")
		}
		return []AssetRow{{Ccy: "PENGU", Amount: 17.9, Kind: "spot_dust", VenueRef: "asset"}}, nil
	})
	if err := s.sync(ctx); err != nil {
		t.Fatal(err)
	}
	var amt float64
	var stale bool
	if err := pool.QueryRow(ctx,
		"SELECT amount,stale FROM fin_holding WHERE ccy='PENGU' AND venue_ref='asset'").Scan(&amt, &stale); err != nil || amt != 17.9 || stale {
		t.Fatalf("台账行 amt=%v stale=%v err=%v", amt, stale, err)
	}
	// 采败→陈旧标记不抹值锚
	ok = false
	if err := s.sync(ctx); err == nil {
		t.Fatal("采败应报")
	}
	if err := pool.QueryRow(ctx,
		"SELECT amount,stale FROM fin_holding WHERE ccy='PENGU'").Scan(&amt, &stale); err != nil || amt != 17.9 || !stale {
		t.Fatalf("陈旧标语义 amt=%v stale=%v err=%v", amt, stale, err)
	}
	// 隔离锚：台账行零入 cashflow
	var n int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_cashflow WHERE ccy='PENGU'").Scan(&n)
	if n != 0 {
		t.Fatal("holding 行泄入 cashflow——运营账污染")
	}
	// NOX 结构单播种锚
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_holding WHERE kind='structured_order' AND inherited_from='nox'").Scan(&n); err != nil || n == 0 {
		t.Fatalf("NOX 播种行=%d err=%v", n, err)
	}
}
