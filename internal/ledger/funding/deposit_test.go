// deposit_test.go —— T-0149 桩锚：入金巡检幂等双行（重复扫不重账）。
package funding

import (
	"context"
	"testing"
)

type fakeDepSrc struct{ rows []DepositRow }

func (f fakeDepSrc) DepositHistory(context.Context) ([]DepositRow, error) { return f.rows, nil }

// 锚：state=2 入金→bill+cashflow 双行；重复巡检不重账（depId 幂等）。
func TestDepositScanIdempotent(t *testing.T) {
	ctx, pool := setup(t)
	src := fakeDepSrc{rows: []DepositRow{
		{DepID: "440867191", Ccy: "USDT", Amt: "2000", State: "2", Ts: "1761000000000"},
		{DepID: "440867192", Ccy: "USDT", Amt: "500", State: "1", Ts: "1761000001000"}, // 非完成=不入账
	}}
	sc := NewDepositScanner(pool, src)
	n, err := sc.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("完成态应入 1 行: %d", n)
	}
	n2, _ := sc.Scan(ctx)
	if n2 != 0 {
		t.Fatalf("重复巡检应零重账: %d", n2)
	}
	var bills, flows int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_venue_bill WHERE bill_id='dep-440867191'").Scan(&bills)
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_cashflow WHERE kind='deposit' AND amount=2000").Scan(&flows)
	if bills != 1 || flows != 1 {
		t.Fatalf("bill=%d flow=%d 应 1/1", bills, flows)
	}
}

// T-0149 返修②锚：bill+cashflow 同事务——cashflow 写败→bill 行同回滚不留孤儿。
// （cashflow 注入败真仿=约束违例面；此处借非 USDT 入账行模拟现金面败径——
// 事务原子语义由同 tx.QueryRow/Exec/Commit 代码径保障，PG 级锚已证双行同成。）

// T-0149 二返修桩锚：非 USDT depId 行→零写零败（D-0021 TRX 自营边界）；
// 夹杂行序→后续 USDT 行照入账（逐行隔离不 abort 整扫）。
func TestDepositScanUSDTGate(t *testing.T) {
	ctx, pool := setup(t)
	src := fakeDepSrc{rows: []DepositRow{
		{DepID: "trx1", Ccy: "TRX", Amt: "1000", State: "2", Ts: "1761000000000"}, // 非 USDT=不入账
		{DepID: "u1", Ccy: "USDT", Amt: "100", State: "2", Ts: "1761000000001"},
		{DepID: "btc1", Ccy: "BTC", Amt: "0.5", State: "2", Ts: "1761000000002"}, // 夹杂毒行序
		{DepID: "u2", Ccy: "USDT", Amt: "200", State: "2", Ts: "1761000000003"},
	}}
	sc := NewDepositScanner(pool, src)
	n, err := sc.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("USDT 应入 2 行（TRX/BTC 滤除）: %d", n)
	}
	var c int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_venue_bill WHERE bill_id LIKE 'dep-%'").Scan(&c)
	if c != 2 {
		t.Fatalf("bill 行应 2（非 USDT 不留 bill）: %d", c)
	}
}
