// tradebook_test.go —— O-33207 手动仓/交易盈亏簿记桩测：交易行幂等
// 重放/O 单排除双簿防线（comment=clOrdID→fin_order_leg 命中不入
// fin_venue_position 但 trade_pnl 守恒仍入）/Charge 邻类 unmapped_bill
// 守恒兜底/逐户 account_id 隔离。
package mt5

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	mt5pb "o/internal/gen/mtapi/mt5"
)

// tradeOrder 交易行桩——posTicket 域平仓仓事实（EXR 实证形：
// OrderType_Buy+DealType_DealSell、Profit 行级聚合额、毫秒时戳）。
func tradeOrder(ticket int64, comment string, profit float64, close time.Time) *mt5pb.Order {
	return &mt5pb.Order{
		Ticket:            ticket,
		OrderType:         mt5pb.OrderType_OrderType_Buy,
		DealType:          mt5pb.DealType_DealType_DealSell,
		Symbol:            "XAUUSDm",
		Profit:            profit,
		Lots:              0.10,
		OpenPrice:         2650.0,
		ClosePrice:        2651.0,
		CloseTime:         timestamppb.New(close),
		CloseTimestampUTC: close.UnixMilli(),
		OpenTimestampUTC:  close.Add(-time.Hour).UnixMilli(),
		Comment:           comment,
		State:             mt5pb.OrderState_OrderState_Filled,
	}
}

func vpCount(t *testing.T, c *Collector, acct, ticket string) int {
	t.Helper()
	var n int
	if err := c.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM fin_venue_position WHERE account_id=$1 AND ticket=$2`,
		acct, ticket).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestTradePnlReplay 交易盈亏幂等——同行三刷 mt5_deal_uk 兜底零双簿。
func TestTradePnlReplay(t *testing.T) {
	c := newBalanceCollector(t)
	ctx := context.Background()
	o := tradeOrder(9100000001, "", 12.34, time.Now().UTC().Truncate(time.Second))
	c.balanceDeal(ctx, testAcct(), o)
	c.balanceDeal(ctx, testAcct(), o)
	c.balanceDeal(ctx, testAcct(), o)
	if n := cashflowCount(t, c, 9100000001); n != 1 {
		t.Fatalf("重放后 trade_pnl 行数=%d want 1", n)
	}
}

// TestOwnOrderExclusion 双簿防线——comment=clOrdID 命中 fin_order_leg
// =O 自执行单：trade_pnl 守恒仍入（venue balance 不分彼此），但
// fin_venue_position 禁入册（O 仓走 fin_position+mtm 既有径）。
func TestOwnOrderExclusion(t *testing.T) {
	c := newBalanceCollector(t)
	ctx := context.Background()
	// O 单夹具：execution+leg（cl_ord_id=comment 域）
	var eid int64
	if err := c.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(inst_id,notional_usd,state) VALUES('XAUUSDm',100,'open') RETURNING id`).
		Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state,ord_type)
		 VALUES($1,1,'ph-exn-x1-abc','exness','buy','XAUUSDm',0.1,'filled','market')`, eid); err != nil {
		t.Fatal(err)
	}
	var pid int64
	if err := c.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,inst_id,notional_usd,state) VALUES($1,'XAUUSDm',100,'open') RETURNING id`,
		eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	o := tradeOrder(9100000002, "ph-exn-x1-abc", -5.5, time.Now().UTC().Truncate(time.Second))
	c.balanceDeal(ctx, testAcct(), o)
	if n := cashflowCount(t, c, 9100000002); n != 1 {
		t.Fatalf("O 单 trade_pnl 未入账 n=%d（守恒缺口）", n)
	}
	if n := vpCount(t, c, "exness:3001157", "9100000002"); n != 0 {
		t.Fatalf("O 单误入手动仓册 n=%d（双簿风险）", n)
	}
	// position_ref 归仓——comment→leg→position 关联
	var pref *int64
	if err := c.pool.QueryRow(ctx,
		`SELECT position_ref FROM fin_cashflow WHERE source_table='mt5_deal' AND source_ref=9100000002`).
		Scan(&pref); err != nil {
		t.Fatal(err)
	}
	if pref == nil {
		t.Fatal("O 单 position_ref=NULL（归仓断链）")
	}
}

// TestChargeUnmapped 邻类守恒兜底——Charge 金额带号入 unmapped_bill
// 不静默吞差，判词注记并存。
func TestChargeUnmapped(t *testing.T) {
	c := newBalanceCollector(t)
	ctx := context.Background()
	ch := &mt5pb.Order{
		Ticket:    9100000003,
		OrderType: mt5pb.OrderType_OrderType_Balance,
		DealType:  mt5pb.DealType_DealType_Charge,
		Profit:    -12.5,
		CloseTime: timestamppb.Now(),
	}
	c.balanceDeal(ctx, testAcct(), ch)
	var kind string
	var amt float64
	if err := c.pool.QueryRow(ctx,
		`SELECT kind,amount FROM fin_cashflow WHERE source_table='mt5_deal' AND source_ref=9100000003`).
		Scan(&kind, &amt); err != nil {
		t.Fatalf("Charge 未守恒入账: %v", err)
	}
	if kind != "unmapped_bill" || amt != -12.5 {
		t.Fatalf("kind=%s amt=%v want unmapped_bill/-12.5", kind, amt)
	}
	var notes int
	if err := c.pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE note LIKE '%exnessdeal:9100000003%'`).Scan(&notes); err != nil || notes != 1 {
		t.Fatalf("判词注记行数=%d err=%v want 1", notes, err)
	}
}

// TestVenueAcctIsolation 逐户账户域隔离——XM 户同票号不与 Exness
// 互踩（account_id 粒度天然隔离，禁 venue 共键）。
func TestVenueAcctIsolation(t *testing.T) {
	c := newBalanceCollector(t)
	ctx := context.Background()
	xm := &acctRun{acct: VenueAcct{Venue: "xmglobal", CredNS: "xmglobal.mt5", Login: "110660304"}}
	o := tradeOrder(9100000004, "", 7.7, time.Now().UTC().Truncate(time.Second))
	c.balanceDeal(ctx, testAcct(), o) // exness 户
	c.balanceDeal(ctx, xm, o)         // xm 户同票号
	var n int
	if err := c.pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_cashflow WHERE source_table='mt5_deal' AND source_ref=9100000004`).
		Scan(&n); err != nil || n != 2 {
		t.Fatalf("逐户隔离失效 n=%d err=%v want 2", n, err)
	}
	if n := vpCount(t, c, "xmglobal:110660304", "9100000004"); n != 1 {
		t.Fatalf("xm 户手动仓行数=%d want 1", n)
	}
}
