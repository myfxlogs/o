package ledger

import (
	"context"
	"fmt"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
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

// T-0087 锚②：划转/执行在途窗 debounce——30s 内 earn_in/earn_out 行（票面
// 触发面）、transfer decision 行（T-0060 停写后实况触发面）、leg/execution
// 行（列名勘误锚）任一在=跳过不判；全拨窗外=恢复判定。
func TestReconcileDebounceWindows(t *testing.T) {
	ctx, r := setup(t)
	var ven map[[2]string]float64
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, error) { return ven, nil })
	if _, err := Post(ctx, r.pool, Entry{Kind: "deposit", Venue: "okx", Ccy: "USDT",
		Amount: 100, SourceRef: 1, SourceTable: "t"}); err != nil {
		t.Fatal(err)
	}
	ven = map[[2]string]float64{{"okx", "USDT"}: 200} // 恒差 100 的判定态
	snap := func() int64 { return r.rej.Snapshot()["ledger/diff_alert"] }
	expectSkip := func(stage string) {
		before := snap()
		if err := r.reconcile(ctx); err != nil {
			t.Fatal(err)
		}
		if snap() != before {
			t.Fatalf("%s: 窗内应跳过不判", stage)
		}
	}
	expectJudge := func(stage string) {
		before := snap()
		if err := r.reconcile(ctx); err != nil {
			t.Fatal(err)
		}
		if snap() == before {
			t.Fatalf("%s: 窗外应恢复判定（差 100 超容差必告警）", stage)
		}
	}
	// ① 票面触发面：earn_in 行窗内→跳过；拨旧→判定。
	if _, err := Post(ctx, r.pool, Entry{Kind: "earn_in", Venue: "okx", Ccy: "USDT",
		Amount: 50, SourceRef: 9, SourceTable: "fin_risk_decision"}); err != nil {
		t.Fatal(err)
	}
	expectSkip("earn 行窗内")
	if _, err := r.pool.Exec(ctx, `UPDATE fin_cashflow SET ts=now()-interval '1 hour' WHERE kind='earn_in'`); err != nil {
		t.Fatal(err)
	}
	expectJudge("earn 行窗外")
	// ② 实况触发面：transfer decision 行窗内→跳过；拨旧→判定。
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{"kind":"transfer"}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	expectSkip("transfer 行窗内")
	if _, err := r.pool.Exec(ctx, `UPDATE fin_risk_decision SET ts=now()-interval '1 hour' WHERE reason='transfer'`); err != nil {
		t.Fatal(err)
	}
	// ③ 执行面：leg/execution 窗内→跳过（旧列名 created_at/opened_at 恒错被
	// 吞=debounce 假在——此段对列名勘误为直接锚）；全拨旧→判定。
	var xid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('X',100,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,'cl-t87','okx','buy','X-USDT',1,'filled')`, xid); err != nil {
		t.Fatal(err)
	}
	expectSkip("leg/execution 窗内")
	if _, err := r.pool.Exec(ctx, `UPDATE fin_order_leg SET ts=now()-interval '1 hour'`); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `UPDATE fin_execution SET created_at=now()-interval '1 hour'`); err != nil {
		t.Fatal(err)
	}
	expectJudge("全窗外")
}

// T-0087 锚①（账本面）：对拍净值=Σcashflow+mtm 偏移——现货资产市值项经 mtm
// 读件入净（读件构成锚在 cmd/core）；偏移读败=0 不加不减，照判不让位。
func TestReconcileMTMOffset(t *testing.T) {
	ctx, r := setup(t)
	var ven map[[2]string]float64
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, error) { return ven, nil })
	if _, err := Post(ctx, r.pool, Entry{Kind: "deposit", Venue: "okx", Ccy: "USDT",
		Amount: 100, SourceRef: 1, SourceTable: "t"}); err != nil {
		t.Fatal(err)
	}
	ven = map[[2]string]float64{{"okx", "USDT"}: 104.5}
	r.WithMTM(func(ctx context.Context) (map[[2]string]float64, error) {
		return map[[2]string]float64{{"okx", "USDT"}: 4.5}, nil
	})
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if n := r.rej.Snapshot()["ledger/diff_alert"]; n != 0 {
		t.Fatalf("mtm 偏移应入净（100+4.5=104.5 平）: diff_alert=%d", n)
	}
	r.WithMTM(func(ctx context.Context) (map[[2]string]float64, error) {
		return nil, fmt.Errorf("mtm 读败")
	})
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["ledger/diff_alert"] == 0 {
		t.Fatal("mtm 读败应按 0 净额判定（差 4.5 必告警）")
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

// T-0095 v2 锚：基金的钱集=USDT∪**open** 腿基币（ETH-USDT→ETH，-SWAP 后缀
// 同剥；未成交/已平腿基币与「有 cashflow 无腿」的币都不入集）——钱在哪过滤
// 与对拍白名单同源消费唯一真源；mt5/USD 恒在册。
func TestFundCcyOpenLegBase(t *testing.T) {
	ctx, r := setup(t)
	var xid, pid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('ETH-USDT-SWAP',100,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd)
		VALUES($1,'ETH-USDT-SWAP',100) RETURNING id`, xid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	var cxid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('DOGE-USDT',50,'closed') RETURNING id`).Scan(&cxid); err != nil {
		t.Fatal(err)
	}
	var cpid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,closed_at)
		VALUES($1,'DOGE-USDT',50,now()) RETURNING id`, cxid).Scan(&cpid); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES ($1,1,'cl-a','okx','buy','ETH-USDT',1,'filled'),
		       ($1,2,'cl-b','okx','sell','ETH-USDT-SWAP',1,'filled'),
		       ($1,3,'cl-c','okx','buy','DOGE-USDT',1,'placed'),
		       ($2,1,'cl-d','okx','sell','DOGE-USDT',1,'filled')`, pid, cpid); err != nil {
		t.Fatal(err)
	}
	// （cashflow 表 ccy 闭集 USDT/USD——「有账无腿」形 schema 层不可存在，
	// v2 集合判据=腿基币自证完备。）
	set, err := FundCcy(ctx, r.pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range [][2]string{{"okx", "USDT"}, {"okx", "ETH"}, {"mt5", "USD"}} {
		if !set[k] {
			t.Fatalf("基金钱集缺 %v", k)
		}
	}
	for _, k := range [][2]string{{"okx", "DOGE"}} {
		if set[k] {
			t.Fatalf("%v 不应入集（腿平/无腿=托管尘）", k)
		}
	}
}

// T-0100 ③锚：非本位键按币量建键——停泊残币 venue=币数 ledger=ParkedAmt
// 台账币数同径比对；台账内漂移静默、超漂真差照报（错配爆雷根除）。
func TestReconcileParkedUnits(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx, "DELETE FROM fin_known_residual"); err != nil {
		t.Fatal(err)
	}
	var ven map[[2]string]float64
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, error) { return ven, nil })
	if _, err := Post(ctx, r.pool, Entry{Kind: "deposit", Venue: "okx", Ccy: "USDT",
		Amount: 100, SourceRef: 1, SourceTable: "t"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{"kind":"transfer","op":"residual_park","venue":"okx","ccy":"DOGE","amt":100}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	// 停泊 100 DOGE 台账=venue 100.2（利息微漂<容差）→静默；同时 USDT 平。
	ven = map[[2]string]float64{{"okx", "USDT"}: 100, {"okx", "DOGE"}: 100.2}
	r.rej = rejects.New("ledger", "diff_alert", "missing_param", "bal_missing", "write_err")
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["ledger/diff_alert"] != 0 {
		t.Fatal("停泊台账币量同径——微漂误报")
	}
	// venue 少 5 币单位→真差照报（按币量差非 USD——台账 100 vs 95）。
	ven = map[[2]string]float64{{"okx", "USDT"}: 100, {"okx", "DOGE"}: 95}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["ledger/diff_alert"] == 0 {
		t.Fatal("停泊币量真差漏报")
	}
}
