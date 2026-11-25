// balanceops_test.go —— O-35753 充提入账桩测：Balance deal→fin_cashflow
// 落行/重放幂等/非充提不收/负值=withdraw/人工先例冲突不双记/Credit 邻类
// 判词注记/水位窗边界。
package mt5

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func newBalanceCollector(t *testing.T) *Collector {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_cashflow WHERE source_table='mt5_deal'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE note LIKE '%mt5deal:%'`); err != nil {
		t.Fatal(err)
	}
	return &Collector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5", "write_err", "balance_ops_err")}
}

// testAcct 按户运行件桩——venue='exness' Exness 兼容锚（写 venue 键断言同旧径）。
func testAcct() *acctRun {
	return &acctRun{acct: VenueAcct{Venue: "exness", CredNS: "exness.mt5", Login: "3001157"}}
}

// balOrder 充提行桩——实证形（EXR 3001157 OrderHistory 实拍：OrderType_
// Balance+DealType_Balance 双标记、Profit=带符号金额、CloseTime+
// CloseTimestampUTC 毫秒双时戳）。
func balOrder(ticket int64, amt float64, close time.Time) *mt5pb.Order {
	return &mt5pb.Order{
		Ticket:            ticket,
		OrderType:         mt5pb.OrderType_OrderType_Balance,
		DealType:          mt5pb.DealType_DealType_Balance,
		Profit:            amt,
		CloseTime:         timestamppb.New(close),
		CloseTimestampUTC: close.UnixMilli(),
		Comment:           "D-ALLINT-USD-INT-test",
		State:             mt5pb.OrderState_OrderState_Filled,
	}
}

func cashflowCount(t *testing.T, c *Collector, ticket int64) int {
	t.Helper()
	var n int
	if err := c.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM fin_cashflow WHERE venue='exness' AND source_table='mt5_deal' AND source_ref=$1`,
		ticket).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestBalanceDealDeposit 正额 Balance deal→deposit 行（spec 验收①）。
func TestBalanceDealDeposit(t *testing.T) {
	c := newBalanceCollector(t)
	ctx := context.Background()
	close := time.Date(2026, 11, 23, 12, 36, 42, 999000000, time.UTC)
	o := balOrder(1313122770, 1.80, close)

	c.balanceDeal(ctx, testAcct(), o)
	if n := cashflowCount(t, c, 1313122770); n != 1 {
		t.Fatalf("deposit 落行数=%d want 1", n)
	}
	var kind, ccy, table string
	var amt float64
	var occurred time.Time
	if err := c.pool.QueryRow(ctx,
		`SELECT kind,ccy,amount,source_table,occurred_at FROM fin_cashflow
		 WHERE venue='exness' AND source_table='mt5_deal' AND source_ref=1313122770`).
		Scan(&kind, &ccy, &amt, &table, &occurred); err != nil {
		t.Fatal(err)
	}
	if kind != "deposit" || ccy != "USD" || table != "mt5_deal" || amt != 1.80 {
		t.Fatalf("行域错 kind=%s ccy=%s amt=%v table=%s", kind, ccy, amt, table)
	}
	if !occurred.Equal(close) {
		t.Fatalf("occurred_at=%v want deal 时刻 %v", occurred, close)
	}
}

// TestBalanceDealReplay 幂等——同 deal 重放不双记（spec 验收②）。
func TestBalanceDealReplay(t *testing.T) {
	c := newBalanceCollector(t)
	ctx := context.Background()
	o := balOrder(9000000001, 50.0, time.Now().UTC().Truncate(time.Second))
	c.balanceDeal(ctx, testAcct(), o)
	c.balanceDeal(ctx, testAcct(), o) // 重放
	c.balanceDeal(ctx, testAcct(), o) // 三刷
	if n := cashflowCount(t, c, 9000000001); n != 1 {
		t.Fatalf("重放后行数=%d want 1（幂等闸失效）", n)
	}
}

