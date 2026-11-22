package main

import (
	"context"
	"testing"
	"time"

	"o/internal/ledger"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

// 工单#22919 锚①：settled 单收敛 open exec 仓——closed_at+settled_* 落位+
// venue_ord_id 回填+exec 态 closed+结算凭据+实拍行，五件同拍。
func TestDcdSettleConverge(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	var eid, pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('BTC-USDT-261204-66000-P',50,'open') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd)
		VALUES($1,'BTC-USDT-261204-66000-P',50) RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	od := okxexec.DcdOrder{OrdID: "11594773", ProductID: "BTC-USDT-261204-66000-P",
		State: "settled", NotionalCcy: "USDT", NotionalSz: "50",
		SettleCcy: "USDT", SettleSz: "50.5", SettlePx: "70100",
		YieldCcy: "USDT", YieldSz: "0.5", SettleTime: "1764979200000", ExpTime: "1764979200000"}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{od})
	var settledAmt float64
	var ccy string
	var closedAt, settledAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT closed_at,settled_at,settled_amount,settled_ccy
		FROM fin_position WHERE id=$1`, pid).Scan(&closedAt, &settledAt, &settledAmt, &ccy); err != nil {
		t.Fatal(err)
	}
	if closedAt == nil || settledAt == nil {
		t.Fatalf("仓未收敛: closed_at=%v settled_at=%v", closedAt, settledAt)
	}
	if settledAmt != 50.5 || ccy != "USDT" {
		t.Fatalf("settled_* 失真: amt=%v ccy=%s", settledAmt, ccy)
	}
	var estate, ordID string
	if err := pool.QueryRow(ctx, `SELECT state,COALESCE(venue_ord_id,'') FROM fin_execution
		WHERE id=$1`, eid).Scan(&estate, &ordID); err != nil {
		t.Fatal(err)
	}
	if estate != "closed" || ordID != "11594773" {
		t.Fatalf("exec 收敛失真: state=%s ordId=%s", estate, ordID)
	}
	var cred bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fin_risk_decision
		WHERE intent->>'op'='dcd_settle' AND intent->>'ordId'='11594773')`).Scan(&cred); err != nil || !cred {
		t.Fatalf("结算凭据缺席: %v", err)
	}
	var ev bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM md_account_events
		WHERE channel='dcd_settle' AND inst_id='11594773' AND data->>'settlePx'='70100')`).Scan(&ev); err != nil || !ev {
		t.Fatalf("settle 实拍行缺席: %v", err)
	}
	// 幂等：重跑不炸不双。
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{od})
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_risk_decision
		WHERE intent->>'op'='dcd_settle'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("凭据不幂等: n=%d", n)
	}
}

// O-33362 锚②：live 名义在持单三覆盖源全缺→自动补挂 dcd_baseline 凭据
// +dcd_out counted 申购腿同事务；重放幂等零重；孤儿升件退役（写败才升
// dcd_baseline_fail）。
func TestDcdLiveAutoBaseline(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	od := okxexec.DcdOrder{OrdID: "920001", ProductID: "BTC-USDT-920001-70000-C",
		State: "live", NotionalCcy: "USDT", NotionalSz: "50",
		CTime: "1767225600000"}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{od})
	var credID int64
	var ccy string
	var amt float64
	if err := pool.QueryRow(ctx, `SELECT id,intent->>'ccy',(intent->>'amt')::float8
		FROM fin_risk_decision
		WHERE intent->>'op'='dcd_baseline' AND intent->>'ordId'='920001'`).Scan(&credID, &ccy, &amt); err != nil {
		t.Fatalf("基线凭据缺: %v", err)
	}
	if ccy != "USDT" || amt != 50 {
		t.Fatalf("基线凭据失真: ccy=%s amt=%v", ccy, amt)
	}
	var legN int
	var legAmt float64
	var legKind, legCcy, legTable string
	var legRef int64
	if err := pool.QueryRow(ctx, `SELECT count(*),min(kind),min(ccy),min(amount),min(source_ref),min(source_table)
		FROM fin_cashflow WHERE kind IN ('dcd_out','dcd_in')`).Scan(&legN, &legKind, &legCcy, &legAmt, &legRef, &legTable); err != nil {
		t.Fatal(err)
	}
	if legN != 1 || legKind != "dcd_out" || legCcy != "USDT" || legAmt != -50 ||
		legRef != credID || legTable != "fin_risk_decision" {
		t.Fatalf("申购腿失真: n=%d kind=%s ccy=%s amt=%v ref=%d tbl=%s",
			legN, legKind, legCcy, legAmt, legRef, legTable)
	}
	var legClass string
	if err := pool.QueryRow(ctx, `SELECT class FROM fin_cashflow WHERE kind='dcd_out'`).Scan(&legClass); err != nil || legClass != "transfer" {
		t.Fatalf("腿 class 未落册: %q err=%v", legClass, err)
	}
	// 孤儿升件退役——自动补挂成功不产升件。
	var esc int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='920001'`).Scan(&esc); err != nil || esc != 0 {
		t.Fatalf("补挂成功误升件: n=%d", esc)
	}
	// 幂等：重放零增。
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{od})
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_risk_decision
		WHERE intent->>'op'='dcd_baseline' AND intent->>'ordId'='920001'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("凭据不幂等: n=%d", n)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow WHERE kind='dcd_out'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("腿不幂等: n=%d", n)
	}
}

