// Package execwiring —— 仓巡×执事×okx 装配缝（T-0426 自 cmd/core/close.go 拆立
// 单源：执行径演习（simdrill）须跑与产线同一装配径——装配缝复制=影子缝漂移源，
// 缝上的语义位（Close=true 必置、Notional 半额口径）错一环演习即失真）。
// guard 不 import execution、execution 不 import guard（方向闸零边）——组合根
// 边集归本包：cmd/core 与 simdrill 同槽消费。
package execwiring

import (
	"context"
	"strconv"
	"strings"

	"o/internal/guard"
	"o/internal/trading/execution"
	"o/internal/trading/okxexec"
)

// CloseExec 仓巡×执事装配缝（guard.Executor 实现）。
type CloseExec struct{ R *execution.Runner }

func (c CloseExec) ExecuteClose(ctx context.Context, spec guard.CloseSpec) (int64, error) {
	if spec.Legs[1].InstID == "" { // 单腿仓（方向族契约——棒5c）走 ExecuteDir
		return c.R.ExecuteDir(ctx, CloseDirIntent(spec))
	}
	return c.R.Execute(ctx, CloseIntent(spec))
}

// CloseDirIntent 单腿平仓 Intent（方向族契约——Legs[1] 空槽不消费）。
func CloseDirIntent(spec guard.CloseSpec) execution.Intent {
	return execution.Intent{Kind: "close", DecisionID: spec.DecisionID,
		StrategyRef: spec.StrategyRef, InstID: spec.InstID, NotionalUSD: spec.NotionalUSD,
		Legs: [2]execution.Leg{CloseLegOf(spec, spec.Legs[0]), {}}}
}

// CloseIntent 平仓 Intent 装配——Kind="close" 免闸免赎语义位+双腿 Close=true 必置。
func CloseIntent(spec guard.CloseSpec) execution.Intent {
	return execution.Intent{
		Kind:       "close", // T-0043：live_switch 翻 false 时平仓被拦=安全律反向
		DecisionID: spec.DecisionID, StrategyRef: spec.StrategyRef, InstID: spec.InstID,
		NotionalUSD: spec.NotionalUSD,
		Legs:        [2]execution.Leg{CloseLegOf(spec, spec.Legs[0]), CloseLegOf(spec, spec.Legs[1])},
	}
}

// CloseLegOf 平仓腿装配——Close=true 必置（T-0043 根修）；Ticket 透传 venue
// 仓凭据（T-0145——mt5 腿 OrderClose 必携 ticket，无票=执行面 fail-closed
// 禁反向单）。
func CloseLegOf(spec guard.CloseSpec, l guard.CloseLeg) execution.Leg {
	return execution.Leg{Venue: l.Venue, InstID: l.InstID, Side: l.Side, Close: true,
		Sz: strconv.FormatFloat(l.Sz, 'f', -1, 64), Notional: spec.NotionalUSD / 2, Ticket: l.Ticket}
}

// OkxSizer guard.VenueSizer okx 适配（T-0046 ②——平仓量决策时点 venue 现读：
// 现货=availBal、合约=positions 张数、lotSz=instruments；读败=fail-closed 拒平）。
type OkxSizer struct{ C *okxexec.Client }

func (s OkxSizer) SpotAvailBal(ctx context.Context, instID string) (float64, error) {
	return s.C.AvailBal(ctx, strings.SplitN(instID, "-", 2)[0])
}

func (s OkxSizer) ContractPos(ctx context.Context, instID string) (float64, error) {
	return s.C.ContractPosSz(ctx, instID)
}

func (s OkxSizer) LotSz(ctx context.Context, instID string) (float64, error) {
	it := "SPOT"
	if strings.HasSuffix(instID, "-SWAP") {
		it = "SWAP"
	}
	m, err := s.C.InstrumentMeta(ctx, it, instID)
	if err != nil {
		return 0, err
	}
	return m.LotSz, nil
}
