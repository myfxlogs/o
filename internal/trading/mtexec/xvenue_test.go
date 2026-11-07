// xvenue_test.go —— T-0145 mt5 执行件分类/仲裁桩：歧义码=unknown 走 Resolve
// 双面查；终局 inst 码（SYMBOL_NOT_FOUND/INVALID_SYMBOL/TRADE_DISABLED）=
// APIError.TerminalInst 入册锚；Resolve 开腿=OpenedOrders+OrderHistory 双面
// 查（history 单面查无≠未成——假 absent 病灶根修）；平腿=OpenedOrders
// ticket 判（票在=live 票消=filled）。
package mtexec

import (
	"context"
	"errors"
	"testing"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/trading/execution"

	"google.golang.org/grpc"
)

// fakeErrTrading 可编程拒件（T-0145 分类锚——不扰 fakeTrading 既有桩）。
type fakeErrTrading struct {
	mt5pb.TradingClient
	sendErr *mt5pb.Error
	sends   int
}

func (f *fakeErrTrading) OrderSend(ctx context.Context, in *mt5pb.OrderSendRequest, opts ...grpc.CallOption) (*mt5pb.OrderSendReply, error) {
	f.sends++
	return &mt5pb.OrderSendReply{Error: f.sendErr}, nil
}
func (f *fakeErrTrading) OrderClose(ctx context.Context, in *mt5pb.OrderCloseRequest, opts ...grpc.CallOption) (*mt5pb.OrderCloseReply, error) {
	return &mt5pb.OrderCloseReply{Error: f.sendErr}, nil
}

// fakeMT5Api OpenedOrders/OrderHistory 双面读桩。
type fakeMT5Api struct {
	mt5pb.MT5Client
	opened []*mt5pb.Order
	hist   []*mt5pb.Order
	err    error
}

func (f *fakeMT5Api) OpenedOrders(context.Context, *mt5pb.OpenedOrdersRequest, ...grpc.CallOption) (*mt5pb.OpenedOrdersReply, error) {
	return &mt5pb.OpenedOrdersReply{Result: f.opened}, f.err
}
func (f *fakeMT5Api) OrderHistory(context.Context, *mt5pb.OrderHistoryRequest, ...grpc.CallOption) (*mt5pb.OrderHistoryReply, error) {
	return &mt5pb.OrderHistoryReply{Result: f.hist}, f.err
}

type fakeCli2 struct{ api mt5pb.MT5Client }

func (f *fakeCli2) Authed(ctx context.Context) context.Context { return ctx }
func (f *fakeCli2) Conn() *grpc.ClientConn                     { return nil }
func (f *fakeCli2) MT5() mt5pb.MT5Client                       { return f.api }

var xvLeg = execution.Leg{Venue: "mt5", InstID: "BTCUSDm", Side: "sell", Sz: "0.5"}

// 锚⑦a：歧义码族（REQUEST_ERROR=网关确认超时栈真户实证）→State=unknown
// 非 absent——歧义禁当未成吞（裸腿险）；APIError.TerminalInst=false 禁入册。
func TestAmbiguousCodeUnknownNotAbsent(t *testing.T) {
	ft := &fakeErrTrading{sendErr: &mt5pb.Error{Code: mt5pb.ErrorCode_REQUEST_ERROR, Message: "gateway confirm timeout"}}
	e := Executor{C: &fakeCli{}, T: ft, X: dxIdentity()}
	res, err := e.Place(context.Background(), xvLeg, "ph1")
	if res.State != "unknown" || err == nil {
		t.Fatalf("REQUEST_ERROR 应 unknown+err: %+v %v", res, err)
	}
	var ae *APIError
	if !errors.As(err, &ae) || ae.TerminalInst {
		t.Fatalf("歧义码 TerminalInst 误置: %+v", ae)
	}
	if TerminalInstErr(err) {
		t.Fatal("歧义码不应入册")
	}
}