// TestBalanceDealWithdraw 负额→withdraw 负值存（OKX funding 同约定）。
func TestBalanceDealWithdraw(t *testing.T) {
	c := newBalanceCollector(t)
	ctx := context.Background()
	o := balOrder(9000000002, -3247.31, time.Now().UTC().Truncate(time.Second))
	c.balanceDeal(ctx, testAcct(), o)
	var kind string
	var amt float64
	if err := c.pool.QueryRow(ctx,
		`SELECT kind,amount FROM fin_cashflow WHERE source_table='mt5_deal' AND source_ref=9000000002`).
		Scan(&kind, &amt); err != nil {
		t.Fatal(err)
	}
	if kind != "withdraw" || amt != -3247.31 {
		t.Fatalf("kind=%s amt=%v want withdraw/-3247.31", kind, amt)
	}
}

// TestNonBalanceDealIgnored 非充提行（交易单）不入账（spec 验收③）。
func TestNonBalanceDealIgnored(t *testing.T) {
	c := newBalanceCollector(t)
	ctx := context.Background()
	trade := &mt5pb.Order{
		Ticket:    9000000003,
		OrderType: mt5pb.OrderType_OrderType_Buy,
		DealType:  mt5pb.DealType_DealType_DealSell,
		Profit:    -0.28,
		CloseTime: timestamppb.Now(),
	}
	c.balanceDeal(ctx, testAcct(), trade)
	if n := cashflowCount(t, c, 9000000003); n != 0 {
		t.Fatalf("交易行误入账 n=%d", n)
	}
}

// TestManualPrecedentConflict 人工先例冲突闸——owner_out#34239 实证形：
// deal -3247.31 vs 人工 -3247.41（差 $0.10）occurred_at Δ33h 窗内→跳过不双记。
func TestManualPrecedentConflict(t *testing.T) {
	c := newBalanceCollector(t)
	ctx := context.Background()
	dealTime := time.Date(2026, 11, 21, 5, 18, 19, 0, time.UTC)
	// 人工簿记先例行（owner_out 负值形——#34239/#1658 实证镜像）
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
		 VALUES('owner_out','exness','USD',-3247.41,99999,'ops_event',$1)`,
		dealTime.Add(33*time.Hour)); err != nil {
		t.Fatal(err)
	}
	o := balOrder(1312436725, -3247.31, dealTime)
	c.balanceDeal(ctx, testAcct(), o)
	if n := cashflowCount(t, c, 1312436725); n != 0 {
		t.Fatalf("人工先例冲突未拦 n=%d（双记风险）", n)
	}
	// 对照：无冲突的负额 deal 照常入账
	far := balOrder(9000000004, -100.0, time.Now().UTC().Truncate(time.Second))
	c.balanceDeal(ctx, testAcct(), far)
	if n := cashflowCount(t, c, 9000000004); n != 1 {
		t.Fatalf("无冲突 withdraw 未入账 n=%d", n)
	}
}

// TestCreditNoteNotBooked Credit 邻类→判词注记不入账不静默丢。
func TestCreditNoteNotBooked(t *testing.T) {
	c := newBalanceCollector(t)
	ctx := context.Background()
	credit := &mt5pb.Order{
		Ticket:    9000000005,
		OrderType: mt5pb.OrderType_OrderType_Credit,
		DealType:  mt5pb.DealType_DealType_Credit,
		Profit:    500.0,
		CloseTime: timestamppb.Now(),
	}
	c.balanceDeal(ctx, testAcct(), credit)
	c.balanceDeal(ctx, testAcct(), credit) // 重扫幂等——判词行只落一条
	if n := cashflowCount(t, c, 9000000005); n != 0 {
		t.Fatalf("Credit 邻类误入账 n=%d", n)
	}
	var notes int
	if err := c.pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE note LIKE '%mt5deal:9000000005%'`).Scan(&notes); err != nil {
		t.Fatal(err)
	}
	if notes != 1 {
		t.Fatalf("判词注记行数=%d want 1（幂等 marker 失效）", notes)
	}
}

// TestBalanceWindow 水位窗边界——首跑 7d 回看/增量 wm-2h 重叠。
func TestBalanceWindow(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	from, to := balanceWindow(time.Time{}, now)
	if !to.Equal(now) || now.Sub(from) != balanceBootLookback {
		t.Fatalf("首跑窗 from=%v to=%v", from, to)
	}
	wm := now.Add(-30 * time.Minute)
	from, to = balanceWindow(wm, now)
	if !from.Equal(wm.Add(-balanceWindowOverlap)) || !to.Equal(now) {
		t.Fatalf("增量窗 from=%v to=%v want wm-2h", from, to)
	}
}
