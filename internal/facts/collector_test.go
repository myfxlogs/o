package facts

import (
	"context"
	"testing"

	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

// fakeSrc 假源（键覆盖锚面）。
type fakeSrc struct {
	venue string
	fs    []Fact
}

func (f *fakeSrc) Venue() string                               { return f.venue }
func (f *fakeSrc) Collect(ctx context.Context) ([]Fact, error) { return f.fs, nil }

// TestUpsertIdempotent UPSERT 幂等锚：重采同行不叠、值刷新、ts 刷新；kind 闭集
// 两层共存（account/instrument 同 venue 并行不串）。
func TestUpsertIdempotent(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	fs := []Fact{
		{Kind: "account", Key: "posMode", Value: "long_short_mode"},
		{Kind: "instrument", Key: "inst:BTC-USDT-SWAP", Value: map[string]any{"ctVal": 0.01, "ctValCcy": "BTC"}},
	}
	c := NewCollector(pool, &fakeSrc{venue: "tstfact", fs: fs})
	c.collectAll(ctx)
	c.collectAll(ctx) // 重采——幂等
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM md_venue_fact WHERE venue='tstfact'").Scan(&n); err != nil || n != 2 {
		t.Fatalf("行数=%d（应 2——重采不叠）", n)
	}
	var posMode string
	if err := pool.QueryRow(ctx,
		"SELECT value#>>'{}' FROM md_venue_fact WHERE venue='tstfact' AND kind='account' AND key='posMode'").Scan(&posMode); err != nil || posMode != "long_short_mode" {
		t.Fatalf("posMode=%s err=%v", posMode, err)
	}
	// 值刷新锚
	if err := Upsert(ctx, pool, "tstfact", Fact{Kind: "account", Key: "posMode", Value: "net_mode"}); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		"SELECT value#>>'{}' FROM md_venue_fact WHERE venue='tstfact' AND kind='account' AND key='posMode'").Scan(&posMode); err != nil || posMode != "net_mode" {
		t.Fatalf("刷新败=%s", posMode)
	}
}

// TestCollectAllSourceIsolation 单源败不连坐：坏源计数、好照落。
func TestCollectAllSourceIsolation(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	bad := &errSrc{}
	good := &fakeSrc{venue: "tstgood", fs: []Fact{{Kind: "account", Key: "equity", Value: 1.5}}}
	c := NewCollector(pool, bad, good)
	c.collectAll(ctx)
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM md_venue_fact WHERE venue='tstgood'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("好源被连坐: n=%d", n)
	}
	if c.rej.Snapshot()["facts/collect_failed"] == 0 {
		t.Fatal("坏源计数未计")
	}
}

type errSrc struct{}

func (e *errSrc) Venue() string { return "tstbad" }
func (e *errSrc) Collect(ctx context.Context) ([]Fact, error) {
	return nil, context.DeadlineExceeded
}
