package ledger

import (
	"context"
	"testing"

	"phase/internal/platform/config"
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
	// 账本净额 100；venue 快照 200→差 100>0.5 容差=告警计门
	if _, err := Post(ctx, r.pool, Entry{Kind: "deposit", Venue: "okx", Ccy: "USDT", Amount: 100, SourceRef: 1, SourceTable: "fin_venue_account_snapshot"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_venue_account_snapshot(venue,venue_type,ccy,equity,src,updated_at)
		VALUES('okx','unified','USDT',200,'x',now())
		ON CONFLICT(venue,venue_type,ccy) DO UPDATE SET equity=200,updated_at=now()`); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["ledger/diff_alert"] == 0 {
		t.Fatal("造差未告警计门")
	}
	// 陈快照不判锚（>5min）
	if _, err := r.pool.Exec(ctx,
		"UPDATE fin_venue_account_snapshot SET updated_at=now()-interval '10 minutes' WHERE ccy='USDT'"); err != nil {
		t.Fatal(err)
	}
	before := r.rej.Snapshot()["ledger/diff_alert"]
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["ledger/diff_alert"] != before {
		t.Fatal("陈快照被单边判")
	}
}

// T-0045 锚：同 venue+ccy 多 venue_type 子户行（unified 交易户+earn 停泊户）
// 按 (venue,ccy) 聚合比对——聚合后账实平=零告警（修前逐行对总额比永响）。
func TestReconcileMultiSubAccountAggregated(t *testing.T) {
	ctx, r := setup(t)
	// 账本净额 2104.09（告警实证场景复刻）；快照两行 75.49+2028.60。
	if _, err := Post(ctx, r.pool, Entry{Kind: "deposit", Venue: "okx", Ccy: "USDT", Amount: 2104.09, SourceRef: 1, SourceTable: "fin_venue_account_snapshot"}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		vt string
		eq float64
	}{
		{"unified", 75.49}, {"earn", 2028.60},
	} {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_venue_account_snapshot(venue,venue_type,ccy,equity,src,updated_at)
			VALUES('okx',$1,'USDT',$2,'x',now())
			ON CONFLICT(venue,venue_type,ccy) DO UPDATE SET equity=$2,updated_at=now()`, row.vt, row.eq); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if n := r.rej.Snapshot()["ledger/diff_alert"]; n != 0 {
		t.Fatalf("聚合平账误报: diff_alert=%d（应 0——75.49+2028.60=2104.09 与账本平）", n)
	}
	// 反向证：抽走一行（聚合额缺 2028.60）→差超容差必须告警。
	if _, err := r.pool.Exec(ctx, "DELETE FROM fin_venue_account_snapshot WHERE venue_type='earn'"); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["ledger/diff_alert"] == 0 {
		t.Fatal("真差不告警=漏报")
	}
}
