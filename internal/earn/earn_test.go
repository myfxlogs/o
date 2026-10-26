package earn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

// fakeAPI 可控 venue 停泊面（行为开关+调用记录——钱动锚全走此件）。
type fakeAPI struct {
	redeemErr, purchaseErr, transferErr, stateErr error
	transferReject                                *okxexec.APIError // venue 业务终拒（T-0059：txFailed 直返禁仲裁）
	transferState                                 string            // 非""=传输败后按 clTId 查得此态
	balance                                       float64
	balOverride                                   string // 非""=SavingsBalance 原文覆写（⑦锚：pendingAmt 在途形）
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
	if f.balOverride != "" {
		return json.RawMessage(f.balOverride), nil
	}
	return json.RawMessage(fmt.Sprintf(`[{"ccy":"USDT","amt":"%v"}]`, f.balance)), nil
}
func (f *fakeAPI) AssetTransfer(ctx context.Context, ccy string, amt float64, from, to string) (string, string, error) {
	f.transfers = append(f.transfers, fmt.Sprintf("%s->%s:%.4f", from, to, amt))
	if f.transferReject != nil {
		return "phT777", "", f.transferReject
	}
	return "phT777", "tr123", f.transferErr
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
	// T-0167 分链：本包测试面跨域（研究域表+判据键）——两链并集态（md 先 fin 后
	// =历史重放方向：md 链瞬态超表层先落，fin 链状态化/守卫后到）。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
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
	if cf := cashflowOf(t, ctx, w, "earn_in"); len(cf) != 0 {
		t.Fatalf("T-0060：内部划转禁落 cashflow，行=%v", cf)
	}
	var reason string
	if err := w.pool.QueryRow(ctx, `SELECT reason FROM fin_risk_decision
		WHERE intent->>'kind'='transfer' AND intent->>'op'='earn_redeem' ORDER BY id DESC LIMIT 1`).
		Scan(&reason); err != nil || reason != "transfer" {
		t.Fatalf("transfer decision 留痕缺 reason=%s err=%v", reason, err)
	}
	var bal float64
	if err := w.pool.QueryRow(ctx, `SELECT (data->>'earn_bal')::float FROM md_account_events
		WHERE venue=$1 AND channel='earn_parking' AND ccy=$2 ORDER BY ts DESC LIMIT 1`, w.venue, w.ccy).Scan(&bal); err != nil || bal != 42 {
		t.Fatalf("停泊事件行 bal=%v err=%v（应=balance 42）", bal, err)
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
	if cf := cashflowOf(t, ctx, w, "earn_in"); len(cf) != 0 {
		t.Fatalf("T-0060：内部划转禁落 cashflow，行=%v", cf)
	}
}

// TestDepositSweep 闲余>阈值（热启）→摆渡+申购+earn_out 留痕+停泊快照。
func TestDepositSweep(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{balance: 42}, map[string]string{"earn.sweep_min_usd": "100"})
	defer w.pool.Exec(ctx, "DELETE FROM fin_cashflow WHERE venue=$1 AND kind IN ('earn_in','earn_out')", w.venue)
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	w.AttachIdle(func(ctx context.Context) (float64, error) { return 150, nil })
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
	if cf := cashflowOf(t, ctx, w, "earn_out"); len(cf) != 0 {
		t.Fatalf("T-0060：内部划转禁落 cashflow，行=%v", cf)
	}
	var op string
	if err := w.pool.QueryRow(ctx, `SELECT intent->>'op' FROM fin_risk_decision
		WHERE intent->>'kind'='transfer' ORDER BY id DESC LIMIT 1`).Scan(&op); err != nil || op != "earn_deposit" {
		t.Fatalf("decision 留痕 op=%s err=%v", op, err)
	}
	var bal float64
	if err := w.pool.QueryRow(ctx, `SELECT (data->>'earn_bal')::float FROM md_account_events
		WHERE venue=$1 AND channel='earn_parking' AND ccy=$2 ORDER BY ts DESC LIMIT 1`, w.venue, w.ccy).Scan(&bal); err != nil || bal != 42 {
		t.Fatalf("停泊事件行=%v err=%v", bal, err)
	}
}

// TestSweepThresholdHot 阈值键热启：150<200 不动；键改 80 后同快照触发。
func TestSweepThresholdHot(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{"earn.sweep_min_usd": "200"})
	defer w.pool.Exec(ctx, "DELETE FROM fin_cashflow WHERE venue=$1 AND kind IN ('earn_in','earn_out')", w.venue)
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	w.AttachIdle(func(ctx context.Context) (float64, error) { return 150, nil })
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

// TestSweepStaleSnapshotSkips 现读败 fail-closed 不动钱+计数门。
func TestSweepStaleSnapshotSkips(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{"earn.sweep_min_usd": "100"})
	w.AttachIdle(func(ctx context.Context) (float64, error) { return 0, fmt.Errorf("读败假件") })
	if err := w.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 0 {
		t.Fatalf("读败不应动钱: %v", fake.transfers)
	}
	if w.rej.Snapshot()["earn/idle_missing"] == 0 {
		t.Fatal("snapshot_stale 计数门未计")
	}
}

