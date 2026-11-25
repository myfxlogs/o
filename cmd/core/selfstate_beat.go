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
	"strings"
	"time"
)

const selfStateEvery = 6 * time.Hour

var depShaRe = regexp.MustCompile(`o-core:([0-9a-f]{7,40})`)

// depSegSha 部署段 sha 提取（指纹件——age 是时漂件非结构件:
// A1 实证指纹含「sha(龄)」=分钟变=gaugeDue 6h 闸永火,
// self_state 4522帧/30d。指纹只认 sha,龄只入帧面 display）。
func depSegSha(note string) string {
	if m := depShaRe.FindStringSubmatch(note); m != nil && len(m[1]) >= 12 {
		return m[1][:12]
	}
	return "?"
}

// selfStateBeat 自态账拍——部署/升件未决/工单/在场认领/NAV/策略态/闸面/
// 会话收口鲜度 七段一帧（各段读败=该段标 ? 不整帧弃——账页面读败如实
// 呈不虚构；INSERT 本身败=零帧让位静默同量表惯例）。
func (c coreService) selfStateBeat(ctx context.Context) {
	seg := map[string]string{}

	// ① 部署面：末 rc=0 部署行 sha+陈（origin/master 尖不在 DB 域,
	//   滞差判归 deploy_watch——本页载部署事实非闸语）。
	var depNote string
	var depTs time.Time
	depSha := "?" // 指纹件——只认 sha 不认龄（A1:龄入指纹=分钟变=闸永火）
	if err := c.pool.QueryRow(ctx, `SELECT note,ts FROM ops_event
		WHERE seat='DEPLOY' AND event='部署' AND note LIKE '%rc=0%'
		AND note LIKE '%o-core:%'
		ORDER BY id DESC LIMIT 1`).Scan(&depNote, &depTs); err == nil { // T-13283：行类滤先——deploy-oci 臂行同 seat/event/rc=0 无 o-core: 标记，取最新行后 extract 必 ?(假滞差源)
		sha := depSegSha(depNote)
		depSha = sha
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

	// ③ OPEN 工单面（口径对齐 dispatch 派活件:A4 实证本页缺挂起/
	//   接续两剔——处置载 挂起条件=∨接续 不销票,挂起暂存期限活=
	//   停放不入OPEN拣面;无活在认领+近限时）。
	var tkN int
	var tkDue *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT count(*), min(t.ts +
		COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour')
		FROM ops_event t WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='处置' AND x.ts>t.ts
		  AND x.note ~ ('工单#'||t.id::text||'([^0-9]|$)')
		  AND x.note NOT LIKE '%挂起条件=%'
		  AND x.note !~ ('工单#'||t.id::text||' ?接续'))
		AND NOT EXISTS (SELECT 1 FROM ops_event s WHERE s.event='处置' AND s.ts>t.ts
		  AND s.note ~ ('工单#'||t.id::text||'([^0-9]|$)') AND s.note LIKE '%挂起条件=%'
		  AND t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour' > now())
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
		(SELECT count(*) FROM fin_position WHERE state<>'closed')
		FROM fin_strategy`).Scan(&liveN, &probeN, &posN); err == nil {
		seg["strat"] = fmt.Sprintf("live%d/probe%d/仓%d", liveN, probeN, posN)
	} else {
		seg["strat"] = "?"
	}

	// ⑦ 闸面：kill_switch 值+族熔断 latch 数+gate.kill.* 数+死刑因子
	// （疼要挂在脸上——factor 不在账页显=隐痛,能力收缩它看不见;
	// 因子变=结构变入纹,梯度换挡自然重报）。
	kill := "?"
	_ = c.pool.QueryRow(ctx,
		"SELECT value#>>'{}' FROM app_config WHERE key='risk.kill_switch'").Scan(&kill)
	death := "1"
	var dv string
	if err := c.pool.QueryRow(ctx,
		"SELECT value#>>'{}' FROM app_config WHERE key='lifecycle.death_factor'").Scan(&dv); err == nil {
		death = dv
	}
	var latchN, gateKillN int
	if err := c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE key LIKE 'guard.dir_broken.%'),
		count(*) FILTER (WHERE key LIKE 'gate.kill.%')
		FROM app_config WHERE value='true'::jsonb`).Scan(&latchN, &gateKillN); err == nil {
		seg["gate"] = fmt.Sprintf("kill=%s dir=%d gate=%d death=%s", kill, latchN, gateKillN, death)
	} else {
		seg["gate"] = fmt.Sprintf("kill=%s dir=? gate=? death=%s", kill, death)
	}

	// ⑧ 会话收口鲜度（唤知环活性——陈=认知环死窗）。
	var lastClose *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT max(ts) FROM ops_event
		WHERE event='会话收口'`).Scan(&lastClose); err == nil && lastClose != nil {
		seg["cog"] = time.Since(*lastClose).Round(time.Minute).String()
	} else {
		seg["cog"] = "?"
	}

	// ⑨ 升件根聚类（O-39420 举一反三触发拍——同根因病族复发摆到
	//   唤知眼前：(esc_class,esc_scope) 二元组窗内计数 topN,n≥2
	//   标「疑似同族」纯计数标不含判断——类化判归会话 canon
	//   class-escalation-rule。窗/截断=cfg 键缺=fail-closed 段标 ?；
	//   读败=读败不炸帧。值入帧不入纹——计数漂移不构结构变,6h
	//   重报闸不被聚类噪声烧穿）。
	seg["cluster"] = c.escClusterSeg(ctx)

	// 纹=结构态指纹（值随浮点漂者只入帧——纹闸只随结构变重报）。
	fp := fmt.Sprintf("%s|%s|%s|%s|%s", depSha, seg["esc"], seg["tk"], seg["strat"], seg["gate"])
	if !gaugeDue(ctx, c, "self_state", fp, selfStateEvery) {
		return
	}
	note := fmt.Sprintf("v=%s 自态账: 部署=%s 升件未决=%s 升件聚类=%s 工单=%s 在场认领=%s NAV=%s %s %s 会话收口=%s——冷启动账页(工单#10358 C件;明细仍自查 ops_event 真源)",
		fp, seg["dep"], seg["esc"], seg["cluster"], seg["tk"], seg["claim"], seg["nav"], seg["strat"], seg["gate"], seg["cog"])
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','self_state',$1)`,
		note); err != nil {
		return
	}
}

