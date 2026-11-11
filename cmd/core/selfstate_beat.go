// selfstate_beat.go —— 自态摘要量表探（工单#10358 C 件——业主令④：
// 「会话冷启动账页」=进仓协议即读件）。每拍聚合冷启动必需的机器自态
// →量表行 ticket='self_state'（指纹闸抑同纹,重报窗 6h——会话随时
// 冷启,账页须近鲜）。会话读径=单行末帧:
//
//	SELECT note FROM ops_event WHERE event='量表' AND ticket='self_state'
//	ORDER BY id DESC LIMIT 1
//
// 账页=取向面非真相源——处置判词仍以 ops_event 直查为准（本页只省
// 冷启动首扫 N 查询）。
// 在场自态账面（业主备记⑩#10489）：在场认领数=此刻有会话在啃票,
// 账页可见化=随 C 件同生的备记交付面。
package main

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

const selfStateEvery = 6 * time.Hour

var depShaRe = regexp.MustCompile(`o-core:([0-9a-f]{7,40})`)

// selfStateBeat 自态账拍——部署/升件未决/工单/在场认领/NAV/策略态/闸面/
// 会话收口鲜度 七段一帧（各段读败=该段标 ? 不整帧弃——账页面读败如实
// 呈不虚构；INSERT 本身败=零帧让位静默同量表惯例）。
func (c coreService) selfStateBeat(ctx context.Context) {
	seg := map[string]string{}

	// ① 部署面：末 rc=0 部署行 sha+陈（origin/master 尖不在 DB 域,
	//   滞差判归 deploy_watch——本页载部署事实非闸语）。
	var depNote string
	var depTs time.Time
	if err := c.pool.QueryRow(ctx, `SELECT note,ts FROM ops_event
		WHERE seat='DEPLOY' AND event='部署' AND note LIKE '%rc=0%'
		ORDER BY id DESC LIMIT 1`).Scan(&depNote, &depTs); err == nil {
		sha := "?"
		if m := depShaRe.FindStringSubmatch(depNote); m != nil && len(m[1]) >= 12 {
			sha = m[1][:12]
		}
		seg["dep"] = fmt.Sprintf("%s(%s)", sha, time.Since(depTs).Round(time.Minute))
	} else {
		seg["dep"] = "?"
	}

	// ② 升件未决面（runbook ① 口径同式——引证/区间引证皆算销）。
	var escN int
	var escOld *int64
	if err := c.pool.QueryRow(ctx, `SELECT count(*), min(e.id) FROM ops_event e
		WHERE e.event='升件' AND e.ts>now()-interval '7 days'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
		  AND (x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')
		   OR EXISTS (SELECT 1 FROM regexp_matches(x.note,'升件#(\d+)-(\d+)','g') AS m(mm)
		     WHERE e.id BETWEEN mm[1]::bigint AND mm[2]::bigint)))`).
		Scan(&escN, &escOld); err == nil {
		old := ""
		if escOld != nil {
			old = fmt.Sprintf("(最陈#%d)", *escOld)
		}
		seg["esc"] = fmt.Sprintf("%d%s", escN, old)
	} else {
		seg["esc"] = "?"
	}

	// ③ OPEN 工单面（runbook 口径：无处置引证+无活在认领）+近限时。
	var tkN int
	var tkDue *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT count(*), min(t.ts +
		COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour')
		FROM ops_event t WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='处置' AND x.ts>t.ts
		  AND x.note ~ ('工单#'||t.id::text||'([^0-9]|$)'))
		AND NOT EXISTS (SELECT 1 FROM ops_event cl WHERE cl.event='认领' AND cl.ts>t.ts
		  AND cl.ts>now()-interval '2 hours'
		  AND cl.note ~ ('工单#'||t.id::text||'([^0-9]|$)'))`).
		Scan(&tkN, &tkDue); err == nil {
		due := ""
		if tkDue != nil {
			due = fmt.Sprintf("(近限%s)", tkDue.UTC().Format("01-02 15:04Z"))
		}
		seg["tk"] = fmt.Sprintf("%d%s", tkN, due)
	} else {
		seg["tk"] = "?"
	}

	// ④ 在场认领数（#10489 在场自态账——此刻有会话在啃票）。
	var claimN int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='认领' AND ts>now()-interval '2 hours'`).Scan(&claimN); err == nil {
		seg["claim"] = fmt.Sprintf("%d", claimN)
	} else {
		seg["claim"] = "?"
	}

	// ⑤ NAV 面：聚合末行值+陈（48h 鲜阈同 idle_capital/dir 判据——
	//   值入帧不入纹：浮点噪声不构结构变,纹闸只随结构态重报）。
	var nav *float64
	var navTs *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT equity,ts FROM md_equity_curve
		WHERE ccy='NAV' ORDER BY ts DESC LIMIT 1`).Scan(&nav, &navTs); err == nil && nav != nil {
		seg["nav"] = fmt.Sprintf("%.2f(%s)", *nav, time.Since(*navTs).Round(time.Minute))
	} else {
		seg["nav"] = "?"
	}

	// ⑥ 策略态+在仓面。
	var liveN, probeN, posN int
	if err := c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE state='live'),
		count(*) FILTER (WHERE state='probe'),
		(SELECT count(*) FROM fin_position WHERE closed_at IS NULL)
		FROM fin_strategy`).Scan(&liveN, &probeN, &posN); err == nil {
		seg["strat"] = fmt.Sprintf("live%d/probe%d/仓%d", liveN, probeN, posN)
	} else {
		seg["strat"] = "?"
	}

	// ⑦ 闸面：kill_switch 值+族熔断 latch 数+gate.kill.* 数。
	kill := "?"
	_ = c.pool.QueryRow(ctx,
		"SELECT value#>>'{}' FROM app_config WHERE key='risk.kill_switch'").Scan(&kill)
	var latchN, gateKillN int
	if err := c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE key LIKE 'guard.dir_broken.%'),
		count(*) FILTER (WHERE key LIKE 'gate.kill.%')
		FROM app_config WHERE value='true'::jsonb`).Scan(&latchN, &gateKillN); err == nil {
		seg["gate"] = fmt.Sprintf("kill=%s dir=%d gate=%d", kill, latchN, gateKillN)
	} else {
		seg["gate"] = fmt.Sprintf("kill=%s dir=? gate=?", kill)
	}

	// ⑧ 会话收口鲜度（唤知环活性——陈=认知环死窗）。
	var lastClose *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT max(ts) FROM ops_event
		WHERE event='会话收口'`).Scan(&lastClose); err == nil && lastClose != nil {
		seg["cog"] = time.Since(*lastClose).Round(time.Minute).String()
	} else {
		seg["cog"] = "?"
	}

	// 纹=结构态指纹（值随浮点漂者只入帧——纹闸只随结构变重报）。
	fp := fmt.Sprintf("%s|%s|%s|%s|%s", seg["dep"], seg["esc"], seg["tk"], seg["strat"], seg["gate"])
	if !gaugeDue(ctx, c, "self_state", fp, selfStateEvery) {
		return
	}
	note := fmt.Sprintf("v=%s 自态账: 部署=%s 升件未决=%s 工单=%s 在场认领=%s NAV=%s %s %s 会话收口=%s——冷启动账页(工单#10358 C件;明细仍自查 ops_event 真源)",
		fp, seg["dep"], seg["esc"], seg["tk"], seg["claim"], seg["nav"], seg["strat"], seg["gate"], seg["cog"])
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','self_state',$1)`,
		note); err != nil {
		return
	}
}
