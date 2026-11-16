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

// intakeDryEscCheck 进水口干涸探——研究进水口自主发现循环（docs/drafts/
// research-intake-spec.md 钦定健康面）:「源普查」登记行 7d 零新增=源发现
// 臂断供（研究臂会话未被触发/规格未成 runbook 触发档/分诊闸过严三因其一,
// 定性归唤知——研究域产物归研究臂,探针只报干涸不代采）。
func (c coreService) intakeDryEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var n int
	var last *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT count(*), max(ts) FROM ops_event
		WHERE seat='研究席' AND note LIKE '源普查%' AND ts>now()-interval '7 days'`).
		Scan(&n, &last); err != nil {
		return // 读件败本拍让位
	}
	dead := n == 0
	c.escEdgeUV(ctx, deadSince, "商业产出", "intake_dry", "@", dead, func() string {
		lst := "从未有登记"
		if last != nil {
			lst = "末登记 " + last.UTC().Format("01-02 15:04Z")
		}
		return fmt.Sprintf("商业产出/intake_dry: 源普查登记 7d 零新增（%s）——研究进水口自主发现循环断供:触发档未立/研究臂未被唤/分诊过严三因其一,定性归唤知会话", lst)
	})
}
