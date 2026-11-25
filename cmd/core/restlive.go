package main

// restlive.go —— REST 保留面活性戳+funding/savings 现读聚（自 venue_readers.go
// 拆出——棘轮 300 行顶线）。

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/trading/okxexec"
)

// restOKAt REST 保留面末次成功读戳（T-0088 真源钉：earn_parking 帧稀疏
// 3min~5h 一跳当活性=假红灯——对拍环每拍实调 funding/savings，成功戳才真）。
var restOKAt atomic.Int64

// tradingFundMap 交易户逐币原生权益（O-34287 验收②：对拍径=币种原生量——
// details[].eq 原币字段逐 ccy 求和，eqUsd=USD 估值字段禁入簿记径；
// 全币种收——对账白名单/基线凭据层裁域，venue 真值不藏）。
// 帧 details 逐 ccy 聚，帧缺/零和→REST /account/balance 兜底同口径
// （fail-closed 上抛禁静默跳腿）。
func tradingFundMap(ctx context.Context, pool *pgxpool.Pool, okcli *okxexec.Client) (map[string]float64, error) {
	sumRows := func(rows []struct {
		Ccy string `json:"ccy"`
		Eq  string `json:"eq"`
	}) map[string]float64 {
		out := map[string]float64{}
		for _, d := range rows {
			if v := parseNum(d.Eq); v != 0 {
				out[d.Ccy] += v
			}
		}
		return out
	}
	var data json.RawMessage
	if err := pool.QueryRow(ctx, `SELECT data FROM md_account_events
		WHERE channel IN ('account','balance_and_position') ORDER BY ts DESC LIMIT 1`).Scan(&data); err == nil {
		var top struct {
			Details []struct {
				Ccy string `json:"ccy"`
				Eq  string `json:"eq"`
			} `json:"details"`
		}
		if json.Unmarshal(data, &top) == nil && len(top.Details) > 0 {
			if m := sumRows(top.Details); len(m) > 0 {
				return m, nil
			}
		}
	}
	if okcli == nil {
		return nil, fmt.Errorf("帧缺且 REST 兜底件缺席")
	}
	raw, rerr := okcli.Get(ctx, "/api/v5/account/balance")
	if rerr != nil {
		return nil, rerr
	}
	var rows []struct {
		Details []struct {
			Ccy string `json:"ccy"`
			Eq  string `json:"eq"`
		} `json:"details"`
	}
	if uerr := json.Unmarshal(raw, &rows); uerr != nil || len(rows) == 0 {
		return nil, fmt.Errorf("balance parse: %v", uerr)
	}
	return sumRows(rows[0].Details), nil
}

// okxCoinsFor 交易户逐币可动余量现读件（T-0095 v3 残币归口判据源——帧先 REST
// 兜底，availBal>0 逐币；读败=err 上抛 fail-closed 归口让位）。
func okxCoinsFor(pool *pgxpool.Pool, okcli *okxexec.Client) func(context.Context) (map[string]float64, error) {
	return func(ctx context.Context) (map[string]float64, error) {
		out := map[string]float64{}
		var data json.RawMessage
		if err := pool.QueryRow(ctx, `SELECT data FROM md_account_events
			WHERE channel IN ('account','balance_and_position') ORDER BY ts DESC LIMIT 1`).Scan(&data); err == nil {
			var top struct {
				Details []struct {
					Ccy      string `json:"ccy"`
					AvailBal string `json:"availBal"`
				} `json:"details"`
			}
			if json.Unmarshal(data, &top) == nil {
				for _, d := range top.Details {
					if v := okxexec.ParseF(d.AvailBal); v > 0 {
						out[d.Ccy] = v
					}
				}
			}
		}
		if len(out) > 0 {
			addSavingsUSDT(ctx, okcli, out)
			return out, nil
		}
		if okcli == nil {
			return nil, fmt.Errorf("coins 判据源缺席（凭据缺）")
		}
		raw, err := okcli.Get(ctx, "/api/v5/account/balance")
		if err != nil {
			return nil, err
		}
		var rows []struct {
			Details []struct {
				Ccy      string `json:"ccy"`
				AvailBal string `json:"availBal"`
			} `json:"details"`
		}
		if json.Unmarshal(raw, &rows) != nil || len(rows) == 0 {
			return nil, fmt.Errorf("balance parse")
		}
		for _, d := range rows[0].Details {
			if v := okxexec.ParseF(d.AvailBal); v > 0 {
				out[d.Ccy] = v
			}
		}
		addSavingsUSDT(ctx, okcli, out)
		return out, nil
	}
}

// addSavingsUSDT USDT 可赎储蓄并入可动余量（工单#20484①：T+0 赎回原语
// earn.RedeemToTrading 在案、dcdexec.Open preRedeem 前置缝已接——余量面
// 不并储蓄=eval 拒 insufficient_balance 而 savings USDT 躺资金户实证
// trading avail=$232/savings=$1014）。读败=fail-open 不并入（保守低估
// 不误放大——eval 面宁缺不混；开仓径 preRedeem 真赎时 venue 二次实判）。
func addSavingsUSDT(ctx context.Context, okcli *okxexec.Client, out map[string]float64) {
	if okcli == nil {
		return
	}
	sav, err := okcli.SavingsBalance(ctx)
	if err != nil {
		return
	}
	var rows []struct {
		Ccy string `json:"ccy"`
		Amt string `json:"amt"`
	}
	if json.Unmarshal(sav, &rows) != nil {
		return
	}
	for _, r := range rows {
		if r.Ccy == "USDT" {
			if v := okxexec.ParseF(r.Amt); v > 0 {
				out["USDT"] += v
			}
		}
	}
}
