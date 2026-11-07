// xvenue_test.go —— T-0145 跨所双腿平仓桩：per-leg venue 保真（旧单 venue
// 字段跨 venue 腿序错位病灶）、mt5 腿 venue_ticket 透传 CloseLeg（OrderClose
// 凭据=票）；venue sizer 路由（mt5 腿→Mt5Sizer OpenedOrders 手数实读、
// okx 腿→默认 okx sizer）；'lot' 量纲 sz 直用。
package guard

import (
	"context"
	"testing"
	"time"
)

// mkXvenuePosition xvenue 双腿仓造具——mt5 cfd 腿（venue_ticket=777/sz_unit=
// lot）+okx perp 腿。
func mkXvenuePosition(t *testing.T, ctx context.Context, w *Warden) int64 {
	t.Helper()
	var stratID, execID, posID int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue) VALUES('BTC-USDT-SWAP','guard-xv',$1,100,'okx') RETURNING id`,
		"live").Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state) VALUES($1,'BTC-USDT-SWAP',100,'open') RETURNING id`,
		stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	for i, lg := range []struct {
		inst, venue, side, unit string
		sz                      float64
		ticket                  int64
	}{
		{"BTCUSDm", "mt5", "sell", "lot", 0.5, 777},
		{"BTC-USDT-SWAP", "okx", "buy", "contract", 10, 0}} {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state,sz_unit,qty_base,venue_ticket)
			VALUES($1,$2,$3,$4,$5,$6,$7,'filled',$8,0.05,NULLIF($9,0))`,
			execID, i+1, "cl-xv-"+lg.inst, lg.venue, lg.side, lg.inst, lg.sz, lg.unit, lg.ticket); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		VALUES($1,$2,'BTC-USDT-SWAP',100,$3) RETURNING id`, execID, stratID, time.Now()).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	return execID
}

// routeSizer venue 路由录调桩（mt5/okx 腿各归各件——跨 venue 错读防线）。
type routeSizer struct {
	calls *[]string
	avail float64
	lot   float64
}

func (s routeSizer) SpotAvailBal(ctx context.Context, inst string) (float64, error) {
	*s.calls = append(*s.calls, "spot:"+inst)
	return s.avail, nil
}
func (s routeSizer) ContractPos(ctx context.Context, inst string) (float64, error) {
	*s.calls = append(*s.calls, "pos:"+inst)
	return s.avail, nil
}
func (s routeSizer) LotSz(ctx context.Context, inst string) (float64, error) {
	return s.lot, nil
}

// 锚⑥a：腿行→CloseLeg——mt5 腿 venue 保真+venue_ticket 透传（OrderClose
// 凭据）；side 翻转（sell→buy 平仓向）；'lot' 量纲 sz 直用（手数=下单方言
// 同量纲——qty_base 币量面值禁当手数发）。
func TestXvenueCloseLegTicket(t *testing.T) {
	ctx, w, _ := setup(t)
	execID := mkXvenuePosition(t, ctx, w)
	ls := w.legs(ctx, execID)
	if ls.one {
		t.Fatal("双腿行未读出")
	}
	cls := ls.reverse()
	if cls[0].Venue != "mt5" || cls[0].Ticket != 777 {
		t.Fatalf("mt5 腿 venue/ticket 未透传: %+v", cls[0])
	}
	if cls[0].Side != "buy" || !cls[0].SzKnown || cls[0].Sz != 0.5 {
		t.Fatalf("mt5 腿翻转/量纲: %+v", cls[0])
	}
	if cls[1].Venue != "okx" || cls[1].Side != "sell" || cls[1].Ticket != 0 {
		t.Fatalf("okx 腿 venue/ticket: %+v", cls[1])
	}
}

// 锚⑥b：sizedCloseLegs venue 路由——mt5 腿走 AttachVenueSizer 专用件
// （OpenedOrders 实读手数）；okx 腿走默认 w.sizer；venue 无路由=fail-closed。
func TestXvenueSizerRouting(t *testing.T) {
	ctx, w, _ := setup(t)
	var m5calls, okcalls []string
	w.AttachVenueSizer("mt5", routeSizer{calls: &m5calls, avail: 0.4, lot: 0.01})
	w.sizer = routeSizer{calls: &okcalls, avail: 100, lot: 0.1}
	cls := [2]CloseLeg{
		{Venue: "mt5", InstID: "BTCUSDm", Side: "buy", Sz: 0.5, SzKnown: true, Ticket: 777},
		{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "sell", Sz: 10, SzKnown: true}}
	out, ok := w.sizedCloseLegs(ctx, cls)
	if !ok {
		t.Fatal("路由件在册仍 fail")
	}
	if len(m5calls) != 1 || m5calls[0] != "pos:BTCUSDm" {
		t.Fatalf("mt5 腿未走专用件: %v", m5calls)
	}
	if len(okcalls) != 1 || okcalls[0] != "pos:BTC-USDT-SWAP" {
		t.Fatalf("okx 腿未走默认件: %v", okcalls)
	}
	// mt5 腿=min(0.5,0.4 avail)→0.4 对齐
	if out[0].Sz != 0.4 {
		t.Fatalf("mt5 腿未按实读对齐: %+v", out[0])
	}
	// venue 无路由=fail-closed（错 venue 件=错读量防线）
	cls[0].Venue = "binance"
	if _, ok = w.sizedCloseLegs(ctx, cls); ok {
		t.Fatal("venue 无路由应 fail-closed")
	}
}
