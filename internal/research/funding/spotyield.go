// spotyield.go —— 现货腿质押/Earn 收益分量率源读件（T-0320 质押腿入 eval）：
// 能力面 product_family='staking' enabled 注册行=资产质押维（stakable 旗+
// rate_source 率源字段——二维积左元）；率源闭集派发：
//
//	trx_onchain  = 链上质押实收口径（fin_trx_vote_apr 投票年化
//	               +fin_trx_txs rental_income/fin_trx_account energy_delegated
//	               租金年化——业主实盘同源口径，rental 样本缺=只计 vote 段如实）；
//	md_earn_rate = Earn 现率行（md_earn_rate 桥件——venue+ccy 键域）。
//
// 域缺席（桥表不在本域=42P01）/行缺/率源未注册=ok=false 如实——分量缺席
// 不虚构（gross 退化单头形）；staleHours>0 且率行更陈=缺席（陈=不存在同款）。
package funding

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SpotYield 质押率源解析：注册行（ccy→rate_source）→闭集派发取年化。
// 返 (Yield, present, err)——present=false=缺席如实（非错误面）；
// err=真读败（42P01 域缺席归 present=false 不算败）。
func SpotYield(ctx context.Context, pool *pgxpool.Pool, ccy string, staleHours float64) (Yield, bool, error) {
	var venue, src string
	err := pool.QueryRow(ctx, `SELECT venue, cost_attrs->>'rate_source' FROM fin_capability_surface
		WHERE product_family='staking' AND enabled
		  AND cost_attrs->>'ccy'=$1 AND (cost_attrs->>'stakable')::boolean
		ORDER BY surface LIMIT 1`, ccy).Scan(&venue, &src)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || pgIsAbsent(err) {
			return Yield{}, false, nil
		}
		return Yield{}, false, err
	}
	switch src {
	case "trx_onchain":
		return trxOnchainYield(ctx, pool, staleHours)
	case "md_earn_rate":
		return earnRateYield(ctx, pool, venue, ccy, staleHours)
	default:
		return Yield{}, false, nil // 率源未注册方言——缺席如实
	}
}

// freshEnough 率行新鲜度闸（staleHours<=0=不闸——慢特征率行如实取）。
func freshEnough(ts time.Time, staleHours float64) bool {
	if staleHours <= 0 {
		return true
	}
	return time.Since(ts) <= time.Duration(staleHours*float64(time.Hour))
}

// earnRateYield Earn 现率分量（md_earn_rate venue+ccy 键域——rate 已小数）。
func earnRateYield(ctx context.Context, pool *pgxpool.Pool, venue, ccy string, staleHours float64) (Yield, bool, error) {
	var rate float64
	var ts time.Time
	err := pool.QueryRow(ctx, `SELECT rate,ts FROM md_earn_rate
		WHERE venue=$1 AND ccy=$2 ORDER BY ts DESC LIMIT 1`, venue, ccy).Scan(&rate, &ts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || pgIsAbsent(err) {
			return Yield{}, false, nil
		}
		return Yield{}, false, err
	}
	if !freshEnough(ts, staleHours) {
		return Yield{}, false, nil
	}
	return Yield{APR: rate, Source: "md_earn_rate:" + venue}, true, nil
}

// trxOnchainYield TRX 链上质押分量=vote 年化（fin_trx_vote_apr 最新
// current_rate%）+rental 年化（近 7d rental_income 实收/最近
// energy_delegated 快照折算——业主实盘同源式：样本不足=rental 段缺席
// 只计 vote 如实）。表域缺席（42P01）=缺席如实。
func trxOnchainYield(ctx context.Context, pool *pgxpool.Pool, staleHours float64) (Yield, bool, error) {
	var votePct float64
	var vts time.Time
	err := pool.QueryRow(ctx, `SELECT current_rate, fetched_at FROM fin_trx_vote_apr
		ORDER BY fetched_at DESC LIMIT 1`).Scan(&votePct, &vts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || pgIsAbsent(err) {
			return Yield{}, false, nil
		}
		return Yield{}, false, err
	}
	if !freshEnough(vts, staleHours) {
		return Yield{}, false, nil
	}
	rentalPct := 0.0
	var delegated int64
	if err := pool.QueryRow(ctx, `SELECT energy_delegated FROM fin_trx_account
		ORDER BY fetched_at DESC LIMIT 1`).Scan(&delegated); err == nil && delegated > 0 {
		var sun float64
		if err := pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_sun),0) FROM fin_trx_txs
			WHERE kind='rental_income' AND ts>now()-interval '7 days'`).Scan(&sun); err == nil && sun > 0 {
			rentalPct = (sun / 1e6 / 7 * 365) / (float64(delegated) / 1e6) * 100
		}
	}
	return Yield{APR: (votePct + rentalPct) / 100, Source: "trx_onchain"}, true, nil
}

// pgIsAbsent 域缺席判（undefined_table 42P01——桥表不在本域=缺席如实非错误）。
func pgIsAbsent(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "42P01"
}
