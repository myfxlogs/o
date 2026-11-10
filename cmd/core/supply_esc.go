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
