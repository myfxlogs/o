// meta_esc.go —— 元认知面探针（escLoop 同拍载,探件本件）：
// 断流/资金闸/对账/孤儿账/唤知/数据缺/商业产出/能力债八类
// （escLoop 探针类谱——断流/资金闸载 feed_esc.go,余载本件）——
// 「系统看不见自己的问题」元盲区补位。
// 探针只述实数/事实,判词归唤知会话 runbook 对应档；
// dedup 全走「未销才抑」（销案≠事实消失,销案后新事证重升不误吞）。
package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"o/internal/platform/excell"
	"o/internal/platform/execfam"
	"o/internal/platform/notify"
)

// reconEscCheck 对账面探（歧态腿机读进水口）——fin_order_leg state='ambiguous'
// 在案=钱径事实（venue 回报歧态,禁补单,原径仅 Feishu 人推）。探针只述
// 腿数+ids,仲裁判词归唤知会话（runbook 对账档:tools/legarb 取证→三态裁）。
func (c coreService) reconEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var n int
	var ids []int64
	if err := c.pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM fin_order_leg WHERE state='ambiguous'),
		(SELECT COALESCE(array_agg(id),'{}') FROM
			(SELECT id FROM fin_order_leg WHERE state='ambiguous'
				ORDER BY id LIMIT 10) q)`).Scan(&n, &ids); err != nil {
		return // 读件败本拍让位
	}
	c.escEdgeUV(ctx, deadSince, "对账", "ambiguous_legs",
		fmt.Sprintf("%d:%v", n, ids), n > 0, func() string {
			return fmt.Sprintf(
				"对账/ambiguous_legs: 歧态腿 %d 条在案（ids %v）——venue 回报歧态势,禁补单待仲裁（runbook 对账档:tools/legarb 取证→三态裁）", n, ids)
		})
}

// orphanProbe 「写了没人读」积压册——（名,积压计数询）一件一册,TTL 在询内钦定。
type orphanProbe struct{ name, query string }

// orphanProbes 孤儿账注册表——消费面死/表退役/真积压三因同形=积压超 TTL。
// TTL=各域消费拍宽限（教训矿 24/48h、假想卡/条件缺口 7d——再低=正常拍误报）。
var orphanProbes = []orphanProbe{
	{"eng_lesson", `SELECT count(*) FROM fin_eng_lesson
		WHERE status='pending' AND ts<now()-interval '24 hours'`},
	{"econ_lesson", `SELECT count(*) FROM fin_econ_lesson
		WHERE status='pending' AND ts<now()-interval '48 hours'`},
	{"hypothesis_proposed", `SELECT count(*) FROM fin_opp_hypothesis
		WHERE state='proposed' AND ts<now()-interval '7 days'`},
	// hypothesis_parked——parked 非终态是挂起：超 30d 未复核=需求黑洞
	// （结构性盲区实证 2026-11-10：#3/#5/#6/#7 骨架齐而卡沉 parked 无人问）。
	// last_reviewed=hypo decide 复核戳（re-park=再议落戳续挂 30d；
	// 无戳卡回退建卡 ts——存量 parked 卡首拍全浮=如实「从未复核」）。
	{"hypothesis_parked", `SELECT count(*) FROM fin_opp_hypothesis
		WHERE state='parked' AND COALESCE(
			CASE WHEN evidence->>'last_reviewed' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}'
				THEN (evidence->>'last_reviewed')::timestamptz END, ts)
			< now()-interval '30 days'`},
	{"condition_gap", `SELECT count(*) FROM fin_condition_gap
		WHERE state='pending' AND requested_at<now()-interval '7 days'`},
}

// orphanEscCheck 孤儿账面探——注册表册逐件判,积压超 TTL=升件 orphan_*。
// 元认知补盲：系统此前看不见「写了没人读」的表（fin_econ_lesson 诊断案
// 曾积压产线无人裁,升件#9063 实证）。
func (c coreService) orphanEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	for _, p := range orphanProbes {
		var n int
		if err := c.pool.QueryRow(ctx, p.query).Scan(&n); err != nil {
			continue // 读件败本册让位
		}
		p := p
		c.escEdgeUV(ctx, deadSince, "孤儿账", "orphan_"+p.name, strconv.Itoa(n), n > 0, func() string {
			return fmt.Sprintf(
				"孤儿账/%s: 未消费积压 %d 行超 TTL——表在写而消费面死/退役/积压（定性归唤知会话孤儿账档）", p.name, n)
		})
	}
}

// cognitionEscCheck 唤知面死看门狗——未决升件在案而处置行 24h 缺席=
// 认知环停摆（钱径照转无人判案）。ops_event 升件落账+Feishu 人端告警
// 双发——脑死时 ops_event 只是日志,人是最后兜底面（人端推只在死沿一拍发,
// 持续死不刷屏）。
func (c coreService) cognitionEscCheck(ctx context.Context, deadSince map[string]time.Time, al *notify.Alerter) {
	var open int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event e
		WHERE e.event='升件' AND e.ts>now()-interval '72 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND (x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')
				OR EXISTS (SELECT 1 FROM regexp_matches(x.note,'升件#(\d+)-(\d+)','g') AS m(mm)
					WHERE e.id BETWEEN mm[1]::bigint AND mm[2]::bigint)))`).Scan(&open); err != nil {
		return
	}
	var last *time.Time
	// 活性锚=认知环自身产出（处置席处置行）——他席处置证的是他环活非
	// 本环活，混入即「探针看产出不看自主产出」遮蔽死（升件#9573）。
	_ = c.pool.QueryRow(ctx, `SELECT max(ts) FROM ops_event
		WHERE event='处置' AND seat='处置席'`).Scan(&last)
	dead := open > 0 && (last == nil || time.Since(*last) > 24*time.Hour)
	const key = "唤知/cognition_dead"
	_, was := deadSince[key]
	c.escEdgeUV(ctx, deadSince, "唤知", "cognition_dead", "@", dead, func() string {
		lastTxt := "无处置行在案"
		if last != nil {
			lastTxt = fmt.Sprintf("末处置 %s（陈 %s）",
				last.UTC().Format("01-02 15:04Z"), time.Since(*last).Round(time.Hour))
		}
		return fmt.Sprintf("唤知面死: 未决升件 %d 件在案,%s——认知环停摆,积压无人判（人端兜底告警已同发）", open, lastTxt)
	})
	if _, now := deadSince[key]; dead && !was && now && al != nil {
		lastTxt := "无处置行在案"
		if last != nil {
			lastTxt = time.Since(*last).Round(time.Hour).String()
		}
		al.Alert(ctx, "cognition_dead",
			fmt.Sprintf("唤知面死——未决升件 %d 件,末处置陈 %s（认知环停摆,积压无人判）", open, lastTxt))
	}
}

