// supply_esc.go —— 机会供给断供+教训矿积压面探针（escLoop 同拍载）。
package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// oppSurfaceStallEscCheck/liveDudEscCheck 已随减法 B 批平移入 ops_state 视件
// （fin_0202——opp_eval_stall/live_dud 判据零漂,「不升件不唤醒」钦定）。

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
