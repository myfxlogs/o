package guard

// ttlzero_test.go —— O-41392 max_hold 口径修对抗锚：零腿产品形
// （DCD/结构品——exec 无 order_leg 行=正常产品形，venue 到期结算径
// dcdSettleOne 收殓）TTL 判词空转根修。畸形腿（>2 腿/读败）不豁免。

import (
	"context"
	"testing"
	"time"
)

// mkZeroLegPosition 造零腿仓 fixture（DCD 产品形——exec 无 fin_order_leg
// 行；strategy/exec/position 三层照 mkPosition 形，腿集刻意缺席）。
func mkZeroLegPosition(t *testing.T, ctx context.Context, w *Warden, stratState string,
	openedAt time.Time) int64 {
	t.Helper()
	var stratID, execID, posID int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue) VALUES('BTC-USDT-SWAP','guard-test',$1,100,'okx') RETURNING id`,
		stratState).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state) VALUES($1,'BTC-USDT-SWAP',100,'open') RETURNING id`,
		stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		VALUES($1,$2,'BTC-USDT-SWAP',100,$3) RETURNING id`, execID, stratID, openedAt).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	return posID
}

// 对抗①（修后断言）：零腿 exec+age>maxH 仓 → max_hold 判词不发——
// 无 fin_guard_close 行（tend 空 trig 早退不产定性行），skip note 在案。
func TestMaxHoldZeroLegProductSkip(t *testing.T) {
	ctx, w, _ := setup(t)
	seedFacts(t, ctx, w, 0.0001)
	posID := mkZeroLegPosition(t, ctx, w, "live", time.Now().Add(-100*time.Hour))
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	closed, state, trig := posClosed(t, ctx, w, posID)
	if closed || state != "open" || trig != "" {
		t.Fatalf("零腿产品形 TTL 不适用——不应产判词行，got closed=%v state=%q trig=%q", closed, state, trig)
	}
	if _, ok := w.notes["ttl_product_skip"]; !ok {
		t.Fatal("ttl_product_skip skip note 未落——零腿豁免须可观测")
	}
	// legs() 直证：查询成功 0 行=one+zero 产品形位。
	var execID int64
	if err := w.pool.QueryRow(ctx,
		"SELECT execution_id FROM fin_position WHERE id=$1", posID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	if ls := w.legs(ctx, execID); !ls.one || !ls.zero {
		t.Fatalf("零腿成功读应 one+zero，got %+v", ls)
	}
}

// legs() 分形直证：查询败（ctx 取消）=one 非 zero 真畸形——零腿豁免不禁读败；
// 正常双腿仓 zero 位恒假（主流径零变）。
func TestLegsQueryFailNotZero(t *testing.T) {
	ctx, w, _ := setup(t)
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	if ls := w.legs(cctx, 1); !ls.one || ls.zero {
		t.Fatalf("查询败应 one:true zero:false（畸形照报），got %+v", ls)
	}
	seedFacts(t, ctx, w, 0.0001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	var execID int64
	if err := w.pool.QueryRow(ctx,
		"SELECT execution_id FROM fin_position WHERE id=$1", posID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	if ls := w.legs(ctx, execID); ls.one || ls.single || ls.zero {
		t.Fatalf("正常双腿应 one:false single:false zero:false，got %+v", ls)
	}
}

// 对抗②：>2 腿畸形仓 age>maxH → max_hold 判词照火（畸形 aging 不豁免——
// legs.one 定性行 close_failed/max_hold 在案）。
func TestMaxHoldMalformedLegsStillFire(t *testing.T) {
	ctx, w, _ := setup(t)
	seedFacts(t, ctx, w, 0.0001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now().Add(-100*time.Hour))
	var execID int64
	if err := w.pool.QueryRow(ctx,
		"SELECT execution_id FROM fin_position WHERE id=$1", posID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state,sz_unit)
		VALUES($1,3,'cl-x3','okx','buy','BTC-USDT-SWAP',1,'filled','contract')`, execID); err != nil {
		t.Fatal(err)
	}
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	closed, state, trig := posClosed(t, ctx, w, posID)
	if closed || state != "close_failed" || trig != "max_hold" {
		t.Fatalf("畸形腿仓 max_hold 应照火定性行，got closed=%v state=%q trig=%q", closed, state, trig)
	}
}
