package ledger

import (
	"context"
	"testing"

	"phase/internal/platform/config"
	"phase/internal/platform/rejects"
	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *Runner) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO app_config(key,value) VALUES('ledger.recon_tol_usd','\"0.5\"') ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, NewRunner(pool, cfg)
}

func TestPostGuards(t *testing.T) {
	ctx, r := setup(t)
	// 缺 source_ref 拒写锚
	if _, err := Post(ctx, r.pool, Entry{Kind: "fee", Venue: "okx", Ccy: "USDT", Amount: 1}); err == nil {
		t.Fatal("缺 source_ref 应拒写")
	}
	// 杂币不入账锚（业主令 USDT/USD 各记各的）
	if _, err := Post(ctx, r.pool, Entry{Kind: "fee", Venue: "okx", Ccy: "PENGU", Amount: 1, SourceRef: 1, SourceTable: "x"}); err == nil {
		t.Fatal("杂币应拒写")
	}
	// kind 闭集外拒
	if _, err := Post(ctx, r.pool, Entry{Kind: "guess", Venue: "okx", Ccy: "USDT", Amount: 1, SourceRef: 1, SourceTable: "x"}); err == nil {
		t.Fatal("kind 闭集外应拒")
	}
}

func TestSettleAndBreakdown(t *testing.T) {
	ctx, r := setup(t)
	var xid, pid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('X',100,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,closed_at)
		VALUES($1,'X',100,now()) RETURNING id`, xid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	for _, e := range []Entry{
		{Kind: "fee", Venue: "okx", Ccy: "USDT", Amount: -0.5, PositionRef: &pid, SourceRef: 1, SourceTable: "fin_order_leg"},
		{Kind: "funding_settle", Venue: "okx", Ccy: "USDT", Amount: 1.2, PositionRef: &pid, SourceRef: 2, SourceTable: "fin_order_leg"},
		{Kind: "trade_pnl", Venue: "okx", Ccy: "USDT", Amount: 0.3, PositionRef: &pid, SourceRef: 3, SourceTable: "fin_order_leg"},
	} {
		if _, err := Post(ctx, r.pool, e); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	var amt float64
	var ccy string
	if err := r.pool.QueryRow(ctx,
		"SELECT settled_amount,settled_ccy FROM fin_position WHERE id=$1", pid).Scan(&amt, &ccy); err != nil || amt != 1.0 || ccy != "USDT" {
		t.Fatalf("结算落位 amt=%v ccy=%s err=%v", amt, ccy, err)
	}
	br, err := Breakdown(ctx, r.pool, pid)
	if err != nil || br["funding_settle"] != 1.2 {
		t.Fatalf("判决书构成: %v err=%v", br, err)
	}
	// 日切快照锚
	if err := r.daySnapshot(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_ledger_snapshot WHERE venue='okx' AND ccy='USDT'").Scan(&n); err != nil || n == 0 {
		t.Fatalf("日切行=%d", n)
	}
}

func TestReconcileDiffAlert(t *testing.T) {
	ctx, r := setup(t)
	// 账本净额 100；venue 现读 200→差 100>0.5 容差=告警计门
	var ven map[[2]string]float64
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, error) { return ven, nil })
	if _, err := Post(ctx, r.pool, Entry{Kind: "deposit", Venue: "okx", Ccy: "USDT", Amount: 100, SourceRef: 1, SourceTable: "t"}); err != nil {
		t.Fatal(err)
	}
	ven = map[[2]string]float64{{"okx", "USDT"}: 200}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["ledger/diff_alert"] == 0 {
		t.Fatal("造差未告警计门")
	}
	// venue 现读件缺席=对账让位不炸锚
	r2 := NewRunner(r.pool, r.cfg)
	before := r2.rej.Snapshot()["ledger/diff_alert"]
	if err := r2.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if r2.rej.Snapshot()["ledger/diff_alert"] != before {
		t.Fatal("读件缺席被单边判")
	}
}

// T-0045 锚：同 venue+ccy 多 venue_type 子户行（unified 交易户+earn 停泊户）
// 按 (venue,ccy) 聚合比对——聚合后账实平=零告警（修前逐行对总额比永响）。
func TestReconcileMultiSubAccountAggregated(t *testing.T) {
	ctx, r := setup(t)
	// 账本净额 2104.09（告警实证场景复刻）；venue 现读聚合额=unified 75.49+earn 2028.60 同口径。
	var ven map[[2]string]float64
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, error) { return ven, nil })
	if _, err := Post(ctx, r.pool, Entry{Kind: "deposit", Venue: "okx", Ccy: "USDT", Amount: 2104.09, SourceRef: 1, SourceTable: "t"}); err != nil {
		t.Fatal(err)
	}
	ven = map[[2]string]float64{{"okx", "USDT"}: 2104.09}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if n := r.rej.Snapshot()["ledger/diff_alert"]; n != 0 {
		t.Fatalf("聚合平账误报: diff_alert=%d（应 0——venue 现读与账本平）", n)
	}
	// 反向证：聚合额缺 2028.60→差超容差必须告警。
	ven = map[[2]string]float64{{"okx", "USDT"}: 75.49}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["ledger/diff_alert"] == 0 {
		t.Fatal("真差不告警=漏报")
	}
}

// T-0055 锚①：Earn 利息事件化入账——bootstrap 播种零入账；累计涨→interest 行
// 落账（source=事件行）；涨不足半分不落。
func TestPostInterest(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE fin_cashflow"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM md_account_events WHERE channel='earn_interest'"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, config.New(pool))
	ens := map[string]float64{"USDT": 72.22}
	r.earnings = func(ctx context.Context) (map[string]float64, error) { return ens, nil }
	r.postInterest(ctx) // bootstrap：播种事件行零入账
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM fin_cashflow WHERE kind='interest'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("bootstrap 应零入账: n=%d err=%v", n, err)
	}
	ens["USDT"] = 72.2245
	r.postInterest(ctx) // delta 0.0045 < 0.005 不落
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM fin_cashflow WHERE kind='interest'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("应计不足半分不落: n=%d", n)
	}
	ens["USDT"] = 72.2312
	r.postInterest(ctx) // delta ≥0.005 落账
	var amt float64
	var src int64
	var tbl string
	if err := pool.QueryRow(ctx, `SELECT amount,source_ref,source_table FROM fin_cashflow WHERE kind='interest'`).
		Scan(&amt, &src, &tbl); err != nil || amt < 0.005 || tbl != "md_account_events" || src == 0 {
		t.Fatalf("interest 行: amt=%v src=%v tbl=%s err=%v", amt, src, tbl, err)
	}
}

// T-0055 锚②：kind 闭集含 interest/recon_adjust（迁移 0030 后语义）。
func TestInterestKindsInClosedSet(t *testing.T) {
	if !kinds["interest"] || !kinds["recon_adjust"] {
		t.Fatal("interest/recon_adjust 应在 kind 闭集")
	}
}

// T-0086 ③锚：对拍白名单=USDT+在册（fin_cashflow 有账币种）——账外币种按身份
// 豁免：钉多大金额都不报警（涨跌不警）。
func TestReconcileOffLedgerExempt(t *testing.T) {
	ctx, r := setup(t)
	// 隔离：清他测留下的已知残差豁免（T-0081 容差补进会吃掉本测造差）。
	if _, err := r.pool.Exec(ctx, "DELETE FROM fin_known_residual"); err != nil {
		t.Fatal(err)
	}
	var ven map[[2]string]float64
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, error) { return ven, nil })
	if _, err := Post(ctx, r.pool, Entry{Kind: "deposit", Venue: "okx", Ccy: "USDT", Amount: 100, SourceRef: 1, SourceTable: "t"}); err != nil {
		t.Fatal(err)
	}
	// 在册 USDT 差 1（>容差 0.5）→告警；账外 DOGE/Earn-BTC 钉巨额→零告警。
	ven = map[[2]string]float64{
		{"okx", "USDT"}: 101,
		{"okx", "DOGE"}: 999999,
		{"okx", "BTC"}:  8888,
	}
	r.rej = rejects.New("ledger", "diff_alert", "missing_param", "bal_missing", "write_err")
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if got := r.rej.Snapshot()["ledger/diff_alert"]; got != 1 {
		t.Fatalf("diff_alert=%d（应恰 1——USDT 告警、账外币豁免零警）", got)
	}
}
