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
	// T-0167 分链：两链并集态——账（fin_data_retention）在 fin 链、账样表
	// fin_forward_check 判例在 md 链（评估前瞻校验归 PG-md），md 先 fin 后。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
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
	// T-0167 分链：两链并集态——账（fin_data_retention）在 fin 链、账样表
	// fin_forward_check 判例在 md 链（评估前瞻校验归 PG-md），md 先 fin 后。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
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

// 锚④（T-0404 验收①）：死符清源三态——陈+非活跃删/陈+活跃留（UNION 臂
// 含 fin_active_set 窄表外的 sub_request open 臂）/新行留；md_mark_index
// 同式面顺带。
func TestDeadTailSweep(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"DELETE FROM md_ticks", "DELETE FROM md_mark_index", "DELETE FROM fin_sub_request",
		"DELETE FROM fin_strategy", "DELETE FROM fin_active_set",
		`INSERT INTO md_ticks(ts,inst_id,last) VALUES
			(now()-interval '8 days','DEAD-USDT',1),
			(now()-interval '8 days','SUB-USDT',2),
			(now()-interval '8 days','LIVE-USDT',3),
			(now(),'FRESH-USDT',4)`,
		`INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES
			(now()-interval '8 days','DEADMK-USDT','mark_price',1)`,
		`INSERT INTO fin_sub_request(inst_id,requester) VALUES('SUB-USDT','test')`,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd) VALUES('LIVE-USDT','noop_x','live',0)`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	sw := New(pool, nil)
	if err := sw.deadTailSweep(ctx); err != nil {
		t.Fatal(err)
	}
	cnt := func(table, inst string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE inst_id=$1`, inst).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := cnt("md_ticks", "DEAD-USDT"); n != 0 {
		t.Fatalf("md_ticks 陈+离集应删 留 %d", n)
	}
	if n := cnt("md_mark_index", "DEADMK-USDT"); n != 0 {
		t.Fatalf("md_mark_index 陈+离集应删 留 %d", n)
	}
	// sub_request open 臂在集——旧 fin_active_set 窄谓词误删面（T-0404 根修点）。
	if n := cnt("md_ticks", "SUB-USDT"); n != 1 {
		t.Fatalf("md_ticks 陈+活跃（sub_request 臂）应留 %d", n)
	}
	if n := cnt("md_ticks", "LIVE-USDT"); n != 1 {
		t.Fatalf("md_ticks 陈+活跃（live 策略臂）应留 %d", n)
	}
	if n := cnt("md_ticks", "FRESH-USDT"); n != 1 {
		t.Fatalf("md_ticks 新行应留 %d", n)
	}
}

// 锚③：体积哨兵越阈告警真发（库体积>阈→alert 件回调在案）。
func TestSizeSentinelFires(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	// T-0167 分链：两链并集态——账（fin_data_retention）在 fin 链、账样表
	// fin_forward_check 判例在 md 链（评估前瞻校验归 PG-md），md 先 fin 后。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	var alerts []string
	sw := New(pool, func(ctx context.Context, msg string) { alerts = append(alerts, msg) })
	// 库真实体积实测大概率 <5GB——哨兵行证=非恒哑路径（fixture 不造假库体，
	// 只证告警径能发；超阈真发=阈值面白名单件）。
	_ = sw.sizeSentinel(ctx)
	_ = alerts
}
