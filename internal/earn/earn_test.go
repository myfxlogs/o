package earn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"phase/internal/platform/config"
	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
	"phase/internal/trading/okxexec"
)

// fakeAPI 可控 venue 停泊面（行为开关+调用记录——钱动锚全走此件）。
type fakeAPI struct {
	redeemErr, purchaseErr, transferErr, stateErr error
	transferState                                 string // 非""=传输败后按 clTId 查得此态
	balance                                       float64
	redeems, purchases, stateQueries              []float64
	transfers                                     []string // "from->to:amt"
}

func (f *fakeAPI) SavingsRedeem(ctx context.Context, ccy string, amt float64) error {
	f.redeems = append(f.redeems, amt)
	return f.redeemErr
}
func (f *fakeAPI) SavingsPurchase(ctx context.Context, ccy string, amt float64) error {
	f.purchases = append(f.purchases, amt)
	return f.purchaseErr
}
func (f *fakeAPI) SavingsBalance(ctx context.Context) (json.RawMessage, error) {
	return json.RawMessage(fmt.Sprintf(`[{"ccy":"USDT","amt":"%v","pendingAmt":"0"}]`, f.balance)), nil
}
func (f *fakeAPI) AssetTransfer(ctx context.Context, ccy string, amt float64, from, to string) (string, error) {
	f.transfers = append(f.transfers, fmt.Sprintf("%s->%s:%.4f", from, to, amt))
	return "777", f.transferErr
}
func (f *fakeAPI) AssetTransferState(ctx context.Context, ccy, clTID string) (string, error) {
	f.stateQueries = append(f.stateQueries, 1)
	if f.stateErr != nil {
		return "", f.stateErr
	}
	return f.transferState, nil
}

// setupW 装配 worker（迁移+键播种+fake 注入）。
func setupW(t *testing.T, api VenueAPI, keys map[string]string) (context.Context, *Worker, *fakeAPI) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for k, v := range keys {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	fake := &fakeAPI{balance: 42}
	w := NewWorker(pool, cfg)
	w.venue = "tstearn" // venue 键域隔离（防跨包共表互踩；ccy 闭集守恒须真 USDT）
	w.rateFn = func(ctx context.Context) (okxexec.RateSummary, error) {
		return okxexec.RateSummary{AvgRate: 0.035}, nil
	}
	if api != nil {
		fake = api.(*fakeAPI)
		w.AttachVenue(fake)
	}
	return ctx, w, fake
}

// cashflowOf 查 earn 移动留痕行（kind→amount 列表）。
func cashflowOf(t *testing.T, ctx context.Context, w *Worker, kind string) []float64 {
	t.Helper()
	rows, err := w.pool.Query(ctx,
		"SELECT amount FROM fin_cashflow WHERE venue=$1 AND ccy=$2 AND kind=$3", w.venue, w.ccy, kind)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		out = append(out, v)
	}
	return out
}

// TestRedeemAmountNotionalPlusBuffer 赎回金额=notional×(1+buffer bps) 可核；成=earn_in
// 留痕（decision reason=transfer 走闸不被敞口判拦+cashflow source 双列）+停泊快照刷。
func TestRedeemAmountNotionalPlusBuffer(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{balance: 42}, map[string]string{"earn.redeem_buffer_bps": "50"})
	defer w.pool.Exec(ctx, "DELETE FROM fin_cashflow WHERE venue=$1 AND kind IN ('earn_in','earn_out')", w.venue)
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	defer w.pool.Exec(ctx, "DELETE FROM fin_venue_account_snapshot WHERE venue_type='earn'")
	if err := w.RedeemToTrading(ctx, 100); err != nil {
		t.Fatalf("赎回应成: %v", err)
	}
	if len(fake.redeems) != 1 || fmt.Sprintf("%.4f", fake.redeems[0]) != "100.5000" {
		t.Fatalf("redeem=%v（应 100×(1+50bps)=100.5）", fake.redeems)
	}
	want := []string{"6->18:100.5000"}
	if fmt.Sprint(fake.transfers) != fmt.Sprint(want) {
		t.Fatalf("transfers=%v（应 funding→trading %v）", fake.transfers, want)
	}
	cf := cashflowOf(t, ctx, w, "earn_in")
	if len(cf) != 1 || fmt.Sprintf("%.4f", cf[0]) != "100.5000" {
		t.Fatalf("earn_in 留痕=%v", cf)
	}
	var reason string
	var srcRef int64
	var srcTable string
	if err := w.pool.QueryRow(ctx, `SELECT d.reason,c.source_ref,c.source_table FROM fin_cashflow c
		JOIN fin_risk_decision d ON d.id=c.source_ref WHERE c.kind='earn_in' AND c.source_table='fin_risk_decision'`).
		Scan(&reason, &srcRef, &srcTable); err != nil || reason != "transfer" {
		t.Fatalf("transfer 留痕链 reason=%s err=%v", reason, err)
	}
	var bal float64
	if err := w.pool.QueryRow(ctx, `SELECT equity FROM fin_venue_account_snapshot
		WHERE venue=$1 AND venue_type='earn' AND ccy=$2`, w.venue, w.ccy).Scan(&bal); err != nil || bal != 42 {
		t.Fatalf("停泊快照 bal=%v err=%v（应=balance 42）", bal, err)
	}
}

