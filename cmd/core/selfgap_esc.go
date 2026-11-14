// selfgap_esc.go —— 短板信号面（业主令 2026-11-13「知道弱≠绕开弱」件①）。
// 自模账是被动帧（域面成败量表，不读不进视野）;本探是主动翼——域复燃
// 超阈=升件把失败史送到会话面前。
//
// 判据：stuck_esc 行 v= 前缀内 scope#id 对（销案宣称「已修/已愈」后再火
// 的 scope——判词不实/修错界的自证），同 scope 30d 内累计出现在 ≥5 张
// stuck_esc 行=该域慢性砸（selfmodel_beat 同径同口径——帧是账,探是哨）。
// fp=域名——一域一纹，30d 记忆期内同域不暴（慢性病一声足够,判词不实
// 复发=到期再验）。处置径=runbook 热域先验尸契约（认领热域票前必读
// 前判词,判词必答「上次为什么砸+这次哪不同」）。
package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// hotDomainMinRefires 热域阈——30d 内 ≥5 张 stuck_esc 行点名同域=慢性砸。
const hotDomainMinRefires = 5

// hotDomainEscCheck 热域探——域级复燃超阈升件（一域一 scope 各自沿）。
func (c coreService) hotDomainEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT m[1], count(DISTINCT e.id) FROM ops_event e,
		LATERAL regexp_matches(e.note,'([a-zA-Z0-9_-]+)#[0-9]+','g') m
		WHERE e.event='升件' AND e.esc_scope='stuck_esc'
		AND e.ts>now()-interval '30 days'
		GROUP BY 1 HAVING count(DISTINCT e.id)>=$1`, hotDomainMinRefires)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	type hot struct {
		dom string
		n   int
	}
	var hs []hot
	for rows.Next() {
		var h hot
		if err := rows.Scan(&h.dom, &h.n); err != nil {
			return
		}
		hs = append(hs, h)
	}
	for _, h := range hs {
		h := h
		c.escEdgeUV(ctx, deadSince, "唤知", "hotdom_"+h.dom, h.dom, true, func() string {
			return fmt.Sprintf(
				"唤知/hotdom_%s: 域「%s」30d 复燃 %d 次（stuck_esc 销案宣称已修后再火——判词不实/修错界自证）;认领此域工单前必读前判词并答「上次为什么砸+这次哪不同」(热域先验尸契约)",
				h.dom, h.dom, h.n)
		})
	}
	// 复绿沿：域下阈=死态撤（escEdgeUV dead=false 行自转恢复——本探域集
	// 由查询行驱动,撤态须逐 key 补发同 capability 对称沿）。
	seen := map[string]bool{}
	for _, h := range hs {
		seen["唤知/hotdom_"+h.dom] = true
	}
	for k, since := range deadSince {
		if strings.HasPrefix(k, "唤知/hotdom_") && !seen[k] {
			delete(deadSince, k)
			c.recoverNote(ctx, "唤知", strings.TrimPrefix(k, "唤知/"), since)
		}
	}
}

// dreamDueEscCheck 做梦到时探（「做梦」器官产线补位 2026-11-14——sleep-time
// compute 对拍:回放写面纯契约(ticket='dream_replay' 行等会话产)而无人
// 唤=器官 15% 刻度实证零产;无米不梦——素材尽=静息合法非僵)。
// 判据=状态非散文:末次回放>7d∨零史,且 30d 内有判毕素材案
// (falsified_verdict/retire_postmortem/trade_postmortem/settled_loss——
// 反事实得有「实态」可对)。产回放行=自复绿;素材尽=不唤。
// 探针只述「到时」不产梦:推演=会话域活(runbook 做梦回放面档——
// 拒虚构 venue 数,无实测只许簿记量差)。
func (c coreService) dreamDueEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var last *time.Time
	var material int
	if err := c.pool.QueryRow(ctx, `SELECT
		(SELECT max(ts) FROM ops_event WHERE ticket='dream_replay'),
		(SELECT count(*) FROM fin_econ_lesson
			WHERE kind IN ('falsified_verdict','retire_postmortem','trade_postmortem','settled_loss')
			AND status='verdicted' AND ts>now()-interval '30 days')`).
		Scan(&last, &material); err != nil {
		return // 读件败本拍让位
	}
	due := material > 0 && (last == nil || time.Since(*last) > 7*24*time.Hour)
	c.escEdgeUV(ctx, deadSince, "唤知", "dream_due", "@", due, func() string {
		since := "零史"
		if last != nil {
			since = fmt.Sprintf("%.0fd", time.Since(*last).Hours()/24)
		}
		return fmt.Sprintf(
			"唤知/dream_due: 回放面 %s 无梦，30d 判毕素材 %d 案——做梦=离线巩固非新信号,产 回放/dream_replay 行自复绿(runbook 做梦回放面档)",
			since, material)
	})
}

// museDueEscCheck muse 醒探（「思想者」件 2026-11-14——自发议程环:
// 自问账=上一个会话出的题,muse 醒=没任务的想,两拼=议程无探针参与,
// 「会答题不会出题」缝的补齐）。sleep-time 全义版:非指派材料回放,
// 是自己出题自己答的离线巩固。
// 判据=状态非散文:自问账有未答题(自问行无「自问#id」处置引)∧
// 未决升件<3(想不抢饭碗——队列重时静默)∧末次 muse>8h(思想不占带宽,
// ticket='muse' 行=想过的痕)。探针只唤不出题也不产答——议程=挑
// 自问账最值钱题想到底(runbook muse 档),答落 muse 行自复绿。
func (c coreService) museDueEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var qN, pendN int
	var lastMuse *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM ops_event e WHERE e.event='自问'
			AND e.ts>now()-interval '30 days'
			AND NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
				AND d.id>e.id AND d.note ~ ('自问#'||e.id||'([^0-9]|$)'))),
		(SELECT count(*) FROM ops_event e WHERE e.event='升件'
			AND e.ts>now()-interval '72 hours'
			AND NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.id>e.id
				AND d.note ~ ('升件#'||e.id||'([^0-9]|$)'))),
		(SELECT max(ts) FROM ops_event WHERE ticket='muse')`).
		Scan(&qN, &pendN, &lastMuse); err != nil {
		return // 读件败本拍让位
	}
	due := qN > 0 && pendN < 3 && (lastMuse == nil || time.Since(*lastMuse) > 8*time.Hour)
	c.escEdgeUV(ctx, deadSince, "唤知", "muse", "@", due, func() string {
		return fmt.Sprintf(
			"唤知/muse: 自问账 %d 题未答+队列 %d 轻——无任务的醒:挑最值钱一题想到底,结论落 muse 行自复绿(runbook muse 档;答不了=升级自问非弃)",
			qN, pendN)
	})
}

