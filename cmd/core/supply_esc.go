// supply_esc.go —— 机会供给断供+教训矿积压面探针（escLoop 同拍载）。
package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// oppSurfaceStallEscCheck 机会面断供探——曾活评估的模板族 48h 零新评估行
// =评估供给侧断流（excell 枚举停/venue 面断/族被摘三因其一,定性归唤知）。
// 判据：30d 窗 ≥20 评估行（曾是活族,非天生冷族）∧ 末拍 >48h。
// 2026-11-10 实案: dcd_call/vol_sell_* 族 11 天零评估无声——断供无信号
// 即探针盲（「安静等供给」与「供给死了」同形，须分开）。
// 2026-11-10 返修：末评真值改读 md_eval_pulse.last_eval_ts——fin_opp_eval
// SG 镜像是 serve 谓词投影（仅提案引用行过桥，全拒族恒零行=结构性假阳；
// projector 10-30 切 serve 当日 C 向三族「断供」即此形）。活性真值=正本
// 聚合脉冲面（T-0485 钦定消费域），曾活判据仍用 fin_opp_eval 30d 计数
// （镜像行随提案链陈化出窗=族转冷出探域，可接受边界）。
func (c coreService) oppSurfaceStallEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT e.template, p.last_eval_ts FROM fin_opp_eval e
		JOIN md_eval_pulse p ON p.template=e.template
		GROUP BY e.template, p.last_eval_ts
		HAVING count(*) FILTER (WHERE e.ts>now()-interval '30 days') >= 20
		AND p.last_eval_ts < now()-interval '48 hours' ORDER BY 2`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	var stalled, tpls []string
	for rows.Next() {
		var tpl string
		var last time.Time
		if err := rows.Scan(&tpl, &last); err != nil {
			return
		}
		stalled = append(stalled, fmt.Sprintf("%s(末评 %s)", tpl, last.UTC().Format("01-02 15:04Z")))
		tpls = append(tpls, tpl)
	}
	sort.Strings(tpls)
	dead := len(stalled) > 0
	c.escEdgeUV(ctx, deadSince, "数据缺", "opp_eval_stall", strings.Join(tpls, ","), dead, func() string {
		return fmt.Sprintf("数据缺/opp_eval_stall: 曾活模板族 %d 支 48h 零评估（%s）——excell 枚举停/venue 面断/族被摘三因其一,定性归唤知会话",
			len(stalled), strings.Join(stalled, "、"))
	})
}

// liveDudEscCheck live 哑弹探——state='live' ≥7d 且零 fin_position=
// 策略在册但从不开仓（执行触发段断/条件恒不满足/腿缺失三因其一,
// 定性归唤知）。2026-11-10 实案：funding-hedge 31 live 中 29 哑,
// 全谱 863 策略 90d 仅 2 结算仓——闸拒之外的更深断点：过闸≠开仓。
// 判据：live ∧ entered_at>7d ∧ NOT EXISTS fin_position(strategy_ref)。
// 曾开过仓的活不在本探针面（至少接线被验过）——哑=从未响过。
func (c coreService) liveDudEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT s.id, s.inst_id, s.entered_at FROM fin_strategy s
		WHERE s.state='live' AND s.entered_at < now()-interval '7 days'
		AND NOT EXISTS (SELECT 1 FROM fin_position fp WHERE fp.strategy_ref=s.id)
		ORDER BY s.entered_at LIMIT 8`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	var duds []string
	n := 0
	for rows.Next() {
		var id int64
		var inst string
		var entered time.Time
		if err := rows.Scan(&id, &inst, &entered); err != nil {
			return
		}
		n++
		if len(duds) < 5 {
			duds = append(duds, fmt.Sprintf("#%d %s(live %s)", id, inst, entered.UTC().Format("01-02")))
		}
	}
	dead := n > 0
	c.escEdgeUV(ctx, deadSince, "商业产出", "live_dud", strconv.Itoa(n), dead, func() string {
		return fmt.Sprintf("商业产出/live_dud: live≥7d 零仓位哑弹策略 %d 条（%s）——执行触发断/条件恒不满足/腿缺失三因其一,定性归唤知会话",
			n, strings.Join(duds, "、"))
	})
}