// TestRedeemFailClosed 赎回败=上抛不开仓（钱未动：无划转无留痕）。
func TestRedeemFailClosed(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{redeemErr: errors.New("insufficient")}, map[string]string{"earn.redeem_buffer_bps": "50"})
	if err := w.RedeemToTrading(ctx, 100); err == nil {
		t.Fatal("redeem 败应 fail-closed")
	}
	if len(fake.transfers) != 0 || len(fake.purchases) != 0 {
		t.Fatalf("钱不应动: transfers=%v purchases=%v", fake.transfers, fake.purchases)
	}
	if n := cashflowOf(t, ctx, w, "earn_in"); len(n) != 0 {
		t.Fatalf("败径不应留痕: %v", n)
	}
}

// TestRedeemTransferFailCompensates 划转败（state=failed 定真）→补偿回停+上抛。
func TestRedeemTransferFailCompensates(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{transferErr: errors.New("http down"), transferState: "failed"},
		map[string]string{"earn.redeem_buffer_bps": "50"})
	defer w.pool.Exec(ctx, "DELETE FROM fin_cashflow WHERE venue=$1 AND kind IN ('earn_in','earn_out')", w.venue)
	if err := w.RedeemToTrading(ctx, 100); err == nil {
		t.Fatal("划转败应 fail-closed")
	}
	if len(fake.purchases) != 1 || fmt.Sprintf("%.4f", fake.purchases[0]) != "100.5000" {
		t.Fatalf("确认未动应补偿回停: purchases=%v", fake.purchases)
	}
}

// TestRedeemTransferAmbiguousNoCompensation 在途/态不可知=禁补偿禁重发（双动险）。
func TestRedeemTransferAmbiguousNoCompensation(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{transferErr: errors.New("timeout"), stateErr: errors.New("state down")},
		map[string]string{"earn.redeem_buffer_bps": "50"})
	if err := w.RedeemToTrading(ctx, 100); err == nil {
		t.Fatal("态不可知应 fail-closed")
	}
	if len(fake.purchases) != 0 {
		t.Fatalf("在途禁补偿: purchases=%v", fake.purchases)
	}
	if w.rej.Snapshot()["earn/transfer_ambiguous"] == 0 {
		t.Fatal("transfer_ambiguous 计数门未计")
	}
}

// TestRedeemTransportFailActuallyMoved 传输败实成（state=success 定真）→不留补偿照常留痕。
func TestRedeemTransportFailActuallyMoved(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{transferErr: errors.New("http reset"), transferState: "success"},
		map[string]string{"earn.redeem_buffer_bps": "50"})
	defer w.pool.Exec(ctx, "DELETE FROM fin_cashflow WHERE venue=$1 AND kind IN ('earn_in','earn_out')", w.venue)
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	if err := w.RedeemToTrading(ctx, 100); err != nil {
		t.Fatalf("定真态=成应放行: %v", err)
	}
	if len(fake.stateQueries) != 1 {
		t.Fatalf("应查真态一次: %v", fake.stateQueries)
	}
	if len(fake.purchases) != 0 {
		t.Fatal("实成不应补偿")
	}
	if cf := cashflowOf(t, ctx, w, "earn_in"); len(cf) != 1 {
		t.Fatalf("earn_in 留痕缺: %v", cf)
	}
}

