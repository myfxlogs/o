// Package mtexec —— MT5 执行适配器（mtapi Trading OrderSend/OrderClose 包
// execution.Executor——跨 venue 双腿的 cfd 腿走此径；mutating 卡 demo 全谱
// 在案，真户=决策席手动 openfirst 授权面，歧态势=unknown 禁补单同 okx 纪律）。
package mtexec

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"strconv"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/trading/execution"
)

// Cli mt5 会话件（authed ctx 携 token——executor 侧不持凭据）。
type Cli interface {
	Authed(ctx context.Context) context.Context
	Conn() *grpc.ClientConn
	MT5() mt5pb.MT5Client
}

// Dialect 官话→venue 符号方言译件窄面（T-0432④ 翻译钉 venue IO 缝——腿携
// 官名，OrderSend 前查表译方言；查无映射=升件（Escalate）+err fail-closed，
// 禁静默 fallback 禁官名硬发。产线=mt5.VenueTranslator（fin_symbol_dialect
// 仓）；仿真层同缝可设桩——T-0426 缝位同款）。
type Dialect interface {
	VenueSymbol(ctx context.Context, canonical string) (string, error)
	Escalate(ctx context.Context, canonical, reason string)
}

func (e Executor) trading() mt5pb.TradingClient {
	if e.T != nil {
		return e.T
	}
	return mt5pb.NewTradingClient(e.C.Conn())
}

// Executor MT5 venue 执行件。
type Executor struct {
	C Cli
	T mt5pb.TradingClient // 测试注假面（产线 nil→自构）
	X Dialect             // 方言译件（T-0432④ 必置——nil=装配错拒单：无译件即无从守官名唯一天）
}

// translate 官名→venue 方言名（fail-closed 核——查无映射=升件落账+err 拒单，
// 唯一合法出路；X 缺=装配错同拒）。
func (e Executor) translate(ctx context.Context, canonical string) (string, error) {
	if e.X == nil {
		return "", fmt.Errorf("mt5 方言译件缺——装配错 fail-closed 拒单（官名硬发=禁线）")
	}
	sym, err := e.X.VenueSymbol(ctx, canonical)
	if err != nil {
		e.X.Escalate(ctx, canonical, err.Error())
		return "", fmt.Errorf("mt5 方言查无映射 fail-closed（升件在案）: %w", err)
	}
	return sym, nil
}

// Place OrderSend 市价单（BUY/SELL；回报空/错=unknown|absent 定态分流）。
// 腿携官名（canonical inst_id——官话唯一天），OrderSend 前方言查表译 venue
// 名（查无=升件 fail-closed 禁官名硬发——T-0432④ 缝钉）。
func (e Executor) Cancel(_ context.Context, _ execution.Leg, _ string) error {
	return fmt.Errorf("maker 径不支持（MT5 无 post_only——OKX 专径）")
}

func (e Executor) Place(ctx context.Context, leg execution.Leg, clOrdID string) (execution.OrderResult, error) {
	if leg.Close {
		return e.placeClose(ctx, leg, clOrdID)
	}
	sym, terr := e.translate(ctx, leg.InstID)
	if terr != nil {
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, terr
	}
	op := mt5pb.OrderType_OrderType_Buy
	if leg.Side == "sell" {
		op = mt5pb.OrderType_OrderType_Sell
	}
	vol, _ := strconv.ParseFloat(leg.Sz, 64)
	r, err := e.trading().OrderSend(e.C.Authed(ctx), &mt5pb.OrderSendRequest{
		Symbol: sym, Operation: op, Volume: vol, Comment: &clOrdID})
	if err != nil {
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
	}
	if ge := r.GetError(); ge != nil && ge.GetCode() != mt5pb.ErrorCode_DONE && ge.GetCode() != mt5pb.ErrorCode_OK {
		return execution.OrderResult{ClOrdID: clOrdID, State: "absent"}, fmt.Errorf("mt5 send: %s", ge.GetCode())
	}
	if ord := r.GetResult(); ord != nil {
		return execution.OrderResult{ClOrdID: clOrdID, State: "filled",
			FillPx: ord.GetOpenPrice(), FillSz: vol, FillTicket: ord.GetTicket()}, nil
	}
	return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, nil
}

// placeClose 平仓径（T-0036 根修：hedge 户「反向单=新开仓」自欺——
// 平仓必走 OrderClose 携 ticket；ticket 缺=unknown fail-closed 禁反向新开）。
func (e Executor) placeClose(ctx context.Context, leg execution.Leg, clOrdID string) (execution.OrderResult, error) {
	if leg.Ticket == 0 {
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"},
			fmt.Errorf("mt5 平仓缺 ticket——fail-closed 禁反向单")
	}
	vol, _ := strconv.ParseFloat(leg.Sz, 64)
	r, err := e.trading().OrderClose(e.C.Authed(ctx), &mt5pb.OrderCloseRequest{
		Ticket: leg.Ticket, Lots: &vol})
	if err != nil {
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, fmt.Errorf("mt5 close: %w", err)
	}
	if ge := r.GetError(); ge != nil && ge.GetCode() != mt5pb.ErrorCode_DONE && ge.GetCode() != mt5pb.ErrorCode_OK {
		return execution.OrderResult{ClOrdID: clOrdID, State: "absent"}, fmt.Errorf("mt5 close: %s", ge.GetCode())
	}
	return execution.OrderResult{ClOrdID: clOrdID, State: "filled", FillSz: vol}, nil
}

// Resolve 按 comment 查仓态（MT5 无 clOrdId 查询——comment 留痕+OpenedOrders 补查。
// 歧态势=unknown 禁补单：查不到 open 且历史无=unknown 升级留痕）。
func (e Executor) Resolve(ctx context.Context, leg execution.Leg, clOrdID string) (execution.OrderResult, error) {
	r, err := e.C.MT5().OrderHistory(e.C.Authed(ctx), &mt5pb.OrderHistoryRequest{})
	if err != nil {
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
	}
	for _, o := range r.GetResult() {
		if o.GetComment() == clOrdID {
			return execution.OrderResult{ClOrdID: clOrdID, State: "filled",
				FillPx: o.GetOpenPrice(), FillSz: float64(o.GetVolume())}, nil
		}
	}
	return execution.OrderResult{ClOrdID: clOrdID, State: "absent"}, nil
}
