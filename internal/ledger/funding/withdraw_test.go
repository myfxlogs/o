// withdraw_test.go —— T-0274 桩锚：出金常设径幂等/闸面/基线/仿真重放。
package funding

import (
	"context"
	"errors"
	"testing"
)

type fakeWdSrc struct{ rows []WithdrawalRow }

func (f fakeWdSrc) WithdrawalHistory(context.Context) ([]WithdrawalRow, error) { return f.rows, nil }

type errWdSrc struct{}

func (errWdSrc) WithdrawalHistory(context.Context) ([]WithdrawalRow, error) {
	return nil, errors.New("读败模拟")
}

// 锚：state=2 出金→bill+cashflow 双行（withdraw 负号=出向）；重复巡检不重账
// （wdId 幂等）。行面用本案真形态（wdId 440849407，-10000，2026-11-01 探针证）。
func TestWithdrawScanIdempotent(t *testing.T) {
	ctx, pool := setup(t)
	seedBaseline(t, ctx, pool)
	src := fakeWdSrc{rows: []WithdrawalRow{
		{WdID: "440849407", Ccy: "USDT", Amt: "10000", Fee: "0", State: "2", Ts: "1793371328000"},
		{WdID: "440849408", Ccy: "USDT", Amt: "500", Fee: "1", State: "0", Ts: "1793371329000"}, // 非完成=不入账
	}}
	sc := NewWithdrawalScanner(pool, src)
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
	var bills int
	var amt float64
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_venue_bill WHERE bill_id='wd-440849407'").Scan(&bills)
	_ = pool.QueryRow(ctx, "SELECT amount FROM fin_cashflow WHERE kind='withdraw' LIMIT 1").Scan(&amt)
	if bills != 1 || amt != -10000 {
		t.Fatalf("bill=%d amount=%v 应 1/-10000", bills, amt)
	}
}

// T-0274 闸面：非 USDT 滤除（deposit 对偶）+夹毒行序逐行隔离不 abort；
// 读败=上抛零写（不炸主径——下拍重试幂等切面不破）。
func TestWithdrawScanGates(t *testing.T) {
	ctx, pool := setup(t)
	seedBaseline(t, ctx, pool)
	src := fakeWdSrc{rows: []WithdrawalRow{
		{WdID: "btc1", Ccy: "BTC", Amt: "0.5", State: "2", Ts: "1793371328000"}, // 非 USDT=不入账
		{WdID: "u1", Ccy: "USDT", Amt: "100", State: "2", Ts: "1793371328001"},
	}}
	sc := NewWithdrawalScanner(pool, src)
	n, err := sc.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("USDT 应入 1 行（BTC 滤除）: %d", n)
	}
	if _, err := NewWithdrawalScanner(pool, errWdSrc{}).Scan(ctx); err == nil {
		t.Fatal("读败应上抛")
	}
	var c int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_venue_bill WHERE bill_id LIKE 'wd-%'").Scan(&c)
	if c != 1 {
		t.Fatalf("读败后零新增写: %d", c)
	}
}

// T-0274 基线锚：pre-baseline 提币史（真户实证 wdId 433855125 等 2026-09/10
// 旧账在册）→零写；post-baseline→入账。
func TestWithdrawScanBaselineCutoff(t *testing.T) {
	ctx, pool := setup(t)
	seedBaseline(t, ctx, pool)
	src := fakeWdSrc{rows: []WithdrawalRow{
		{WdID: "pre1", Ccy: "USDT", Amt: "1000", State: "2", Ts: "1790672748000"}, // 2026-10-09=pre 基线
		{WdID: "post1", Ccy: "USDT", Amt: "300", State: "2", Ts: "1793371328000"}, // 2026-10-30=post
	}}
	sc := NewWithdrawalScanner(pool, src)
	n, _ := sc.Scan(ctx)
	if n != 1 {
		t.Fatalf("post-baseline 应入 1 行: %d", n)
	}
	var c int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_venue_bill WHERE bill_id='wd-pre1'").Scan(&c)
	if c != 0 {
		t.Fatalf("pre-baseline wdId 应零写: %d", c)
	}
}

// 仿真锚（票面）：重放 2001103/2001104 形态——/account/bills type=1 双腿
// （sub11 +10000 内转进腿/sub12 -10000 内转出腿）零 cashflow（源行留档）；
// withdrawal-history 案行→恰一行 withdraw（-10000）。行面自证「划转行禁
// 记账+外向唯自 withdrawal-history 终证」全链。
func TestWithdrawTranslationReplay(t *testing.T) {
	ctx, pool := setup(t)
	seedBaseline(t, ctx, pool)
	ing := NewIngester(pool, nil) // record 分派径直拍——不依赖 src
	for _, b := range []Bill{
		{BillID: "2001104", Type: "1", SubType: "11", Ccy: "USDT", TsMs: 1793370961000, BalChg: 10000},
		{BillID: "2001103", Type: "1", SubType: "12", Ccy: "USDT", TsMs: 1793370962000, BalChg: -10000},
	} {
		if _, err := ing.record(ctx, b); err != nil {
			t.Fatal(err)
		}
	}
	var inner int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow cf
		JOIN fin_venue_bill vb ON cf.source_ref=vb.id
		WHERE vb.bill_id IN ('2001103','2001104')`).Scan(&inner)
	if inner != 0 {
		t.Fatalf("type=1 划转双腿应零 cashflow: %d", inner)
	}
	sc := NewWithdrawalScanner(pool, fakeWdSrc{rows: []WithdrawalRow{
		{WdID: "440849407", Ccy: "USDT", Amt: "10000", State: "2", Ts: "1793371328000"},
	}})
	n, err := sc.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var wd int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_cashflow WHERE kind='withdraw'").Scan(&wd)
	if n != 1 || wd != 1 {
		t.Fatalf("案行应恰一行 withdraw -10000: n=%d rows=%d", n, wd)
	}
}
