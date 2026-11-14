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
