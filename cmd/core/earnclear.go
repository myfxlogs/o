// earnclear.go —— T-0439 杂币清仓缝组合根翻译（earn.Clearer ←
// execution.OkxExecutor——earn 不 import execution，D-514 方向闸零边，
// execwiring 装配缝同例）。
package main

import (
	"context"

	"o/internal/earn"
	"o/internal/trading/execution"
)

// okxClearer earn.Clearer 适配件（产线=OkxExecutor 同一件 T-0008 已验径）。
type okxClearer struct{ e *execution.OkxExecutor }

func (o okxClearer) leg(l earn.ClearLeg) execution.Leg {
	return execution.Leg{Venue: l.Venue, InstID: l.InstID, Side: l.Side,
		Close: l.Close, Sz: l.Sz}
}

func (o okxClearer) res(r execution.OrderResult) earn.ClearResult {
	return earn.ClearResult{ClOrdID: r.ClOrdID, State: r.State,
		FillPx: r.FillPx, FillSz: r.FillSz, VenueCode: r.VenueCode, VenueMsg: r.VenueMsg}
}

func (o okxClearer) Place(ctx context.Context, leg earn.ClearLeg, clOrdID string) (earn.ClearResult, error) {
	r, err := o.e.Place(ctx, o.leg(leg), clOrdID)
	return o.res(r), err
}

func (o okxClearer) Resolve(ctx context.Context, leg earn.ClearLeg, clOrdID string) (earn.ClearResult, error) {
	r, err := o.e.Resolve(ctx, o.leg(leg), clOrdID)
	return o.res(r), err
}
