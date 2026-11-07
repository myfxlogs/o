package mtexec

import (
	"context"
	"fmt"
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
	lastSendSymbol        string
}

func (f *fakeTrading) OrderSend(ctx context.Context, in *mt5pb.OrderSendRequest, opts ...grpc.CallOption) (*mt5pb.OrderSendReply, error) {
	f.sendCalls++
	f.lastSendSymbol = in.GetSymbol()
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

// fakeDialect 方言译件桩（T-0432④——mapping 查表/查无=err；escCalls 升件
// 断言面）。
type fakeDialect struct {
	mapping  map[string]string
	escCalls []string
}

func (f *fakeDialect) VenueSymbol(_ context.Context, canonical string) (string, error) {
	if s, ok := f.mapping[canonical]; ok {
		return s, nil
	}
	return "", fmt.Errorf("查无映射: %s", canonical)
}
func (f *fakeDialect) Escalate(_ context.Context, canonical, reason string) {
	f.escCalls = append(f.escCalls, canonical+"|"+reason)
}

// dxIdentity 全同映射桩（历史语义腿=venue 名直用——T-0036 既有锚零扰动）。
func dxIdentity() *fakeDialect {
	return &fakeDialect{mapping: map[string]string{"BTCUSDm": "BTCUSDm"}}
}

// T-0036 hedge 平仓语义锚：Close=true→OrderClose 携 ticket 非 OrderSend；
// ticket 缺=unknown fail-closed 禁反向新开。
func TestCloseSemantics(t *testing.T) {
	ft := &fakeTrading{}
	// Executor.trading() 内部径——注假 Cli 须改实现口：直接测 placeClose 走 OrderClose。
	e := Executor{C: &fakeCli{}, T: ft, X: dxIdentity()}
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

// T-0432④ 翻译缝全环锚：腿携官名→OrderSend 符号=方言（venue 实报名）——
// 官话唯一天：内部件只见 canonical，venue 边界缝内译。
func TestDialectTranslateFullLoop(t *testing.T) {
	ft := &fakeTrading{}
	dx := &fakeDialect{mapping: map[string]string{"BTC-USDT-SWAP": "BTCUSDm"}}
	e := Executor{C: &fakeCli{}, T: ft, X: dx}
	res, err := e.Place(context.Background(),
		execution.Leg{Venue: "mt5", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: "0.01"}, "c1")
	if err != nil || res.State != "filled" {
		t.Fatalf("place: %+v %v", res, err)
	}
	if ft.lastSendSymbol != "BTCUSDm" {
		t.Fatalf("OrderSend 符号非方言译名: %q", ft.lastSendSymbol)
	}
	if len(dx.escCalls) != 0 {
		t.Fatalf("映射在位不应升件: %v", dx.escCalls)
	}
}

// T-0432④ 红桩①（断言红桩）：查无映射调用→升件落账+venue IO 零触（不入闸
// ——禁官名硬发禁静默 fallback 的构造性证）。
func TestDialectMissingMappingFailClosed(t *testing.T) {
	ft := &fakeTrading{}
	dx := &fakeDialect{mapping: map[string]string{}} // 空映射=查无
	e := Executor{C: &fakeCli{}, T: ft, X: dx}
	res, err := e.Place(context.Background(),
		execution.Leg{Venue: "mt5", InstID: "GOLD-ALIASED", Side: "buy", Sz: "0.01"}, "c1")
	if err == nil {
		t.Fatal("查无映射未拒单——fail-closed 破防")
	}
	if res.State != "unknown" {
		t.Fatalf("态=%s——拒单应 unknown", res.State)
	}
	if ft.sendCalls != 0 {
		t.Fatalf("venue IO 已触（sendCalls=%d）——不入闸红桩破", ft.sendCalls)
	}
	if len(dx.escCalls) != 1 || dx.escCalls[0] != "GOLD-ALIASED|查无映射: GOLD-ALIASED" {
		t.Fatalf("升件未落: %v", dx.escCalls)
	}
}

// T-0432④ 译件缺=装配错拒单（nil X——官名硬发禁线的构造面）。
func TestDialectNilTranslatorRejected(t *testing.T) {
	ft := &fakeTrading{}
	e := Executor{C: &fakeCli{}, T: ft}
	if _, err := e.Place(context.Background(),
		execution.Leg{Venue: "mt5", InstID: "BTCUSDm", Side: "buy", Sz: "0.01"}, "c1"); err == nil {
		t.Fatal("译件缺未拒——官名硬发位敞开")
	}
	if ft.sendCalls != 0 {
		t.Fatalf("venue IO 已触: %d", ft.sendCalls)
	}
}
