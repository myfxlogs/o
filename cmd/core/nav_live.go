// nav_live.go —— NAV 三户活读件（T-0086 ①④：资金户/Earn REST 保留面——
// 零快照零落地，venue 唯一事实源；自主文件拆分，行数棘轮）。
// T-0096：+MT5 户活读件（AccountSummary 现读——每真实账号一行）。
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/trading/okxexec"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mt5Acct MT5 户行（T-0096——login=真实账号，equity=净值现读 USD 本位；
// balance=可用余额附标）。
type mt5Acct struct {
	Login   string
	Equity  float64
	Balance float64
}

// mt5LiveFor MT5 户活读件（OpenSession 逐会话现读——AccountFacts 出 login、
// AccountSummary 出净值；凭据缺席/读败=err 上抛，卡面显「读件缺」不假零）。
func mt5LiveFor(pool *pgxpool.Pool, cfg *config.Client) func(context.Context) ([]mt5Acct, error) {
	return func(ctx context.Context) ([]mt5Acct, error) {
		cl, err := mt5.OpenSession(ctx, pool, cfg)
		if err != nil {
			return nil, err
		}
		defer cl.Close()
		login, _, _, ferr := cl.AccountFacts(ctx)
		if ferr != nil {
			return nil, fmt.Errorf("mt5 account facts: %w", ferr)
		}
		eq, _, bal, aerr := cl.AccountSummary(ctx)
		if aerr != nil {
			return nil, fmt.Errorf("mt5 account summary: %w", aerr)
		}
		return []mt5Acct{{Login: fmt.Sprintf("%d", login), Equity: eq, Balance: bal}}, nil
	}
}

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
