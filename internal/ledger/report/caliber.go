// caliber.go —— 实现盈亏统一口径单源（T-0422 三面同读钦定式——口径定义节
// 载 docs/运维/周报口径.md）：
//
//	实现盈亏 = 现金盈亏钉式集（occurred_at 业务时窗）
//	          + 期内平仓仓腿差（closed_at 窗归属——实现日=平仓日，仓内置换
//	            流量不按腿发生日计损，防「开仓买腿当亏」假亏）
//	          + 裸袋挂账冲回（同 closed_at 窗——未回资产挂账估值非实现损，
//	            T-0419 settled_amount=Σcashflow+bare_bag_usd 同口径）。
//
//	未平仓仓 spot 腿=置换流量（资产仍持、NAV 已计值）不入实现盈亏；
//	earn_*/deposit/withdraw/owner_*/dust_*/recon_adjust=移动/校准非盈亏。
//	消费面：fin_report net/cum_nav、Overview pnl_today/pnl_cum、
//	驾驶舱 NetMonth/Daily——四面同读一式，禁面侧另写口径 SQL。
package report

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// realizedKinds 现金盈亏钉式集（class='income' 现金侧——spot 腿另行按仓归属；
// earn_*/transfer/adjustment 全出）。rollback_cost 在册（fin_0133 补 class 洞）。
const realizedKinds = `('interest','funding_settle','trade_pnl','fee','rollback_cost')`

// realizedSQL [$1,$2) 窗实现盈亏——现金侧按 occurred_at，腿/袋侧按 closed_at
// （实现时点）；bare_bag_usd NULL=无价不挂虚值同 settle 口径（sum 自然跳 NULL）。
// 升件#10063②：venue<>'sim' 显拒夹具域（simdrill 以真 inst 播 venue='sim'
// 行——演习成本不得入业主盈亏面；meta_esc 商业探针同口径先滤）+仓侧
// simdrill 模板域同拒（纸面仓腿差/裸袋非实现损益）。
const realizedSQL = `SELECT COALESCE(
	  (SELECT sum(amount) FROM fin_cashflow
	    WHERE kind IN ` + realizedKinds + ` AND venue<>'sim' AND occurred_at>=$1 AND occurred_at<$2)
	+ COALESCE((SELECT sum(c.amount) FROM fin_cashflow c
	    JOIN fin_position p ON p.id=c.position_ref
	    WHERE c.kind IN ('spot_buy','spot_sell') AND p.closed_at>=$1 AND p.closed_at<$2
	      AND NOT EXISTS(SELECT 1 FROM fin_strategy s WHERE s.id=p.strategy_ref
	        AND s.template_ref LIKE 'simdrill%')),0)
	+ COALESCE((SELECT sum(p.bare_bag_usd) FROM fin_position p
	    WHERE p.closed_at>=$1 AND p.closed_at<$2
	      AND NOT EXISTS(SELECT 1 FROM fin_strategy s WHERE s.id=p.strategy_ref
	        AND s.template_ref LIKE 'simdrill%')),0), 0)`

// Realized 实现盈亏 [start,end)——start=零值=左无界（累计面：全史至 end）。
// 界为 timestamptz 绝对参——无会话时区歧（UTC 界由调用方定）。
func Realized(ctx context.Context, pool *pgxpool.Pool, start, end time.Time) (float64, error) {
	var v float64
	err := pool.QueryRow(ctx, realizedSQL, start, end).Scan(&v)
	return v, err
}

// RealizedDaily 逐日实现盈亏序列（since 起、UTC 日分桶升序——零行日不出桶，
// 稀疏语义同驾驶舱原 ts::date 序列）。
func RealizedDaily(ctx context.Context, pool *pgxpool.Pool, since time.Time) ([]float64, error) {
	rows, err := pool.Query(ctx, `SELECT d, sum(v) FROM (
	  SELECT (occurred_at AT TIME ZONE 'UTC')::date d, amount::float8 v FROM fin_cashflow
	    WHERE kind IN `+realizedKinds+` AND venue<>'sim' AND occurred_at>=$1
	  UNION ALL
	  SELECT (p.closed_at AT TIME ZONE 'UTC')::date, c.amount::float8 FROM fin_cashflow c
	    JOIN fin_position p ON p.id=c.position_ref
	    WHERE c.kind IN ('spot_buy','spot_sell') AND p.closed_at>=$1
	      AND NOT EXISTS(SELECT 1 FROM fin_strategy s WHERE s.id=p.strategy_ref
	        AND s.template_ref LIKE 'simdrill%')
	  UNION ALL
	  SELECT (p.closed_at AT TIME ZONE 'UTC')::date, p.bare_bag_usd::float8 FROM fin_position p
	    WHERE p.closed_at>=$1 AND p.bare_bag_usd IS NOT NULL
	      AND NOT EXISTS(SELECT 1 FROM fin_strategy s WHERE s.id=p.strategy_ref
	        AND s.template_ref LIKE 'simdrill%')
	) u GROUP BY d ORDER BY d`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []float64
	for rows.Next() {
		var d time.Time
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