// lessonBacklogEscCheck 教训矿积压探——fin_eng_lesson pending ≥5 且最陈
// >24h = 矿在产无人清（会话③段漏跑/矿写径坏三因其一,定性归唤知）。
// 判词档=runbook③段裁定流（可归纳→canon 条目,判弃→verdicted）。
func (c coreService) lessonBacklogEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var n int
	var oldest *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT count(*), min(ts) FROM fin_eng_lesson
		WHERE status='pending'`).Scan(&n, &oldest); err != nil {
		return // 读件败本拍让位
	}
	dead := n >= 5 && oldest != nil && time.Since(*oldest) > 24*time.Hour
	c.escEdgeUV(ctx, deadSince, "唤知", "lesson_backlog", strconv.Itoa(n), dead, func() string {
		return fmt.Sprintf("唤知/lesson_backlog: 教训矿 pending=%d（最陈 %s）——③段漏跑/矿写径坏,候选积压无人裁定,定性归唤知会话",
			n, oldest.UTC().Format("01-02 15:04Z"))
	})
}

// pairZeroEscCheck 配对恒零探——venue接入草案 beat 同 scope 连续零命中=
// 结构性死局（恒值信号非事件不入会话队——盲区实证 2026-11-11：mt5:3001157
// 57 连零 28h 无会话看见,业主点破方知）。判据：36h 窗内零命中草案 beat
// ≥10 ∧ 首零 >12h 前（持续非偶发）∧ 末零 <2h 前（仍活）∧ 末零后无非零
// 草案 beat（未自复——复绿再死=新回合照火）。
func (c coreService) pairZeroEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `WITH z AS (
		SELECT esc_scope,
			count(*) FILTER (WHERE note LIKE '%零命中%') AS zeros,
			min(ts) FILTER (WHERE note LIKE '%零命中%') AS first_zero,
			max(ts) FILTER (WHERE note LIKE '%零命中%') AS last_zero,
			max(ts) FILTER (WHERE note NOT LIKE '%零命中%') AS last_nonzero
		FROM ops_event
		WHERE event='venue接入' AND note LIKE 'beat=草案 %'
		AND ts>now()-interval '36 hours' GROUP BY esc_scope)
		SELECT esc_scope, zeros, first_zero FROM z
		WHERE zeros>=10 AND first_zero<now()-interval '12 hours'
		AND last_zero>now()-interval '2 hours'
		AND COALESCE(last_nonzero,'-infinity'::timestamptz)<last_zero`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	var stalled []string
	for rows.Next() {
		var sc string
		var zeros int
		var first time.Time
		if err := rows.Scan(&sc, &zeros, &first); err != nil {
			return
		}
		stalled = append(stalled, fmt.Sprintf("%s(连零%d 自%s)", sc, zeros, first.UTC().Format("01-02 15:04Z")))
	}
	dead := len(stalled) > 0
	c.escEdgeUV(ctx, deadSince, "数据缺", "pair_zero", "@", dead, func() string {
		return fmt.Sprintf("数据缺/pair_zero: venue 配对草案恒零 scope %d 支（%s）——符号宇宙方言不互译/宇宙字面消费/探测符号表残缺三因其一,定性归唤知会话",
			len(stalled), strings.Join(stalled, "、"))
	})
}

// stuckEscCheck 卡壳/复核面探——升件获「修|愈」类处置后同 scope 复燃=
// 状态可证伪宣称被戳穿（宣称已修→修未成立;宣称已愈→愈是假愈——
// 修没修成愈没愈成,状态不说谎）。判词质量复核的机械臂:
// 复审不审词,审「宣称的事实」——两类宣称都可被状态戳穿。
// 销类处置复燃不在此面（误发/同判词销=对判词不对状态,条件存续
// 复火属正常流转;定性销无状态宣称=本探针域外,归审计抽查面）。
// 判据:7d 内 ∃升件 e1,其后处置行载完成态宣称词∧同 scope 又火 e2。
// 词表钉完成态(已修/已部署/上岸…非裸「修」——「要码修/待修」
// 是待办非宣称,裸词误中曾滚出 13 代自指链 2026-11-10 夜)。
// 2026-11-10 业主令:机器卡壳不自知是常事,复审要机械可证伪。
// 收窄计谱(升件#9739 判词钉,15+代自指链实证):①除自 scope——
// 自 scope 销案判词与复燃同写一探针域,自指链无认知产出;
// ②宣称词负前文——宣称词被「宣称/说/谓/引号」裹述时不算宣称
// (#9743 案:处置文内引「宣称已修实未愈」被计成已修宣称)。
func (c coreService) stuckEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	// 处置端起步（2026-11-10 SG CPU 烧实证改写）：旧形 e1×e2 自连+
	// 逐对 EXISTS regex 扫——1154 升件×8k 行 pkey 范围扫，单后端 100%
	// 单核烧 2min+/拍。新形=宣称处置行先物化抽 升件#id（处置面行少，
	// regex 单程），pkey 回 e1，scope 索引扫 e2——毫秒级。
	rows, err := c.pool.Query(ctx, `WITH claims AS MATERIALIZED (
		SELECT d.id did, (m.m)[1]::bigint e1id
		FROM ops_event d, LATERAL regexp_matches(d.note,$1,'g') m(m)
		WHERE d.event='处置' AND d.ts>now()-interval '7 days'
		AND d.note ~ '(^|[^称说谓「『])(已修|已部署|已落成|已修复|已愈|自愈|已恢复|复绿|上岸)')
		SELECT e2.esc_scope, e1.id, e2.id FROM claims c
		JOIN ops_event e1 ON e1.id=c.e1id AND e1.id<c.did
		JOIN ops_event e2 ON e2.esc_scope=e1.esc_scope AND e2.event='升件' AND e2.id>c.did
		WHERE e1.event='升件' AND e1.ts>now()-interval '7 days'
		AND e1.esc_scope IS NOT NULL AND e1.esc_scope<>'' AND e1.esc_scope<>'stuck_esc'
		ORDER BY e2.id DESC LIMIT 10`, `升件#([0-9]+)`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	type pair struct {
		scope        string
		prev, refire int64
	}
	seen := map[string]bool{}
	var stuck []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.scope, &p.prev, &p.refire); err != nil {
			return
		}
		if !seen[p.scope] {
			seen[p.scope] = true
			stuck = append(stuck, p)
		}
	}
	dead := len(stuck) > 0
	var parts, fps []string
	for _, s := range stuck {
		parts = append(parts, fmt.Sprintf("%s(#%d销后→#%d又火)", s.scope, s.prev, s.refire))
		fps = append(fps, fmt.Sprintf("%s#%d", s.scope, s.refire))
	}
	sort.Strings(fps)
	c.escEdgeUV(ctx, deadSince, "唤知", "stuck_esc", strings.Join(fps, ","), dead, func() string {
		return fmt.Sprintf("唤知/stuck_esc: %d 个 scope 销案后复燃（%s）——判词宣称已修|已愈而条件复发,宣称不实,疑卡壳建议值守介入",
			len(stuck), strings.Join(parts, "、"))
	})
}

