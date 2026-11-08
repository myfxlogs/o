package holding

import (
	"context"
	"fmt"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestSyncAndStale(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
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

// T-0478 锚：缺席即删四断言——fetch 覆盖域存量行不在返回集→DELETE；
// 同拍返回行照 UPSERT；bag 显形行（kind='spot_dust' 且 venue_ref 空）/
// 手工行（source='owner_declared'）/structured_order 不误收；
// fetch 败拍全标 stale 不删（拉败非真消失）。
func TestSyncAbsentDelete(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 存量行五型：幻影镜像行×2（fetch 域内缺席）、bag 显形行同形
	// （bag.go 写径 venue_ref=''——kind 同 fetch 但对域外）、手工行
	// （覆盖对域内但 source≠api）、在拍返回行（UPSERT 锚）。
	seed := []struct{ kind, ccy, vref, source string }{
		{"spot_dust", "GONE", "asset", "api"},
		{"earn_flexible", "GHOST", "savings", "api"},
		{"spot_dust", "BAG", "", "api"},
		{"spot_dust", "MANUAL", "asset", "owner_declared"},
		{"spot_dust", "PENGU", "asset", "api"},
	}
	for _, r := range seed {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_holding(venue,kind,ccy,amount,venue_ref,source)
			VALUES('okx',$1,$2,1.0,$3,$4)`, r.kind, r.ccy, r.vref, r.source); err != nil {
			t.Fatal(err)
		}
	}
	ok := true
	s := NewSyncer(pool, func(ctx context.Context) ([]AssetRow, error) {
		if !ok {
			return nil, fmt.Errorf("venue down")
		}
		return []AssetRow{
			{Ccy: "PENGU", Amount: 17.9, Kind: "spot_dust", VenueRef: "asset"},
			{Ccy: "USDT", Amount: 99.0, Kind: "earn_flexible", VenueRef: "savings", Product: "savings"},
		}, nil
	})
	if err := s.sync(ctx); err != nil {
		t.Fatal(err)
	}
	// 断言①：覆盖域内缺席行已删（GONE spot_dust×asset / GHOST earn×savings）
	for _, ccy := range []string{"GONE", "GHOST"} {
		var n int
		if err := pool.QueryRow(ctx,
			"SELECT count(*) FROM fin_holding WHERE ccy=$1", ccy).Scan(&n); err != nil || n != 0 {
			t.Fatalf("缺席行 %s 未核销 n=%d err=%v", ccy, n, err)
		}
	}
	// 断言②：同拍返回行照 UPSERT（旧值 1.0→17.9，stale=false）
	var amt float64
	var stale bool
	if err := pool.QueryRow(ctx,
		"SELECT amount,stale FROM fin_holding WHERE ccy='PENGU' AND venue_ref='asset'").Scan(&amt, &stale); err != nil || amt != 17.9 || stale {
		t.Fatalf("UPSERT 行 amt=%v stale=%v err=%v", amt, stale, err)
	}
	// 断言③：bag 显形行/手工行/结构单不误收
	for _, ccy := range []string{"BAG", "MANUAL"} {
		var n int
		if err := pool.QueryRow(ctx,
			"SELECT count(*) FROM fin_holding WHERE ccy=$1", ccy).Scan(&n); err != nil || n != 1 {
			t.Fatalf("护线行 %s 误收 n=%d err=%v", ccy, n, err)
		}
	}
	var n int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_holding WHERE kind='structured_order' AND inherited_from='nox'").Scan(&n); err != nil || n == 0 {
		t.Fatalf("NOX 播种行误收 n=%d err=%v", n, err)
	}
	// 断言④：fetch 败拍——新铺缺席行标 stale 不删（拉败非真消失）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_holding(venue,kind,ccy,amount,venue_ref,source)
		VALUES('okx','earn_flexible','GHOST2',5.0,'savings','api')`); err != nil {
		t.Fatal(err)
	}
	ok = false
	if err := s.sync(ctx); err == nil {
		t.Fatal("采败应报")
	}
	if err := pool.QueryRow(ctx,
		"SELECT count(*),bool_and(stale) FROM fin_holding WHERE ccy='GHOST2'").Scan(&n, &stale); err != nil || n != 1 || !stale {
		t.Fatalf("败拍缺席行应存且 stale n=%d stale=%v err=%v", n, stale, err)
	}
}
