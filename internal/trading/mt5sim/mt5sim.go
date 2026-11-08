// Package mt5sim —— MT5 CFD 仿真 venue（T-0461 simvenue 矩阵跨所格——
// venue 语义纸面跑：ticket 生命周期/OrderClose 平仓/OpenedOrders 在仓读/
// OrderHistory 残史/SymbolParams 乐器面全产线同缝）。缝面/真相读面见
// mt5sim_seam.go（行数棘轮拆件）。
//
// 只替 venue IO 缝：实现 mt5pb.TradingClient（下单面三方法）+
// mt5pb.MT5Client 三读面（OpenedOrders/OrderHistory/SymbolParams——嵌入
// nil 接口面，他法不触即不炸）+mtexec.Cli/Dialect/guard.VenueSizer 三缝
// 结构化满足。消费件（mtexec.Executor 腿序/回滚/ticket 透传/fin_order_leg
// 留痕）一律产线同码——禁影子实现。
//
// 拒码=mt5pb.ErrorCode 实枚举（真 venue 方言——mtexec.apiErr 分类锚同源：
// SYMBOL_NOT_FOUND/TRADE_DISABLED=inst 级终局入册锚、INVALID_VOLUME=参数族
// 禁入册、MARKET_CLOSED=瞬态、POSITION_NOT_EXISTS=平侧查无）。拒单槽序
// （确定性写死——首命中即拒）：
//
//	① symbol 不在册 → ② TradeMode≠FullAccess/市闭 → ③ volume 网格
//	（≤0/<minLot/不落 lotStep） → ④ 平侧 ticket/volume 查无
package mt5sim

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// Instrument CFD 乐器面（fin_symbol_dialect 实约同词汇——ct_val=1 手
// base 币量/min_sz=最小手数/lot_step=手数步长/trade_mode=SymGroup。
// TradeMode 取 mt5pb.TradeMode 数值（4=FullAccess——BuildXvenueLegs
// 可开性预检同判据）；State=live|closed（MARKET_CLOSED 瞬态注入位）。
type Instrument struct {
	VenueSymbol  string  // broker 方言名（venue 边界词——OrderSend 收此键）
	Canonical    string  // 官话 inst_id（内部仓位/台账唯一用名——对账键）
	ContractSize float64 // ct_val（1 手 base 币量）
	MinLot       float64
	LotStep      float64
	TradeMode    int32   // mt5pb.TradeMode 数值
	Px           float64 // 仿真成交价（标的资产价——CFD 跟标的）
	State        string  // live|closed
}

// Fill CFD 成交流水（venue 内部真相——对账源；Close=true=平仓腿）。
type Fill struct {
	Ticket    int64
	Canonical string
	Side      string // buy|sell
	Lots      float64
	Px        float64
	Close     bool
	Ts        time.Time
}

type position struct {
	ticket    int64
	symbol    string
	canonical string
	side      mt5pb.OrderType
	lots      float64 // 在仓手数（部分平递减）
	openLots  float64 // 开仓手数（残史 Lots 面）
	openPx    float64
	comment   string
	openTs    time.Time
}

// Sim MT5 CFD 仿真 venue（内存态——仓位/ticket 序/历史/流水全自持）。
type Sim struct {
	mt5pb.MT5Client // 嵌入 nil 面——未覆写方法调用即炸（禁触即安全）

	mu         sync.Mutex
	instr      map[string]*Instrument // venue_symbol→乐器
	canon      map[string]string      // canonical→venue_symbol
	positions  map[int64]*position    // ticket→在仓
	history    []*mt5pb.Order         // 平仓残史（comment=clOrdID 幂等键保留）
	nextTicket int64
	fills      []Fill
	bal        float64 // 纸面权益 USDT（保证金域记账面——open 扣名义全额占用，close 回补）
	escalated  []string
}

// New 仿真 venue 构造。
func New() *Sim {
	return &Sim{
		instr:      map[string]*Instrument{},
		canon:      map[string]string{},
		positions:  map[int64]*position{},
		nextTicket: 700001, // 仿真 ticket 域（真 venue ticket 空间不撞——审计可辨）
	}
}

// WithInstrument 乐器入册（链式——canonical↔venue_symbol 双键互查）。
func (s *Sim) WithInstrument(i Instrument) *Sim {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i.State == "" {
		i.State = "live"
	}
	cp := i
	s.instr[i.VenueSymbol] = &cp
	if i.Canonical != "" {
		s.canon[i.Canonical] = i.VenueSymbol
	}
	return s
}

