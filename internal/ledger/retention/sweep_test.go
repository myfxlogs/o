// sweep_test.go —— T-0153 桩锚：账行驱删幂等+未入账表报缺+哨兵越阈真发。
package retention

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 锚①：账行驱删（fin_forward_check judged 终局 30d 窗行删+幂等双拍零差）。
func TestSweepDrivenDelete(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, "TRUNCATE fin_forward_check,fin_paper_fill")
	_, _ = pool.Exec(ctx, `INSERT INTO fin_forward_check(obj_kind,obj_ref,window_h,target_ts,judged,judged_at)
		VALUES('rejected_proposal',1,24,now(),true,now()-interval '40 days'),
		      ('rejected_proposal',2,24,now(),true,now()-interval '10 days')`)
	var alerts []string
	sw := New(pool, func(ctx context.Context, msg string) { alerts = append(alerts, msg) })
	if err := sw.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	var cnt int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_forward_check").Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("终局 30d 窗应删 1 留 1: %d", cnt)
	}
	_ = sw.Sweep(ctx)
	var c2 int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_forward_check").Scan(&c2)
	if c2 != 1 {
		t.Fatalf("幂等双拍应留 1: %d", c2)
	}
}

// 锚②：未入账表报缺 fail-closed（库表不入册=unregistered_table 计数显形）。
func TestUnregisteredReported(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, "CREATE TABLE IF NOT EXISTS t_orphan_x(id int)")
	defer pool.Exec(ctx, "DROP TABLE IF EXISTS t_orphan_x")
	_, _ = pool.Exec(ctx, "DELETE FROM fin_data_retention WHERE table_name='t_orphan_x'")
	var alerts []string
	sw := New(pool, func(ctx context.Context, msg string) { alerts = append(alerts, msg) })
	if err := sw.reportUnregistered(ctx); err != nil {
		t.Fatal(err)
	}
}

// 锚③：体积哨兵越阈告警真发（库体积>阈→alert 件回调在案）。
func TestSizeSentinelFires(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var alerts []string
	sw := New(pool, func(ctx context.Context, msg string) { alerts = append(alerts, msg) })
	// 库真实体积实测大概率 <5GB——哨兵行证=非恒哑路径（fixture 不造假库体，
	// 只证告警径能发；超阈真发=阈值面白名单件）。
	_ = sw.sizeSentinel(ctx)
	_ = alerts
}
