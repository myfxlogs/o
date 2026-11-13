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
	"strconv"
	"strings"
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

// paperDryEscCheck 练习场旱情探（业主令 2026-11-13「经验管接通」件②——
// paper/probe 槽全空=无候选在练：手艺不进则退，practice field 空转=饿
// 探之外的第二钱面信号）。二元死态（≥1 即湿不自扰），fp="@" 回合起刻
// 纹——复干新回合照火。探针只述旱不开药：补料径=提案→admitProposals→
// paper 开户全机读在案，干的是上游提案供水（研究员/目标票域）。
func (c coreService) paperDryEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var n int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy
		WHERE state IN ('paper','probe')`).Scan(&n); err != nil {
		return // 读件败本拍让位
	}
	c.escEdgeUV(ctx, deadSince, "动机", "paper_dry", "@", n == 0, func() string {
		return "动机/paper_dry: 练习场零候选（paper+probe=0——无策略在练,经验管断料;补料归目标票/研究员提案径,admitProposals 开户面在案）"
	})
}

// goalIdleEscCheck 目标票怠探（「想≠做」缝实证 2026-11-14：goal-nav-apr
// 立票 10h 零推进——目标票躺工单面无拉力,升件面必拣而工单面可无视）。
// 判据=状态非散文:OPEN goal 票(同 codeworkStall 未销形)票龄>4h 且
// 近 8h 无任何处置行点名工单#id(推进/接续/挂起皆算在场触)。触出现=
// 自复绿,复怠=新回合照火(escalateUnresolved 律)。探针只喊不开药:
// 认领票干活∨接续挂起留据∨证伪销票,裁径归会话(runbook 动机档)。
func (c coreService) goalIdleEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT e.id, left(e.note,50),
		round(extract(epoch from now()-e.ts)/3600)
		FROM ops_event e WHERE e.event='工单' AND e.note LIKE '工单/goal-%'
		AND e.ts > now() - interval '30 days' AND e.ts < now() - interval '4 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
			AND d.id>e.id AND d.note ~ ('工单#'||e.id||'([^0-9]|$)')
			AND d.note NOT LIKE '%挂起条件=%' AND d.note !~ ('工单#'||e.id||' ?接续'))
		AND NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
			AND d.id>e.id AND d.note ~ ('工单#'||e.id||'([^0-9]|$)')
			AND d.ts > now() - interval '8 hours')
		ORDER BY e.id LIMIT 8`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	var parts []string
	var ids []string
	for rows.Next() {
		var id int64
		var note string
		var age float64
		if rows.Scan(&id, &note, &age) != nil {
			return
		}
		ids = append(ids, strconv.FormatInt(id, 10))
		parts = append(parts, fmt.Sprintf("#%d(%.0fh):%s", id, age, note))
	}
	c.escEdgeUV(ctx, deadSince, "动机", "goal_idle",
		strings.Join(ids, "+"), len(ids) > 0, func() string {
			return fmt.Sprintf(
				"动机/goal_idle: %d 张目标票期限内零推进（%s）——想要空转:认领=拣票干活写推进行∨接续挂起留据∨证伪销票,禁无视(runbook 动机档)",
				len(ids), strings.Join(parts, "、"))
		})
}