// WithBalance 纸面权益播种（USDT——CFD 保证金域简化全额占用）。
func (s *Sim) WithBalance(usd float64) *Sim {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bal = usd
	return s
}

func sendErr(code mt5pb.ErrorCode, msg string) *mt5pb.OrderSendReply {
	return &mt5pb.OrderSendReply{Error: &mt5pb.Error{Code: code, Message: msg}}
}

func closeErr(code mt5pb.ErrorCode, msg string) *mt5pb.OrderCloseReply {
	return &mt5pb.OrderCloseReply{Error: &mt5pb.Error{Code: code, Message: msg}}
}

// OrderSend 开仓市价单（mt5pb.TradingClient 面——拒单槽序①②③）。
func (s *Sim) OrderSend(_ context.Context, req *mt5pb.OrderSendRequest, _ ...grpc.CallOption) (*mt5pb.OrderSendReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sym := req.GetSymbol()
	in, ok := s.instr[sym]
	if !ok {
		return sendErr(mt5pb.ErrorCode_SYMBOL_NOT_FOUND,
			fmt.Sprintf("symbol %s not found", sym)), nil
	}
	if in.State != "live" {
		return sendErr(mt5pb.ErrorCode_MARKET_CLOSED,
			fmt.Sprintf("market closed: %s state=%s", sym, in.State)), nil
	}
	if in.TradeMode != 4 { // FullAccess——可开性断言（BuildXvenueLegs 预检同判据双保险）
		return sendErr(mt5pb.ErrorCode_TRADE_DISABLED,
			fmt.Sprintf("trade disabled: %s TradeMode=%d", sym, in.TradeMode)), nil
	}
	vol := req.GetVolume()
	if vol <= 0 || vol < in.MinLot ||
		(in.LotStep > 0 && math.Abs(vol/in.LotStep-math.Round(vol/in.LotStep)) > 1e-6) {
		return sendErr(mt5pb.ErrorCode_INVALID_VOLUME,
			fmt.Sprintf("invalid volume %.8g（min=%.8g step=%.8g）", vol, in.MinLot, in.LotStep)), nil
	}
	// 资金判（保证金域简化——名义全额占用纸面钱；不足=NO_MONEY 参数族拒）。
	notional := vol * in.ContractSize * in.Px
	if s.bal < notional {
		return sendErr(mt5pb.ErrorCode_NO_MONEY,
			fmt.Sprintf("no money: need %.4f have %.4f", notional, s.bal)), nil
	}
	s.bal -= notional
	ticket := s.nextTicket
	s.nextTicket++
	op := req.GetOperation()
	p := &position{ticket: ticket, symbol: sym, canonical: in.Canonical,
		side: op, lots: vol, openLots: vol, openPx: in.Px,
		comment: req.GetComment(), openTs: time.Now()}
	s.positions[ticket] = p
	s.fills = append(s.fills, Fill{Ticket: ticket, Canonical: in.Canonical,
		Side: sideName(op), Lots: vol, Px: in.Px, Ts: p.openTs})
	return &mt5pb.OrderSendReply{Result: &mt5pb.Order{
		Ticket: ticket, Symbol: sym, OpenPrice: in.Px, Lots: vol,
		Comment: p.comment, OrderType: op, ContractSize: in.ContractSize,
		OpenTime: timestamppb.New(p.openTs)}}, nil
}

// OrderModify 挂单修改面（市价单域无挂单——幂等 OK 回执：产线径不触，
// 实现面备齐防 nil 接口裸奔）。
func (s *Sim) OrderModify(_ context.Context, _ *mt5pb.OrderModifyRequest, _ ...grpc.CallOption) (*mt5pb.OrderModifyReply, error) {
	return &mt5pb.OrderModifyReply{Error: &mt5pb.Error{Code: mt5pb.ErrorCode_OK, Message: "sim no-op"}}, nil
}

