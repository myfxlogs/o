// xvenue_sizer.go —— T-0145 mt5 腿平仓量现读面（自 xvenue.go 拆件——棘轮 D-588）。
package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/trading/mtexec"
)

// xvenueSess mt5 会话窄面（xvenueOpen/Mt5Sizer 取用缝——产线=*mt5LazyCli；
// T-0145 返修②：桩测可注假，nil-conn 取用点守契约随面走）。
type xvenueSess interface {
	account(ctx context.Context) (string, error)
	Authed(ctx context.Context) context.Context
	Conn() *grpc.ClientConn
	MT5() mt5pb.MT5Client
	Suspect()
}

// Mt5Sizer mt5 腿平仓量现读面（guard.VenueSizer——T-0145 venue 路由专用件：
// ContractPos=OpenedOrders 实读手数（我方仓筛 comment 'ph' 幂等前缀——
// 他席/手工仓不混读）；LotSz=方言行 lot_step（台账基线——SymbolParams
// 活读预检在开仓径另位）；SpotAvailBal=CFD 无现货腿恒拒）。
type Mt5Sizer struct {
	C xvenueSess
	X mtexec.Dialect
	D *mt5.DialectStore
}

func (s Mt5Sizer) ContractPos(ctx context.Context, canonical string) (float64, error) {
	sym, err := s.X.VenueSymbol(ctx, canonical)
	if err != nil {
		return 0, err
	}
	// T-0145 返修②：MT5()=nil=会话未建——nil 守 err 上浮（OpenedOrders
	// 实读径懒连败=panic 病灶根修，guard 现读面同案）。
	mc := s.C.MT5()
	if mc == nil {
		return 0, fmt.Errorf("mt5 会话未建（懒连败）fail-closed——ContractPos 现读不可得")
	}
	or, err := mc.OpenedOrders(s.C.Authed(ctx), &mt5pb.OpenedOrdersRequest{})
	if err != nil {
		s.C.Suspect()
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
func (s Mt5Sizer) LotSz(ctx context.Context, canonical string) (float64, error) {
	acct, err := s.C.account(ctx)
	if err != nil {
		return 0, err
	}
	r, err := s.D.Lookup(ctx, "mt5", acct, canonical)
	if err != nil {
		return 0, err
	}
	return r.LotStep, nil
}
func (s Mt5Sizer) SpotAvailBal(context.Context, string) (float64, error) {
	return 0, fmt.Errorf("mt5 无现货腿——CFD 腿走 ContractPos 径")
}

// hasExecComment 我方执行件幂等键判（clOrdID 'ph' 前缀——execution.flow
// ph%dx%dx%d/ph%drb%d 两形；他席/手工仓无此前缀不混读）。
func hasExecComment(c string) bool { return len(c) >= 2 && c[:2] == "ph" }
