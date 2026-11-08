// xvenue_test.go —— T-0145 跨所 carry 双腿原子序桩测：腿序钦定 mt5 先
// okx 后、leg2 败→mt5 腿 OrderClose(ticket) 回滚、venue_ticket/'lot' 量纲
// 落列、歧义=告警不静默。
package execution

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"o/internal/trading/okxexec"
)

// xvFake 分 venue 记录桩（调用序共享指针——双腿序断言）。
type xvFake struct {
	seq       *[]string
	name      string
	fail      bool              // Place 传输败→Resolve unknown=歧义（假 absent 病灶径）
	apiReject *okxexec.APIError // envelope 业务拒（返修①终局透传锚）
	ticket    int64
	legs      []Leg
}

func (f *xvFake) Cancel(_ context.Context, _ Leg, _ string) error { return nil }
func (f *xvFake) Place(_ context.Context, leg Leg, clOrdID string) (OrderResult, error) {
	f.legs = append(f.legs, leg)
	tag := f.name + ":place"
	if leg.Close {
		tag = fmt.Sprintf("%s:close:t%d", f.name, leg.Ticket)
	}
	*f.seq = append(*f.seq, tag)
	if f.apiReject != nil {
		return OrderResult{ClOrdID: clOrdID, State: "absent"}, f.apiReject
	}
	if f.fail {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, errors.New("transport down")
	}
	return OrderResult{ClOrdID: clOrdID, State: "filled", FillPx: 100, FillSz: 1, FillTicket: f.ticket}, nil
}
func (f *xvFake) Resolve(_ context.Context, _ Leg, clOrdID string) (OrderResult, error) {
	if f.fail {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, errors.New("resolve down")
	}
	return OrderResult{ClOrdID: clOrdID, State: "filled", FillPx: 100, FillSz: 1, FillTicket: f.ticket}, nil
}

func xvLegs() [2]Leg {
	return [2]Leg{
		{Venue: "mt5", InstID: "BTCUSDm", Side: "sell", Sz: "0.5", Notional: 50, CtVal: 1},
		{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: "10", Notional: 50},
	}
}

// 锚①：双腿序钦定——mt5 cfd 腿先于 okx perp 腿；venue_ticket+sz_unit='lot'
// +qty_base=fillSz×CtVal 落列；仓行 inst=okx 主腿乐器。
func TestXvenueTwoLegOrder(t *testing.T) {
	var seq []string
	m5, ok := &xvFake{seq: &seq, name: "mt5", ticket: 777}, &xvFake{seq: &seq, name: "okx"}
	ctx, r := setup(t, RouterExecutor{"mt5": m5, "okx": ok})
	did := mkToken(t, ctx, r, false, false, "allow")
	in := mkIntent(did)
	in.Legs = xvLegs()
	eid, err := r.Execute(ctx, in)
	if err != nil {
		t.Fatalf("双腿开: %v", err)
	}
	if len(seq) != 2 || seq[0] != "mt5:place" || seq[1] != "okx:place" {
		t.Fatalf("腿序错位: %v（钦定 mt5 先 okx 后）", seq)
	}
	var venue, unit string
	var qb float64
	var tk int64
	if err := r.pool.QueryRow(ctx, `SELECT venue,sz_unit,qty_base,venue_ticket FROM fin_order_leg
		WHERE execution_id=$1 AND leg_no=1`, eid).Scan(&venue, &unit, &qb, &tk); err != nil {
		t.Fatal(err)
	}
	if venue != "mt5" || unit != "lot" || qb != 1 || tk != 777 {
		t.Fatalf("mt5 腿行落位错: venue=%s unit=%s qty_base=%v ticket=%d", venue, unit, qb, tk)
	}
	var posInst string
	if err := r.pool.QueryRow(ctx, `SELECT inst_id FROM fin_position WHERE execution_id=$1`, eid).Scan(&posInst); err != nil {
		t.Fatalf("仓行: %v", err)
	}
	if posInst != "BTC-USDT-SWAP" {
		t.Fatalf("仓行 inst=%s（应=okx 主腿乐器）", posInst)
	}
}

// 锚②：leg2 败→LEGGED→mt5 腿 OrderClose(ticket=777) 回滚——venue 仓凭据
// 透传断言（无票=hedge 户反向新开禁线）；state=rolled_back+仓行零落。
func TestXvenueLeg2FailRollback(t *testing.T) {
	var seq []string
	m5, ok := &xvFake{seq: &seq, name: "mt5", ticket: 777}, &xvFake{seq: &seq, name: "okx", fail: true}
	ctx, r := setup(t, RouterExecutor{"mt5": m5, "okx": ok})
	did := mkToken(t, ctx, r, false, false, "allow")
	in := mkIntent(did)
	in.Legs = xvLegs()
	eid, err := r.Execute(ctx, in)
	if err == nil {
		t.Fatal("leg2 败应 LEGGED")
	}
	var state string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_execution WHERE id=$1`, eid).Scan(&state); err != nil || state != "rolled_back" {
		t.Fatalf("state=%s（应 rolled_back）", state)
	}
	var rbSeen bool
	for _, s := range seq {
		if s == "mt5:close:t777" {
			rbSeen = true
		}
	}
	if !rbSeen {
		t.Fatalf("mt5 回滚未携 ticket=777 OrderClose: %v", seq)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_position WHERE execution_id=$1`, eid).Scan(&n); err != nil || n != 0 {
		t.Fatalf("回滚后仓行=%d", n)
	}
}