// codeworkStallEscCheck 工单滞留探——双臂:
//
//	①一等票型: event='工单' 真行,自载期限(note 期限=Nh,默认24h),
//	  销=处置行「工单#id」——过期未销=码修待启而无人盯;
//	②旧形臂: 非探测席立的升件(会话验码/复证/挂起件)陈>24h未销,
//	  向后兼容一等票型前的工单形状。
//
// 「说要修没开修」面,与 stuck_esc 孪生(一个盯说修了没修住,
// 一个盯说要修没开修)。判据皆为状态非散文——票行是真行。
// 2026-11-11 业主令:转票无滞留哨=条件请求同族缺陷(径在无人盯走);
// 同日升级:直接上一等票型,不养散文判定。
func (c coreService) codeworkStallEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT e.id, e.seat, left(e.note,40) FROM ops_event e
		WHERE e.event='工单' AND e.ts>now()-interval '30 days'
		AND e.ts + COALESCE((substring(e.note from '期限=([0-9]+)h'))::int,24) * interval '1 hour' < now()
		AND NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
			AND d.id>e.id AND d.note ~ ('工单#'||e.id||'([^0-9]|$)'))
		UNION ALL
		SELECT e.id, e.seat, left(e.note,40) FROM ops_event e
		WHERE e.event='升件' AND e.seat<>'探测席'
		AND e.ts>now()-interval '14 days' AND e.ts<now()-interval '24 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
			AND d.id>e.id AND d.note ~ ('升件#'||e.id||'([^0-9]|$)'))
		ORDER BY 1 DESC LIMIT 20`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	var parts []string
	n := 0
	for rows.Next() {
		var id int64
		var seat, note string
		if rows.Scan(&id, &seat, &note) != nil {
			return
		}
		n++
		if len(parts) < 6 {
			parts = append(parts, fmt.Sprintf("#%d(%s:%s)", id, seat, note))
		}
	}
	dead := n > 0
	c.escEdgeUV(ctx, deadSince, "唤知", "codework_stall", strconv.Itoa(n), dead, func() string {
		return fmt.Sprintf("唤知/codework_stall: %d 件工单过期未销（%s）——转要码修件滞留,定性归唤知会话分诊(在修/挂起/死单/真滞留)",
			n, strings.Join(parts, "、"))
	})
}
