// evalqueue_face.go —— 「评估与队列」哨兵域四行（T-0218——鲜度哨兵管「数据
// 来没来」管不到「判词说什么」：unevaluable 语义停摆挂数周无人见实证在案）。
// 行义=判词分布/转化漏斗/审批积压/模板名册——语义判态非鲜度判态，host=sg、
// cat=ops、exch=评估队列（系统页 SG 卡内独立收拢组，前端零改动）。
// 判据钦定（服务端算定，前端零判据）：
//
//	① eval 分布：24h 窗 unevaluable 占比 >50% 黄 >75% 红（实测基线≈33%，
//	  主因 coins_absent=结构缺席非病灶；骤变主因=写径病灶信号）。
//	② proposal 漏斗：armed 最老龄 >12h 黄 >24h 红（24h=lifecycle.armed_ttl_h
//	  缺省——超时未退役=淤积闸失效）。
//	③ hypothesis 队列：proposed 最老 >7d 黄 >14d 红（审批=决策席人工件，
//	  数周积压=盲窗复现）。
//	④ template 名册：恒绿——candidate（enabled=f）灰显列名属正常态。
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"o/internal/gen/dashboardv1"
)

const evalQueueExch = "评估队列"

func eqComp(name, fact string) *dashboardv1.SystemComponent {
	return &dashboardv1.SystemComponent{Component: name, Fact: fact,
		Host: "sg", Cat: "ops", Exch: evalQueueExch}
}

func eqFail(c *dashboardv1.SystemComponent, err error) *dashboardv1.SystemComponent {
	log.Printf("[evalqueue] %s 查询败: %v", c.Component, err)
	c.State, c.Note = 2, "查询败（读件缺）"
	return c
}

// evalQueueRows 评估与队列四行——逐行独立查询，败=该行红不拖全集。
func (c coreService) evalQueueRows(ctx context.Context) []*dashboardv1.SystemComponent {
	return []*dashboardv1.SystemComponent{
		c.evalVerdictRow(ctx), c.proposalFunnelRow(ctx),
		c.hypothesisQueueRow(ctx), c.templateRosterRow(ctx)}
}

// evalVerdictRow fin_opp_eval 24h verdict 三色分布+unevaluable top3 reason
// （evidence->>reason 机读面——判词漂移一眼见）。
func (c coreService) evalVerdictRow(ctx context.Context) *dashboardv1.SystemComponent {
	comp := eqComp("评估判词分布", "fin_opp_eval verdict 24h 分布")
	var prop, rej, unev int64
	var last *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE verdict='proposed'),
		count(*) FILTER (WHERE verdict='rejected'),
		count(*) FILTER (WHERE verdict='unevaluable'), max(ts)
		FROM fin_opp_eval WHERE ts>now()-interval '24 hours'`).
		Scan(&prop, &rej, &unev, &last); err != nil {
		return eqFail(comp, err)
	}
	total := prop + rej + unev
	if total == 0 {
		comp.State, comp.Note = 2, "24h 零评估行——评估环停摆疑（读件缺）"
		return comp
	}
	if last != nil {
		comp.LastWrite = tsProto(*last)
	}
	ratio := float64(unev) / float64(total)
	switch {
	case ratio > 0.75:
		comp.State = 2
	case ratio > 0.5:
		comp.State = 1
	}
	var tops []string
	rrows, err := c.pool.Query(ctx, `SELECT coalesce(evidence->>'reason','<无 reason>'), count(*)
		FROM fin_opp_eval WHERE verdict='unevaluable' AND ts>now()-interval '24 hours'
		GROUP BY 1 ORDER BY 2 DESC LIMIT 3`)
	if err != nil {
		return eqFail(comp, err)
	}
	for rrows.Next() {
		var r string
		var n int64
		if err := rrows.Scan(&r, &n); err != nil {
			rrows.Close()
			return eqFail(comp, err)
		}
		tops = append(tops, fmt.Sprintf("%s×%d", r, n))
	}
	rrows.Close()
	comp.Note = fmt.Sprintf("24h proposed %d · rejected %d · unevaluable %d（%.0f%%）",
		prop, rej, unev, ratio*100)
	if len(tops) > 0 {
		comp.Note += "｜unev top: " + strings.Join(tops, " ")
	}
	if comp.State != 0 {
		comp.Note += "；占比超阈——查 SG o-core 评估环日志，unev 主因骤变=写径病灶非市场态"
	}
	return comp
}

// proposalFunnelRow fin_proposed→fin_armed 计数+armed 最老龄+24h 退役量
// （转化吞吐可见——armed 超 armed_ttl 未退役=淤积闸失效告警）。
func (c coreService) proposalFunnelRow(ctx context.Context) *dashboardv1.SystemComponent {
	comp := eqComp("提案状态漏斗", "fin_proposal proposed→armed 龄期+24h 退役")
	var prop, armed, retired24 int64
	var armedOldest, last *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE state='fin_proposed'),
		count(*) FILTER (WHERE state='fin_armed'),
		count(*) FILTER (WHERE state='fin_retired' AND retired_at>now()-interval '24 hours'),
		min(created_at) FILTER (WHERE state='fin_armed'), max(created_at)
		FROM fin_proposal`).Scan(&prop, &armed, &retired24, &armedOldest, &last); err != nil {
		return eqFail(comp, err)
	}
	if last != nil {
		comp.LastWrite = tsProto(*last)
	}
	armedAge := "—"
	if armedOldest != nil {
		d := time.Since(*armedOldest)
		armedAge = durHuman(d)
		switch {
		case d > 24*time.Hour:
			comp.State = 2
		case d > 12*time.Hour:
			comp.State = 1
		}
	}
	comp.Note = fmt.Sprintf("proposed %d → armed %d（最老 %s）｜24h 退役 %d",
		prop, armed, armedAge, retired24)
	if comp.State != 0 {
		comp.Note += "；armed 超龄未退役=retireStaleArmed 淤积闸失效——查 lifecycle.armed_ttl_h"
	}
	return comp
}

