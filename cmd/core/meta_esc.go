// meta_esc.go —— 元认知面探针（escLoop 同拍载,探件本件）：
// 断流/资金闸/对账/唤知/数据缺/能力债/预期产出等类（孤儿账/商业产出
// 已随减法B批平移入 ops_state 视件+状况桥——fin_0201,判据零漂）
// （escLoop 探针类谱——断流/资金闸载 feed_esc.go,预期产出通用形载
// expect_esc.go,余载本件）——「系统看不见自己的问题」元盲区补位。
// 探针只述实数/事实,判词归唤知会话 runbook 对应档；
// dedup 全走「未销才抑」（销案≠事实消失,销案后新事证重升不误吞）。
package main

import (
	"context"
	"fmt"
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

// orphanProbes 孤儿账注册表已随减法 B 批平移入 ops_state 视件
// （fin_0201——「写了没人读」积压册判据零漂,scope=orphan_<name> 同款；
// 元认知补盲来历=fin_econ_lesson 诊断案积压产线无人裁实证,升件#9063）。

// cognitionEscCheck 唤知面死看门狗——未决升件在案而处置行 24h 缺席=
// 认知环停摆（钱径照转无人判案）。ops_event 升件落账+Feishu 人端告警
// 双发——脑死时 ops_event 只是日志,人是最后兜底面（人端推只在死沿一拍发,
// 持续死不刷屏）。
func (c coreService) cognitionEscCheck(ctx context.Context, deadSince map[string]time.Time, al *notify.Alerter) {
	var open int
	// 引用端物化（SG CPU 烧实证同型改——逐升件相关 regex 全表扫→
	// 引用集单程抽出反连,毫秒级）。
	if err := c.pool.QueryRow(ctx, `WITH cited AS MATERIALIZED (
		SELECT x.id xid, (m.mm)[1]::bigint lo,
			COALESCE((m.mm)[2]::bigint,(m.mm)[1]::bigint) hi
		FROM ops_event x, LATERAL regexp_matches(x.note,$1,'g') m(mm)
		WHERE x.note LIKE '%升件#%')
		SELECT count(*) FROM ops_event e
		WHERE e.event='升件' AND e.ts>now()-interval '72 hours'
		AND NOT EXISTS (SELECT 1 FROM cited c WHERE c.xid>e.id
			AND e.id BETWEEN c.lo AND c.hi)`, `升件#([0-9]+)(?:-([0-9]+))?`).Scan(&open); err != nil {
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

// commercialEscCheck 商业面同拍副探容器——原 commercial_stall 升件
// （30d净≤0∧live/probe在册——「机器在转但转不出钱」元盲区补位，
// 损益 kind 集 canon 见 econ-earn-interest-as-strategy-profit）已随
// 减法 B 批平移入 ops_state 视件+状况桥（fin_0201，判据零漂）；
// 本件只余钱面同域副探（feed_esc 棘轮满线不入册唤点故搭拍）。
func (c coreService) commercialEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	c.hungerGapEscCheck(ctx, deadSince) // 动机探同拍（钱面同域;feed_esc 棘轮满线不入册唤点）
	c.paperDryEscCheck(ctx, deadSince)  // 练习场旱探同拍（动机面第二信号——零候选即旱）
	c.goalIdleEscCheck(ctx, deadSince)  // 目标票怠探同拍（想要空转——OPEN票零推进即喊）
	c.breedDryEscCheck(ctx, deadSince)  // 育种旱探同拍（进化环停滞——有种不繁/无种可繁）
	c.deathGradCheck(ctx, deadSince)    // 死刑梯度同拍（饿的牙——年化→能力因子→敞口阈）
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
// 态不探；凭据补上=况行自消（能力债清偿=事实消失非销案）。
// 减法B批（fin_0202）：升件写改 ops_state_row 行写（「不升件不唤醒,
// 变了才更新一行」）——excell.Enumerate 占用判定不可 SQL 化故走表不走视。
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
	want := map[string]string{}
	for _, t := range tmpls {
		var drillN, evalN int
		// 工单#17938 启用窗起算：新启用族(<7d)演习债宽限——凭据窗起点=enabled_at（采纳日∨参数调整启行最新ts），无据=-infinity回落旧7d；评估债臂不修（零评估=真债不误伤）。
		var young bool
		if err := c.pool.QueryRow(ctx, `SELECT GREATEST(
			COALESCE((SELECT max(h.ts) FROM fin_opp_hypothesis h
				WHERE h.state='adopted' AND EXISTS (
					SELECT 1 FROM fin_opp_template tt
					WHERE tt.template=$1 AND tt.venue=h.venue AND tt.surface=h.surface)),
			 '-infinity'::timestamptz),
			COALESCE((SELECT max(e.ts) FROM ops_event e
				WHERE e.event='参数调整'
				  AND strpos(e.note,'target=fin_opp_template key='||$1||'@')=1
				  AND e.note LIKE '%新=enabled=true%'),
			 '-infinity'::timestamptz)) > now()-interval '7 days'`, t).Scan(&young); err != nil {
			continue
		}
		needDrill := occupied[t] && !young
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
		if gap := !reg || (needDrill && drillN == 0) || evalN == 0; gap {
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
			want["capability_"+t] = fmt.Sprintf(
				"启用模板 %s 缺项=[%s]——在册即须被行使,不行使即信号（处置径=补凭据/补评估/回禁用裁,定性归唤知会话能力债档）",
				t, strings.Join(lacks, "+"))
		}
	}
	c.syncStateRows(ctx, "能力债", "capability_", want, 50)
}

// syncStateRows 状况行同步——ops_state_row 写面（fin_0202,减法B批
// 「不升件不唤醒,变了才更新一行,自我席每拍读」钦定）：want=本拍真况集
// （scope→detail）,差出删/同值留（detail 未变 updated_at 不动=变了才更）/
// 新现插。写败=本拍让位。
func (c coreService) syncStateRows(ctx context.Context, kind, prefix string, want map[string]string, valence int) {
	scopes := make([]string, 0, len(want))
	for scope, detail := range want {
		scopes = append(scopes, scope)
		if _, err := c.pool.Exec(ctx, `INSERT INTO ops_state_row(kind,scope,valence,detail)
			VALUES($1,$2,$3,$4)
			ON CONFLICT(kind,scope) DO UPDATE SET detail=EXCLUDED.detail,
				updated_at=CASE WHEN ops_state_row.detail IS DISTINCT FROM EXCLUDED.detail
					THEN now() ELSE ops_state_row.updated_at END`,
			kind, scope, valence, detail); err != nil {
			return // 写件败本拍让位
		}
	}
	if _, err := c.pool.Exec(ctx, `DELETE FROM ops_state_row
		WHERE kind=$1 AND scope LIKE $2 AND NOT (scope=ANY($3))`,
		kind, prefix+"%", scopes); err != nil {
		_ = err // 差出删败=陈行多留一拍,下拍补
	}
}

// dataGapEscCheck 数据缺面探——fin_opp_eval verdict='unevaluable' 同真因3d窗≥3行=结构缺（非瞬态）；真因消失=恢复自闭（销案不抑死态——事实仍死只清未决面）。
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
	for k, since := range deadSince { // 真因消失=缺口愈合——对挂死键补恢复（沿对称:死键在而实测非死）
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
