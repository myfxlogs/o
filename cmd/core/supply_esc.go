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

// stuckEscCheck 卡壳/复核面探——升件获「修|愈」类处置后同 scope 复燃=
// 状态可证伪宣称被戳穿（宣称已修→修未成立;宣称已愈→愈是假愈——
// 修没修成愈没愈成,状态不说谎）。判词质量复核的机械臂:
// 复审不审词,审「宣称的事实」——两类宣称都可被状态戳穿。
// 销类处置复燃不在此面（误发/同判词销=对判词不对状态,条件存续
// 复火属正常流转;定性销无状态宣称=本探针域外,归审计抽查面）。
// 判据:7d 内 ∃升件 e1,其后处置行载修愈类词∧同 scope 又火 e2。
// 2026-11-10 业主令:机器卡壳不自知是常事,复审要机械可证伪。
func (c coreService) stuckEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT e2.esc_scope, e1.id, e2.id FROM ops_event e1
		JOIN ops_event e2 ON e2.esc_scope=e1.esc_scope AND e2.event='升件' AND e2.id>e1.id
		WHERE e1.event='升件' AND e1.ts>now()-interval '7 days'
		AND e1.esc_scope IS NOT NULL AND e1.esc_scope<>''
		AND EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
			AND d.id>e1.id AND d.id<e2.id
			AND d.note ~ ('升件#'||e1.id||'([^0-9]|$)')
			AND d.note ~ '(修|已修|部署|落成|修复|已愈|自愈|恢复|复绿)')
		ORDER BY e2.id DESC LIMIT 10`)
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
	var parts []string
	for _, s := range stuck {
		parts = append(parts, fmt.Sprintf("%s(#%d销后→#%d又火)", s.scope, s.prev, s.refire))
	}
	c.escEdgeU(ctx, deadSince, "唤知", "stuck_esc", dead, func() string {
		return fmt.Sprintf("唤知/stuck_esc: %d 个 scope 销案后复燃（%s）——判词宣称已修|已愈而条件复发,宣称不实,疑卡壳建议值守介入",
			len(stuck), strings.Join(parts, "、"))
	})
}
