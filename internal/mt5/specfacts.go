// specfacts.go —— 单符 SymbolParams 全谱事实面（O-41378 件2 普查落列料：
// proto 型不出 mt5 包界 D-514——返回 typed struct 纯字段；spec_hash=SpecHash
// 同 dialect 面同函数；tick_sz=TickSize>0 否则 Points 方言判据同 dialectTickSz）。
package mt5

import (
	"context"
	"fmt"
)

// SpecFacts 单符合约面事实（fin_venue_symbols 列料+params jsonb 序列化源——
// venue 自报裸值留证，配对判词可复演）。
type SpecFacts struct {
	Symbol    string  `json:"symbol"`
	Desc      string  `json:"desc"`
	CtVal     float64 `json:"ct_val"`
	TickSz    float64 `json:"tick_sz"`
	MinSz     float64 `json:"min_sz"`
	LotStep   float64 `json:"lot_step"`
	Digits    int32   `json:"digits"`
	TradeMode int32   `json:"trade_mode"`
	ProfitCcy string  `json:"profit_ccy"`
	MarginCcy string  `json:"margin_ccy"`
	SpreadPts float64 `json:"spread_pts"`
	SpecHash  string  `json:"spec_hash"`
}

// SymbolSpecFacts 单符 SymbolParams 实读全谱（一次 RPC 出全列料——
// 普查拍逐符调用，dialectPace 同尺限速由调用方把）。
func (c *Client) SymbolSpecFacts(ctx context.Context, sym string) (SpecFacts, error) {
	si, sg, err := c.SymbolSpec(ctx, sym)
	if err != nil {
		return SpecFacts{}, err
	}
	if si == nil || sg == nil {
		return SpecFacts{}, fmt.Errorf("symbolParams %s 半空体", sym)
	}
	f := SpecFacts{
		Symbol:    sym,
		Desc:      si.GetDescription(),
		CtVal:     si.GetContractSize(),
		TickSz:    si.GetTickSize(),
		MinSz:     sg.GetMinLots(),
		LotStep:   sg.GetLotsStep(),
		Digits:    si.GetDigits(),
		TradeMode: int32(sg.GetTradeMode()),
		ProfitCcy: si.GetProfitCurrency(),
		MarginCcy: si.GetMarginCurrency(),
		SpreadPts: float64(si.GetSpread()),
		SpecHash:  SpecHash(si, sg),
	}
	if f.TickSz <= 0 {
		f.TickSz = si.GetPoints() // T-0143 Exness 实拍：tick 双字段 0 不填 points 实填
	}
	return f, nil
}
