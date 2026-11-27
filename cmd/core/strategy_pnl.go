// strategy_pnl.go —— O-41386 策略盈亏榜（ListStrategyPnl）：
// fin_pnl_net 钦定账本行 join 仓→策略聚合——策略账非流水账；
// position_ref NULL 行（账外腿/现货转换）经 inner join 天然不入围；
// net=realized+fees+funding 三者和（服务端算定前端零判据）。
package main

import (
	"context"
	"sort"
	"time"

	"o/internal/gen/dashboardv1"
)

// ListStrategyPnl 策略盈亏榜（days<=0=30，上限 180——窗外截断不猜）。
func (c coreService) ListStrategyPnl(ctx context.Context,
	req *dashboardv1.ListStrategyPnlRequest) (*dashboardv1.ListStrategyPnlResponse, error) {
	days := int(req.GetDays())
	if days <= 0 {
		days = 30
	}
	if days > 180 {
		days = 180
	}
	rows, err := c.pool.Query(ctx, `
		SELECT s.id, s.template_ref,
			COALESCE(SUM(n.amount) FILTER (WHERE n.kind='trade_pnl'),0),
			COALESCE(SUM(n.amount) FILTER (WHERE n.kind='fee'),0),
			COALESCE(SUM(n.amount) FILTER (WHERE n.kind='funding_settle'),0),
			COUNT(DISTINCT n.position_ref),
			MAX(n.occurred_at)
		FROM fin_pnl_net n
		JOIN fin_position p ON p.id=n.position_ref
		JOIN fin_strategy s ON s.id=p.strategy_ref
		WHERE n.occurred_at > now() - make_interval(days=>$1)
		GROUP BY s.id, s.template_ref`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*dashboardv1.StrategyPnlRow
	for rows.Next() {
		var (
			sidI             int64
			tmpl             string
			real, fees, fund float64
			posN             int32
			lastTS           time.Time
		)
		if err := rows.Scan(&sidI, &tmpl, &real, &fees, &fund, &posN, &lastTS); err != nil {
			return nil, err
		}
		list = append(list, &dashboardv1.StrategyPnlRow{
			StrategyRef: sidI, Template: tmpl,
			RealizedUsd: real, FeesUsd: fees, FundingUsd: fund,
			NetUsd: real + fees + fund, Positions: posN,
			LastTs: tsProto(lastTS),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].NetUsd > list[j].NetUsd })
	return &dashboardv1.ListStrategyPnlResponse{Rows: list}, nil
}