// 返修①端到端锚：Execute leg2 终局 APIError %w 透传——wiring 全链断言
// （venue 拒→placeAndResolve→Execute→err 链 errors.As 可达 TerminalInstErr）。
// 旧径 leg2 出口裸 fmt.Errorf 无 %w 全吞=okx 终局联动死径（delist/停盘拒→
// 不入册→策略不退→armed 每拍 leg1 真开+leg2 拒+OrderClose 真平=开平 churn
// 烧真钱）。本锚堵「handler 过线径断」测试缺口——分类判据在 Execute 边界可判。
func TestXvenueLeg2TerminalPassthrough(t *testing.T) {
	var seq []string
	m5 := &xvFake{seq: &seq, name: "mt5", ticket: 777}
	ok := &xvFake{seq: &seq, name: "okx",
		apiReject: &okxexec.APIError{Code: "51001", Msg: "Instrument ID doesn't exist"}}
	ctx, r := setup(t, RouterExecutor{"mt5": m5, "okx": ok})
	did := mkToken(t, ctx, r, false, false, "allow")
	in := mkIntent(did)
	in.Legs = xvLegs()
	eid, err := r.Execute(ctx, in)
	if err == nil {
		t.Fatal("leg2 终局拒应 LEGGED")
	}
	if !okxexec.TerminalInstErr(err) {
		t.Fatalf("终局 APIError 被吞——%%w 透传断链（untradeable 联动死径）: %v", err)
	}
	var state string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_execution WHERE id=$1`, eid).Scan(&state); err != nil || state != "rolled_back" {
		t.Fatalf("state=%s（应 rolled_back）", state)
	}
	var rbSeen bool
	for _, s := range seq {
		if s == "mt5:close:t777" {
			rbSeen = true
		}
	}
	if !rbSeen {
		t.Fatalf("mt5 回滚未携 ticket=777 OrderClose: %v", seq)
	}
}

// 锚③：leg1 歧态势（传输败+Resolve 同败=定不了真态）——failed+歧义腿行
// 落痕非静默（告警面 WithAlerter 空守零炸）。
func TestXvenueLeg1Ambiguous(t *testing.T) {
	var seq []string
	m5 := &xvFake{seq: &seq, name: "mt5", fail: true}
	ctx, r := setup(t, RouterExecutor{"mt5": m5, "okx": &xvFake{seq: &seq, name: "okx"}})
	did := mkToken(t, ctx, r, false, false, "allow")
	in := mkIntent(did)
	in.Legs = xvLegs()
	eid, err := r.Execute(ctx, in)
	if err == nil {
		t.Fatal("leg1 歧义应 err")
	}
	var state string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_execution WHERE id=$1`, eid).Scan(&state); err != nil || state != "failed" {
		t.Fatalf("state=%s", state)
	}
	if len(seq) != 1 { // leg2 未触=歧义禁补单链
		t.Fatalf("leg1 歧义后 leg2 仍触 venue: %v", seq)
	}
}

// 锚④：BuildXvenueLegs——向位双态映射+TradeMode 断言+min-lot fail-closed+
// Notional 实值对齐（T-0062 律：perp 反推 cfd 实名义）。
func TestBuildXvenueLegs(t *testing.T) {
	perpM := InstMeta{CtVal: 0.01, CtValCcy: "BTC", LotSz: 1, MinSz: 0.1}
	cfdM := CfdMeta{ContractSize: 1, MinLot: 0.01, LotStep: 0.01, TradeMode: 4}
	// long_perp_short_cfd→mt5 sell+okx buy；notional=100@px=100→lots=0.5→
	// cfdUSD=50→perp 张=50/(0.01×100)=50 张→perpUSD=50。
	legs, err := BuildXvenueLegs("long_perp_short_cfd", "BTC-USDT-SWAP", "BTCUSDm", 100, perpM, cfdM, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	if legs[0].Venue != "mt5" || legs[0].Side != "sell" || legs[0].Sz != "0.5" || legs[0].CtVal != 1 {
		t.Fatalf("leg1: %+v", legs[0])
	}
	if legs[1].Venue != "okx" || legs[1].Side != "buy" || legs[1].Sz != "50" {
		t.Fatalf("leg2: %+v", legs[1])
	}
	if legs[0].Notional != 50 || legs[1].Notional != 50 {
		t.Fatalf("名义错位: %v/%v", legs[0].Notional, legs[1].Notional)
	}
	// short_perp_long_cfd→mt5 buy+okx sell
	legs, err = BuildXvenueLegs("short_perp_long_cfd", "BTC-USDT-SWAP", "BTCUSDm", 100, perpM, cfdM, 100, 100)
	if err != nil || legs[0].Side != "buy" || legs[1].Side != "sell" {
		t.Fatalf("short_perp_long_cfd 向位错: %+v %v", legs, err)
	}
	// 他向位=fail-closed
	if _, err = BuildXvenueLegs("long_cfd_short_perp", "BTC-USDT-SWAP", "BTCUSDm", 100, perpM, cfdM, 100, 100); err == nil {
		t.Fatal("他向位未拒")
	}
	// TradeMode≠4=拒构（CloseOnly 预检）
	if _, err = BuildXvenueLegs("long_perp_short_cfd", "BTC-USDT-SWAP", "BTCUSDm", 100, perpM,
		CfdMeta{ContractSize: 1, MinLot: 0.01, LotStep: 0.01, TradeMode: 3}, 100, 100); err == nil {
		t.Fatal("CloseOnly 未拒")
	}
	// min-lot 构不出=fail-closed（规模不达标不裸放）
	if _, err = BuildXvenueLegs("long_perp_short_cfd", "BTC-USDT-SWAP", "BTCUSDm", 1, perpM, cfdM, 100, 100); err == nil {
		t.Fatal("min-lot 构不出未拒")
	}
}
