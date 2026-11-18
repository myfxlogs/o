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

// 工单#22919 锚②：live 单账本三覆盖源全缺→反向孤儿升件；有覆盖→不升。
func TestDcdOrphanEscalate(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	orphan := okxexec.DcdOrder{OrdID: "999", ProductID: "BTC-USDT-999999-1-C",
		State: "live", NotionalCcy: "USDT", NotionalSz: "10"}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{orphan})
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='dcd_orphan' AND esc_scope='999'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("孤儿升件缺: n=%d err=%v", n, err)
	}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{orphan})
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_class='dcd_orphan' AND esc_scope='999'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件不幂等: n=%d", n)
	}
	// 基线凭据覆盖源：有 dcd_baseline 同 ordId→不升。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{"op":"dcd_baseline","ordId":"888","venue":"okx","ccy":"BTC","amt":0.01}','allow','transfer',1)`); err != nil {
		t.Fatal(err)
	}
	covered := okxexec.DcdOrder{OrdID: "888", ProductID: "BTC-USDT-999999-2-C",
		State: "live", NotionalCcy: "BTC", NotionalSz: "0.01"}
	dcdSettleReconcile(ctx, pool, []okxexec.DcdOrder{covered})
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_class='dcd_orphan' AND esc_scope='888'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("有覆盖误升: n=%d", n)
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
