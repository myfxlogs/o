// surveyfacts.go —— 上岗普查读面件（O-41375——T-0099 swapsurvey 字段全集
// 的 Client 方法化：SymbolParams 全字段+GetQuote+RequiredMargin 只读面；
// mutating 面零触。证据列形与 swapsurvey JSON 原档同键名——判词文档可
// 直接对拍 T-0099 基线）。
package mt5

import (
	"context"
	"fmt"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// SymbolSurveyFacts 单符 SymbolParams 全字段快照（swapsurvey 同款 17 键——
// SymbolInfo+SymGroup 双段全取；err/空体=上浮不虚构）。
func (c *Client) SymbolSurveyFacts(ctx context.Context, sym string) (map[string]any, error) {
	r, err := c.mt5.SymbolParams(c.authed(ctx),
		&mt5pb.SymbolParamsRequest{Id: c.token, Symbol: sym})
	if err != nil {
		return nil, err
	}
	if r.Error != nil {
		return nil, fmt.Errorf("symbolParams %s apierr: %s", sym, r.Error.Message)
	}
	if r.Result == nil {
		return nil, fmt.Errorf("symbolParams %s 空体", sym)
	}
	si, sg := r.Result.SymbolInfo, r.Result.SymbolGroup
	return map[string]any{
		"desc":            si.GetDescription(),
		"digits":          si.GetDigits(),
		"spread_pts":      si.GetSpread(),
		"tick_size":       si.GetTickSize(),
		"tick_value":      si.GetTickValue(),
		"contract_size":   si.GetContractSize(),
		"margin_ccy":      si.GetMarginCurrency(),
		"profit_ccy":      si.GetProfitCurrency(),
		"trade_mode":      sg.GetTradeMode().String(),
		"trade_mode_id":   int32(sg.GetTradeMode()),
		"min_lots":        sg.GetMinLots(),
		"lots_step":       sg.GetLotsStep(),
		"swap_type":       sg.GetSwapType().String(),
		"swap_long":       sg.GetSwapLong(),
		"swap_short":      sg.GetSwapShort(),
		"triple_swap_day": sg.GetThreeDaysSwap().String(),
		"init_margin":     sg.GetInitialMargin(),
		"mntn_margin":     sg.GetMaintenanceMargin(),
	}, nil
}

// Margin1LotBuy 单符 1 lot buy 向 RequiredMargin 试算（swapsurvey 同款——
// err/apierr=上浮由调用方如实录行不虚构）。
func (c *Client) Margin1LotBuy(ctx context.Context, sym string) (float64, error) {
	buy := mt5pb.DealType_DealType_DealBuy
	r, err := c.mt5.RequiredMargin(c.authed(ctx),
		&mt5pb.RequiredMarginRequest{Id: c.token, Symbol: sym, Lots: 1, Type: &buy})
	if err != nil {
		return 0, err
	}
	if r.Error != nil {
		return 0, fmt.Errorf("requiredMargin %s apierr: %s", sym, r.Error.Message)
	}
	return r.Result, nil
}
