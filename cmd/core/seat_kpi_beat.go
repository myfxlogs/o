// seat_kpi_beat.go —— 各岗 KPI 量表探（工单#19121——唤知钦定口径
// ops_event #19588 备忘全文转录；八段全 ops_event∪fin_opp_hypothesis
// 派生读域，零新表/零推送/零依赖）。量表拍群同挂位（metricBeats 尾加
// 一行），严格日拍=seatKpiDue（业主令#20930——指纹闸面值一抖即重发，
// 改末行 ts 距今<档即抑）。
// probe 在 o-core 内跑=全读域无 roster_rw 界；各段读败=该段标 ?
// 不整帧弃（账页面读败如实呈不虚构，同 selfStateBeat 先例）。
//
// 钦定行 schema：event='量表' seat='探测席' ticket='seat_kpi'，
// note=唤知:销中位Xh·误销Y%|施工:完工中位Zh·打回W%|审计:P1n|
// 探针:误报m%|自我:兑现k/n|研究:转化j/90d（除注明外窗=30d）。
package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// seatKpiEvery 严格日拍档期（业主令#20930——指纹含小数，面值一抖
// 就重发；改末行 ts 距今<档即抑，不看指纹）。
const seatKpiEvery = 20 * time.Hour

// seatKpiM 八段读数（nil=该段读败——note 面呈 ? 不虚构）。
type seatKpiM struct {
	escH      *float64 // ①唤知销案中位 h（钦定 SQL 照抄）
	refirePct *float64 // ②误销复燃 %
	doneH     *float64 // ③施工完工中位 h
	rejectPct *float64 // ④打回 %
	p1        *int     // ⑤审计疑点 P1
	falsePct  *float64 // ⑥探针误报 %
	selfK     *int     // ⑦自我兑现分子
	selfN     *int     // ⑦自我兑现分母
	research  *int     // ⑧研究转化 90d
	moneyK    *int     // ⑨施工钱环占比分子
	moneyN    *int     // ⑨施工钱环占比分母
}

// seatKpiDue 严格日拍闸——末行 ticket='seat_kpi' 距今 < seatKpiEvery
// 即抑（不看指纹：面值一抖不重发）；无末行=首报。读败放行——量表
// 惯例读败不虚构抑制。
func (c coreService) seatKpiDue(ctx context.Context) bool {
	var last *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT max(ts) FROM ops_event
		WHERE event='量表' AND ticket='seat_kpi'`).Scan(&last); err != nil {
		return true
	}
	return last == nil || time.Since(*last) >= seatKpiEvery
}

// seatKpiBeat 各岗 KPI 日行——严格日拍（seatKpiDue 同档期内全抑，
// 面值变动不再重发——业主令#20930 减法 C 批）。
func (c coreService) seatKpiBeat(ctx context.Context) {
	if !c.seatKpiDue(ctx) {
		return
	}
	m := c.seatKpiRead(ctx)
	fp := strings.Join(seatKpiVals(m), "|")
	note := seatKpiNote(fp, m)
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','seat_kpi',$1)`,
		note); err != nil {
		return // 量表惯例：写败本拍让位静默
	}
}