// commercialEscCheck 商业产出面探——「机器在转但转不出钱」元盲区补位：
// 策略面 30d 净产出≤0 且系统名义在赚（live/probe 在册>0）=升件
// 商业产出/commercial_stall。口径=周净利 canon 剔 Earn 基线
// （econ-earn-interest-as-strategy-profit 钦定：interest 非策略利润，
// 损益 kind 集={fee,funding_settle,trade_pnl,rollback_cost,spot_buy,spot_sell}；
// deposit/withdraw/earn_*/recon_adjust 非损益禁入）。
// 探针只述净产出+在册面，「为何空转」定性归唤知会话；净产出回正=恢复自闭。
func (c coreService) commercialEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var net float64
	var liveN, openPos int
	if err := c.pool.QueryRow(ctx, `SELECT
		(SELECT COALESCE(sum(amount),0) FROM fin_cashflow
			WHERE venue<>'sim' AND occurred_at>now()-interval '30 days'
			AND kind IN ('fee','funding_settle','trade_pnl','rollback_cost','spot_buy','spot_sell')),
		(SELECT count(*) FROM fin_strategy WHERE state IN ('live','probe')),
		(SELECT count(*) FROM fin_position WHERE closed_at IS NULL)`).
		Scan(&net, &liveN, &openPos); err != nil {
		return // 读件败本拍让位
	}
	stall := liveN > 0 && net <= 0
	c.escEdgeUV(ctx, deadSince, "商业产出", "commercial_stall",
		fmt.Sprintf("%.4f", net), stall, func() string {
			return fmt.Sprintf(
				"商业产出/commercial_stall: 策略面 30d 净产出 %.4f≤0（live+probe=%d 在仓=%d——机器名义在赚而实付净产出非正；Earn 息不入账已剔。空转定性归唤知会话）",
				net, liveN, openPos)
		})
}