// learnGapEscCheck 求知滞探(求知环拉起缝根修):「求知:」前缀工单=
// 研究席触发④专用票(处置席契不领),但工单面=可沉默的施工面——
// O 环内无「工单→研究席」自发派发件,票能唤起无主的会话空转轨道。
// 判据:OPEN 求知票(未销∧无活在认领)票龄>2h→唤知/learn_gap 升件
// (升件面=保证注意力的面,滞票转升件即接上注意链——goal_idle 同款
// 传唤语义:本件非新需求,是把无主票推回必读流)。处置写到点名票:
// 认领干活(题域内外调→digest/canon 交付)∨escalate 转研究席派发
// ∨证伪销票;禁裸销本件。
func (c coreService) learnGapEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT t.id,
		EXTRACT(EPOCH FROM (now()-t.ts))/3600
	FROM ops_event t
	WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND t.note ~ '求知[:：]'
		AND t.ts<now()-interval '2 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='处置'
			AND x.id>t.id AND x.note ~ ('工单#'||t.id||'([^0-9]|$)'))
		AND NOT EXISTS (SELECT 1 FROM ops_event cl WHERE cl.event='认领'
			AND cl.id>t.id AND cl.ts>now()-interval '2 hours'
			AND cl.note ~ ('工单#'||t.id||'([^0-9]|$)'))
	ORDER BY t.id LIMIT 5`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	var stale []string
	for rows.Next() {
		var id int64
		var h float64
		if rows.Scan(&id, &h) == nil {
			stale = append(stale, fmt.Sprintf("#%d(%.0fh)", id, h))
		}
	}
	c.escEdgeUV(ctx, deadSince, "唤知", "learn_gap", "@", len(stale) > 0, func() string {
		return fmt.Sprintf(
			"唤知/learn_gap: %d 张求知票滞工单面零触（%s）——无主外调票转升件:认领干活(题域内外调→digest/canon)∨escalate转研究席∨证伪销;禁裸销",
			len(stale), strings.Join(stale, "、"))
	})
}

// museClaimEscCheck muse 落实探(自证绿缝根修):muse 会话自己答题
// 自己销题——产出形强制载「落=<canon∨工单#id∨回放∨升级自问>」,
// 机查得出来的字段没人查=散文形状噪声照样绿。判据:7d 内 muse 行
// 缺「落=」引用∧无后随处置核行→唤知/muse_claim 升件。处置=核该
// muse 行判词真落实(canonical/工单实存∨判不实追修);muse 行皆载
// 落=∨核毕处置落账=自复绿。
func (c coreService) museClaimEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var n int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event m
		WHERE m.event='muse' AND m.ts>now()-interval '7 days'
			AND m.note !~ '落='
			AND NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
				AND d.id>m.id AND d.note ~ 'muse')`).Scan(&n); err != nil {
		return // 读件败本拍让位
	}
	c.escEdgeUV(ctx, deadSince, "唤知", "muse_claim", "@", n > 0, func() string {
		return fmt.Sprintf(
			"唤知/muse_claim: %d 张 muse 行无「落=」引用=自证绿——核判词落实(canon/工单/回放/升级自问实存)∨判不实追修;禁裸销",
			n)
	})
}
