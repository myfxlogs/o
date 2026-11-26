package main

// margin_trackgap_test.go —— O-41350 验收五案：
// ①flatten_failed 重枚举+息腿续记（枚举窗扩根修——息腿/健康闸不再失明）
// ②唯一索引 flatten_failed 同币双借拒 ③裸负债升件 margin_naked_liab
// ④spot_sell 毛额=fillSz×成交avgPx（非报价 px）⑤flatten 连击封顶升件+
// 防抖+封顶后停自动重试。

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/okxexec"
)

// ①flatten_failed=仍计息负债：枚举在账+息腿续记（pre-fix 三隐身链根修锚）。
func TestMarginFlattenFailedEnumerated(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO md_margin_borrow(ccy,inst_id,mgn_mode,borrow_apr,max_loan,ts)
		VALUES('FFL','FFL-USDT','cross',0.5,10,now())`); err != nil {
		t.Fatal(err)
	}
	var loanID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr,status,last_accrual_at)
		VALUES('okx','FFL','FFL-USDT',2,0.4,'flatten_failed',now()-interval '10 hours') RETURNING id`).Scan(&loanID); err != nil {
		t.Fatal("loan seed:", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `UPDATE fin_margin_loan SET status='repaid',repaid_at=now() WHERE id=$1`, loanID)
	}()
	loans, err := marginOpenLoans(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	var found *marginLoan
	for i := range loans {
		if loans[i].ID == loanID {
			found = &loans[i]
		}
	}
	if found == nil || found.Status != "flatten_failed" {
		t.Fatalf("flatten_failed 未重枚举 found=%+v", found)
	}
	if err := marginAccrue(ctx, pool, loans); err != nil {
		t.Fatal("accrue:", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow
		WHERE source_table='fin_margin_loan' AND source_ref=$1 AND kind='borrow_interest'`, loanID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("flatten_failed 息腿未续记 n=%d err=%v", n, err)
	}
}

// ②fin_margin_loan_one_open 部分索引含 flatten_failed——同币双借封死。
func TestMarginLoanUniqueFlattenFailed(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr,status)
		VALUES('okx','UNQ','UNQ-USDT',1,0.1,'open')`); err != nil {
		t.Fatal("open seed:", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `UPDATE fin_margin_loan SET status='repaid',repaid_at=now() WHERE ccy='UNQ'`)
	}()
	if _, err := pool.Exec(ctx, `INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr,status)
		VALUES('okx','UNQ','UNQ-USDT',1,0.1,'flatten_failed')`); err == nil {
		t.Fatal("flatten_failed 同币双借未被唯一索引拒")
	}
	// 对照：repaid 终态不占唯一键（同币可再借）。
	if _, err := pool.Exec(ctx, `UPDATE fin_margin_loan SET status='repaid',repaid_at=now()
		WHERE ccy='UNQ' AND status='open'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr,status)
		VALUES('okx','UNQ','UNQ-USDT',1,0.1,'flatten_failed')`); err != nil {
		t.Fatalf("repaid 后 flatten_failed 应可入 err=%v", err)
	}
}

// marginOpenTestSetup 开仓测公共夹具：闸键+证据+价+equity+redeem。
func marginOpenTestSetup(t *testing.T, pool *pgxpool.Pool, cfg *config.Client,
	ccy string, histErr error) (*fakeMarginVenue, func(context.Context, float64) error, float64) {
	t.Helper()
	ctx := context.Background()
	for _, kv := range [][2]string{
		{"margin.exec_enabled", "true"}, {"margin.max_open_usd", "500"},
		{"funding.margin_borrow_stale_hours", "6"}, {"funding.margin_borrow_max_apr", "0.5"},
		{"funding.margin_min_loan_usd", "1"},
	} {
		if _, err := pool.Exec(ctx, `UPDATE app_config SET value=$1 WHERE key=$2`, kv[1], kv[0]); err != nil {
			t.Fatal(kv, err)
		}
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_margin_borrow(ccy,inst_id,mgn_mode,borrow_apr,max_loan,ts)
		VALUES($1,$2,'cross',0.1,1000,now())`, ccy, ccy+"-USDT"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now(),'okx',$1,50)`, ccy+"-USDT"); err != nil {
		t.Fatal(err)
	}
	cap1, _ := cfg.GetFloat("exec.probe_notional_usd")
	cap2, _ := cfg.GetFloat("risk.max_notional_usd")
	wantN := min(100.0, 500.0, cap1, cap2)
	wantQty := wantN / 50.0
	fv := &fakeMarginVenue{maxLoan: 1000, adjEq: 50000, histErr: histErr}
	if histErr == nil {
		fv.hist = []okxexec.BorrowRepayHistRow{{
			Ccy: ccy, Amt: fmt.Sprintf("%g", wantQty), Side: "borrow",
			TS: fmt.Sprintf("%d", time.Now().UnixMilli())}}
	}
	return fv, func(context.Context, float64) error { return nil }, wantQty
}

// ③借成确认败=裸负债升件（venue 侧可能真借而台账零行——人工可见不静默）。
func TestMarginNakedLiabEscalation(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	cfg := marginCfg(t, pool)
	// histErr→借后确认即时败（不等 5s 轮询窗）。
	fv, redeem, _ := marginOpenTestSetup(t, pool, cfg, "NKD", fmt.Errorf("hist read fail"))
	gate := risk.NewGate(pool, cfg).WithEquity(func(context.Context, string) (float64, error) { return 100000, nil })
	h := hedgeOpener{pool: pool, cfg: cfg, gate: gate, mvenue: fv, redeem: redeem}
	_, err := h.marginShortOpen(ctx, lifecycle.OpenSpec{
		StrategyRef: 999997, Venue: "okx", InstID: "NKD-USDT-SWAP",
		Direction: "long_perp_short_spot", NotionalUSD: 100})
	if err == nil {
		t.Fatal("确认败应上抛")
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_class='margin_naked_liab' AND note LIKE '%NKD%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("裸负债升件 n=%d err=%v want 1", n, err)
	}
	var loans int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_margin_loan WHERE ccy='NKD'`).Scan(&loans)
	if loans != 0 {
		t.Fatalf("未簿记应零 loan 行 got=%d", loans)
	}
	// 卖腿不得触发（借了不卖=负债裸持如实停）。
	for _, c := range fv.calls {
		if c == "place:sell" {
			t.Fatalf("确认败仍卖腿 calls=%v", fv.calls)
		}
	}
}

// ④spot_sell 毛额=fillSz×成交 avgPx（滑点实收——报价 px 仅兜底）。
func TestMarginSpotSellAvgPx(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	cfg := marginCfg(t, pool)
	fv, redeem, wantQty := marginOpenTestSetup(t, pool, cfg, "APX", nil)
	fv.avgPx = 51 // ≠报价 50——成交均价带滑点
	fv.fillSz = wantQty
	fv.resolveSt = okxexec.StateFilled
	gate := risk.NewGate(pool, cfg).WithEquity(func(context.Context, string) (float64, error) { return 100000, nil })
	h := hedgeOpener{pool: pool, cfg: cfg, gate: gate, mvenue: fv, redeem: redeem}
	execID, err := h.marginShortOpen(ctx, lifecycle.OpenSpec{
		StrategyRef: 999996, Venue: "okx", InstID: "APX-USDT-SWAP",
		Direction: "long_perp_short_spot", NotionalUSD: 100})
	if err != nil {
		t.Fatal("开仓败:", err)
	}
	var amt float64
	if err := pool.QueryRow(ctx, `SELECT amount FROM fin_cashflow
		WHERE kind='spot_sell' AND source_table='fin_execution' AND source_ref=$1`, execID).Scan(&amt); err != nil {
		t.Fatal("spot_sell 行缺:", err)
	}
	want := wantQty * 51
	if math.Abs(math.Abs(amt)-want) > want*0.001 {
		t.Fatalf("spot_sell amount=%.6f want=%.6f（fillSz×avgPx）", amt, want)
	}
	// 清场。
	_, _ = pool.Exec(ctx, `UPDATE fin_margin_loan SET status='repaid',repaid_at=now() WHERE ccy='APX'`)
	_, _ = pool.Exec(ctx, `UPDATE fin_position SET closed_at=now() WHERE inst_id='APX-USDT' AND closed_at IS NULL`)
}

// ⑤flatten_failed 重试有界：防抖窗内不动→过窗重试→连击封顶升件一次→
// 封顶后永久停自动重试（负债仍在枚举窗=息腿/健康闸不受影响）。
func TestMarginFlattenRetryBounded(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	cfg := marginCfg(t, pool)
	for _, kv := range [][2]string{
		{"margin.flatten_max_fails", "1"}, {"margin.flatten_retry_ivl_s", "1"},
	} {
		if _, err := pool.Exec(ctx, `UPDATE app_config SET value=$1 WHERE key=$2`, kv[1], kv[0]); err != nil {
			t.Fatal(kv, err)
		}
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	var loanID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr,status,
		flatten_fails,last_flatten_try_at)
		VALUES('okx','BND','BND-USDT',3,0.1,'flatten_failed',0,now()-interval '1 hour') RETURNING id`).Scan(&loanID); err != nil {
		t.Fatal("loan seed:", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `UPDATE fin_margin_loan SET status='repaid',repaid_at=now() WHERE id=$1`, loanID)
	}()
	fv := &fakeMarginVenue{adjEq: 100000, placeErr: fmt.Errorf("venue reject")}
	// 拍一：fails=0<max=1∧距上试 1h>1s→重试→flatten 败→fails=1。
	if err := marginLoanTick(ctx, pool, cfg, fv); err != nil {
		t.Fatal("tick1:", err)
	}
	var fails int
	var st string
	if err := pool.QueryRow(ctx, `SELECT flatten_fails,status FROM fin_margin_loan WHERE id=$1`, loanID).Scan(&fails, &st); err != nil {
		t.Fatal(err)
	}
	if fails != 1 || st != "flatten_failed" {
		t.Fatalf("拍一后 fails=%d status=%s want 1/flatten_failed", fails, st)
	}
	places := 0
	for _, c := range fv.calls {
		if c == "place:buy" {
			places++
		}
	}
	if places != 1 {
		t.Fatalf("拍一 place:buy=%d want 1 calls=%v", places, fv.calls)
	}
	// 拍二：fails=1≥max=1→封顶升件一次。
	if err := marginLoanTick(ctx, pool, cfg, fv); err != nil {
		t.Fatal("tick2:", err)
	}
	var escN int
	var escd bool
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_class='margin_flatten_fail' AND note LIKE $1`,
		fmt.Sprintf("%%loan#%d%%", loanID)).Scan(&escN); err != nil || escN != 1 {
		t.Fatalf("封顶升件 n=%d err=%v want 1", escN, err)
	}
	if err := pool.QueryRow(ctx, `SELECT flatten_escalated FROM fin_margin_loan WHERE id=$1`, loanID).Scan(&escd); err != nil || !escd {
		t.Fatalf("flatten_escalated=%v err=%v want true", escd, err)
	}
	// 拍三：封顶态——零 venue 平仓调用零新升件（永久停自动重试）。
	callsBefore := len(fv.calls)
	if err := marginLoanTick(ctx, pool, cfg, fv); err != nil {
		t.Fatal("tick3:", err)
	}
	newPlaces := 0
	for _, c := range fv.calls[callsBefore:] {
		if c == "place:buy" {
			newPlaces++
		}
	}
	if newPlaces != 0 {
		t.Fatalf("封顶后仍重试 place=%d", newPlaces)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_class='margin_flatten_fail' AND note LIKE $1`,
		fmt.Sprintf("%%loan#%d%%", loanID)).Scan(&escN); err != nil || escN != 1 {
		t.Fatalf("封顶后升件重复 n=%d want 1", escN)
	}
}