// capabilityEscCheck 能力债面探（T-0523 感知升维——「在册即须被行使,
// 不行使即信号」）：启用模板三不变量逐件核——
// ①execfam 执行径在册 ②占用格模板 7d 内有绿演习凭据（fin_exec_drill
// cell pass）③14d 内有评估产出（fin_opp_eval——evaluator 在枚举该族）
// 任一缺=升件 能力债/capability_<template>。结构性盲区实证 2026-11-10：
// quarter_basis_long 矩阵格 11-08 fail 多日无人问+vol_sell 族错映沉睡+
// #3/#5/#6/#7 骨架齐而卡沉——症状探针只看得见「坏」，看不见「缺」。
// 占用感知（2026-11-10 自修假阳）：excell⑤「词表零占免演」钦定——零占用
// 模板无格可钻，结构性产不出凭据；对其火演习债=永火假阳。故②只对
// 有占用格（excell.Enumerate 非 Exempt 格）的模板成立；零占 enabled 族
// 健康度由③单独判（enabled 而永不被枚举=真债，枚举着没候选=市场平窗
// 非债——eval 拒行也计行）。占用判定唯走 excell.Enumerate 真源（禁自查
// fin_strategy 复述——枚举径演进探针不漂）。
// 边界：enabled=false 模板/registered-only 执行族（无模板行）=钦定挂起
// 态不探；凭据补上=恢复自闭（能力债清偿=事实消失非销案）。
func (c coreService) capabilityEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT template FROM fin_opp_template
		WHERE enabled AND state='enabled' GROUP BY template`)
	if err != nil {
		return // 读件败本拍让位
	}
	var tmpls []string
	for rows.Next() {
		var t string
		if rows.Scan(&t) == nil {
			tmpls = append(tmpls, t)
		}
	}
	rows.Close()
	cells, err := excell.Enumerate(ctx, c.pool)
	if err != nil {
		return // 枚举读件败本拍让位（同径下次再核）
	}
	occupied := map[string]bool{}
	for i := range cells {
		if !cells[i].Exempt {
			occupied[cells[i].Template] = true
		}
	}
	for _, t := range tmpls {
		var drillN, evalN int
		needDrill := occupied[t]
		if needDrill {
			if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM fin_exec_drill
				WHERE cell_key LIKE $1||'@%' AND verdict='pass'
				AND ts>now()-interval '7 days'`, t).Scan(&drillN); err != nil {
				continue
			}
		}
		if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
			WHERE template=$1 AND ts>now()-interval '14 days'`, t).Scan(&evalN); err != nil {
			continue
		}
		reg := execfam.Has(t)
		gap := !reg || (needDrill && drillN == 0) || evalN == 0
		t := t
		c.escEdgeUV(ctx, deadSince, "能力债", "capability_"+t, "@", gap, func() string {
			var lacks []string
			if !reg {
				lacks = append(lacks, "执行族未注册")
			}
			if needDrill && drillN == 0 {
				lacks = append(lacks, "7d 无绿演习凭据")
			}
			if evalN == 0 {
				lacks = append(lacks, "14d 零评估产出")
			}
			return fmt.Sprintf(
				"能力债/capability_%s: 启用模板 %s 缺项=[%s]——在册即须被行使,不行使即信号（处置径=补凭据/补评估/回禁用裁,定性归唤知会话能力债档）",
				t, t, strings.Join(lacks, "+"))
		})
	}
	// 已不启用模板的能力债补恢复（对称沿——禁用后键死而实测非债）。
	seen := map[string]bool{}
	for _, t := range tmpls {
		seen["能力债/capability_"+t] = true
	}
	for k, since := range deadSince {
		if strings.HasPrefix(k, "能力债/capability_") && !seen[k] {
			delete(deadSince, k)
			c.recoverNote(ctx, "能力债", strings.TrimPrefix(k, "能力债/"), since)
		}
	}
}

// dataGapEscCheck 数据缺面探——fin_opp_eval verdict='unevaluable' 同真因
// 3d 窗 ≥3 行=结构缺（评估器重评估仍缺=非瞬态）。逐真因一 scope,
// 数据到位=真因行消失→恢复自闭（销案不抑死态——事实仍死只清未决面）。
func (c coreService) dataGapEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT evidence->>'reason' r, count(*) FROM fin_opp_eval
		WHERE verdict='unevaluable' AND ts>now()-interval '3 days'
		GROUP BY 1 HAVING count(*)>=3`)
	if err != nil {
		return
	}
	type gap struct {
		reason string
		n      int
	}
	var gaps []gap
	for rows.Next() {
		var g gap
		if rows.Scan(&g.reason, &g.n) == nil && g.reason != "" {
			gaps = append(gaps, g)
		}
	}
	rows.Close()
	seen := map[string]bool{}
	for _, g := range gaps {
		seen[g.reason] = true
		g := g
		c.escEdgeUV(ctx, deadSince, "数据缺", "data_gap_"+g.reason, "@", true, func() string {
			return fmt.Sprintf(
				"数据缺/%s: 评估 unevaluable 3d 窗 %d 行同真因——数据面缺腿是结构缺非瞬态（数据/凭据/启用 定性归唤知会话）", g.reason, g.n)
		})
	}
	// 真因消失=缺口愈合——对挂死键补恢复（沿对称:死键在而实测非死）。
	for k, since := range deadSince {
		if !strings.HasPrefix(k, "数据缺/data_gap_") {
			continue
		}
		reason := strings.TrimPrefix(k, "数据缺/data_gap_")
		if !seen[reason] {
			delete(deadSince, k)
			c.recoverNote(ctx, "数据缺", "data_gap_"+reason, since)
		}
	}
}