// seatKpiRead 八口径聚合读（各段独立——一段败不拖全帧）。
func (c coreService) seatKpiRead(ctx context.Context) seatKpiM {
	var m seatKpiM

	// ① 唤知销案中位——钦定 SQL 照抄（升件→首个引证处置对，
	// median(d.ts-e.ts)；e.id 前缀碰撞为钦定原样不修）。
	if err := c.pool.QueryRow(ctx, `SELECT percentile_cont(0.5) WITHIN GROUP (
		ORDER BY extract(epoch FROM (d.ts-e.ts))/3600)
		FROM ops_event e JOIN ops_event d
		  ON d.event='处置' AND d.ts>e.ts AND d.note ~ ('升件#'||e.id)
		WHERE e.event='升件' AND e.ts>now()-interval '30 days'`).
		Scan(&m.escH); err != nil {
		m.escH = nil
	}

	// ② 误销复燃率——处置行销的升件（处置动词=销）中，同 esc_scope
	// 销后 24h 内再升件占比（scope 取升件行 esc_scope 列；锚=销时
	// min(d.ts)——复燃语义=被销后同源再火）。
	var refireN, refireHit int
	if err := c.pool.QueryRow(ctx, `WITH disposed AS (
		SELECT e.id, e.esc_scope, min(d.ts) AS dts
		FROM ops_event e JOIN ops_event d
		  ON d.event='处置' AND d.ts>e.ts AND d.note ~ ('升件#'||e.id||' 销')
		WHERE e.event='升件' AND e.ts>now()-interval '30 days'
		  AND e.esc_scope IS NOT NULL AND e.esc_scope<>''
		GROUP BY e.id, e.esc_scope)
		SELECT count(*), count(*) FILTER (WHERE EXISTS (
		  SELECT 1 FROM ops_event e2 WHERE e2.event='升件'
		    AND e2.esc_scope=disposed.esc_scope
		    AND e2.ts>disposed.dts AND e2.ts<=disposed.dts+interval '24 hours'))
		FROM disposed`).Scan(&refireN, &refireHit); err == nil && refireN > 0 {
		p := float64(refireHit) / float64(refireN) * 100
		m.refirePct = &p
	}

	// ③④ 施工完工对——done=施工席处置含 完工∨回执（30d）；③配对
	// 同票最新前置施工席认领行（event='认领' 记法实测唯一可配——
	// 领单行零工单#参不入对）；④唤知处置=seat='处置席' 处置行
	// （实测唤知会话落账席——升件销判 1202/1202 全出其手）。
	var doneN, doneHit int
	if err := c.pool.QueryRow(ctx, `WITH done AS (
		SELECT substring(d.note from '工单#([0-9]+)') AS tk, d.ts
		FROM ops_event d
		WHERE d.seat='施工席' AND d.event='处置'
		  AND (d.note LIKE '%完工%' OR d.note LIKE '%回执%')
		  AND d.ts>now()-interval '30 days')
		SELECT count(*), count(*) FILTER (WHERE EXISTS (
		  SELECT 1 FROM ops_event w
		  WHERE w.event='处置' AND w.seat='处置席' AND w.ts>d.ts
		    AND w.note ~ ('工单#'||d.tk||'([^0-9]|$)')
		    AND w.note ~ '(打回|不通过|退)'))
		FROM done d`).Scan(&doneN, &doneHit); err == nil && doneN > 0 {
		p := float64(doneHit) / float64(doneN) * 100
		m.rejectPct = &p
	}
	if err := c.pool.QueryRow(ctx, `WITH done AS (
		SELECT substring(d.note from '工单#([0-9]+)') AS tk, d.ts
		FROM ops_event d
		WHERE d.seat='施工席' AND d.event='处置'
		  AND (d.note LIKE '%完工%' OR d.note LIKE '%回执%')
		  AND d.ts>now()-interval '30 days')
		SELECT percentile_cont(0.5) WITHIN GROUP (
		  ORDER BY extract(epoch FROM (d.ts-c.cts))/3600)
		FROM done d JOIN LATERAL (
		  SELECT max(c2.ts) AS cts FROM ops_event c2
		  WHERE c2.event='认领' AND c2.seat='施工席' AND c2.ts<d.ts
		    AND c2.note ~ ('工单#'||d.tk||'([^0-9]|$)')
		) c ON c.cts IS NOT NULL`).Scan(&m.doneH); err != nil {
		m.doneH = nil
	}

	// ⑤ 审计疑点 P1 数（30d）。
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='审计疑点' AND note LIKE '%P1%'
		  AND ts>now()-interval '30 days'`).Scan(&m.p1); err != nil {
		m.p1 = nil
	}

	// ⑥ 探针误报率——30d 升件中有处置引证者（分母）里，处置行含
	// 无事∨误报∨噪声者（分子）占比。
	var falseN, disposedN int
	if err := c.pool.QueryRow(ctx, `WITH esc AS (
		SELECT e.id, e.ts FROM ops_event e
		WHERE e.event='升件' AND e.ts>now()-interval '30 days')
		SELECT count(*) FILTER (WHERE EXISTS (
		    SELECT 1 FROM ops_event d WHERE d.event='处置' AND d.ts>e.ts
		      AND d.note ~ ('升件#'||e.id||'([^0-9]|$)'))),
		  count(*) FILTER (WHERE EXISTS (
		    SELECT 1 FROM ops_event d WHERE d.event='处置' AND d.ts>e.ts
		      AND d.note ~ ('升件#'||e.id||'([^0-9]|$)')
		      AND d.note ~ '(无事|误报|噪声)'))
		FROM esc e`).Scan(&disposedN, &falseN); err == nil && disposedN > 0 {
		p := float64(falseN) / float64(disposedN) * 100
		m.falsePct = &p
	}

	// ⑦ 自我兑现率——判词 note 钦定正则 工单#([0-9]+) 引用逐枚，下
	// 判词前有 认领∨处置 行引同票者占比（实测判词面现行裸 #N 记
	// 法→钦定正则 n=0 如实呈 0/0，词面不达报复审不自裁）。
	if err := c.pool.QueryRow(ctx, `WITH v AS (
		SELECT p.id, p.ts, mm[1] AS tk
		FROM ops_event p CROSS JOIN LATERAL
		  regexp_matches(p.note,'工单#([0-9]+)','g') AS m(mm)
		WHERE p.event='判词' AND p.ts>now()-interval '30 days')
		SELECT count(*), count(*) FILTER (WHERE EXISTS (
		  SELECT 1 FROM ops_event x
		  WHERE x.event IN ('认领','处置') AND x.ts<v.ts
		    AND x.note ~ ('工单#'||v.tk||'([^0-9]|$)')))
		FROM v`).Scan(&m.selfN, &m.selfK); err != nil {
		m.selfK, m.selfN = nil, nil
	}

	// ⑧ 研究转化——fin_opp_hypothesis 90d 内 proposed→adopted 推进
	// 链计数（state 闭集止于 adopted——paper/live 腿在下游
	// fin_strategy 模板键域无 hypothesis 外键，adopted=表可测推进
	// 终点，如实口径报复审）。
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_hypothesis
		WHERE state='adopted' AND ts>now()-interval '90 days'`).Scan(&m.research); err != nil {
		m.research = nil
	}

	// ⑨ 施工钱环占比（减法E批·业主令#20930「施工会话中钱环票不少于
	// 2/3」预算墙实测面）：分母=30d 施工席完工回执票（同③④选域），
	// 分子=票文命中钱环词面者。钱环词=钱径环面（策略/venue/执行/账/
	// 评估/研究/族）钦定词册——自管词（探针/工单/会话/部署/台账）
	// 不在册故零命中票=自管票如实判（词册演进=改本行一处）。
	if err := c.pool.QueryRow(ctx, `WITH done AS (
		SELECT substring(d.note from '工单#([0-9]+)') AS tk
		FROM ops_event d
		WHERE d.seat='施工席' AND d.event='处置'
		  AND (d.note LIKE '%完工%' OR d.note LIKE '%回执%')
		  AND d.ts>now()-interval '30 days')
		SELECT count(*), count(*) FILTER (WHERE e.note ~*
		  '(venue|okx|mt5|cmc|策略|评估|假想|hypo|执行|exec|平仓|开仓|腿|ledger|净值|资金|risk|演习|校准|drill|族|template|funding|研究|育种|进水口|dcd|行情|pnl|收益|position|order|leg|fill|settle|结算|利息|earn|利润|候选)')
		FROM ops_event e JOIN done ON e.id::text=done.tk WHERE e.event='工单'`).
		Scan(&m.moneyN, &m.moneyK); err != nil {
		m.moneyK, m.moneyN = nil, nil
	}
	return m
}