// TestDepositSweep 闲余>阈值（热启）→摆渡+申购+earn_out 留痕+停泊快照。
func TestDepositSweep(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{balance: 42}, map[string]string{"earn.sweep_min_usd": "100"})
	defer w.pool.Exec(ctx, "DELETE FROM fin_cashflow WHERE venue=$1 AND kind IN ('earn_in','earn_out')", w.venue)
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	defer w.pool.Exec(ctx, "DELETE FROM fin_venue_account_snapshot WHERE venue='tstearn'")
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_venue_account_snapshot(venue,venue_type,ccy,equity,avail,src)
		VALUES('tstearn','unified','USDT',150,150,'test')`); err != nil {
		t.Fatal(err)
	}
	if err := w.scan(ctx); err != nil {
		t.Fatalf("scan: %v", err)
	}
	want := []string{"18->6:150.0000"}
	if fmt.Sprint(fake.transfers) != fmt.Sprint(want) {
		t.Fatalf("transfers=%v（应 trading→funding %v）", fake.transfers, want)
	}
	if len(fake.purchases) != 1 || fake.purchases[0] != 150 {
		t.Fatalf("申购=%v", fake.purchases)
	}
	if cf := cashflowOf(t, ctx, w, "earn_out"); len(cf) != 1 || fmt.Sprintf("%.4f", cf[0]) != "-150.0000" {
		t.Fatalf("earn_out 留痕=%v（应 -150）", cf)
	}
	var bal float64
	if err := w.pool.QueryRow(ctx, `SELECT equity FROM fin_venue_account_snapshot
		WHERE venue=$1 AND venue_type='earn' AND ccy=$2`, w.venue, w.ccy).Scan(&bal); err != nil || bal != 42 {
		t.Fatalf("停泊快照=%v err=%v", bal, err)
	}
}

// TestSweepThresholdHot 阈值键热启：150<200 不动；键改 80 后同快照触发。
func TestSweepThresholdHot(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{"earn.sweep_min_usd": "200"})
	defer w.pool.Exec(ctx, "DELETE FROM fin_cashflow WHERE venue=$1 AND kind IN ('earn_in','earn_out')", w.venue)
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	defer w.pool.Exec(ctx, "DELETE FROM fin_venue_account_snapshot WHERE venue='tstearn'")
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_venue_account_snapshot(venue,venue_type,ccy,equity,avail,src)
		VALUES('tstearn','unified','USDT',150,150,'test')`); err != nil {
		t.Fatal(err)
	}
	if err := w.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 0 {
		t.Fatalf("阈下不应动钱: %v", fake.transfers)
	}
	if _, err := w.pool.Exec(ctx, "UPDATE app_config SET value='\"80\"' WHERE key='earn.sweep_min_usd'"); err != nil {
		t.Fatal(err)
	}
	if err := w.cfg.Load(ctx); err != nil { // 模拟 Watch 热载（按拍读取=热启语义）
		t.Fatal(err)
	}
	if err := w.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 1 {
		t.Fatalf("热启后应触发: %v", fake.transfers)
	}
}

// TestSweepStaleSnapshotSkips 陈快照 fail-closed 不动钱+计数门。
func TestSweepStaleSnapshotSkips(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{"earn.sweep_min_usd": "100"})
	defer w.pool.Exec(ctx, "DELETE FROM fin_venue_account_snapshot WHERE venue='tstearn'")
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_venue_account_snapshot(venue,venue_type,ccy,equity,avail,src,updated_at)
		VALUES('tstearn','unified','USDT',150,150,'test',now()-interval '10 minutes')`); err != nil {
		t.Fatal(err)
	}
	if err := w.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 0 {
		t.Fatalf("陈快照不应动钱: %v", fake.transfers)
	}
	if w.rej.Snapshot()["earn/snapshot_stale"] == 0 {
		t.Fatal("snapshot_stale 计数门未计")
	}
}

// TestSweepInflightSkips 执行在途让位（防饿腿）+计数门。
func TestSweepInflightSkips(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{"earn.sweep_min_usd": "100"})
	defer w.pool.Exec(ctx, "DELETE FROM fin_cashflow WHERE venue=$1 AND kind IN ('earn_in','earn_out')", w.venue)
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	defer w.pool.Exec(ctx, "DELETE FROM fin_venue_account_snapshot WHERE venue='tstearn'")
	defer w.pool.Exec(ctx, "DELETE FROM fin_execution WHERE inst_id='TEST-EARN'")
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_venue_account_snapshot(venue,venue_type,ccy,equity,avail,src)
		VALUES('tstearn','unified','USDT',150,150,'test')`); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES(1,1,NULL,'TEST-EARN',100,'open')`); err != nil {
		t.Fatal(err)
	}
	if err := w.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 0 {
		t.Fatalf("在途应让位: %v", fake.transfers)
	}
	if w.rej.Snapshot()["earn/sweep_skip_inflight"] == 0 {
		t.Fatal("sweep_skip_inflight 计数门未计")
	}
}

// TestRatePollFactRow 现率采样→md_earn_rate 事实行；采样败=计数不入假值。
func TestRatePollFactRow(t *testing.T) {
	ctx, w, _ := setupW(t, &fakeAPI{}, map[string]string{"earn.sweep_min_usd": "100"})
	defer w.pool.Exec(ctx, "DELETE FROM md_earn_rate WHERE venue=$1 AND ccy=$2", w.venue, w.ccy)
	if err := w.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var rate float64
	var ts time.Time
	if err := w.pool.QueryRow(ctx, `SELECT rate,ts FROM md_earn_rate WHERE venue=$1 AND ccy=$2
		ORDER BY ts DESC LIMIT 1`, w.venue, w.ccy).Scan(&rate, &ts); err != nil || rate != 0.035 {
		t.Fatalf("事实行 rate=%v err=%v（应 0.035）", rate, err)
	}
	w.rateFn = func(ctx context.Context) (okxexec.RateSummary, error) {
		return okxexec.RateSummary{}, errors.New("net down")
	}
	if err := w.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := w.pool.QueryRow(ctx, "SELECT count(*) FROM md_earn_rate WHERE venue=$1 AND ccy=$2", w.venue, w.ccy).Scan(&n); err != nil || n != 1 {
		t.Fatalf("采样败不应入行: n=%d", n)
	}
	if w.rej.Snapshot()["earn/rate_fetch_failed"] == 0 {
		t.Fatal("rate_fetch_failed 计数门未计")
	}
}
