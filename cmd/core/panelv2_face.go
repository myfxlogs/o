// panelv2_face.go —— 运作面板 v2 三径读面（docs/drafts/dashboard-spec.md：
// 钱径 GetFunnel/设备径 GetFaceGrid/决策径 GetDecisionTrail；判词判态转化
// 全后端算定，前端零计算零词表——同源探针判据双径不复述）。
package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"o/internal/gen/dashboardv1"
)

// ── 钱径 GetFunnel ──

// GetFunnel 评估漏斗：eval→提案→策略→仓位 四级窗内吞吐（各层独立
// fail-soft——单层读败=该层 n=-1 显缺不炸面，同 flow_pulse 律）。
func (c coreService) GetFunnel(ctx context.Context,
	req *dashboardv1.GetFunnelRequest) (*dashboardv1.GetFunnelResponse, error) {
	win := int(req.GetWindowDays())
	if win <= 0 {
		win = 30
	}
	resp := &dashboardv1.GetFunnelResponse{AsOf: timestamppb.Now(), WindowDays: int32(win)}
	iv := fmt.Sprintf("%d days", win)

	lvl := func(key, label string, n, openN int64, note string) {
		resp.Levels = append(resp.Levels, &dashboardv1.FunnelLevel{
			Key: key, Label: label, N: n, OpenN: openN, Note: note})
	}
	var n, openN int64
	// ①eval 层：窗内评估行+verdict 分布（evalqueue 同源口径）。
	var prop, rej, unev int64
	if err := c.pool.QueryRow(ctx, `SELECT count(*),
		count(*) FILTER (WHERE verdict='proposed'),
		count(*) FILTER (WHERE verdict='rejected'),
		count(*) FILTER (WHERE verdict='unevaluable')
		FROM fin_opp_eval WHERE ts>now()-$1::interval`, iv).
		Scan(&n, &prop, &rej, &unev); err != nil {
		log.Printf("[funnel] eval 层查询败: %v", err)
		lvl("eval", "评估", -1, -1, "读件缺")
	} else {
		lvl("eval", "评估", n, prop,
			fmt.Sprintf("proposed %d · rejected %d · unevaluable %d", prop, rej, unev))
	}
	// ②提案层：窗内提案行+现态在册（proposed+armed）。
	if err := c.pool.QueryRow(ctx, `SELECT count(*),
		count(*) FILTER (WHERE state IN ('fin_proposed','fin_armed'))
		FROM fin_proposal WHERE created_at>now()-$1::interval`, iv).
		Scan(&n, &openN); err != nil {
		log.Printf("[funnel] 提案层查询败: %v", err)
		lvl("proposal", "提案", -1, -1, "读件缺")
	} else {
		lvl("proposal", "提案", n, openN, fmt.Sprintf("现态在册 %d", openN))
	}
	// ③策略层：窗内入场策略+现态 live/probe 在册。
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE entered_at>now()-$1::interval),
		count(*) FILTER (WHERE state IN ('live','probe'))
		FROM fin_strategy`, iv).Scan(&n, &openN); err != nil {
		log.Printf("[funnel] 策略层查询败: %v", err)
		lvl("strategy", "策略", -1, -1, "读件缺")
	} else {
		lvl("strategy", "策略", n, openN, fmt.Sprintf("live+probe 在册 %d", openN))
	}
	// ④仓位层：窗内开仓+现态未平（fin_position 时标列=opened_at 非 ts）。
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE opened_at>now()-$1::interval),
		count(*) FILTER (WHERE closed_at IS NULL)
		FROM fin_position`, iv).Scan(&n, &openN); err != nil {
		log.Printf("[funnel] 仓位层查询败: %v", err)
		lvl("position", "仓位", -1, -1, "读件缺")
	} else {
		lvl("position", "仓位", n, openN, fmt.Sprintf("在仓 %d", openN))
	}

	// 拒因榜：eval 层 verdict≠proposed 分布+unev reason top3（evalqueue 同源），
	// 提案层 rejected 计数（窗内）。
	if rs, err := c.pool.Query(ctx, `SELECT verdict, count(*) FROM fin_opp_eval
		WHERE ts>now()-$1::interval AND verdict<>'proposed' GROUP BY 1`, iv); err == nil {
		for rs.Next() {
			var v string
			var cnt int64
			if rs.Scan(&v, &cnt) == nil {
				resp.Rejects = append(resp.Rejects, &dashboardv1.FunnelReject{
					Level: "eval", Reason: "verdict=" + v, N: cnt})
			}
		}
		rs.Close()
	}
	if rs, err := c.pool.Query(ctx, `SELECT coalesce(evidence->>'reason','<无reason>'), count(*)
		FROM fin_opp_eval WHERE verdict='unevaluable' AND ts>now()-$1::interval
		GROUP BY 1 ORDER BY 2 DESC LIMIT 3`, iv); err == nil {
		for rs.Next() {
			var r string
			var cnt int64
			if rs.Scan(&r, &cnt) == nil {
				resp.Rejects = append(resp.Rejects, &dashboardv1.FunnelReject{
					Level: "eval", Reason: r, N: cnt})
			}
		}
		rs.Close()
	}
	var prj int64
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE created_at>now()-$1::interval AND state='fin_retired'`, iv).
		Scan(&prj); err == nil && prj > 0 {
		resp.Rejects = append(resp.Rejects, &dashboardv1.FunnelReject{
			Level: "proposal", Reason: "fin_retired", N: prj})
	}

	// 哑弹榜：live_dud 探针同判据（live∧entered>7d∧零 fin_position）。
	if rs, err := c.pool.Query(ctx, `SELECT s.id, s.inst_id, s.template_ref, s.notional_usd, s.entered_at
		FROM fin_strategy s
		WHERE s.state='live' AND s.entered_at < now()-interval '7 days'
		AND NOT EXISTS (SELECT 1 FROM fin_position fp WHERE fp.strategy_ref=s.id)
		ORDER BY s.entered_at LIMIT 20`); err == nil {
		for rs.Next() {
			var id int64
			var inst, tpl string
			var ntl float64
			var entered time.Time
			if rs.Scan(&id, &inst, &tpl, &ntl, &entered) != nil {
				break
			}
			resp.Duds = append(resp.Duds, &dashboardv1.DudStrategy{
				Id: id, InstId: inst, TemplateRef: tpl, NotionalUsd: ntl,
				EnteredAt: timestamppb.New(entered.UTC())})
		}
		rs.Close()
	} else {
		log.Printf("[funnel] 哑弹榜查询败: %v", err)
	}
	return resp, nil
}

// ── 设备径 GetFaceGrid ──

// faceEval 单件面判态（healthComps 注册表同源——与 GetSystemHealth 同判据
// 双出口：SystemComponent vs FaceRow 渲染形异判据一）。
func faceEval(b compBeat, ts *time.Time, qerr error) (state int32, note string, lagS int64) {
	if qerr != nil {
		return 2, "查询败（读件缺）", 0
	}
	if ts == nil {
		if b.zeroNote != "" {
			return b.zeroState, b.zeroNote, 0
		}
		return 2, "零行——读件缺", 0
	}
	lagS = int64(time.Since(*ts).Seconds())
	switch d := time.Since(*ts); {
	case d > b.dead:
		state = 2
	case d > b.warn:
		state = 1
	}
	if state != 0 && b.fix != "" {
		note = b.fix
	}
	return state, note, lagS
}

// GetFaceGrid 设备径：组件面（healthComps 同源判态）+探针心跳（esc_scope×
// 最近升/复绿+未决）+会话面（唤起/收口尾流）+部署面（近 3 部署行）。
func (c coreService) GetFaceGrid(ctx context.Context,
	_ *dashboardv1.GetFaceGridRequest) (*dashboardv1.GetFaceGridResponse, error) {
	resp := &dashboardv1.GetFaceGridResponse{AsOf: timestamppb.Now()}
	for _, b := range healthComps() {
		row := &dashboardv1.FaceRow{Name: b.name, Fact: b.fact, Exch: b.exch, Cat: b.cat, Host: b.host}
		var ts *time.Time
		err := c.pool.QueryRow(ctx, b.query).Scan(&ts)
		row.State, row.Note, row.LagS = faceEval(b, ts, err)
		if err != nil {
			log.Printf("[facegrid] %s 查询败: %v", b.name, err)
		}
		if ts != nil {
			row.LastWrite = timestamppb.New(ts.UTC())
		}
		resp.Faces = append(resp.Faces, row)
	}

	// 探针心跳：esc_scope 聚合最近升/复绿；未决=未销升件在案（runbook①
	// 同谓词——无界窗对齐，老案未销亦显 open）。
	openScopes := map[string]bool{}
	if rs, err := c.pool.Query(ctx, `SELECT DISTINCT esc_scope FROM ops_event e
		WHERE e.event='升件' AND e.esc_scope IS NOT NULL AND e.esc_scope<>''
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND (x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')
				OR EXISTS (SELECT 1 FROM regexp_matches(x.note,'升件#(\d+)-(\d+)','g') AS m(mm)
					WHERE e.id BETWEEN mm[1]::bigint AND mm[2]::bigint)))`); err == nil {
		for rs.Next() {
			var s string
			if rs.Scan(&s) == nil {
				openScopes[s] = true
			}
		}
		rs.Close()
	}
	if rs, err := c.pool.Query(ctx, `SELECT esc_class, esc_scope,
		max(ts) FILTER (WHERE event='升件') AS last_esc,
		max(ts) FILTER (WHERE event='恢复') AS last_recover
		FROM ops_event WHERE esc_scope IS NOT NULL AND esc_scope<>''
		GROUP BY 1,2 ORDER BY last_esc DESC NULLS LAST`); err == nil {
		for rs.Next() {
			var cls, scope string
			var esc, rec *time.Time
			if rs.Scan(&cls, &scope, &esc, &rec) != nil {
				break
			}
			pb := &dashboardv1.ProbeBeat{EscClass: cls, EscScope: scope, Open: openScopes[scope]}
			if esc != nil {
				pb.LastEsc = timestamppb.New(esc.UTC())
			}
			if rec != nil {
				pb.LastRecover = timestamppb.New(rec.UTC())
			}
			resp.Probes = append(resp.Probes, pb)
		}
		rs.Close()
	} else {
		log.Printf("[facegrid] 探针心跳查询败: %v", err)
	}
	// open 先行排（未决件不被新火序挤出视面——审计 P1）。
	sort.SliceStable(resp.Probes, func(i, j int) bool {
		if resp.Probes[i].Open != resp.Probes[j].Open {
			return resp.Probes[i].Open
		}
		return false
	})

	// 会话面：唤起/收口/唤起败尾流——双臂 seat 全集（唤知机/唤知/审计机——
	// 审计 P1：审计机|唤起败 在案，双臂滤单臂漏数）。
	sf := &dashboardv1.SessionFace{}
	var lastSpawn, lastClose *time.Time
	var spawns, closes, fails int64
	if err := c.pool.QueryRow(ctx, `SELECT
		max(ts) FILTER (WHERE event='唤起'),
		max(ts) FILTER (WHERE event='会话收口'),
		count(*) FILTER (WHERE event='唤起' AND ts>now()-interval '24 hours'),
		count(*) FILTER (WHERE event='会话收口' AND ts>now()-interval '24 hours'),
		count(*) FILTER (WHERE event='唤起败' AND ts>now()-interval '24 hours')
		FROM ops_event WHERE seat IN ('唤知机','唤知','审计机')`).
		Scan(&lastSpawn, &lastClose, &spawns, &closes, &fails); err != nil {
		log.Printf("[facegrid] 会话面查询败: %v", err)
	} else {
		if lastSpawn != nil {
			sf.LastSpawn = timestamppb.New(lastSpawn.UTC())
			sf.Running = (lastClose == nil || lastSpawn.After(*lastClose)) &&
				time.Since(*lastSpawn) < 2*time.Hour
		}
		sf.Spawns_24H, sf.Closes_24H, sf.Fails_24H = spawns, closes, fails
	}
	resp.Session = sf

	// 部署面：近 3 部署行（note 形「deploy <sha> rc=<n> image=o-core:<sha>」）。
	if rs, err := c.pool.Query(ctx, `SELECT id, ts, note FROM ops_event
		WHERE seat='DEPLOY' AND event='部署' ORDER BY id DESC LIMIT 3`); err == nil {
		re := regexp.MustCompile(`deploy ([0-9a-f]+) rc=(\d+)`)
		for rs.Next() {
			var id int64
			var ts time.Time
			var note string
			if rs.Scan(&id, &ts, &note) != nil {
				break
			}
			d := &dashboardv1.DeployRow{Id: id, Ts: timestamppb.New(ts.UTC())}
			if m := re.FindStringSubmatch(note); m != nil {
				d.Sha = m[1]
				fmt.Sscanf(m[2], "%d", &d.Rc)
			}
			resp.Deploys = append(resp.Deploys, d)
		}
		rs.Close()
	} else {
		log.Printf("[facegrid] 部署面查询败: %v", err)
	}
	return resp, nil
}
