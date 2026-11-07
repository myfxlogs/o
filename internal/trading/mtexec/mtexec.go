// Package mtexec —— MT5 执行适配器（mtapi Trading OrderSend/OrderClose 包
// execution.Executor——跨 venue 双腿的 cfd 腿走此径；mutating 卡 demo 全谱
// 在案，真户=决策席手动 openfirst 授权面，歧态势=unknown 禁补单同 okx 纪律）。
//
// T-0145 跨所 carry 真执行三件根修：
// ① venue 拒码分类 APIError{Code,Msg,TerminalInst}——inst 级终局
//
//	（SYMBOL_NOT_FOUND/INVALID_SYMBOL/TRADE_DISABLED）→fin_venue_untradeable
//	入册锚；MARKET_CLOSED/REQUOTE 族=瞬态非终局禁入册；
//
// ② 歧义码集——REQUOTE*/PRICE_CHANGED/REQUEST_*/TIMEOUT/SERVER_BUSY/
//
//	NO_PRICES 等瞬态/处理态→unknown 走 Resolve 定真态（单或已建未定=歧态
//	势禁补单），禁一码躺 absent 误报未成交；
//
// ③ Resolve 双面——开腿仓在 OpenedOrders（comment 幂等键）非仅
//
//	OrderHistory（历史面查不到=假 absent 病灶）；平腿走 OpenedOrders 按
//	ticket 判定（票在=未平成 live，票消=平成 filled）。
package mtexec

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/trading/execution"
)

// Cli mt5 会话件（authed ctx 携 token——executor 侧不持凭据）。
// 产线=cmd 懒连会话件：可选实现 Suspecter——RPC 传输败标失效下拍重连。
type Cli interface {
	Authed(ctx context.Context) context.Context
	Conn() *grpc.ClientConn
	MT5() mt5pb.MT5Client
}

// Suspecter 会话失效标记面（懒连会话件实现——executor 在 RPC 败径调用，
// 下次取用重连自愈；未实现=零行为兼容）。
type Suspecter interface{ Suspect() }

// Dialect 官话→venue 符号方言译件窄面（T-0432④ 翻译钉 venue IO 缝——腿携
// 官名，OrderSend 前查表译方言；查无映射=升件（Escalate）+err fail-closed，
// 禁静默 fallback 禁官名硬发。产线=mt5.VenueTranslator（fin_symbol_dialect
// 仓）；仿真层同缝可设桩——T-0426 缝位同款）。
type Dialect interface {
	VenueSymbol(ctx context.Context, canonical string) (string, error)
	Escalate(ctx context.Context, canonical, reason string)
}

// APIError venue 业务拒（mtapi Error 载荷——Code=mt5pb 枚举串、Msg=venue
// 拒文；TerminalInst=inst 级终局位——T-0145 untradeable 联动分类锚）。
type APIError struct {
	Code         string
	Msg          string
	TerminalInst bool
}

func (e *APIError) Error() string { return fmt.Sprintf("mt5 %s: %s", e.Code, e.Msg) }

// TerminalInstErr mt5 腿 inst 级终局判定（T-0145——SYMBOL_NOT_FOUND/
// INVALID_SYMBOL/TRADE_DISABLED 三码钦定入册锚；瞬态族（MARKET_CLOSED/
// REQUOTE/REQUEST_ERROR 等）/参数族（INVALID_VOLUME/STOPS/PRICE）/资金族
// （NO_MONEY）皆禁入册——市闭=会话复开非永拒，参数=我方病灶非 venue 事实）。
func TerminalInstErr(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.TerminalInst
}

