// worldmodel_esc.go —— 世模探家（O 对世界的信念面——非自模看自己，
// 是看市场）。首件 calib_drift：信念(eval 预测净APR)×兑现(后代
// 实测净APR)系统性偏离=族信念不可信。
package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// calibDriftEscCheck 校准失真探(世模第一桩——O 对世界下过信念:
// 每张 fin_opp_eval 载 net_apr 预测=可证伪预期,后代实绩=兑现面;
// 预测-兑现系统性偏离=族信念不可信)。funding-hedge 实证:1613 后代
// 预测均值 -2.8% APR 仍照常生——预测面早存在,校准面缺席=失真
// 无信号面,世界信念写了没人对账。
// 时代守卫(工单#15866 取证钉案):eval 键(inst,template,direction)
// upsert 原地覆写——e.ts>pr.created_at 的行=开户后重估非入场信念,
// 拿它对兑现=时代错配幻影(funding-hedge 负簇 eval.ts 均开户后18d,
// 「预测-14.4% vs 实测+7.5%」即此病灶非判官判反)。判据:仅入场
// 时代信念(e.ts<=pr.created_at)入 n/pred/real;被改写户计 era_skip
// 如实报数不藏——不可对账的失真探沉默=如实(工单载改写根修面)。
// 判据:族内入场时代实测后代≥10 ∧ |mean(pred)−mean(real)|>5pp APR
// →升件;实测=fin_cashflow(live)+fin_paper_fill(paper)合并年化,
// 与种群榜同口径(simdrill 剔——演习单非世界信念兑现)。
// 探针只报失真不裁处置(处置=核 eval 模型病灶∨标 lethal 断粮
// ∨校准后放行——归唤知会话判词)。
func (c coreService) calibDriftEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT template_ref,n,pred,real,era_skip FROM (
		SELECT s.template_ref,
			count(*) FILTER (WHERE e.ts<=pr.created_at) n,
			count(*) FILTER (WHERE e.ts>pr.created_at) era_skip,
			avg(e.net_apr) FILTER (WHERE e.ts<=pr.created_at) pred,
			avg((COALESCE(cf.net,0)+COALESCE(pf.net,0))/NULLIF(s.notional_usd,0)
				/GREATEST(EXTRACT(EPOCH FROM(now()-COALESCE(s.entered_at,s.created_at)))/86400,1)*365)
				FILTER (WHERE e.ts<=pr.created_at) real
		FROM fin_strategy s
		JOIN fin_proposal pr ON pr.id=s.proposal_ref
		JOIN fin_opp_eval e ON e.id=pr.eval_id
		LEFT JOIN (SELECT p.strategy_ref,sum(c.amount) net FROM fin_position p
			JOIN fin_cashflow c ON c.position_ref=p.id GROUP BY 1) cf
			ON cf.strategy_ref=s.id
		LEFT JOIN (SELECT f.strategy_id,sum(f.amount_usd) net
			FROM fin_paper_fill f GROUP BY 1) pf ON pf.strategy_id=s.id
		WHERE s.template_ref NOT LIKE 'simdrill%' AND e.net_apr IS NOT NULL
			AND (cf.net IS NOT NULL OR pf.net IS NOT NULL)
		GROUP BY 1) t
	WHERE n>=10 AND abs(pred-real)>0.05
	ORDER BY abs(pred-real) DESC LIMIT 5`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	var drifted []string
	for rows.Next() {
		var tpl string
		var n, eraSkip int
		var pred, real float64
		if rows.Scan(&tpl, &n, &pred, &real, &eraSkip) == nil {
			drifted = append(drifted,
				fmt.Sprintf("%s(预测%+.1f%% vs 实测%+.1f%%,n=%d,改写%d)", tpl, pred*100, real*100, n, eraSkip))
		}
	}
	c.escEdgeUV(ctx, deadSince, "世模", "calib_drift", "@", len(drifted) > 0, func() string {
		return fmt.Sprintf(
			"世模/calib_drift: %d 族预测-兑现系统性失真（%s）——族信念不可信:核 eval 模型病灶∨标 lethal 断粮∨校准后放行;禁裸销",
			len(drifted), strings.Join(drifted, "、"))
	})
}
