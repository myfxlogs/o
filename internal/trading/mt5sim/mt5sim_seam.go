// mt5sim_seam.go —— T-0461 mt5sim 产线缝面+真相读面（行数棘轮拆件）：
// mtexec.Cli/Dialect/guard.VenueSizer 三缝结构化满足+Meta 乐器面+
// 升件留痕+在仓/残史/流水真相读（对账源）。
package mt5sim

import (
	"context"
	"fmt"
	"sort"

	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// Cli mtexec.Cli 缝（Authed 透传 ctx——仿真无 token；Conn=nil 取用点
// 守契约——Executor.T 注入径不触 Conn；MT5=本件读面）。
type cli struct{ s *Sim }

func (c cli) Authed(ctx context.Context) context.Context { return ctx }
func (c cli) Conn() *grpc.ClientConn                     { return nil }
func (c cli) MT5() mt5pb.MT5Client                       { return c.s }

// Cli 会话缝取件。
func (s *Sim) Cli() cli { return cli{s: s} }

// Trading 下单面取件（mtexec.Executor.T 注入缝——产线=懒连 TradingClient
// 同位）。
func (s *Sim) Trading() mt5pb.TradingClient { return s }

// Dialect 方言缝（canonical→venue_symbol 唯一译径——查无=err+Escalate
// 留痕，禁官名硬发同产线律）。
type dialect struct{ s *Sim }

func (s *Sim) Dialect() dialect { return dialect{s: s} }

func (d dialect) VenueSymbol(_ context.Context, canonical string) (string, error) {
	d.s.mu.Lock()
	defer d.s.mu.Unlock()
	sym, ok := d.s.canon[canonical]
	if !ok {
		return "", fmt.Errorf("mt5sim 方言查无映射: %s", canonical)
	}
	return sym, nil
}

func (d dialect) Escalate(_ context.Context, canonical, reason string) {
	d.s.mu.Lock()
	defer d.s.mu.Unlock()
	d.s.escalated = append(d.s.escalated, canonical+":"+reason)
}

// Meta 乐器面取件（canonical→ContractSize/MinLot/LotStep/TradeMode——
// 产线 SymbolParams 活读同语义的仿真册直读；BuildXvenueLegs CfdMeta 与
// guard ctVal 现读件装配源）。
func (s *Sim) Meta(canonical string) (contractSize, minLot, lotStep float64, tradeMode int32, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sym, ok := s.canon[canonical]
	if !ok {
		return 0, 0, 0, 0, false
	}
	in, ok := s.instr[sym]
	if !ok {
		return 0, 0, 0, 0, false
	}
	return in.ContractSize, in.MinLot, in.LotStep, in.TradeMode, true
}

// Escalations 方言升件留痕读面（桩测锚——查无映射必显形）。
func (s *Sim) Escalations() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.escalated...)
}

// ContractPos guard.VenueSizer 现读面——OpenedOrders 实读手数（我方仓筛
// comment 'ph' 幂等前缀——Mt5Sizer 同判据，canonical→方言名先译）。
func (s *Sim) ContractPos(ctx context.Context, canonical string) (float64, error) {
	sym, err := s.Dialect().VenueSymbol(ctx, canonical)
	if err != nil {
		return 0, err
	}
	or, err := s.OpenedOrders(ctx, &mt5pb.OpenedOrdersRequest{})
	if err != nil {
		return 0, err
	}
	var lots float64
	for _, o := range or.GetResult() {
		if o.GetSymbol() == sym && hasExecComment(o.GetComment()) {
			lots += o.GetLots()
		}
	}
	return lots, nil
}

// LotSz guard.VenueSizer 现读面——乐器 lot_step（方言行基线同产线径）。
func (s *Sim) LotSz(ctx context.Context, canonical string) (float64, error) {
	sym, err := s.Dialect().VenueSymbol(ctx, canonical)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	in, ok := s.instr[sym]
	if !ok {
		return 0, fmt.Errorf("mt5sim 乐器不在册 %s", sym)
	}
	return in.LotStep, nil
}

// SpotAvailBal guard.VenueSizer——CFD 无现货腿恒拒（Mt5Sizer 同语义）。
func (s *Sim) SpotAvailBal(context.Context, string) (float64, error) {
	return 0, fmt.Errorf("mt5sim 无现货腿——CFD 腿走 ContractPos 径")
}

// OpenedLots 真相读——在仓手数合计（canonical 键——对账源；不筛 comment
// 前缀=全量真相，演习私有 venue 无他席仓）。
func (s *Sim) OpenedLots(canonical string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var lots float64
	for _, p := range s.positions {
		if p.canonical == canonical {
			lots += p.lots
		}
	}
	return lots
}

// OpenTickets 真相读——在仓 ticket 序（升序——生命周期断言锚）。
func (s *Sim) OpenTickets() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]int64, 0, len(s.positions))
	for t := range s.positions {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// HistoryCount 真相读——残史条数（开完即平=history 命中面锚）。
func (s *Sim) HistoryCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.history)
}

// Fills 真相读——成交流水快照。
func (s *Sim) Fills() []Fill {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Fill{}, s.fills...)
}

// Balance 真相读——纸面权益。
func (s *Sim) Balance() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bal
}
