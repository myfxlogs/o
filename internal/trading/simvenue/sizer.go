// sizer.go —— venue 真相读面（T-0426 自 simvenue.go 拆件——行数棘轮）：
// Sizer 三方法（guard.VenueSizer 结构化满足）+真相访问器（演习对拍源）。
package simvenue

import (
	"context"
	"fmt"
	"math"
)

// Balance 真相读：现货余额。
func (v *SimVenue) Balance(ccy string) float64 {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.bal[ccy]
}

// Position 真相读：合约净持仓张数（带号）。
func (v *SimVenue) Position(instID string) float64 {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.pos[instID]
}

// Instrument 真相读：乐器面拷贝（不在册=false——演习乐器面直取源）。
func (v *SimVenue) Instrument(instID string) (Instrument, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	in, ok := v.instr[instID]
	return in, ok
}

// CtValOf 真相读：合约张面值（guard.AttachCtVal 注入源——不在册=false）。
func (v *SimVenue) CtValOf(instID string) (float64, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	in, ok := v.instr[instID]
	if !ok || in.CtVal <= 0 {
		return 0, false
	}
	return in.CtVal, true
}

// Fills 真相读：成交流水快照。
func (v *SimVenue) Fills() []Fill {
	v.mu.Lock()
	defer v.mu.Unlock()
	out := make([]Fill, len(v.fills))
	copy(out, v.fills)
	return out
}

// SpotAvailBal 平仓量现读——现货可用余额（guard.VenueSizer 结构化满足；
// 产线同语义 okx.AvailBal：乐器不在册=错上抛 fail-closed）。
func (v *SimVenue) SpotAvailBal(_ context.Context, instID string) (float64, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	in, ok := v.instr[instID]
	if !ok {
		return 0, fmt.Errorf("simvenue: 乐器不在册 %s", instID)
	}
	return v.bal[baseCcy(in)], nil
}

// ContractPos 平仓量现读——合约净持仓张数 abs（okx.ContractPosSz 同语义）。
func (v *SimVenue) ContractPos(_ context.Context, instID string) (float64, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return math.Abs(v.pos[instID]), nil
}

// LotSz 平仓量现读——数量步长（okx.InstrumentMeta 同语义）。
func (v *SimVenue) LotSz(_ context.Context, instID string) (float64, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	in, ok := v.instr[instID]
	if !ok {
		return 0, fmt.Errorf("simvenue: 乐器不在册 %s", instID)
	}
	return in.LotSz, nil
}