// escClusterRow 聚类行（esc_class×esc_scope 计数——读面纯件）。
type escClusterRow struct {
	class, scope string
	n            int
}

// escClusterSeg 升件根聚类段——cfg 缺=?、读败=读败、零行=Nd:0
// 如实（O-39420：帧只投影不定性——疑似同族标=纯计数标,类化判
// 归会话；值入帧不入纹）。
func (c coreService) escClusterSeg(ctx context.Context) string {
	days, top, ok := c.escClusterCfg(ctx)
	if !ok {
		return "?"
	}
	rows, err := c.pool.Query(ctx, `SELECT esc_class,esc_scope,count(*)
		FROM ops_event
		WHERE event='升件' AND ts>now()-make_interval(days=>$1)
		  AND esc_class IS NOT NULL AND esc_class<>''
		  AND esc_scope IS NOT NULL AND esc_scope<>''
		GROUP BY 1,2 ORDER BY 3 DESC,1,2 LIMIT $2`, days, top)
	if err != nil {
		return "读败"
	}
	defer rows.Close()
	var cls []escClusterRow
	for rows.Next() {
		var r escClusterRow
		if err := rows.Scan(&r.class, &r.scope, &r.n); err != nil {
			return "读败"
		}
		cls = append(cls, r)
	}
	if err := rows.Err(); err != nil {
		return "读败"
	}
	return fmt.Sprintf("%dd:%s", days, escClusterText(cls))
}

// escClusterCfg 窗/截断参数（app_config selfstate.esc_cluster_{days,
// topn}——双键齐才探,缺一=fail-closed；值域 days∈(0,365]∧top∈(0,50]）。
func (c coreService) escClusterCfg(ctx context.Context) (days, top int, ok bool) {
	var d, t float64
	if err := c.pool.QueryRow(ctx,
		`SELECT (value#>>'{}')::float8 FROM app_config WHERE key='selfstate.esc_cluster_days'`).Scan(&d); err != nil || d <= 0 || d > 365 {
		return 0, 0, false
	}
	if err := c.pool.QueryRow(ctx,
		`SELECT (value#>>'{}')::float8 FROM app_config WHERE key='selfstate.esc_cluster_topn'`).Scan(&t); err != nil || t <= 0 || t > 50 {
		return 0, 0, false
	}
	return int(d), int(t), true
}

// escClusterText 聚类段文纯件——「class@scope×n[疑似同族]」| 分隔,
// n≥2 标疑似同族（纯计数标——类化判归会话）；零行=「0」如实。
func escClusterText(rows []escClusterRow) string {
	if len(rows) == 0 {
		return "0"
	}
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		s := fmt.Sprintf("%s@%s×%d", r.class, r.scope, r.n)
		if r.n >= 2 {
			s += "[疑似同族]"
		}
		if len(s) > 120 {
			s = s[:117] + "…"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "|")
}