// hypothesisQueueRow fin_opp_hypothesis proposed 卡数+最老龄期（审批队列=
// 决策席人工件——积压数周=盲窗复现告警）。
func (c coreService) hypothesisQueueRow(ctx context.Context) *dashboardv1.SystemComponent {
	comp := eqComp("假想卡审批队列", "fin_opp_hypothesis state=proposed 龄期")
	var n int64
	var oldest, last *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT count(*), min(ts), max(ts)
		FROM fin_opp_hypothesis WHERE state='proposed'`).
		Scan(&n, &oldest, &last); err != nil {
		return eqFail(comp, err)
	}
	if last != nil {
		comp.LastWrite = tsProto(*last)
	}
	if n == 0 {
		comp.Note = "审批队列空"
		return comp
	}
	d := time.Since(*oldest)
	switch {
	case d > 14*24*time.Hour:
		comp.State = 2
	case d > 7*24*time.Hour:
		comp.State = 1
	}
	comp.Note = fmt.Sprintf("proposed %d · 最老 %.1f 天", n, d.Hours()/24)
	if comp.State != 0 {
		comp.Note += "；审批积压——决策席 adjudication 待办（卡非执行单，堆积=盲窗）"
	}
	return comp
}

// templateRosterRow fin_opp_template enabled/candidate 分色——新族未启=
// 候选态灰显属正常（恒绿，名册可见即值）。
func (c coreService) templateRosterRow(ctx context.Context) *dashboardv1.SystemComponent {
	comp := eqComp("机会模板名册", "fin_opp_template enabled/candidate")
	var enabled, cand int64
	var names *string
	if err := c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE enabled), count(*) FILTER (WHERE NOT enabled),
		string_agg(template||':'||venue,'、') FILTER (WHERE NOT enabled)
		FROM fin_opp_template`).Scan(&enabled, &cand, &names); err != nil {
		return eqFail(comp, err)
	}
	if enabled+cand == 0 {
		comp.State, comp.Note = 2, "零行——读件缺"
		return comp
	}
	comp.Note = fmt.Sprintf("enabled %d · candidate %d", enabled, cand)
	if cand > 0 && names != nil {
		comp.Note += "（" + *names + "）"
	}
	return comp
}