// O-33362 锚③：BTC 名义 live 单——基线凭据照落（venue 名义币量对拍键），
// 现金腿零落（fin_cashflow.ccy 闭集物理拒杂币——不猜汇率不硬写）。
func TestDcdLiveBtcBaselineNoLeg(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	od := okxexec.DcdOrder{OrdID: "920002", ProductID: "BTC-USDT-920002-95000-C",
		State: "live", NotionalCcy: "BTC", NotionalSz: "0.0207"}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{od})
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_risk_decision
		WHERE intent->>'op'='dcd_baseline' AND intent->>'ordId'='920002'
		AND intent->>'ccy'='BTC'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("BTC 基线缺: n=%d", n)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow
		WHERE kind IN ('dcd_out','dcd_in')`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("BTC 名义误落现金腿: n=%d", n)
	}
}

// O-33362 锚④：覆盖源逐源核——exec 活仓/归属域册/既有基线凭据任一在
// →零补挂；orphaned 仓（脱钩探针态非健康覆盖）不阻补挂。
func TestDcdLiveCoverageSources(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	baseN := func(ord string) int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_risk_decision
			WHERE intent->>'op'='dcd_baseline' AND intent->>'ordId'=$1`, ord).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	// ① exec 活仓覆盖（open 仓挂 exec）→零补挂。
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('BTC-USDT-920003-1-C',10,'open') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd)
		VALUES($1,'BTC-USDT-920003-1-C',10)`, eid); err != nil {
		t.Fatal(err)
	}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{{OrdID: "920003",
		ProductID: "BTC-USDT-920003-1-C", State: "live", NotionalCcy: "USDT", NotionalSz: "10"}})
	if n := baseN("920003"); n != 0 {
		t.Fatalf("exec 活仓覆盖误补挂: n=%d", n)
	}
	// ② orphaned 仓不算覆盖（脱钩探针态——无腿无现金贡献）→照补挂。
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('BTC-USDT-920004-1-C',10,'closed') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,state)
		VALUES($1,'BTC-USDT-920004-1-C',10,'orphaned')`, eid); err != nil {
		t.Fatal(err)
	}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{{OrdID: "920004",
		ProductID: "BTC-USDT-920004-1-C", State: "live", NotionalCcy: "USDT", NotionalSz: "10"}})
	if n := baseN("920004"); n != 1 {
		t.Fatalf("orphaned 仓误判覆盖漏补挂: n=%d", n)
	}
	// ③ 归属域册覆盖（业主域 ordId 锚）→零补挂零腿=业主资产不入策略账。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position_domain(decision_id,domain,inst_id,ord_id,note)
		VALUES(987654321,'owner','','920005','test owner domain')`); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM fin_position_domain WHERE ord_id='920005'`)
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{{OrdID: "920005",
		ProductID: "BTC-USDT-920005-1-C", State: "live", NotionalCcy: "USDT", NotionalSz: "10"}})
	if n := baseN("920005"); n != 0 {
		t.Fatalf("域册覆盖误补挂: n=%d", n)
	}
	var legN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow WHERE kind='dcd_out'`).Scan(&legN); err != nil || legN != 1 {
		t.Fatalf("dcd_out 腿计数失真（仅 920004 orphaned 补挂应产一腿）: n=%d", legN)
	}
}

// O-33362 锚⑤：settled 本位单——结算凭据+本金腿 dcd_in+息腿 interest
// 拆分同事务；重放幂等；基线同 ordId 凭据自动退役（dcd_settle 排除面）。
func TestDcdSettleCashLegs(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{"op":"dcd_baseline","ordId":"920006","venue":"okx","ccy":"USDT","amt":10,"kind":"transfer"}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	od := okxexec.DcdOrder{OrdID: "920006", ProductID: "ETH-USDT-920006-2000-P",
		State: "settled", NotionalCcy: "USDT", NotionalSz: "10",
		SettleCcy: "USDT", SettleSz: "10.0260288", SettlePx: "2000",
		YieldCcy: "USDT", YieldSz: "0.0260288",
		SettleTime: "1767225600000", SettledTime: "1767225660000", ExpTime: "1767225600000"}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{od})
	var credID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM fin_risk_decision
		WHERE intent->>'op'='dcd_settle' AND intent->>'ordId'='920006'`).Scan(&credID); err != nil {
		t.Fatalf("结算凭据缺: %v", err)
	}
	rows, err := pool.Query(ctx, `SELECT kind,amount FROM fin_cashflow
		WHERE source_table='fin_risk_decision' AND source_ref=$1 ORDER BY kind`, credID)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]float64{}
	for rows.Next() {
		var k string
		var a float64
		if err := rows.Scan(&k, &a); err != nil {
			t.Fatal(err)
		}
		got[k] = a
	}
	rows.Close()
	if got["dcd_in"] != 10.0 || got["interest"] != 0.0260288 || len(got) != 2 {
		t.Fatalf("结算腿拆分失真: %v", got)
	}
	// 幂等重放零增。
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{od})
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow
		WHERE source_table='fin_risk_decision' AND source_ref=$1`, credID).Scan(&n); err != nil || n != 2 {
		t.Fatalf("结算腿不幂等: n=%d", n)
	}
	// 基线退役：同 ordId dcd_settle 在册→DcdBaselineAmt 不再计。
	base, err := ledger.DcdBaselineAmt(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if base[[2]string{"okx", "USDT"}] != 0 {
		t.Fatalf("已结算基线未退役: %v", base)
	}
}

// O-33362 锚⑥：杂币回款结算单——凭据照落 FundCcy 域注册，现金腿零落。
func TestDcdSettleBtcNoLeg(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	od := okxexec.DcdOrder{OrdID: "920007", ProductID: "BTC-USDT-920007-90000-P",
		State: "settled", NotionalCcy: "BTC", NotionalSz: "0.01",
		SettleCcy: "BTC", SettleSz: "0.0105", SettlePx: "90000",
		YieldCcy: "BTC", YieldSz: "0.0005",
		SettleTime: "1767225600000", SettledTime: "1767225660000"}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{od})
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_risk_decision
		WHERE intent->>'op'='dcd_settle' AND intent->>'ordId'='920007'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("BTC 结算凭据缺: n=%d", n)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow
		WHERE kind IN ('dcd_in','dcd_out','interest')`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("BTC 回款误落现金腿: n=%d", n)
	}
}

