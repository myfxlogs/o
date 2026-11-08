// mt5sim_test.go —— T-0461 MT5 CFD 仿真 venue 桩测：ticket 生命周期/
// OrderClose 平仓/OpenedOrders 在仓读/拒单槽序/方言缝——mtexec 产线
// 消费面同缝断言（venue 语义纸面跑锚）。
package mt5sim

import (
	"context"
	"testing"

	mt5pb "o/internal/gen/mtapi/mt5"
)

func testSim() *Sim {
	return New().WithBalance(1e6).WithInstrument(Instrument{
		VenueSymbol: "BTCUSDm", Canonical: "BTCUSD",
		ContractSize: 0.001, MinLot: 0.001, LotStep: 0.001,
		TradeMode: 4, Px: 50000, State: "live"})
}

// 锚①：开仓市价单→ticket 签发+OpenedOrders 在仓读+comment 幂等键保留。
func TestOrderSendTicketLifecycle(t *testing.T) {
	s := testSim()
	ctx := context.Background()
	r, err := s.OrderSend(ctx, &mt5pb.OrderSendRequest{
		Symbol: "BTCUSDm", Operation: mt5pb.OrderType_OrderType_Sell,
		Volume: 0.5, Comment: strptr("ph1x1x1")})
	if err != nil || r.GetError() != nil {
		t.Fatalf("OrderSend 败: %v %+v", err, r.GetError())
	}
	tk := r.GetResult().GetTicket()
	if tk == 0 {
		t.Fatal("ticket 未签发")
	}
	if got := s.OpenedLots("BTCUSD"); got != 0.5 {
		t.Fatalf("在仓手数=%v want 0.5", got)
	}
	or, _ := s.OpenedOrders(ctx, &mt5pb.OpenedOrdersRequest{})
	if len(or.GetResult()) != 1 || or.GetResult()[0].GetComment() != "ph1x1x1" {
		t.Fatalf("OpenedOrders 在仓读异常: %+v", or.GetResult())
	}
}

// 锚②：OrderClose ticket 平仓→OpenedOrders 票消+OrderHistory 残史命中
// （comment 幂等键保留——mtexec.Resolve 开腿秒平残史面锚）。
func TestOrderCloseTicketLifecycle(t *testing.T) {
	s := testSim()
	ctx := context.Background()
	r, _ := s.OrderSend(ctx, &mt5pb.OrderSendRequest{
		Symbol: "BTCUSDm", Operation: mt5pb.OrderType_OrderType_Buy, Volume: 0.3})
	tk := r.GetResult().GetTicket()
	vol := 0.3
	cr, err := s.OrderClose(ctx, &mt5pb.OrderCloseRequest{Ticket: tk, Lots: &vol})
	if err != nil || cr.GetError() != nil {
		t.Fatalf("OrderClose 败: %v %+v", err, cr.GetError())
	}
	if got := s.OpenedLots("BTCUSD"); got != 0 {
		t.Fatalf("平后在仓=%v want 0", got)
	}
	if len(s.OpenTickets()) != 0 {
		t.Fatal("平后 OpenTickets 应空")
	}
	if s.HistoryCount() != 1 {
		t.Fatal("平后残史应 1 条")
	}
	// 残史面 OpenedOrders 读不到（票消）——mtexec resolveClose 票消=filled 律。
	or, _ := s.OpenedOrders(ctx, &mt5pb.OpenedOrdersRequest{})
	for _, o := range or.GetResult() {
		if o.GetTicket() == tk {
			t.Fatal("平后票应消")
		}
	}
}

// 锚③：拒单槽序——symbol 不在册/TradeMode≠FullAccess/量不落网格/
// 平侧 ticket 查无=venue 方言拒码（mtexec.apiErr 分类锚同源）。
func TestRejectSlots(t *testing.T) {
	s := testSim()
	ctx := context.Background()
	// ① symbol 不在册.
	if r, _ := s.OrderSend(ctx, &mt5pb.OrderSendRequest{
		Symbol: "NOPE", Operation: mt5pb.OrderType_OrderType_Buy, Volume: 0.1}); r.GetError().GetCode() != mt5pb.ErrorCode_SYMBOL_NOT_FOUND {
		t.Fatalf("不在册拒码=%v", r.GetError().GetCode())
	}
	// ③ 量不落 lotStep.
	if r, _ := s.OrderSend(ctx, &mt5pb.OrderSendRequest{
		Symbol: "BTCUSDm", Operation: mt5pb.OrderType_OrderType_Buy, Volume: 0.1234}); r.GetError().GetCode() != mt5pb.ErrorCode_INVALID_VOLUME {
		t.Fatalf("网格外拒码=%v", r.GetError().GetCode())
	}
	// ④ 平侧 ticket 查无.
	ghost := int64(999999)
	vol := 0.1
	if r, _ := s.OrderClose(ctx, &mt5pb.OrderCloseRequest{Ticket: ghost, Lots: &vol}); r.GetError().GetCode() != mt5pb.ErrorCode_POSITION_NOT_EXISTS {
		t.Fatalf("平侧查无拒码=%v", r.GetError().GetCode())
	}
	// TradeMode 拒——另册 CloseOnly 乐器.
	s2 := New().WithBalance(1e6).WithInstrument(Instrument{
		VenueSymbol: "XAUUSDm", Canonical: "XAUUSD",
		ContractSize: 100, MinLot: 0.01, LotStep: 0.01,
		TradeMode: 3, Px: 2000}) // CloseOnly
	if r, _ := s2.OrderSend(ctx, &mt5pb.OrderSendRequest{
		Symbol: "XAUUSDm", Operation: mt5pb.OrderType_OrderType_Buy, Volume: 0.01}); r.GetError().GetCode() != mt5pb.ErrorCode_TRADE_DISABLED {
		t.Fatalf("TradeMode 拒码=%v", r.GetError().GetCode())
	}
}

// 锚④：方言缝+乐器面——canonical→venue_symbol 唯一译径+Meta/SymbolParams
// 三面实填（产线 xvenueOpen TradeMode 活读同缝）。
func TestDialectAndMeta(t *testing.T) {
	s := testSim()
	ctx := context.Background()
	sym, err := s.Dialect().VenueSymbol(ctx, "BTCUSD")
	if err != nil || sym != "BTCUSDm" {
		t.Fatalf("方言译=%q err=%v", sym, err)
	}
	if _, err := s.Dialect().VenueSymbol(ctx, "NOPE"); err == nil {
		t.Fatal("方言查无应 err fail-closed")
	}
	// 升件缝=mtexec 侧 Escalate 调用留痕（sim 不自记——产线分层律）。
	s.Dialect().Escalate(ctx, "NOPE", "test-miss")
	if len(s.Escalations()) != 1 {
		t.Fatalf("方言升件留痕=%v", s.Escalations())
	}
	cs, minLot, lotStep, tm, ok := s.Meta("BTCUSD")
	if !ok || cs != 0.001 || minLot != 0.001 || lotStep != 0.001 || tm != 4 {
		t.Fatalf("Meta 面=%v %v %v %v %v", cs, minLot, lotStep, tm, ok)
	}
	// VenueSizer 三面（guard 消费缝）.
	lots, err := s.ContractPos(ctx, "BTCUSD")
	if err != nil || lots != 0 {
		t.Fatalf("ContractPos 空仓=%v err=%v", lots, err)
	}
	if _, err := s.SpotAvailBal(ctx, "BTCUSD"); err == nil {
		t.Fatal("CFD 无现货腿应恒拒")
	}
}

func strptr(s string) *string { return &s }
