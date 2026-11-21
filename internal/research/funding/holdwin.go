package funding

// holdwin.go —— 实测持仓窗摊销判据（工单#19797 三分量②：eval 分母口径复核
// 根修——T-0521 opener 侧已实测定案，eval 侧补同源合流）。md_funding 该
// inst 近 30d sign 号收租连续段的天数中位——正号段=short_perp 收租期,
// 负号段=long_perp 收租期。sign(funding_rate)=0 行按 0 号分段（中断两侧段）。
// 段长=段内 max−min+一个结算窗（0.334d≈8h——末行后仍收一窗租）。
// 该 inst 无 sign 号段→0=facts 缺（caller fail-closed,不虚构摊销窗）。
// opener_gate.breakevenOK 同式调用（判据一源律——评估卡与开闸判据同底）。

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MeasuredHoldDays 实测持仓窗天数中位（30d 窗；无该号段=0 不虚构）。
func MeasuredHoldDays(ctx context.Context, pool *pgxpool.Pool, inst string, sign float64) (float64, error) {
	var d *float64
	err := pool.QueryRow(ctx, `WITH s AS (
		SELECT ts,funding_rate,LAG(funding_rate) OVER (ORDER BY ts) prev
		FROM md_funding WHERE inst_id=$1 AND ts>now()-interval '30 days'),
	r AS (SELECT ts,funding_rate,
		SUM(CASE WHEN sign(funding_rate)<>sign(COALESCE(prev,0)) THEN 1 ELSE 0 END)
		OVER (ORDER BY ts) rid FROM s),
	st AS (SELECT EXTRACT(EPOCH FROM (max(ts)-min(ts)))/86400.0+0.334 d,
		sign(max(funding_rate)) sg FROM r GROUP BY rid)
	SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY d) FROM st WHERE sg=$2`,
		inst, sign).Scan(&d)
	if err != nil {
		return 0, err
	}
	if d == nil {
		return 0, nil
	}
	return *d, nil
}
