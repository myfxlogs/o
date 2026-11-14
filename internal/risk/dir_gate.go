// dir_gate.go —— 方向书子预算闸（棒3 真户前置件,棒5 改百分比口径——业主
// 语义是「OKX 余额的百分数」非定值美元,余额动则定值漂移）。strategy.
// direction∈{long_perp,short_perp}=方向书成员,开仓三判（权益=venue
// 现读,读败=fail-closed 同 venue 集中度径）：
//
//	①亏完断粮:书内已实现净亏破 loss_budget_pct×equity=方向书全拒新开
//	  （累计亏损预算是账不是建议;close 径不入本闸——退仓权保留）;
//	②单笔帽:本意图名义×止损距(lifecycle.dir.stop_frac——与 dirpaper
//	  出场同一真源)破 trade_loss_pct×equity=拒——单笔最坏亏损界;
//	③书名义帽:open 持仓+在途 exec dir 名义+本意图>
//	  max_notional_pct×equity（聚合口径同总敞口读面——簇止损最坏亏
//	  =帽×stop_frac,预算内收敛）。
//
// 键缺=missing_param fail-closed;阀值=棒5 系统自推基线（单笔0.5%=
// 10%断粮÷20笔样本——3%只够3笔实验自产证据前先死;名义20%×15%止损
// =簇亏3%<断粮线;owner 写域业主可覆）。查无行/非方向=nil 放行——
// 闸只约束可证实的方向书成员。
package risk

import "context"

// dirBookDeny 方向书成员开仓判（open 径内嵌——Check 判序尾段）；
// nil=过闸（非成员或三判俱过）。
func (g *Gate) dirBookDeny(ctx context.Context, in Intent) *Decision {
	var dir string
	if err := g.pool.QueryRow(ctx, `SELECT direction FROM fin_strategy
		WHERE id=$1`, in.StrategyRef).Scan(&dir); err != nil ||
		(dir != "long_perp" && dir != "short_perp") {
		return nil
	}
	budgetPct, e1 := g.getF("risk.dir.loss_budget_pct")
	tradePct, e2 := g.getF("risk.dir.trade_loss_pct")
	notionalPct, e3 := g.getF("risk.dir.max_notional_pct")
	stopFrac, e4 := g.getF("lifecycle.dir.stop_frac")
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		d := g.deny(ctx, in, "missing_param")
		return &d
	}
	if g.equity == nil {
		d := g.deny(ctx, in, "missing_param")
		return &d
	}
	eq, err := g.equity(ctx, in.Venue)
	if err != nil {
		d := g.deny(ctx, in, "equity_missing")
		return &d
	}
	var dirPnl float64
	_ = g.pool.QueryRow(ctx, `SELECT COALESCE(sum(c.amount),0)
		FROM fin_cashflow c JOIN fin_position p ON p.id=c.position_ref
		JOIN fin_strategy s ON s.id=p.strategy_ref
		WHERE s.direction IN ('long_perp','short_perp')`).Scan(&dirPnl)
	if dirPnl < -budgetPct*eq {
		d := g.deny(ctx, in, "dir_budget_exhausted")
		return &d
	}
	if in.NotionalUSD*stopFrac > tradePct*eq {
		d := g.deny(ctx, in, "dir_trade_loss_cap")
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
	if dirExp+in.NotionalUSD > notionalPct*eq {
		d := g.deny(ctx, in, "dir_notional_cap")
		return &d
	}
	return nil
}
