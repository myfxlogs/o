// supply_esc.go —— 机会供给断供+教训矿积压面探针（escLoop 同拍载）。
package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// oppSurfaceStallEscCheck 机会面断供探——曾活评估的模板族 48h 零新评估行
// =评估供给侧断流（excell 枚举停/venue 面断/族被摘三因其一,定性归唤知）。
// 判据：30d 窗 ≥20 评估行（曾是活族,非天生冷族）∧ 末拍 >48h。
// 2026-11-10 实案: dcd_call/vol_sell_* 族 11 天零评估无声——断供无信号
// 即探针盲（「安静等供给」与「供给死了」同形，须分开）。
func (c coreService) oppSurfaceStallEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT template, max(ts) FROM fin_opp_eval
		GROUP BY template
		HAVING count(*) FILTER (WHERE ts>now()-interval '30 days') >= 20
		AND max(ts) < now()-interval '48 hours' ORDER BY 2`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	var stalled []string
	for rows.Next() {
		var tpl string
		var last time.Time
		if err := rows.Scan(&tpl, &last); err != nil {
			return
		}
		stalled = append(stalled, fmt.Sprintf("%s(末评 %s)", tpl, last.UTC().Format("01-02 15:04Z")))
	}
	dead := len(stalled) > 0
	c.escEdgeU(ctx, deadSince, "数据缺", "opp_eval_stall", dead, func() string {
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
	c.escEdgeU(ctx, deadSince, "商业产出", "live_dud", dead, func() string {
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
	c.escEdgeU(ctx, deadSince, "唤知", "lesson_backlog", dead, func() string {
		return fmt.Sprintf("唤知/lesson_backlog: 教训矿 pending=%d（最陈 %s）——③段漏跑/矿写径坏,候选积压无人裁定,定性归唤知会话",
			n, oldest.UTC().Format("01-02 15:04Z"))
	})
}