// ambiguousCodes 歧义/瞬态码集——OrderSend 回报此族=单或已建未定态→
// State=unknown 走 Resolve 双面查定真态（真户实证族：REQUEST_ERROR=
// MarketOpenWaiterAsync 网关确认超时栈——退避重试即过，实未成交；但
// 超时窗内可能已成交=歧态势，Resolve 仲裁唯一正径）。价漂/重报族
// （REQUOTE*/PRICE_CHANGED）同归歧义——拒因是价非单，旧码躺 absent
// 吞歧义=裸腿险。
var ambiguousCodes = map[mt5pb.ErrorCode]bool{
	mt5pb.ErrorCode_REQUEST_ERROR: true, mt5pb.ErrorCode_REQUEST_TIMEOUT: true,
	mt5pb.ErrorCode_REQUEST_ACCEPTED: true, mt5pb.ErrorCode_REQUEST_ON_WAY: true,
	mt5pb.ErrorCode_REQUEST_RETURNED_IN_QUEUE:           true,
	mt5pb.ErrorCode_REQUEST_REJECTED_TO_ABSENCE_CONNECT: true,
	mt5pb.ErrorCode_REQUOTE:                             true, mt5pb.ErrorCode_REQUOTE_EX: true,
	mt5pb.ErrorCode_PRICE_CHANGED: true, mt5pb.ErrorCode_NO_PRICES: true,
	mt5pb.ErrorCode_TIMEOUT: true, mt5pb.ErrorCode_SERVER_BUSY: true,
	mt5pb.ErrorCode_ANOTHER_PROCESS_IN_PROGRESS: true,
	mt5pb.ErrorCode_NO_CONNECTION:               true, mt5pb.ErrorCode_CONNECTION_NOT_PERMITTED: true,
}

// terminalInstCodes inst 级终局码集（untradeable 入册锚——三码钦定：
// 符号不存在/非法=目录终局；TRADE_DISABLED=broker 关停非瞬态市闭）。
var terminalInstCodes = map[mt5pb.ErrorCode]bool{
	mt5pb.ErrorCode_SYMBOL_NOT_FOUND: true, mt5pb.ErrorCode_INVALID_SYMBOL: true,
	mt5pb.ErrorCode_TRADE_DISABLED: true,
}

// apiErr mtapi Error 载荷→APIError（终局位/歧义判定同源——Code.String()
// 枚举串为唯一语义载体，禁字符串匹配拒文）。
func apiErr(ge *mt5pb.Error) *APIError {
	code := ge.GetCode()
	return &APIError{Code: code.String(), Msg: ge.GetMessage(),
		TerminalInst: terminalInstCodes[code]}
}

func (e Executor) trading() mt5pb.TradingClient {
	if e.T != nil {
		return e.T
	}
	return mt5pb.NewTradingClient(e.C.Conn())
}

// suspect 会话失效标记（RPC 传输败径调用——懒连会话件下拍重连自愈；
// Cli 未实现=零行为）。
func (e Executor) suspect() {
	if s, ok := e.C.(Suspecter); ok {
		s.Suspect()
	}
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
// T-0145：venue 拒码携 APIError 终局位上浮（untradeable 联动分类锚——
// 装配层 errors.As 取 *APIError 判 TerminalInst 入册）。
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
		e.suspect() // 传输败=会话疑似失效——懒连件下拍重连自愈
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
	}
	if ge := r.GetError(); ge != nil && ge.GetCode() != mt5pb.ErrorCode_DONE && ge.GetCode() != mt5pb.ErrorCode_OK {
		ae := apiErr(ge)
		state := "absent"
		if ambiguousCodes[ge.GetCode()] {
			state = "unknown" // T-0145：歧义码=单或已建未定态——Resolve 仲裁唯一正径
		}
		return execution.OrderResult{ClOrdID: clOrdID, State: state}, ae
	}
	if ord := r.GetResult(); ord != nil {
		return execution.OrderResult{ClOrdID: clOrdID, State: "filled",
			FillPx: ord.GetOpenPrice(), FillSz: vol, FillTicket: ord.GetTicket()}, nil
	}
	return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, nil
}