// seatKpiVals 八段面值渲染（fp 与 note 共用——面变=纹变=重报；
// nil 段呈 ? 不虚构，同账页面读败如实呈先例）。
func seatKpiVals(m seatKpiM) []string {
	f := func(p *float64) string {
		if p == nil {
			return "?"
		}
		return fmt.Sprintf("%.1f", *p)
	}
	i := func(p *int) string {
		if p == nil {
			return "?"
		}
		return fmt.Sprintf("%d", *p)
	}
	fulfill := "?/?"
	if m.selfK != nil && m.selfN != nil {
		fulfill = fmt.Sprintf("%d/%d", *m.selfK, *m.selfN)
	}
	money := "?/?"
	if m.moneyK != nil && m.moneyN != nil {
		money = fmt.Sprintf("%d/%d", *m.moneyK, *m.moneyN)
	}
	return []string{f(m.escH), f(m.refirePct), f(m.doneH), f(m.rejectPct),
		i(m.p1), f(m.falsePct), fulfill, i(m.research), money}
}

// seatKpiNote 量表文+指纹（纯函数供 shape 测——钦定六段词形）。
func seatKpiNote(fp string, m seatKpiM) string {
	v := seatKpiVals(m)
	return fmt.Sprintf("v=%s 唤知:销中位%sh·误销%s%%|施工:完工中位%sh·打回%s%%|审计:P1%s|探针:误报%s%%|自我:兑现%s|研究:转化%s/90d|施工钱环:%s",
		fp, v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8])
}