// TestSweepInflightSkips 执行在途让位（防饿腿）+计数门。
func TestSweepInflightSkips(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{"earn.sweep_min_usd": "100"})
	defer w.pool.Exec(ctx, "DELETE FROM fin_cashflow WHERE venue=$1 AND kind IN ('earn_in','earn_out')", w.venue)
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	defer w.pool.Exec(ctx, "DELETE FROM fin_execution WHERE inst_id='TEST-EARN'")
	w.AttachIdle(func(ctx context.Context) (float64, error) { return 150, nil })
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

// T-0046 ⑦锚：amt 已含在途 pendingAmt（loanAmt+pendingAmt=amt 实证）——
// 停泊快照取 amt 单字段，amt+pendingAmt 求和=双计虚高（2129≠2229.4）。
func TestParkingAmtNoDoubleCount(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{"earn.park_ccy": "USDT"})
	_ = ctx
	fake.balance = 2129.00
	// savings 响应含在途 pendingAmt=100.40（amt 已含——canon 实证口径）。
	fake.balOverride = `[{"ccy":"USDT","amt":"2129.00","pendingAmt":"100.40"}]`
	if err := w.refreshParkingSnapshot(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	var bal float64
	if err := w.pool.QueryRow(context.Background(), `SELECT (data->>'earn_bal')::float FROM md_account_events
		WHERE venue=$1 AND channel='earn_parking' AND ccy=$2 ORDER BY ts DESC LIMIT 1`, w.venue, w.ccy).Scan(&bal); err != nil || bal != 2129.00 {
		t.Fatalf("停泊快照 bal=%v err=%v（应=amt 2129.00 非 2229.40 双计）", bal, err)
	}
}

// T-0059 锚②：58xxx 业务终拒→txFailed 直返+零仲裁调用（58350 终拒误送仲裁→
// 58129「态未定」ALERT→5s 同参重试死锁环实证的反面）。
func TestTransferTerminalRejectNoArbitration(t *testing.T) {
	fake := &fakeAPI{transferReject: &okxexec.APIError{Code: "58350", Msg: "Insufficient balance"}}
	ctx, w, _ := setupW(t, fake, nil)
	out, err := w.transfer(ctx, "USDT", 75.92713388, "18", "6")
	if out != txFailed {
		t.Fatalf("终拒应 txFailed: out=%v err=%v", out, err)
	}
	if len(fake.stateQueries) != 0 {
		t.Fatalf("终拒禁送仲裁: stateQueries=%d", len(fake.stateQueries))
	}
	// 摆渡径消费面：depositIdle 终拒→moved=false（退避钩挂上游 sweepIdle）。
	if moved := w.depositIdle(ctx, 75.92713388); moved {
		t.Fatal("终拒后 moved 应 false")
	}
}

// T-0059 锚②b：传输败→仲裁查询自身业务拒（*APIError）=无单在案→txFailed
// （58129/51000 双坑的正解——非「态未定」死循环）。
func TestTransferArbitrationQueryRejectIsFailed(t *testing.T) {
	fake := &fakeAPI{transferErr: fmt.Errorf("conn reset"), stateErr: &okxexec.APIError{Code: "58129", Msg: "clientId does not match with 'type'"}}
	ctx, w, _ := setupW(t, fake, nil)
	out, _ := w.transfer(ctx, "USDT", 10, "18", "6")
	if out != txFailed || len(fake.stateQueries) != 1 {
		t.Fatalf("查询业务拒应 txFailed+一次仲裁: out=%v queries=%d", out, len(fake.stateQueries))
	}
}

// T-0149 ②锚：funding 户闲资超阈→直购 Earn 零摆渡（无 transfer 帧）；歧态停手同退避。
func TestFundingIdleDirectPurchase(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{"earn.sweep_min_usd": "100"})
	w.AttachIdleFunding(func(context.Context) (float64, error) { return 500, nil }) // funding 500>thr
	w.AttachIdle(func(context.Context) (float64, error) { return 0, nil })          // 交易户零——不触发三跳
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.purchases) != 1 || fake.purchases[0] != 500 {
		t.Fatalf("funding 直购应 500 一笔: %v", fake.purchases)
	}
	if len(fake.transfers) != 0 {
		t.Fatalf("funding 段应零摆渡: %v", fake.transfers)
	}
}

// T-0149 返修①锚：funding 直购连败两拍→第二拍 sweep_backoff 不重发。
func TestFundingIdleBackoff(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{purchaseErr: fmt.Errorf("58350")}, map[string]string{"earn.sweep_min_usd": "100"})
	w.AttachIdleFunding(func(context.Context) (float64, error) { return 500, nil })
	w.AttachIdle(func(context.Context) (float64, error) { return 0, nil })
	_ = w.sweepIdle(ctx) // 一拍败
	n1 := len(fake.purchases)
	_ = w.sweepIdle(ctx) // 二拍=退避闸应拒重发
	if len(fake.purchases) != n1 {
		t.Fatalf("连败两拍第二拍应 sweep_backoff 不重发: %d→%d", n1, len(fake.purchases))
	}
}