// placeClose 平仓径（T-0036 根修：hedge 户「反向单=新开仓」自欺——
// 平仓必走 OrderClose 携 ticket；ticket 缺=unknown fail-closed 禁反向新开。
// 真户实证：平必带 Lots 字段——缺 Lots=「Request processing error」瞬态
// 拒族；开完即平竞态=败后 OpenedOrders 仲裁残仓禁信报错=未成交）。
func (e Executor) placeClose(ctx context.Context, leg execution.Leg, clOrdID string) (execution.OrderResult, error) {
	if leg.Ticket == 0 {
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"},
			fmt.Errorf("mt5 平仓缺 ticket——fail-closed 禁反向单")
	}
	vol, _ := strconv.ParseFloat(leg.Sz, 64)
	r, err := e.trading().OrderClose(e.C.Authed(ctx), &mt5pb.OrderCloseRequest{
		Ticket: leg.Ticket, Lots: &vol})
	if err != nil {
		e.suspect()
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, fmt.Errorf("mt5 close: %w", err)
	}
	if ge := r.GetError(); ge != nil && ge.GetCode() != mt5pb.ErrorCode_DONE && ge.GetCode() != mt5pb.ErrorCode_OK {
		ae := apiErr(ge)
		state := "absent"
		if ambiguousCodes[ge.GetCode()] {
			state = "unknown" // 平侧歧义同律——ResolvedOrders 仲裁定真态
		}
		return execution.OrderResult{ClOrdID: clOrdID, State: state}, ae
	}
	return execution.OrderResult{ClOrdID: clOrdID, State: "filled", FillSz: vol}, nil
}

// Resolve 歧态势定真态双面查（T-0145 根修——原仅 OrderHistory 单面，开腿
// 仓在 OpenedOrders 集=假 absent 病灶；平腿=OpenedOrders 按 ticket 判：
// 票在=未平成 live，票消=平成 filled）。
// 开腿：comment=clOrdID 幂等键——OpenedOrders 先查（在仓=已成仓）→
// OrderHistory 补查（秒平残史）；两面皆无=absent。歧义=unknown 禁补单。
func (e Executor) Resolve(ctx context.Context, leg execution.Leg, clOrdID string) (execution.OrderResult, error) {
	if leg.Close {
		return e.resolveClose(ctx, leg, clOrdID)
	}
	or, err := e.C.MT5().OpenedOrders(e.C.Authed(ctx), &mt5pb.OpenedOrdersRequest{})
	if err != nil {
		e.suspect()
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
	}
	for _, o := range or.GetResult() {
		if o.GetComment() == clOrdID {
			return execution.OrderResult{ClOrdID: clOrdID, State: "filled",
				FillPx: o.GetOpenPrice(), FillSz: o.GetLots(), FillTicket: o.GetTicket()}, nil
		}
	}
	hr, err := e.C.MT5().OrderHistory(e.C.Authed(ctx), &mt5pb.OrderHistoryRequest{})
	if err != nil {
		e.suspect()
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
	}
	for _, o := range hr.GetResult() {
		if o.GetComment() == clOrdID {
			return execution.OrderResult{ClOrdID: clOrdID, State: "filled",
				FillPx: o.GetOpenPrice(), FillSz: o.GetLots(), FillTicket: o.GetTicket()}, nil
		}
	}
	return execution.OrderResult{ClOrdID: clOrdID, State: "absent"}, nil
}

// resolveClose 平腿歧义仲裁——OpenedOrders 按 ticket 定真态（票在=close
// 未落 live；票消=close 已成 filled）。comment 幂等键不可靠（OrderClose
// 回报注释域非委托键——T-0145 实证面：ticket 唯一锚）。
func (e Executor) resolveClose(ctx context.Context, leg execution.Leg, clOrdID string) (execution.OrderResult, error) {
	or, err := e.C.MT5().OpenedOrders(e.C.Authed(ctx), &mt5pb.OpenedOrdersRequest{})
	if err != nil {
		e.suspect()
		return execution.OrderResult{ClOrdID: clOrdID, State: "unknown"}, err
	}
	for _, o := range or.GetResult() {
		if o.GetTicket() == leg.Ticket {
			return execution.OrderResult{ClOrdID: clOrdID, State: "live"}, nil
		}
	}
	vol, _ := strconv.ParseFloat(leg.Sz, 64)
	return execution.OrderResult{ClOrdID: clOrdID, State: "filled", FillSz: vol}, nil
}
