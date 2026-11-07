// xvenue_sizer.go —— T-0145 mt5 腿平仓量现读面（自 xvenue.go 拆件——棘轮 D-588）。
package main

import (
	"context"
	"fmt"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/trading/mtexec"
)

// Mt5Sizer mt5 腿平仓量现读面（guard.VenueSizer——T-0145 venue 路由专用件：
// ContractPos=OpenedOrders 实读手数（我方仓筛 comment 'ph' 幂等前缀——
// 他席/手工仓不混读）；LotSz=方言行 lot_step（台账基线——SymbolParams
// 活读预检在开仓径另位）；SpotAvailBal=CFD 无现货腿恒拒）。
type Mt5Sizer struct {
	C *mt5LazyCli
	X mtexec.Dialect
	D *mt5.DialectStore
}

func (s Mt5Sizer) ContractPos(ctx context.Context, canonical string) (float64, error) {
	sym, err := s.X.VenueSymbol(ctx, canonical)
	if err != nil {
		return 0, err
	}
	or, err := s.C.MT5().OpenedOrders(s.C.Authed(ctx), &mt5pb.OpenedOrdersRequest{})
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
