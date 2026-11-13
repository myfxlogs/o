// hunger_esc.go —— 动机器官（「饿」信号面,业主令 2026-11-13「长动机」件）。
// 人饿了自己找吃的;O 此前所有动作源头=探针报病∨业主令——没有一条源头是
// 「我想赚更多」。本探补该源头:目标差距=信号,目标由认知环自立（探针只
// 述饿不开药——目标票=会话写的才算它想要,runbook 动机档为契）。
//
// 判据=真账年化(30d净×365/30/NAV,owner_beat 同径同口径)<目标 8%
// （口径:adulthood-bar——成年判据②,达线前扩投禁）;NAV 缺/陈>48h 不判
// （权益盲时饿判无据,nav_stale 另有探）。commercial_stall 只问「净≤0 转而
// 不出水」,本探问「正也不够」——8% 线下皆饿,差距数随行。
//
// 环形:饿→升件→会话立 目标票(scope=hunger_gap)→票 OPEN 探针静默(票抑键)
// →票销/过期→差距仍在再饿（新目标）。差距闭合(年化≥8%)→复绿自销。
// 值纹=年化整数位——日漂不重火,未销在案不暴(escalateUnresolved 律)。
package main

import (
	"context"
	"fmt"
	"time"
)

const hungerTargetPct = 8.0 // 口径:adulthood-bar 真账年化线

// hungerGapEscCheck 饿探——真账年化 vs 8% 目标差分升件。
func (c coreService) hungerGapEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var nav *float64
	var net30 float64
	var liveN int
	if err := c.pool.QueryRow(ctx, `SELECT
		(SELECT equity FROM md_equity_curve WHERE ccy='NAV'
			AND ts>now()-interval '48 hours' ORDER BY ts DESC LIMIT 1),
		(SELECT COALESCE(sum(amount),0) FROM fin_cashflow
			WHERE venue<>'sim' AND occurred_at>now()-interval '30 days'
			AND kind IN ('fee','funding_settle','trade_pnl','rollback_cost','spot_buy','spot_sell')),
		(SELECT count(*) FROM fin_strategy WHERE state IN ('live','probe'))`).
		Scan(&nav, &net30, &liveN); err != nil {
		return // 读件败本拍让位
	}
	if nav == nil || *nav <= 0 {
		return // 权益盲=饿判无据（nav_stale 探另报）
	}
	annual := net30 * 365 / 30 / *nav * 100
	hungry := annual < hungerTargetPct
	c.escEdgeUV(ctx, deadSince, "动机", "hunger_gap",
		fmt.Sprintf("%.0f", annual), hungry, func() string {
			return fmt.Sprintf(
				"动机/hunger_gap: 真账年化 %.2f%% < 目标 %.0f%%（30d净 %.4f NAV %.0f live+probe %d）——差距 %.2f 个点;饿=信号非药方,目标票归认知环自立(runbook 动机档:工单/goal-* scope=hunger_gap,禁裸销)",
				annual, hungerTargetPct, net30, *nav, liveN, hungerTargetPct-annual)
		})
}