// OrderClose 平仓（mt5pb.TradingClient 面——ticket 必携；拒单槽序④：
// ticket 查无=POSITION_NOT_EXISTS、lots>持=INVALID_VOLUME、lots≤0=INVALID_VOLUME；
// 部分平=减仓留仓、全平=移 history 残史（comment 幂等键保留——Resolve
// OrderHistory 面命中锚））。
func (s *Sim) OrderClose(_ context.Context, req *mt5pb.OrderCloseRequest, _ ...grpc.CallOption) (*mt5pb.OrderCloseReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.positions[req.GetTicket()]
	if !ok {
		return closeErr(mt5pb.ErrorCode_POSITION_NOT_EXISTS,
			fmt.Sprintf("position %d not exists", req.GetTicket())), nil
	}
	lots := req.GetLots()
	if lots <= 0 || lots > p.lots+1e-9 {
		return closeErr(mt5pb.ErrorCode_INVALID_VOLUME,
			fmt.Sprintf("close volume %.8g exceeds position %.8g", lots, p.lots)), nil
	}
	in := s.instr[p.symbol]
	px := in.Px
	p.lots -= lots
	s.bal += lots * in.ContractSize * px // 平仓回补名义（简化保证金域——P&L 零和）
	s.fills = append(s.fills, Fill{Ticket: p.ticket, Canonical: p.canonical,
		Side: sideName(p.side), Lots: lots, Px: px, Close: true, Ts: time.Now()})
	if p.lots <= 1e-9 {
		delete(s.positions, p.ticket)
		s.history = append(s.history, &mt5pb.Order{
			Ticket: p.ticket, Symbol: p.symbol, OpenPrice: p.openPx,
			Lots: p.openLots, Comment: p.comment, OrderType: p.side,
			ContractSize: in.ContractSize, ClosePrice: px, CloseLots: lots,
			CloseTime: timestamppb.New(time.Now())})
	}
	return &mt5pb.OrderCloseReply{Result: &mt5pb.Order{
		Ticket: p.ticket, Symbol: p.symbol, ClosePrice: px,
		CloseLots: lots, CloseTime: timestamppb.New(time.Now())}}, nil
}

// OpenedOrders 在仓读面（mtexec.Resolve 开腿/平腿双面仲裁+Mt5Sizer
// ContractPos 现读——comment=clOrdID 幂等键/票号双锚）。
func (s *Sim) OpenedOrders(_ context.Context, _ *mt5pb.OpenedOrdersRequest, _ ...grpc.CallOption) (*mt5pb.OpenedOrdersReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := &mt5pb.OpenedOrdersReply{}
	for _, p := range s.positions {
		out.Result = append(out.Result, &mt5pb.Order{
			Ticket: p.ticket, Symbol: p.symbol, OpenPrice: p.openPx,
			Lots: p.lots, Comment: p.comment, OrderType: p.side,
			ContractSize: s.instr[p.symbol].ContractSize,
			OpenTime:     timestamppb.New(p.openTs)})
	}
	sort.Slice(out.Result, func(i, j int) bool { return out.Result[i].GetTicket() < out.Result[j].GetTicket() })
	return out, nil
}

// OrderHistory 残史读面（mtexec.Resolve 开腿秒平残史命中面——comment
// 幂等键保留）。
func (s *Sim) OrderHistory(_ context.Context, _ *mt5pb.OrderHistoryRequest, _ ...grpc.CallOption) (*mt5pb.OrderHistoryReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := &mt5pb.OrderHistoryReply{Result: append([]*mt5pb.Order{}, s.history...)}
	return out, nil
}

// SymbolParams 乐器面读（产线 xvenueOpen TradeMode 活读同缝——SymGroup
// TradeMode/MinLots/LotsStep+SymbolInfo ContractSize 三面实填）。
func (s *Sim) SymbolParams(_ context.Context, req *mt5pb.SymbolParamsRequest, _ ...grpc.CallOption) (*mt5pb.SymbolParamsReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in, ok := s.instr[req.GetSymbol()]
	if !ok {
		return &mt5pb.SymbolParamsReply{Error: &mt5pb.Error{
			Code: mt5pb.ErrorCode_SYMBOL_NOT_FOUND, Message: "symbol not found"}}, nil
	}
	return &mt5pb.SymbolParamsReply{Result: &mt5pb.SymbolParams{
		Symbol:     req.GetSymbol(),
		SymbolInfo: &mt5pb.SymbolInfo{ContractSize: in.ContractSize},
		SymbolGroup: &mt5pb.SymGroup{
			TradeMode: mt5pb.TradeMode(in.TradeMode),
			MinLots:   in.MinLot, LotsStep: in.LotStep}}}, nil
}

func sideName(op mt5pb.OrderType) string {
	if op == mt5pb.OrderType_OrderType_Sell {
		return "sell"
	}
	return "buy"
}

func hasExecComment(c string) bool { return len(c) >= 2 && c[:2] == "ph" }
