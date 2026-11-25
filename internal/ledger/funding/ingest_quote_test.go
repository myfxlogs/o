package funding

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mkOwnLeg 造 O 自执行腿+legCash 现金行（判别键实证件——clOrdId 命中
// fin_order_leg 且 source_table='fin_order_leg' 现金行在账=legCash 已记）。
// withFee=true 加 legCash fee 实记行（feeCcy=USDT 径——bill fee 判别键）。
func mkOwnLeg(t *testing.T, ctx context.Context, pool *pgxpool.Pool, clOrdID string, withFee bool) {
	t.Helper()
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES(1,1,gen_random_uuid(),'BTC-USDT',100,'closed') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	var lid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,$2,'okx','buy','BTC-USDT',0.001,'filled') RETURNING id`, eid, clOrdID).Scan(&lid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
		VALUES('spot_buy','okx','USDT',-100,$1,'fin_order_leg',now())`, lid); err != nil {
		t.Fatal(err)
	}
	if withFee {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
			VALUES('fee','okx','USDT',-0.1,$1,'fin_order_leg',now())`, lid); err != nil {
			t.Fatal(err)
		}
	}
}

// mkClearDecision 造 residual_clear 自记账锚（fin_risk_decision intent
// cl_ord_id——gross 域已记=spot_sell cashflow 同事务实证）。
func mkClearDecision(t *testing.T, ctx context.Context, pool *pgxpool.Pool, clOrdID string) {
	t.Helper()
	var did int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind)
		VALUES($1::jsonb,'allow','transfer',1,'open') RETURNING id`,
		`{"op":"residual_clear","cl_ord_id":"`+clOrdID+`","ccy":"TRX"}`).Scan(&did); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
		VALUES('spot_sell','okx','USDT',9999,$1,'fin_risk_decision',now())`, did); err != nil {
		t.Fatal(err)
	}
}

func cashCnt(t *testing.T, ctx context.Context, pool *pgxpool.Pool, billID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow c
		JOIN fin_venue_bill b ON c.source_ref=b.id AND c.source_table='fin_venue_bill'
		WHERE b.bill_id=$1`, billID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// O-39310 验收①：clOrdId 命中已记腿（legCash 现金行在账）→USDT 计价腿
