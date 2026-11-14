// dir_gate.go —— 方向书子预算闸（棒3 真户前置件——carry 出血形死法与
// 方向爆仓形断崖不同,子账隔离）。strategy.direction∈{long_perp,
// short_perp}=方向书成员,开仓额外两判：
//
//	①亏完断粮:书内已实现净亏破 risk.dir.loss_budget_usd=方向书全拒
//	  新开（亏损预算是账不是建议;close 径不入本闸——退仓权保留）;
//	②书名义帽:open 持仓+在途 exec dir 名义+本意图≤
//	  risk.dir.max_notional_usd（聚合口径同总敞口读面）。
//
// 键缺=missing_param fail-closed;种子 0=方向书零预算全拒（毕业=业主
// 供数,owner 域——与 max_notional 同类）。查无行/非方向=nil 放行——
// 闸只约束可证实的方向书成员。
package risk

import "context"

// dirBookDeny 方向书成员开仓判（open 径内嵌——Check 判序尾段）；
// nil=过闸（非成员或两判俱过）。
func (g *Gate) dirBookDeny(ctx context.Context, in Intent) *Decision {
	var dir string
	if err := g.pool.QueryRow(ctx, `SELECT direction FROM fin_strategy
		WHERE id=$1`, in.StrategyRef).Scan(&dir); err != nil ||
		(dir != "long_perp" && dir != "short_perp") {
		return nil
	}
	dirCap, e1 := g.getF("risk.dir.max_notional_usd")
	dirBudget, e2 := g.getF("risk.dir.loss_budget_usd")
	if e1 != nil || e2 != nil {
		d := g.deny(ctx, in, "missing_param")
		return &d
	}
	var dirPnl float64
	_ = g.pool.QueryRow(ctx, `SELECT COALESCE(sum(c.amount),0)
		FROM fin_cashflow c JOIN fin_position p ON p.id=c.position_ref
		JOIN fin_strategy s ON s.id=p.strategy_ref
		WHERE s.direction IN ('long_perp','short_perp')`).Scan(&dirPnl)
	if dirPnl < -dirBudget {
		d := g.deny(ctx, in, "dir_budget_exhausted")
		return &d
	}
	var dirExp float64
	_ = g.pool.QueryRow(ctx, `SELECT COALESCE(sum(x),0) FROM (
		SELECT sum(p.notional_usd) x FROM fin_position p
			JOIN fin_strategy s ON s.id=p.strategy_ref
			WHERE p.closed_at IS NULL AND s.direction IN ('long_perp','short_perp')
		UNION ALL SELECT sum(e.notional_usd) FROM fin_execution e
			JOIN fin_strategy s ON s.id=e.strategy_ref
			WHERE e.state='open' AND s.direction IN ('long_perp','short_perp')
			AND NOT EXISTS(SELECT 1 FROM fin_position p
				WHERE p.execution_id=e.id)) t`).Scan(&dirExp)
	if dirExp+in.NotionalUSD > dirCap {
		d := g.deny(ctx, in, "dir_notional_cap")
		return &d
	}
	return nil
}
