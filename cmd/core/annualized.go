// annualized.go —— 真账年化统一算式（工单O-34703/业主令#34702③）：
// 成年判据年化分母由 NAV 改部署本金——部署率低时 NAV 基数学失真
// （部署$100 月赚$2=NAV年化0.22%，策略真实年化24%被摊没，判据
// 不可达）。分子口径不动=fin_pnl_net（O-34619 净产单源）；分母
// =部署腿窗时加权均值，腿集与 moneyDist 同源单源：fin_position
// 按 opened_at/closed_at 与查询窗交叠加权（无 closed_at 计至窗
// 尾）+ fin_position_view kind='dcd' 策略域产品仓凭据现值快照
// （凭据无区间史——快照径，读面带 annualSnapMark）；dcd_owner/
// paper/Earn 腿不入。全部年化判据读面（盈利报告/自观账/饿探/死刑
// 梯度）同调 annualJudgment 判据输入单源出口——第二处算式=口径
// 漂移温床。O-34963 下限补丁：部署均值<floor（app_config
// annualized.deploy_floor_usd 缺省500）→判据回退 NAV 口径
// （小分母放大失真实证#34937/#34942）。
package main

import (
	"context"
	"fmt"
	"time"

	"o/internal/platform/config"
)

// annualSnapMark 快照径读面标（spec 钦定字面——DCD 凭据无区间史，
// 现值快照代时加权，审计面须可见降级径）。
const annualSnapMark = "（部署=现值快照）"

// deployedBasis 部署本金窗时加权均值。返回 mean=均值、ok=分母
// 可得（exec/dcd 双腿皆读成才算得——缺腿分母不可信，读面"—"
// 不虚计，同 moneyDist 缺段降级律）、snap=DCD 快照成分在场
// （>0 时读面须带 annualSnapMark）。
func (c coreService) deployedBasis(ctx context.Context, start, end time.Time) (mean float64, ok bool, snap bool) {
	var execMean float64
	execOK := c.pool.QueryRow(ctx, `SELECT COALESCE(sum(p.notional_usd *
		EXTRACT(EPOCH FROM (LEAST(COALESCE(p.closed_at,$2),$2)
			- GREATEST(p.opened_at,$1)))
		/ NULLIF(EXTRACT(EPOCH FROM ($2-$1)),0)),0)
	FROM fin_position p
	WHERE p.opened_at<$2 AND COALESCE(p.closed_at,$2)>$1`,
		start, end).Scan(&execMean) == nil
	var dcd float64
	dcdOK := c.pool.QueryRow(ctx, `SELECT COALESCE(sum(v.notional*CASE
		WHEN v.ccy IN ('USDT','USD') THEN 1 ELSE COALESCE(px.p,0) END),0)
	FROM fin_position_view v
	LEFT JOIN LATERAL (SELECT p FROM (
		SELECT ts,last AS p FROM md_ticks WHERE inst_id=v.ccy||'-USDT'
		UNION ALL
		SELECT ts,px AS p FROM md_mark_index WHERE inst_id=v.ccy||'-USDT'
	  ) u ORDER BY ts DESC LIMIT 1) px ON true
	WHERE v.kind='dcd'`).Scan(&dcd) == nil
	if !execOK || !dcdOK {
		return 0, false, false
	}
	return execMean + dcd, true, dcd > 0
}

// annualizedPct 统一年化算式：窗净×(365/窗天数)/部署均值×100。
// deployed<=0→ok=false（读面"—"，不虚计 0% 不炸帧）。
func annualizedPct(net float64, days, deployed float64) (float64, bool) {
	if deployed <= 0 {
		return 0, false
	}
	return net * 365 / days / deployed * 100, true
}

// annualDepTag 部署分母读面签——可得带值+快照标，不可得"?"，
// 零部署显式 $0（与"—"分档：0=实测零部署，?=读败）。
func annualDepTag(dep float64, ok, snap bool) string {
	if !ok {
		return "（部署?）"
	}
	tag := fmt.Sprintf("（部署$%.0f）", dep)
	if snap {
		tag += annualSnapMark
	}
	return tag
}

// annualDeployFloor 判据下限（工单O-34963/值守裁定#34962）：
// app_config annualized.deploy_floor_usd，缺省 500 fail-closed——
// 读败/键缺/非正值回落缺省（0 floor=部署口径恒生效=小分母病灶
// 复辟口）。
func annualDeployFloor(cfg *config.Client) float64 {
	if cfg != nil {
		if v, err := cfg.GetFloat("annualized.deploy_floor_usd"); err == nil && v > 0 {
			return v
		}
	}
	return 500
}

// annualJudgment 判据输入单源出口（O-34963——部署本金口径下限补丁，
// #34702③同令下限非改令）：部署腿时加权均值≥floor→部署口径判；
// <floor 或分母读败→回退 NAV 口径（net×365/days÷NAV——底线真相，
// 部署不足不得以小分母豁免达标判定；「分母不足」语义=证据不足
// 回退 NAV 判，非达标非不达标）。NAV 回退径沿用权益盲律：缺/
// 陈>48h/非正→ok=false（读面"—"不虚计）。返回 pct/ok/tag——
// tag 随实际消费口径同源，读面显什么判据就吃什么。
func (c coreService) annualJudgment(ctx context.Context, start, end time.Time,
	net, days float64) (pct float64, ok bool, tag string) {
	dep, depOK, snap := c.deployedBasis(ctx, start, end)
	floor := annualDeployFloor(c.cfg)
	if depOK && dep >= floor {
		p, ok := annualizedPct(net, days, dep)
		return p, ok, annualDepTag(dep, depOK, snap)
	}
	depTxt := "?"
	if depOK {
		depTxt = fmt.Sprintf("$%.0f", dep)
	}
	tag = fmt.Sprintf("（部署%s<下限%.0f→NAV口径）", depTxt, floor)
	if snap {
		tag += annualSnapMark
	}
	var nav *float64
	if err := c.pool.QueryRow(ctx, `SELECT equity FROM md_equity_curve
		WHERE ccy='NAV' AND ts>now()-interval '48 hours'
		ORDER BY ts DESC LIMIT 1`).Scan(&nav); err != nil || nav == nil || *nav <= 0 {
		return 0, false, tag
	}
	p, ok := annualizedPct(net, days, *nav)
	return p, ok, tag
}