// 锚⑦b：终局 inst 码三钦定（SYMBOL_NOT_FOUND/INVALID_SYMBOL/TRADE_DISABLED）
// →absent+TerminalInst 入册锚；MARKET_CLOSED=市闭瞬态非终局禁入册。
func TestTerminalInstCodes(t *testing.T) {
	for _, code := range []mt5pb.ErrorCode{
		mt5pb.ErrorCode_SYMBOL_NOT_FOUND, mt5pb.ErrorCode_INVALID_SYMBOL,
		mt5pb.ErrorCode_TRADE_DISABLED} {
		ft := &fakeErrTrading{sendErr: &mt5pb.Error{Code: code, Message: "x"}}
		e := Executor{C: &fakeCli{}, T: ft, X: dxIdentity()}
		res, err := e.Place(context.Background(), xvLeg, "ph1")
		if res.State != "absent" {
			t.Fatalf("%s 应 absent: %+v", code, res)
		}
		var ae *APIError
		if !errors.As(err, &ae) || !ae.TerminalInst || ae.Code != code.String() {
			t.Fatalf("%s TerminalInst 位/码面: %+v %v", code, ae, err)
		}
		if !TerminalInstErr(err) {
			t.Fatalf("%s 应终局入册", code)
		}
	}
	// MARKET_CLOSED=市闭瞬态——absent 但 TerminalInst=false 禁入册
	ft := &fakeErrTrading{sendErr: &mt5pb.Error{Code: mt5pb.ErrorCode_MARKET_CLOSED, Message: "closed"}}
	e := Executor{C: &fakeCli{}, T: ft, X: dxIdentity()}
	_, err := e.Place(context.Background(), xvLeg, "ph1")
	if TerminalInstErr(err) {
		t.Fatal("MARKET_CLOSED 市闭瞬态误判终局入册")
	}
}

// 锚⑦c：Resolve 开腿双面——OpenedOrders 命中（假 absent 病灶根修：
// 仓在 open 集=已成仓）；history 补查（秒平残史）；双面皆无=absent。
func TestResolveOpenTwoFaced(t *testing.T) {
	// OpenedOrders 命中——comment=clOrdID 幂等键
	api := &fakeMT5Api{opened: []*mt5pb.Order{
		{Ticket: 888, Comment: "ph9x1x2", Symbol: "BTCUSDm", Lots: 0.5, OpenPrice: 90000}}}
	e := Executor{C: &fakeCli2{api: api}, X: dxIdentity()}
	res, err := e.Resolve(context.Background(), xvLeg, "ph9x1x2")
	if err != nil || res.State != "filled" || res.FillTicket != 888 || res.FillSz != 0.5 {
		t.Fatalf("OpenedOrders 命中: %+v %v", res, err)
	}
	// OpenedOrders 无→history 补查（开完秒平残史面）
	api.opened = nil
	api.hist = []*mt5pb.Order{{Ticket: 889, Comment: "ph9x1x2", Lots: 0.5, OpenPrice: 90000}}
	res, err = e.Resolve(context.Background(), xvLeg, "ph9x1x2")
	if err != nil || res.State != "filled" || res.FillTicket != 889 {
		t.Fatalf("history 补查: %+v %v", res, err)
	}
	// 双面皆无=absent（真未成交——可安标败）
	api.hist = nil
	res, err = e.Resolve(context.Background(), xvLeg, "ph9x1x2")
	if err != nil || res.State != "absent" {
		t.Fatalf("双面皆无应 absent: %+v %v", res, err)
	}
}

// 锚⑦d：resolveClose 平腿——OpenedOrders 按 ticket 判：票在=close 未落
// live；票消=平成 filled（comment 幂等键不可靠——T-0145 实证 ticket 唯一锚）。
func TestResolveCloseByTicket(t *testing.T) {
	api := &fakeMT5Api{opened: []*mt5pb.Order{{Ticket: 777, Symbol: "BTCUSDm", Lots: 0.5}}}
	e := Executor{C: &fakeCli2{api: api}, X: dxIdentity()}
	cl := execution.Leg{Venue: "mt5", InstID: "BTCUSDm", Close: true, Ticket: 777, Sz: "0.5"}
	res, err := e.Resolve(context.Background(), cl, "rb1")
	if err != nil || res.State != "live" {
		t.Fatalf("票在应 live（close 未落）: %+v %v", res, err)
	}
	api.opened = nil
	res, err = e.Resolve(context.Background(), cl, "rb1")
	if err != nil || res.State != "filled" {
		t.Fatalf("票消应 filled（close 已成）: %+v %v", res, err)
	}
}
