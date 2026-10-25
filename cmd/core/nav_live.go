// nav_live.go —— NAV 三户活读件（T-0086 ①④：资金户/Earn REST 保留面——
// 零快照零落地，venue 唯一事实源；自主文件拆分，行数棘轮）。
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"o/internal/trading/okxexec"
)

// T-0086 ①④：NAV 三户活读件（资金户=逐币余额、Earn=逐行 amt——REST 保留面
// 非快照零落地；凭据缺席=空表 0 不误报）。
func fundingLiveFor(okcli *okxexec.Client) func(context.Context) (map[string]float64, error) {
	return func(ctx context.Context) (map[string]float64, error) {
		out := map[string]float64{}
		if okcli == nil {
			return out, nil
		}
		bals, err := okcli.AssetBalances(ctx)
		if err != nil {
			return nil, err
		}
		for _, b := range bals {
			out[b.Ccy] += okxexec.ParseF(b.Bal)
		}
		return out, nil
	}
}

func earnLiveFor(okcli *okxexec.Client) func(context.Context) (map[string]float64, error) {
	return func(ctx context.Context) (map[string]float64, error) {
		out := map[string]float64{}
		if okcli == nil {
			return out, nil
		}
		sav, err := okcli.SavingsBalance(ctx)
		if err != nil {
			return nil, err
		}
		var rows []struct {
			Ccy string `json:"ccy"`
			Amt string `json:"amt"`
		}
		if json.Unmarshal(sav, &rows) != nil {
			return nil, fmt.Errorf("savings parse")
		}
		for _, r2 := range rows {
			out[r2.Ccy] += okxexec.ParseF(r2.Amt)
		}
		return out, nil
	}
}
