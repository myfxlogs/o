package execution

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// fakeEarn 可控停泊赎回面（redeem 败/成双态+共享调用序记录——前置赎回锚）。
type fakeEarn struct {
	err   error
	calls []float64
	order *[]string // 共享调用序（earn 赎回应先于 leg1 下单）
}

func (f *fakeEarn) RedeemToTrading(ctx context.Context, usdAmt float64) error {
	*f.order = append(*f.order, "earn")
	f.calls = append(f.calls, usdAmt)
	return f.err
}

// TestEarnRedeemFailClosed 前置赎回败=不开仓（fail-closed）：无腿行+exec 翻 failed+计数门。
func TestEarnRedeemFailClosed(t *testing.T) {
	order := []string{}
	fe := &fakeEarn{err: errors.New("earn down"), order: &order}
	ctx, r := setup(t, &fakeVenue{})
	r.WithEarn(fe)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err == nil {
		t.Fatal("赎回败应不开仓")
	}
	var state string
	if err := r.pool.QueryRow(ctx, "SELECT state FROM fin_execution WHERE id=$1", eid).Scan(&state); err != nil || state != "failed" {
		t.Fatalf("state=%s err=%v（应 failed 留痕）", state, err)
	}
	var n int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_order_leg WHERE execution_id=$1", eid).Scan(&n); err != nil || n != 0 {
		t.Fatalf("腿行=%d（应零腿）", n)
	}
	if r.rej.Snapshot()["execution/earn_redeem_failed"] == 0 {
		t.Fatal("earn_redeem_failed 计数门未计")
	}
}

// TestEarnRedeemBeforeLeg1 赎回先于 leg1 下单；redeem 金额=notional（buffer 加成在 mover 面核）。
func TestEarnRedeemBeforeLeg1(t *testing.T) {
	order := []string{}
	fe := &fakeEarn{order: &order}
	fv := &fakeVenue{onPlace: func() { order = append(order, "leg") }}
	ctx, r := setup(t, fv)
	r.WithEarn(fe)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err != nil {
		t.Fatalf("赎回成+腿成应开仓: %v", err)
	}
	if len(fe.calls) != 1 || fe.calls[0] != 100 {
		t.Fatalf("redeem calls=%v（应一次且=notional 100）", fe.calls)
	}
	if fmt.Sprint(order) != "[earn leg leg]" { // 赎回①→leg1②→leg2③
		t.Fatalf("调用序=%v（赎回应先于双腿）", order)
	}
	var n int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_order_leg WHERE execution_id=$1 AND state='filled'", eid).Scan(&n); err != nil || n != 2 {
		t.Fatalf("腿行=%d（应双腿 filled）", n)
	}
}

// TestEarnAbsentNoRedeem 未挂接停泊面=跳过赎回（停泊关闭径——资金常驻交易户）。
func TestEarnAbsentNoRedeem(t *testing.T) {
	ctx, r := setup(t, &fakeVenue{})
	did := mkToken(t, ctx, r, false, false, "allow")
	if _, err := r.Execute(ctx, mkIntent(did)); err != nil {
		t.Fatalf("未挂接应照常开仓: %v", err)
	}
}