// O-33362 验收锚⑧：生产三单簿记态回放——机制上拍对存量单=全幂等零新行
// （覆盖/凭据在册即跳，不重写历史 append-only 事实）。
func TestDcdProductionReplayNoop(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 生产实证形种子：11594773=baseline 在册（账本侧 recon_adjust 历史行
	// 不动）；11448262=baseline+业主域册双覆盖；11401353=dcd_settle 凭据
	// 在册（结算腿历史行不动）。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen) VALUES
		('{"op":"dcd_baseline","ordId":"920101","venue":"okx","ccy":"USDT","amt":50,"kind":"transfer"}','allow','transfer',1),
		('{"op":"dcd_baseline","ordId":"920102","venue":"okx","ccy":"BTC","amt":0.0207,"kind":"transfer"}','allow','transfer',1),
		('{"op":"dcd_settle","ordId":"920103","venue":"okx","ccy":"USDT","amt":10.0260288,"kind":"transfer"}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position_domain(decision_id,domain,inst_id,ord_id,note)
		VALUES(987654322,'owner','BTC-USDT-920102-95000-C','920102','test owner domain')`); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM fin_position_domain WHERE ord_id='920102'`)
	ords := []okxexec.DcdOrder{
		{OrdID: "920101", ProductID: "BTC-USDT-920101-66000-P", State: "live",
			NotionalCcy: "USDT", NotionalSz: "50"},
		{OrdID: "920102", ProductID: "BTC-USDT-920102-95000-C", State: "live",
			NotionalCcy: "BTC", NotionalSz: "0.0207"},
		{OrdID: "920103", ProductID: "ETH-USDT-920103-2280-P", State: "settled",
			NotionalCcy: "USDT", NotionalSz: "10", SettleCcy: "USDT",
			SettleSz: "10.0260288", YieldCcy: "USDT", YieldSz: "0.0260288",
			SettleTime: "1767225600000", SettledTime: "1767225660000"},
	}
	var credBefore, legBefore int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_risk_decision`).Scan(&credBefore); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow`).Scan(&legBefore); err != nil {
		t.Fatal(err)
	}
	dcdSettleReconcile(ctx, pool, ords)
	var credAfter, legAfter int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_risk_decision`).Scan(&credAfter); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow`).Scan(&legAfter); err != nil {
		t.Fatal(err)
	}
	// 凭据+3=迁移种子基线(0043)+本测三笔——全为播种；对账拍零增量。
	if credAfter != credBefore || legAfter != legBefore {
		t.Fatalf("存量单回放非幂等零增: cred %d→%d leg %d→%d",
			credBefore, credAfter, legBefore, legAfter)
	}
}

