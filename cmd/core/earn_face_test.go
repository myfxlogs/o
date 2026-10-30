package main

import (
	"context"
	"testing"

	"o/internal/gen/dashboardv1"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// FEAT-01 锚③：SetEarnAutoPark(false)→app_config 行变+ops_event 审计行+generation 涨；
// 再切回 true→回读一致；GetCockpit.earn_auto_park 与写态同径。
func TestSetEarnAutoPark(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ctx := context.Background()
	c := coreService{pool: pool}

	r1, err := c.SetEarnAutoPark(ctx, &dashboardv1.SetEarnAutoParkRequest{Enabled: false})
	if err != nil {
		t.Fatalf("set false: %v", err)
	}
	var val string
	if err := pool.QueryRow(ctx, `SELECT value::text FROM app_config WHERE key='earn.sweep_enabled'`).Scan(&val); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if val != "false" || !r1.Enabled && r1.Enabled {
		t.Fatalf("写态不回读：val=%s", val)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='earn_autopark_toggle'`).Scan(&n); err != nil || n < 1 {
		t.Fatalf("审计行缺：n=%d err=%v", n, err)
	}
	r2, err := c.SetEarnAutoPark(ctx, &dashboardv1.SetEarnAutoParkRequest{Enabled: true})
	if err != nil || !r2.Enabled || r2.Generation <= 0 {
		t.Fatalf("set true: r2=%+v err=%v", r2, err)
	}
}
