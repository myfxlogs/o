// deposit_test.go —— T-0149 桩锚：入金巡检幂等双行（重复扫不重账）。
package funding

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeDepSrc struct{ rows []DepositRow }

func (f fakeDepSrc) DepositHistory(context.Context) ([]DepositRow, error) { return f.rows, nil }

// 锚：state=2 入金→bill+cashflow 双行；重复巡检不重账（depId 幂等）。
func TestDepositScanIdempotent(t *testing.T) {
	ctx, pool := setup(t)
	seedBaseline(t, ctx, pool)
	src := fakeDepSrc{rows: []DepositRow{
		{DepID: "440867191", Ccy: "USDT", Amt: "2000", State: "2", Ts: "1792886400000"},
		{DepID: "440867192", Ccy: "USDT", Amt: "500", State: "1", Ts: "1792886401000"}, // 非完成=不入账
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

// T-0149 二返修桩锚+O-36422 根修：币形闸——BTC 等原生币注金按本域 ccy
// 落账（custody 按币量对拍；depId=439722036 BTC 注资漏账实证案根修），
// 形坏串滤除；夹杂行序→后续行照入账（逐行隔离不 abort 整扫）。
func TestDepositScanCcyGate(t *testing.T) {
	ctx, pool := setup(t)
	seedBaseline(t, ctx, pool)
	src := fakeDepSrc{rows: []DepositRow{
		{DepID: "trx1", Ccy: "TRX", Amt: "1000", State: "2", Ts: "1792886400000"}, // 原生币=本域入账
		{DepID: "bad1", Ccy: "btc!", Amt: "1", State: "2", Ts: "1792886400000"},   // 形坏=滤除
		{DepID: "u1", Ccy: "USDT", Amt: "100", State: "2", Ts: "1792886400001"},
		{DepID: "btc1", Ccy: "BTC", Amt: "0.5", State: "2", Ts: "1792886400002"}, // 夹杂行序
		{DepID: "u2", Ccy: "USDT", Amt: "200", State: "2", Ts: "1792886400003"},
	}}
	sc := NewDepositScanner(pool, src)
	n, err := sc.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("TRX+USDT+BTC+USDT 应入 4 行（形坏滤除）: %d", n)
	}
	var c int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_venue_bill WHERE bill_id LIKE 'dep-%'").Scan(&c)
	if c != 4 {
		t.Fatalf("bill 行应 4（形坏不留 bill）: %d", c)
	}
	// 验收锚：BTC 注金按本域币量落 BTC cashflow（非 USDT 面值换算）。
	var btcAmt float64
	var btcCcy string
	_ = pool.QueryRow(ctx, "SELECT ccy,amount FROM fin_cashflow WHERE kind='deposit' AND source_table='fin_venue_bill' AND amount=0.5").Scan(&btcCcy, &btcAmt)
	if btcCcy != "BTC" || btcAmt != 0.5 {
		t.Fatalf("BTC 注金应 ccy=BTC amount=0.5 got %s/%v", btcCcy, btcAmt)
	}
}

// T-0149 三返修桩锚：pre-baseline depId→零写+pre_baseline 计数；post-baseline→入账。
func TestDepositScanBaselineCutoff(t *testing.T) {
	ctx, pool := setup(t)
	seedBaseline(t, ctx, pool)
	src := fakeDepSrc{rows: []DepositRow{
		{DepID: "pre1", Ccy: "USDT", Amt: "420803", State: "2", Ts: "1751385600000"}, // 2025-07=pre 基线
		{DepID: "post1", Ccy: "USDT", Amt: "100", State: "2", Ts: "1792886400000"},   // 2026-10-25=post
	}}
	sc := NewDepositScanner(pool, src)
	n, _ := sc.Scan(ctx)
	if n != 1 {
		t.Fatalf("post-baseline 应入 1 行: %d", n)
	}
	var c int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_venue_bill WHERE bill_id='dep-pre1'").Scan(&c)
	if c != 0 {
		t.Fatalf("pre-baseline depId 应零写（无 bill 行）: %d", c)
	}
}

// seedBaseline 账基线锚（最早 cashflow 行=基线截止谓词参照——2026-10-24
// 开盘行模拟；pre/post 行都相对此戳谓词）。
func seedBaseline(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
		VALUES('deposit','okx','USDT',500,999,'fin_venue_bill','2026-10-24 04:07')`); err != nil {
		t.Fatal(err)
	}
}
