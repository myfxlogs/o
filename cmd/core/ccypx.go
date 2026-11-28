package main

// ccypx.go —— O-41395 件① NAV 非 USD 腿价源硬化：鲜度优先三源估值件。
//
//	病灶：valCcy/cockpit 敞口内联只吃 md_ticks.last ∪ md_mark_index.px
//	落库行——订阅死（BTC-USDT 2026-11-17 出活跃集实证）后陈价静默入
//	NAV（$64,348 vs 实值 ~$60.4k 差 6% 实证）。
//	修法：加 OKX 公开 index-tickers REST 现拉源，判序=鲜度优先（任源
//	ts 最新者胜，非「先 tick 后 index」顺位）——陈 tick(>24h) 被鲜
//	index 压过；REST 败/缺席回退 DB 最鲜行；全无=false 不虚计（现语义保）。
//	两面同改律：valCcy 与 cockpit 敞口同吃 ccyPxUSD 本件，散修即分叉禁复制。

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/singleflight"

	"o/internal/trading/okxexec"
)

// ccyIdxPxFn index-tickers 现拉价件（assemble 注入 okxexec.PublicGet——
// 60s TTL+singleflight 缓存在 indexPxFor 闭包内；nil/读败=DB 兜底径
// 现语义）。at=拉取完成时刻=鲜度戳（缓存命中返原戳——陈缓存不冒充鲜）。
var ccyIdxPx atomic.Value // func(ctx, instID string) (px float64, at time.Time, err error)

const indexPxTTL = 60 * time.Second

type indexPxVal struct {
	px float64
	at time.Time
}

// indexPxFor 装配件：单次拉 /api/v5/market/index-tickers?instId=<inst>
// （公开面零凭据——钦定保留面同 PublicGet 径）+60s 单飞缓存：并发同键
// 合一发、TTL 内命中零拉、负结果同缓存（断流期不锤 venue）。
func indexPxFor(okcli *okxexec.Client) func(context.Context, string) (float64, time.Time, error) {
	if okcli == nil { // 凭据缺席期——absent 桩=DB-only 现语义（读面不炸）
		return func(context.Context, string) (float64, time.Time, error) {
			return 0, time.Time{}, fmt.Errorf("okcli 缺席——index 价件停")
		}
	}
	var sf singleflight.Group
	var mu sync.Mutex
	type ent struct {
		val indexPxVal
		at  time.Time
		err error
	}
	cache := map[string]ent{}
	return func(ctx context.Context, instID string) (float64, time.Time, error) {
		mu.Lock()
		if e, ok := cache[instID]; ok && time.Since(e.at) < indexPxTTL {
			mu.Unlock()
			return e.val.px, e.at, e.err
		}
		mu.Unlock()
		v, err, _ := sf.Do(instID, func() (any, error) {
			raw, rerr := okcli.PublicGet(ctx, "/api/v5/market/index-tickers?instId="+instID)
			if rerr != nil {
				return nil, rerr
			}
			var rows []struct {
				InstID string `json:"instId"`
				IdxPx  string `json:"idxPx"`
			}
			if uerr := json.Unmarshal(raw, &rows); uerr != nil {
				return nil, uerr
			}
			if len(rows) == 0 {
				return nil, fmt.Errorf("index-tickers %s 空 data", instID)
			}
			px, perr := strconv.ParseFloat(rows[0].IdxPx, 64)
			if perr != nil || px <= 0 {
				return nil, fmt.Errorf("index-tickers %s idxPx=%q 非正值", instID, rows[0].IdxPx)
			}
			return indexPxVal{px: px, at: time.Now()}, nil
		})
		e := ent{at: time.Now(), err: err}
		if err == nil {
			e.val = v.(indexPxVal)
			e.at = e.val.at
		}
		mu.Lock()
		cache[instID] = e
		mu.Unlock()
		return e.val.px, e.at, e.err
	}
}

// ccyPxUSD 非 USD 币→USD 单价（鲜度优先——候选=落库 UNION 最鲜行
// (md_ticks.last ∪ md_mark_index.px，index-tickers channel 行天然含本
// UNION) + REST index 现拉 (at=拉取时刻)；ts 最新者胜。REST 缺席/败
// 回退 DB；全无 ok=false 不虚计）。USDT/USD 直返 1（估值面直入现语义）。
func ccyPxUSD(ctx context.Context, pool *pgxpool.Pool, ccy string) (float64, bool) {
	if ccy == "USDT" || ccy == "USD" {
		return 1, true
	}
	inst := ccy + "-USDT"
	var last *float64
	var lastTs time.Time
	_ = pool.QueryRow(ctx, `SELECT v,ts FROM (SELECT last v,ts FROM md_ticks WHERE inst_id=$1
		UNION ALL SELECT px,ts FROM md_mark_index WHERE inst_id=$1) q ORDER BY ts DESC LIMIT 1`,
		inst).Scan(&last, &lastTs)
	best, bestTs, have := 0.0, time.Time{}, false
	if last != nil {
		best, bestTs, have = *last, lastTs, true
	}
	if f, ok := ccyIdxPx.Load().(func(context.Context, string) (float64, time.Time, error)); ok && f != nil {
		if ipx, iat, err := f(ctx, inst); err == nil && ipx > 0 && (!have || iat.After(bestTs)) {
			best, have = ipx, true
		}
	}
	return best, have
}

// dcdExposureUSD 产品仓 dcd 凭据 Σamt×ccyPxUSD 估值（cockpit 敞口分子
// 腿——fin_position_view kind='dcd'；T-0364 kind='dcd_owner' 不入同旧；
// NULL 行跳过=原 SUM 语义保，无价币贡献 0 不虚计）。
func dcdExposureUSD(ctx context.Context, pool *pgxpool.Pool) (float64, error) {
	rows, err := pool.Query(ctx,
		`SELECT ccy,notional FROM fin_position_view WHERE kind='dcd'`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var sum float64
	for rows.Next() {
		var ccy *string
		var notional *float64
		if err := rows.Scan(&ccy, &notional); err != nil {
			return 0, err
		}
		if ccy == nil || notional == nil {
			continue
		}
		if px, ok := ccyPxUSD(ctx, pool, *ccy); ok {
			sum += *notional * px
		}
	}
	return sum, rows.Err()
}

// valCcy 币本位→USD 估值（units×单价；无价=0 不虚计——现语义保）。
func valCcy(ctx context.Context, pool *pgxpool.Pool, ccy string, units float64) float64 {
	if ccy == "USDT" || ccy == "USD" {
		return units
	}
	px, ok := ccyPxUSD(ctx, pool, ccy)
	if !ok {
		return 0
	}
	return units * px
}
