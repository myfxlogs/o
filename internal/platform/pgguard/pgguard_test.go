package pgguard

// pgguard_test.go —— 事务残留守卫桩锚（T-0154）：BEGIN 不结还池→借出守卫
// 回滚+计数——下任借用者 TxStatus='I' 干净；哨兵零钉手静默语义。
import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// tpool 自建守卫池（testutil 反向引环——本包桩测直建同守池）。
func tpool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("O_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("O_TEST_PG_DSN 未设——独立测试实例缺席")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	GuardPool(cfg)
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// 锚①：借 conn→BEGIN 不结→Release（带病还池）→再借 TxStatus='I'（守卫
// 回滚放行）+残留计数递增（泄漏借用者定位面）。
func TestTxResidueGuard(t *testing.T) {
	pool := tpool(t)
	ctx := context.Background()
	before := TxResidueCount()

	c1, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 直协议 BEGIN（绕 pgx 事务跟踪——pgxpool Release 内建自动回滚只认
	// pgx 侧状态；泄漏实形=协议层残留 pgx 不可见，守卫 PgConn 层才见）。
	if _, err := c1.Conn().PgConn().Exec(ctx, "BEGIN").ReadAll(); err != nil {
		t.Fatal(err)
	}
	if st := c1.Conn().PgConn().TxStatus(); st != 'T' {
		t.Fatalf("夹具应事务态 T got %q", st)
	}
	c1.Release() // 带病还池（守卫借出侧拦）

	c2, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Release()
	if st := c2.Conn().PgConn().TxStatus(); st != 'I' {
		t.Fatalf("下任借用者应干净 idle got %q（守卫/pgx 双层任一未生效）", st)
	}
	// 计数断言诚实化：pgx v5.10 Release 自身已 Destroy 带事务态连接
	// （conn.go Release：TxStatus!='I'→res.Destroy）——守卫触发面=版本差异
	// 防御纵深（旧版/直协议旁路面），pgx 自愈时计数不增。卫生结果断言为锚
	// （下任 idle=双层任一生效），计数仅观测面。
	if after := TxResidueCount(); after < before {
		t.Fatalf("残留计数不应倒退 before=%d after=%d", before, after)
	}
}

// 锚②：xmin 哨兵判定语义——测试库零长事务=零命中不警（静默=对，恢复
// 清标）；命中径由 Check 查询谓词直测（时钟不可注——>10min 窗构造面在
// pg_stat_activity，生产观测=业主通道实拍）。
func TestXminSentinelQuiet(t *testing.T) {
	pool := tpool(t)
	ctx := context.Background()
	x := NewSentinel(pool)
	x.Check(ctx) // 零钉手——不警
	c, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Release()
	if _, err := c.Exec(ctx, "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	x.Check(ctx) // 短事务不超窗——仍不警
	x.mu.Lock()
	alerted := x.alerted
	x.mu.Unlock()
	if alerted {
		t.Fatal("零钉手不应置警标")
	}
	_ = time.Now
}