// gross 域跳过防双计；fee 域独立判别——腿无 fee 实记行时 bill fee 腿
// 是唯一 USDT 费账源照记（legCash feeCcy 漏报案在案——分量级判别）。
func TestSpotQuoteOwnOrderSkipped(t *testing.T) {
	ctx, pool := setup(t)
	mkOwnLeg(t, ctx, pool, "ph999x1x123456", false)
	nowMs := time.Now().UnixMilli()
	bills := []Bill{
		{BillID: "q-own", SubType: "2", InstID: "BTC-USDT", InstType: "SPOT",
			Ccy: "USDT", TsMs: nowMs, BalChg: -100.5, Fee: -0.1, ClOrdID: "ph999x1x123456",
			Raw: []byte(`{"billId":"q-own","clOrdId":"ph999x1x123456"}`)},
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 { // gross 跳过+fee 腿照记（腿无 fee 行=bill fee 唯一账源）
		t.Fatalf("自执行单计价腿应仅 fee 入账 got=%d", n)
	}
	var kinds string
	_ = pool.QueryRow(ctx, `SELECT string_agg(kind,',') FROM fin_cashflow c
		JOIN fin_venue_bill b ON c.source_ref=b.id AND c.source_table='fin_venue_bill'
		WHERE b.bill_id='q-own'`).Scan(&kinds)
	if kinds != "fee" {
		t.Fatalf("自执行单计价腿应只产 fee 行 got=%q", kinds)
	}
	var srcN int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_venue_bill WHERE bill_id='q-own'").Scan(&srcN)
	if srcN != 1 {
		t.Fatalf("源行应留档 got=%d", srcN)
	}
}

// O-39310 验收②：clOrdId 不命中（venue 侧 bot/手动单）→USDT 计价腿落
// spotNative 对称口径——买=计价币流出记 spot_buy 负额+fee 腿；卖=流入
// 记 spot_sell 正额。
func TestSpotQuoteVenueSideRecorded(t *testing.T) {
	ctx, pool := setup(t)
	nowMs := time.Now().UnixMilli()
	bills := []Bill{
		// venue 侧买单 USDT 腿：balChg=-100.5 fee=-0.1 → gross=-100.4 spot_buy
		{BillID: "q-ven-buy", SubType: "2", InstID: "BTC-USDT", InstType: "SPOT",
			Ccy: "USDT", TsMs: nowMs, BalChg: -100.5, Fee: -0.1, ClOrdID: "O4128808031748269058",
			Raw: []byte(`{}`)},
		// venue 侧卖单 USDT 腿：balChg=+50 fee=-0.05 → gross=+50.05 spot_sell
		{BillID: "q-ven-sell", SubType: "1", InstID: "ETH-USDT", InstType: "SPOT",
			Ccy: "USDT", TsMs: nowMs + 1, BalChg: 50, Fee: -0.05, ClOrdID: "vf9988776655",
			Raw: []byte(`{}`)},
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 { // 两腿各 spot+fee=4
		t.Fatalf("venue 侧计价腿应 4 行入账 got=%d", n)
	}
	var buy, sell, fee float64
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='spot_buy' AND ccy='USDT'").Scan(&buy)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='spot_sell' AND ccy='USDT'").Scan(&sell)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='fee' AND ccy='USDT'").Scan(&fee)
	// gross=balChg−fee：买 −100.5−(−0.1)=−100.4；卖 50−(−0.05)=50.05；费 −0.1−0.05
	if buy != -100.4 || sell != 50.05 {
		t.Fatalf("计价腿 buy=%v sell=%v 应 -100.4/50.05", buy, sell)
	}
	if fee != -0.15 {
		t.Fatalf("fee 合计=%v 应 -0.15", fee)
	}
}

// O-39310 验收③：clOrdId 缺载=无判别键保守跳过+rej 显形（漏记>双计）。
func TestSpotQuoteMissingClOrdIDSkipped(t *testing.T) {
	ctx, pool := setup(t)
	nowMs := time.Now().UnixMilli()
	bills := []Bill{
		{BillID: "q-noid", SubType: "2", InstID: "BTC-USDT", InstType: "SPOT",
			Ccy: "USDT", TsMs: nowMs, BalChg: -100.5, Fee: -0.1, // ClOrdID 空
			Raw: []byte(`{}`)},
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("缺载计价腿应零入账 got=%d", n)
	}
	if cashCnt(t, ctx, pool, "q-noid") != 0 {
		t.Fatal("缺载计价腿产 cashflow=裸放违例")
	}
	if ing.rej.Snapshot()["funding_ingest/clordid_missing"] != 1 {
		t.Fatal("clordid_missing rej 未显形")
	}
}

// O-39310 验收④：腿行在账但 legCash 未记（ack/live 未成交确认）→EXISTS
// 连接不成立→bill 侧正常入账（迟成交单两腿对称补记，不双计不漏计）。
func TestSpotQuoteLegWithoutCashRecorded(t *testing.T) {
	ctx, pool := setup(t)
	// 造腿行但不造 legCash 现金行（ack 态迟成交场景）
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES(1,1,gen_random_uuid(),'BTC-USDT',100,'closed') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,'ph888x1x999','okx','buy','BTC-USDT',0.001,'ack')`, eid); err != nil {
		t.Fatal(err)
	}
	nowMs := time.Now().UnixMilli()
	bills := []Bill{
		{BillID: "q-ack", SubType: "2", InstID: "BTC-USDT", InstType: "SPOT",
			Ccy: "USDT", TsMs: nowMs, BalChg: -100.5, Fee: -0.1, ClOrdID: "ph888x1x999",
			Raw: []byte(`{}`)},
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 { // spot+fee=2——腿在账但无现金行=未记，自执行语义不成立照记
		t.Fatalf("无现金行腿的计价腿应入账 got=%d", n)
	}
}

// O-39310 验收⑤：clOrdId 命中 fin_order_leg 且 legCash fee 实记行在账
// （feeCcy=USDT 径）→gross+fee 双域全覆盖零入账（全跳过实证）。
func TestSpotQuoteOwnOrderFeeCovered(t *testing.T) {
	ctx, pool := setup(t)
	mkOwnLeg(t, ctx, pool, "ph777x1x555", true)
	nowMs := time.Now().UnixMilli()
	bills := []Bill{
		{BillID: "q-ownfee", SubType: "2", InstID: "BTC-USDT", InstType: "SPOT",
			Ccy: "USDT", TsMs: nowMs, BalChg: -100.5, Fee: -0.1, ClOrdID: "ph777x1x555",
			Raw: []byte(`{}`)},
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("fee 已记自执行单计价腿应零入账 got=%d", n)
	}
	if cashCnt(t, ctx, pool, "q-ownfee") != 0 {
		t.Fatal("fee 已记自执行单计价腿产 cashflow=双计")
	}
}

// O-39310 验收⑥：clOrdId 命中 fin_risk_decision（residual_clear 自记账
// ——gross 域已记 spot_sell）→gross 跳过防双计；fee 域照记（clear 径
// 不记 fee——bill fee 腿唯一 USDT 费账源）。生产实证：TRX 清仓
// $9999.08 双计案根修判据。
func TestSpotQuoteClearOrderFeeOnly(t *testing.T) {
	ctx, pool := setup(t)
	mkClearDecision(t, ctx, pool, "ec1795234507438182144")
	nowMs := time.Now().UnixMilli()
	bills := []Bill{
		{BillID: "q-clear", SubType: "1", InstID: "TRX-USDT", InstType: "SPOT",
			Ccy: "USDT", TsMs: nowMs, BalChg: 1082.05, Fee: -1.08, ClOrdID: "ec1795234507438182144",
			Raw: []byte(`{}`)},
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 { // gross 跳过+fee 腿照记=1
		t.Fatalf("clear 单计价腿应仅 fee 入账 got=%d", n)
	}
	var kinds string
	_ = pool.QueryRow(ctx, `SELECT string_agg(kind,',') FROM fin_cashflow c
		JOIN fin_venue_bill b ON c.source_ref=b.id AND c.source_table='fin_venue_bill'
		WHERE b.bill_id='q-clear'`).Scan(&kinds)
	if kinds != "fee" {
		t.Fatalf("clear 单计价腿应只产 fee 行 got=%q", kinds)
	}
}
