// coins.go —— okxCoinsFor 复制件（T-0165：研究节点同形装配——从 SG
// cmd/core/restlive.go 原样搬，零改写；T-0170 参数换 authGetter 窄面——
// account/balance 同属私有读面，走 readcred 持守件同形）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/trading/okxexec"
)

// okxCoinsFor 交易户逐币可动余量现读件（T-0095 v3 残币归口判据源——帧先 REST
// 兜底，availBal>0 逐币；读败=err 上抛 fail-closed 归口让位）。
func okxCoinsFor(pool *pgxpool.Pool, okcli authGetter) func(context.Context) (map[string]float64, error) {
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

// addSavingsUSDT 储蓄 USDT 可赎加算（O-CAP1 工单#20484：eval 余量读件
// trading avail 之外另计 /finance/savings/balance USDT 行 amt=本金面
// T+0 可赎——实证 trading avail=$232 时储蓄 USDT=$1014 判
// insufficient_balance 拒单）。仅 ccy=='USDT' 行计——savings 里
// DOGE/FIL/BTC/ETH=业主自持停泊资产不入基金钱。遍历全行找 ccy——
// 首行序不定坑在案（okx-asset-funding 能力账）。okcli 缺席/读败/
// 解析败=out 原样+WARN 降级——储蓄读败不盲拒交易户余量（半瞎好过
// 全瞎，如实留痕）。
func addSavingsUSDT(ctx context.Context, okcli authGetter, out map[string]float64) {
	if okcli == nil {
		log.Printf("WARN [coins] savings/balance 读件缺席（凭据缺）——储蓄可赎不计 eval 余量")
		return
	}
	raw, err := okcli.Get(ctx, "/api/v5/finance/savings/balance")
	if err != nil {
		log.Printf("WARN [coins] savings/balance 读败（%v）——储蓄可赎不计 eval 余量", err)
		return
	}
	var rows []struct {
		Ccy string `json:"ccy"`
		Amt string `json:"amt"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		log.Printf("WARN [coins] savings/balance 解析败（%v）——储蓄可赎不计 eval 余量", err)
		return
	}
	for _, r := range rows {
		if r.Ccy != "USDT" {
			continue
		}
		amt, perr := strconv.ParseFloat(r.Amt, 64)
		if perr != nil {
			log.Printf("WARN [coins] savings/balance USDT amt 解析败（%q: %v）——该行不计", r.Amt, perr)
			continue
		}
		out["USDT"] += amt
	}
}
