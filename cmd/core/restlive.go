package main

// restlive.go —— REST 保留面活性戳+funding/savings 现读聚（自 venue_readers.go
// 拆出——棘轮 300 行顶线）。

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/ledger"
	"o/internal/trading/okxexec"
)

// restOKAt REST 保留面末次成功读戳（T-0088 真源钉：earn_parking 帧稀疏
// 3min~5h 一跳当活性=假红灯——对拍环每拍实调 funding/savings，成功戳才真）。
var restOKAt atomic.Int64

// fundSaveBalances funding+savings 现读聚（T-0058：任一读败=err 上抛——静默跳腿禁）。
// T-0100 ①③：非本位币只收停泊溯源币（okx:funding/earn 域 FundCcy 成员——
// Earn 利息尘/腿基币渗漏=托管不入 venue 侧键）；usd=true=非 USDT 按 md_ticks
// 估值归一（权益求和径），false=原币量（对拍 ledger 侧按币量建键径）。
func fundSaveBalances(ctx context.Context, pool *pgxpool.Pool, okcli *okxexec.Client,
	out map[[2]string]float64, usd bool) error {
	set, err := ledger.FundCcy(ctx, pool)
	if err != nil {
		return err
	}
	emit := func(ccy string, amt float64) {
		if ccy != "USDT" && ccy != "USD" &&
			!set[[2]string{"okx:funding", ccy}] && !set[[2]string{"okx:earn", ccy}] {
			return // 非停泊托管币=域外不键（T-0100 ① venue 侧同源排除）
		}
		if usd && ccy != "USDT" && ccy != "USD" {
			amt = valCcy(ctx, pool, ccy, amt)
		}
		out[[2]string{"okx", ccy}] += amt
	}
	bals, err := okcli.AssetBalances(ctx)
	if err != nil {
		return fmt.Errorf("funding 读败: %w", err)
	}
	for _, b := range bals {
		emit(b.Ccy, okxexec.ParseF(b.Bal))
	}
	sav, err := okcli.SavingsBalance(ctx)
	if err != nil {
		return fmt.Errorf("savings 读败: %w", err)
	}
	var rows []struct {
		Ccy string `json:"ccy"`
		Amt string `json:"amt"`
	}
	if json.Unmarshal(sav, &rows) != nil {
		return fmt.Errorf("savings parse")
	}
	for _, r2 := range rows {
		emit(r2.Ccy, okxexec.ParseF(r2.Amt))
	}
	// T-0101：dcd 在持产品单无条件入键（venue LIVE 单=已部署资本铁证——
	// 不走 FundCcy 闸：未基线的新购仓 venue 侧照发，基线缺席由对拍面兜差）；
	// 读败=err 上抛不假零。
	ords, err := okcli.DcdLiveOrders(ctx)
	if err != nil {
		return fmt.Errorf("dcd 持仓读败: %w", err)
	}
	for _, od := range ords {
		amt := okxexec.ParseF(od.NotionalSz)
		if usd && od.NotionalCcy != "USDT" && od.NotionalCcy != "USD" {
			amt = valCcy(ctx, pool, od.NotionalCcy, amt)
		}
		out[[2]string{"okx", od.NotionalCcy}] += amt
	}
	dcdSyncEvents(ctx, pool, ords)
	restOKAt.Store(time.Now().UnixNano())
	return nil
}

// dcdSyncEvents LIVE 集指纹对账（T-0101 到期收口：在持集变动=LIVE→SETTLED
// 迁移/新增部署——md_account_events channel='dcd_live' 落事件行，指纹相同
// 幂等跳写不复读）。写败只 log 不炸对账面。
func dcdSyncEvents(ctx context.Context, pool *pgxpool.Pool, ords []okxexec.DcdOrder) {
	var ids []string
	payload := []map[string]any{}
	for _, od := range ords {
		ids = append(ids, od.OrdID)
		payload = append(payload, map[string]any{
			"ordId": od.OrdID, "productId": od.ProductID, "notionalCcy": od.NotionalCcy,
			"notionalSz": od.NotionalSz, "settleTime": od.SettleTime})
	}
	sort.Strings(ids)
	fp := strings.Join(ids, ",")
	var prev string
	_ = pool.QueryRow(ctx, `SELECT COALESCE(data->>'live_ids','') FROM md_account_events
		WHERE channel='dcd_live' ORDER BY ts DESC LIMIT 1`).Scan(&prev)
	if prev == fp {
		return // 在持集无变——幂等跳写
	}
	data, _ := json.Marshal(map[string]any{"live_ids": fp, "orders": payload})
	if _, err := pool.Exec(ctx, `INSERT INTO md_account_events(venue,channel,ccy,data,ts)
		VALUES('okx','dcd_live','', $1, now())
		ON CONFLICT (venue,channel,ccy,inst_id) DO UPDATE SET ts=EXCLUDED.ts,
		data=EXCLUDED.data,recv_ts=now()`, data); err != nil {
		log.Printf("[dcd] 持仓集事件行写败: %v", err)
	}
}

// tradingFundEq 交易户基金口径 eqUsd（T-0100 ①：trading 域=USDT∪open 腿基币
// 行过滤求和——不按全账户取；账外尘〔腿平残币/杂币〕口径内天然排除）。
// 帧 details 逐 ccy 过滤，帧缺/零和→REST /account/balance 兜底同口径
// （fail-closed 上抛禁静默跳腿）。
func tradingFundEq(ctx context.Context, pool *pgxpool.Pool, okcli *okxexec.Client) (float64, error) {
	wl, err := ledger.FundCcy(ctx, pool)
	if err != nil {
		return 0, err
	}
	fund := map[string]bool{}
	for k := range wl {
		if k[0] == "okx:trading" {
			fund[k[1]] = true
		}
	}
	sumRows := func(rows []struct {
		Ccy   string `json:"ccy"`
		EqUsd string `json:"eqUsd"`
	}) float64 {
		var sum float64
		for _, d := range rows {
			if !fund[d.Ccy] {
				continue
			}
			sum += parseNum(d.EqUsd)
		}
		return sum
	}
	var data json.RawMessage
	if err := pool.QueryRow(ctx, `SELECT data FROM md_account_events
		WHERE channel IN ('account','balance_and_position') ORDER BY ts DESC LIMIT 1`).Scan(&data); err == nil {
		var top struct {
			Details []struct {
				Ccy   string `json:"ccy"`
				EqUsd string `json:"eqUsd"`
			} `json:"details"`
		}
		if json.Unmarshal(data, &top) == nil && len(top.Details) > 0 {
			if sum := sumRows(top.Details); sum > 0 {
				return sum, nil
			}
		}
	}
	if okcli == nil {
		return 0, fmt.Errorf("帧缺且 REST 兜底件缺席")
	}
	raw, rerr := okcli.Get(ctx, "/api/v5/account/balance")
	if rerr != nil {
		return 0, rerr
	}
	var rows []struct {
		Details []struct {
			Ccy   string `json:"ccy"`
			EqUsd string `json:"eqUsd"`
		} `json:"details"`
	}
	if uerr := json.Unmarshal(raw, &rows); uerr != nil || len(rows) == 0 {
		return 0, fmt.Errorf("balance parse: %v", uerr)
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
		return out, nil
	}
}