// O-33362 锚⑦：基线补挂写败→dcd_baseline_fail 专属升件（24h 同 ordId
// 去重窗——孤儿升件退役后唯一告警径）。
func TestDcdBaselineFailEsc(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	od := okxexec.DcdOrder{OrdID: "920008", ProductID: "BTC-USDT-920008-1-C",
		State: "live", NotionalCcy: "USDT", NotionalSz: "10"}
	dcdBaselineFailEsc(ctx, pool, od, context.DeadlineExceeded)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='dcd_baseline_fail' AND esc_scope='920008'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("补挂败升件缺: n=%d err=%v", n, err)
	}
	dcdBaselineFailEsc(ctx, pool, od, context.DeadlineExceeded)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_class='dcd_baseline_fail' AND esc_scope='920008'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件不幂等: n=%d", n)
	}
}

// 工单#20904(b) 锚③：dcd_settle 凭据→FundCcy funding/earn 域注册回款币；
// 同 ordId 基线退役（dcd 域键消失=幻影持仓断根）。
func TestFundCcyDcdSettle(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 桩币取独占键隔离（测试库共享——他锚 dcd_baseline BTC live 桩合法供
	// okx:dcd,BTC 域键，同币撞面=误红）。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen) VALUES
		('{"op":"dcd_baseline","ordId":"777","venue":"okx","ccy":"XRP","amt":20}','allow','transfer',1),
		('{"op":"dcd_settle","ordId":"777","venue":"okx","ccy":"XRP","amt":20.1}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	set, err := ledger.FundCcy(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if !set[[2]string{"okx:funding", "XRP"}] || !set[[2]string{"okx:earn", "XRP"}] {
		t.Fatal("结算回款币未注册 funding/earn 域")
	}
	if set[[2]string{"okx:dcd", "XRP"}] {
		t.Fatal("已结算基线未退役——幻影持仓面")
	}
}

// 工单#24798 锚④：孤儿裁定仓形（exec 早已 closed+position state='orphaned'）
// ——settle 仍收敛仓（closed_at 置位 FSM 随 closed）+exec ordId 确证键回填
// 不动态（原 WHERE state='open' 断根形根修）。
func TestDcdSettleOrphanedConverge(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	var eid, pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('BTC-USDT-261204-66000-P',50,'closed') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,state)
		VALUES($1,'BTC-USDT-261204-66000-P',50,'orphaned') RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	od := okxexec.DcdOrder{OrdID: "11594773", ProductID: "BTC-USDT-261204-66000-P",
		State: "settled", NotionalCcy: "USDT", NotionalSz: "50",
		SettleCcy: "USDT", SettleSz: "50.5", SettlePx: "70100",
		YieldCcy: "USDT", YieldSz: "0.5", SettleTime: "1764979200000", ExpTime: "1764979200000"}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{od})
	var estate, ordID, pstate string
	var closedAt *time.Time
	var samt float64
	var sccy string
	if err := pool.QueryRow(ctx, `SELECT state,COALESCE(venue_ord_id,'') FROM fin_execution
		WHERE id=$1`, eid).Scan(&estate, &ordID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT state,closed_at,settled_amount,settled_ccy
		FROM fin_position WHERE id=$1`, pid).Scan(&pstate, &closedAt, &samt, &sccy); err != nil {
		t.Fatal(err)
	}
	if estate != "closed" || ordID != "11594773" {
		t.Fatalf("exec 确证键断根: state=%s ordId=%s", estate, ordID)
	}
	if pstate != "closed" || closedAt == nil || samt != 50.5 || sccy != "USDT" {
		t.Fatalf("孤儿仓未收敛: state=%s closed_at=%v amt=%v ccy=%s", pstate, closedAt, samt, sccy)
	}
}
