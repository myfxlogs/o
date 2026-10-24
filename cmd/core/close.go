// closeExec 仓巡×执事装配缝（cmd 组合根——guard 不 import execution，方向闸零边）：
// guard CloseSpec → execution 反向双腿 Intent；腿 sz 沿 fin_order_leg 串传
// （okx 张数口径换算归 venue 层）；Notional 取名义半额（delta 中性带检入参口径）。
// T-0043 根修：平仓腿 Close=true 必置（漏置=按开仓腿装配→short 仓上 buy
// 反向开 long 非平仓——NOX 反向新开病灶平仓径漏网）。
package main

import (
	"context"
	"strconv"

	"phase/internal/guard"
	"phase/internal/trading/execution"
)

type closeExec struct{ r *execution.Runner }

func (c closeExec) ExecuteClose(ctx context.Context, spec guard.CloseSpec) (int64, error) {
	return c.r.Execute(ctx, closeIntent(spec))
}

// closeIntent 平仓 Intent 装配——Kind="close" 免闸免赎语义位+双腿 Close=true 必置。
func closeIntent(spec guard.CloseSpec) execution.Intent {
	return execution.Intent{
		Kind:       "close", // T-0043：live_switch 翻 false 时平仓被拦=安全律反向
		DecisionID: spec.DecisionID, StrategyRef: spec.StrategyRef, InstID: spec.InstID,
		NotionalUSD: spec.NotionalUSD,
		Legs:        [2]execution.Leg{closeLegOf(spec, spec.Legs[0]), closeLegOf(spec, spec.Legs[1])},
	}
}

// closeLegOf 平仓腿装配——Close=true 必置（T-0043 根修）。
func closeLegOf(spec guard.CloseSpec, l guard.CloseLeg) execution.Leg {
	return execution.Leg{Venue: l.Venue, InstID: l.InstID, Side: l.Side, Close: true,
		Sz: strconv.FormatFloat(l.Sz, 'f', -1, 64), Notional: spec.NotionalUSD / 2}
}
