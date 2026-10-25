package mtexec

import (
	"context"
	"testing"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/trading/execution"

	"google.golang.org/grpc"
)

type fakeCli struct{}

func (f *fakeCli) Authed(ctx context.Context) context.Context { return ctx }
func (f *fakeCli) Conn() *grpc.ClientConn                     { return nil }
func (f *fakeCli) MT5() mt5pb.MT5Client                       { return nil }

type fakeTrading struct {
	sendCalls, closeCalls int
	lastTicket            int64
}

func (f *fakeTrading) OrderSend(ctx context.Context, in *mt5pb.OrderSendRequest, opts ...grpc.CallOption) (*mt5pb.OrderSendReply, error) {
	f.sendCalls++
	return &mt5pb.OrderSendReply{Result: &mt5pb.Order{Ticket: 777, OpenPrice: 90000}}, nil
}
func (f *fakeTrading) OrderModify(ctx context.Context, in *mt5pb.OrderModifyRequest, opts ...grpc.CallOption) (*mt5pb.OrderModifyReply, error) {
	return &mt5pb.OrderModifyReply{}, nil
}
func (f *fakeTrading) OrderClose(ctx context.Context, in *mt5pb.OrderCloseRequest, opts ...grpc.CallOption) (*mt5pb.OrderCloseReply, error) {
	f.closeCalls++
	f.lastTicket = in.GetTicket()
	return &mt5pb.OrderCloseReply{}, nil
}

// T-0036 hedge 平仓语义锚：Close=true→OrderClose 携 ticket 非 OrderSend；
// ticket 缺=unknown fail-closed 禁反向新开。
func TestCloseSemantics(t *testing.T) {
	ft := &fakeTrading{}
	// Executor.trading() 内部径——注假 Cli 须改实现口：直接测 placeClose 走 OrderClose。
	e := Executor{C: &fakeCli{}, T: ft}
	// 开仓腿→OrderSend
	res, err := e.Place(context.Background(), execution.Leg{Venue: "mt5", InstID: "BTCUSDm", Side: "buy", Sz: "0.01"}, "c1")
	if err != nil || res.State != "filled" || res.FillTicket != 777 {
		t.Fatalf("open: %+v %v", res, err)
	}
	if ft.sendCalls != 1 {
		t.Fatal("OrderSend 未调")
	}
	// 平仓腿→OrderClose 携 ticket
	res, err = e.Place(context.Background(), execution.Leg{Venue: "mt5", InstID: "BTCUSDm", Side: "sell", Sz: "0.01", Close: true, Ticket: 777}, "c2")
	if err != nil || res.State != "filled" {
		t.Fatalf("close: %+v %v", res, err)
	}
	if ft.closeCalls != 1 || ft.lastTicket != 777 {
		t.Fatalf("OrderClose 未携 ticket: calls=%d ticket=%d", ft.closeCalls, ft.lastTicket)
	}
	// ticket 缺=unknown fail-closed 禁反向新开
	res, err = e.Place(context.Background(), execution.Leg{Venue: "mt5", InstID: "BTCUSDm", Side: "sell", Sz: "0.01", Close: true}, "c3")
	if err == nil || res.State != "unknown" {
		t.Fatalf("缺 ticket 应 unknown: %+v %v", res, err)
	}
}
