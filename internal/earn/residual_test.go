// residual_test.go —— 腿平残币归口桩测（T-0095 v3 验收锚）：
//   - 申购径落凭据行（decision intent ccy=残币币种，零换手无现金流）；
//   - 低额=托管标注非假装成功（30 天幂等窗）；
//   - 在持腿币不误搬；申购拒=回摆补偿+托管标。
package earn

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// residualSetup 开仓腿 fixture（venue 域=worker 同域——FundCcy venue 过滤对齐）。
func residualSetup(t *testing.T, ctx context.Context, pool *pgxpool.Pool, venue string) {
	t.Helper()
	var xid, pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('ETH-USDT',100,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd)
		VALUES($1,'ETH-USDT',100) RETURNING id`, xid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,'cl-res',$2,'buy','ETH-USDT',1,'filled')`, xid, venue); err != nil {
		t.Fatal(err)
	}
}

func custodyCount(t *testing.T, ctx context.Context, w *Worker, ccy string) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_residual_custody WHERE venue=$1 AND ccy=$2`, w.venue, ccy).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// 锚①：够起投残币→摆渡+申购径落凭据行（decision intent ccy=FIL；kind 空位
// 零现金流——零换手无 earn_* 假行）。
func TestResidualPurchasePath(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, nil)
	residualSetup(t, ctx, w.pool, w.venue)
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,inst_id,last) VALUES(now(),'FIL-USDT',5)`); err != nil {
		t.Fatal(err)
	}
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"FIL": 200, "ETH": 0.1}, nil // ETH=在持腿币候选外
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 1 || fake.transfers[0] != "18->6:200.0000" {
		t.Fatalf("残币摆渡径: %v", fake.transfers)
	}
	if len(fake.purchases) != 1 || fake.purchases[0] != 200 {
		t.Fatalf("申购径: %v", fake.purchases)
	}
	var ccy string
	if err := w.pool.QueryRow(ctx, `SELECT intent->>'ccy' FROM fin_risk_decision
		WHERE reason='transfer' ORDER BY id DESC LIMIT 1`).Scan(&ccy); err != nil || ccy != "FIL" {
		t.Fatalf("凭据行 ccy=%q err=%v（应 FIL——归口扩币种留真）", ccy, err)
	}
	if n := len(cashflowOf(t, ctx, w, "earn_in")) + len(cashflowOf(t, ctx, w, "earn_out")); n != 0 {
		t.Fatalf("零换手不得落现金流行: %d", n)
	}
	if custodyCount(t, ctx, w, "FIL") != 0 {
		t.Fatal("申购成不落托管标")
	}
}

// 锚②：低额=托管如实标非假装成功；幂等（30 天窗内重复扫不重复落行）。
func TestResidualLowAmountCustody(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, nil)
	residualSetup(t, ctx, w.pool, w.venue)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"DOGE": 5}, nil // 无 tick=估值 0 <起投底
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 0 || len(fake.purchases) != 0 {
		t.Fatalf("低额禁钱动: %v %v", fake.transfers, fake.purchases)
	}
	if n := custodyCount(t, ctx, w, "DOGE"); n != 1 {
		t.Fatalf("托管标应幂等 1 行: %d", n)
	}
}

// 锚③：在持腿基币不误搬（fund 集内——ETF 式候选外）。
func TestResidualOpenLegNotSwept(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, nil)
	residualSetup(t, ctx, w.pool, w.venue)
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,inst_id,last) VALUES(now(),'ETH-USDT',3000)`); err != nil {
		t.Fatal(err)
	}
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"ETH": 0.032}, nil
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 0 || len(fake.purchases) != 0 || custodyCount(t, ctx, w, "ETH") != 0 {
		t.Fatal("在持腿基币被误搬（fund 集守恒破）")
	}
}

// 锚④：申购业务拒=确认未动回摆补偿+托管标（钱最终留交易户，非假装成功）。
func TestResidualPurchaseRejectCompensates(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{purchaseErr: errors.New("58110 risk locked")}, nil)
	residualSetup(t, ctx, w.pool, w.venue)
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,inst_id,last) VALUES(now(),'FIL-USDT',5)`); err != nil {
		t.Fatal(err)
	}
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"FIL": 200}, nil
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 2 || fake.transfers[1] != "6->18:200.0000" {
		t.Fatalf("申购拒须回摆补偿: %v", fake.transfers)
	}
	if custodyCount(t, ctx, w, "FIL") != 1 {
		t.Fatal("申购拒须落托管标")
	}
}
