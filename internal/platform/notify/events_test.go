package notify

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestEventPushOffNoSend 开关缺/false→零发（fail-closed 静默锚）。
func TestEventPushOffNoSend(t *testing.T) {
	f := &fakeSender{}
	e := NewEventPush(nil, f, func() string { return "oc_x" }, func() bool { return false })
	_ = e.Scan(context.Background())
	if len(f.sent) != 0 {
		t.Fatalf("off 应零发 got=%v", f.sent)
	}
}

// TestEventPushDedup 同 key 一键一推——重注入不重复。
func TestEventPushDedup(t *testing.T) {
	f := &fakeSender{}
	e := NewEventPush(nil, f, func() string { return "oc_x" }, func() bool { return true })
	ctx := context.Background()
	e.push(ctx, "opened:1", "开仓 BTC-USDT-SWAP")
	e.push(ctx, "opened:1", "开仓 BTC-USDT-SWAP")
	if len(f.sent) != 1 || !strings.Contains(f.sent[0], "[经营事件]") {
		t.Fatalf("去重推=%v", f.sent)
	}
}

// TestEventPushSimdrillFiltered 工单#13304：simdrill 胞(template_ref='simdrill:%')
// 开平仓不入经营推送；真策略行照推——两谓词同面双证。
func TestEventPushSimdrillFiltered(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	mk := func(tmpl string) int64 {
		var sid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
			VALUES('BTC-USDT-SWAP',$1,'live',50,'okx') RETURNING id`, tmpl).Scan(&sid); err != nil {
			t.Fatal(err)
		}
		var eid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
			VALUES($1,'BTC-USDT-SWAP',50,'closed') RETURNING id`, sid).Scan(&eid); err != nil {
			t.Fatal(err)
		}
		return eid
	}
	eSim := mk("simdrill:cell:funding-hedge@okx/short_perp_long_spot#usd=50")
	eReal := mk("funding-hedge")
	for _, eid := range []int64{eSim, eReal} {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
			SELECT $1,strategy_ref,'BTC-USDT-SWAP',50 FROM fin_execution WHERE id=$1`, eid); err != nil {
			t.Fatal(err)
		}
	}
	f := &fakeSender{}
	e := NewEventPush(pool, f, func() string { return "oc_x" }, func() bool { return true })
	e.pushOpened(ctx)
	if len(f.sent) != 1 {
		t.Fatalf("演习滤后应余 1 推 got=%v", f.sent)
	}
	e.pushClosed(ctx) // 未关——不产推
	if len(f.sent) != 1 {
		t.Fatalf("open 件未关 pushClosed 应零推 got=%v", f.sent)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_position SET closed_at=now(),settled_amount=1.0,settled_ccy='USDT'`); err != nil {
		t.Fatal(err)
	}
	e.pushClosed(ctx)
	if len(f.sent) != 2 {
		t.Fatalf("真平仓应再推 1 got=%v", f.sent)
	}
}
