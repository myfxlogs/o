// condreq_test.go —— T-0177 桩锚：Request upsert 语义（pending 撞=刷新非重行）
// +Resolve 结案径。
package condreq

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestRequestUpsertResolve(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, "DELETE FROM fin_condition_gap WHERE venue='cmc'")
	if err := Request(ctx, pool, "credential", "cmc", "cmc.email", `{"a":1}`); err != nil {
		t.Fatal(err)
	}
	if err := Request(ctx, pool, "credential", "cmc", "cmc.email", `{"a":2}`); err != nil {
		t.Fatal(err)
	}
	var cnt int
	var id int64
	_ = pool.QueryRow(ctx, `SELECT count(*),max(id) FROM fin_condition_gap
		WHERE venue='cmc' AND kind='credential' AND state='pending'`).Scan(&cnt, &id)
	if cnt != 1 {
		t.Fatalf("pending 同键应单行: %d", cnt)
	}
	if err := Resolve(ctx, pool, id, "fulfilled", "凭据已供"); err != nil {
		t.Fatal(err)
	}
	var st string
	_ = pool.QueryRow(ctx, "SELECT state FROM fin_condition_gap WHERE id=$1", id).Scan(&st)
	if st != "fulfilled" {
		t.Fatalf("resolve 应 fulfilled: %s", st)
	}
}
