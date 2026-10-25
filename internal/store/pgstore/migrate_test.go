package pgstore

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
)

func TestMigrateIdempotent(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate 二次幂等: %v", err)
	}
	for _, tbl := range []string{"app_config", "app_secrets", "fin_employee_registry", "schema_migrations"} {
		var exists bool
		if err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name=$1)", tbl,
		).Scan(&exists); err != nil || !exists {
			t.Fatalf("表 %s 缺（exists=%v err=%v）", tbl, exists, err)
		}
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != len(migrationTable) {
		t.Fatalf("schema_migrations=%d 登记表=%d 不齐", n, len(migrationTable))
	}
}
