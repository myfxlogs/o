// postmortem.go —— 结案判定机（T-0420 平仓监察件——每平必审补读侧）：
// settle 同槽机械件——settled_* 落位即结案判（「结案」=结算拍内判定，拍外
// 零烧；正常件 ops_decision outcome 行案闭环，异常件监察案入裁定席 pending
// 集——业主判词「平仓了没人管/亏了如何处理没人管」病灶对治：ops_decision
// 已记开平理由，读理由复盘的本件补上）。
//
// 输入真源四件（票面钦定）：
//
//	触发类  fin_guard_close.state='closed' 行 trigger_kind（guard 平）→
//	        ops_decision 本仓 close 行 payload->>'trigger'（异写径留痕）
//	        →'manual'（两源皆缺=手平）；
//	持仓时长 opened_at→closed_at；
//	settled 幅 settled_amount/notional_usd（本拍刚落位的实现额——
//	        Σcashflow+bare_bag 挂账口径）；
//	理由行  ops_decision 本仓最近 close 行 reason（监察案载荷——裁席可读
//	        「为什么平」；行缺=decision_row_found=false 载案不虚构）。
//
// 判据集（guard.postmortem.* 热启键，前三件任一键缺=fail-closed 本仓跳判+
// missing_param 计数——判据不虚构与 guard 判据链同律；第四件缺只降新增
// 复发支不跳全链，GW-001 段D1）：
//
//	min_hold_minutes    秒级开平阈（T-0419 实证 15s 类）；
//	settled_anomaly_pct 结算幅阈 %（|settled|>notional×阈——-49% 实证类）；
//	known_triggers      触发认知集（guard 触发词谱+manual——集外=异常）；
//	fast_close_repeat_n fast_close 复发闸 N（7d 窗同 strategy_ref∨inst_id
//	                    含本仓共 ≥N 次∨settled<0 才入诊断闸——单次秒平
//	                    不烧诊断额度）。
//
// 异常件（任一）：秒级开平 / 结算幅超阈 / 触发类不在认知集 / 残袋>0
// →fin_econ_lesson kind='trade_postmortem' entity_ref=position#<id>
// pending 入裁（UNIQUE(kind,entity_ref) 幂等——一仓一案；写败计数+log
// 不阻钱径，与记忆写面纪律②同式）。量低期零烧——判在拍内，有案才耗
// 裁定席会话（verdict_tick 水印闸 pending>0 自醒）。
//
// 判词回写（T-0428）：fin_position.trigger_verdict 拍内同槽落——正常件
// 'confirmed'（复核确认原判）、异常件=机械修正码集（fast_close/settled_over/
// trigger_unknown/bare_bag）；trigger_kind 原貌不动=审计链，裁定席语义修正
// （unit_mismatch_hallucination 类）可后覆写。
package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"o/internal/platform/decision"
)

// postmortem 单仓结案判（settle 内 per-row 调用——settled_amount 已落位同拍）。
// 非阻塞：读败/写败计数留痕，不回滚结算事实。
func (r *Runner) postmortem(ctx context.Context, posID int64) {
	minHold, err := r.cfg.GetFloat("guard.postmortem.min_hold_minutes")
	if err != nil {
		r.rej.Inc("missing_param")
		log.Printf("WARN [ledger] 仓 #%d guard.postmortem.min_hold_minutes 读败——结案判据链跳过: %v", posID, err)
		return
	}
	pct, err := r.cfg.GetFloat("guard.postmortem.settled_anomaly_pct")
	if err != nil {
		r.rej.Inc("missing_param")
		log.Printf("WARN [ledger] 仓 #%d guard.postmortem.settled_anomaly_pct 读败——结案判据链跳过: %v", posID, err)
		return
	}
	known := r.cfg.GetStringList("guard.postmortem.known_triggers")
	if len(known) == 0 {
		r.rej.Inc("missing_param")
		log.Printf("WARN [ledger] 仓 #%d guard.postmortem.known_triggers 缺——结案判据链跳过", posID)
		return
	}
	// GW-001 段D1 fast_close 复发闸 N 键（热启）。缺键/读败=复发支不激活
	// （单次秒平维持不入闸现制）——不 fail-closed 整链：判据三键缺才跳
	// 全链，本键只盖新增支，WARN 留痕待 DM 补键。
	repeatN := int64(0)
	if v, err := r.cfg.GetInt("guard.postmortem.fast_close_repeat_n"); err != nil {
		log.Printf("WARN [ledger] 仓 #%d guard.postmortem.fast_close_repeat_n 读败——fast_close 复发支不激活（单次秒平不入诊断额度）: %v", posID, err)
	} else {
		repeatN = v
	}
	var (
		execID, stratRef                  int64
		inst                              string
		notional, settled, bagQty, bagUSD float64
		opened, closed                    time.Time
		guardTrig                         string
		guardAttempts                     int
		guardLastErr                      string
	)
	if err := r.pool.QueryRow(ctx, `SELECT p.execution_id,COALESCE(p.strategy_ref,0),
		p.inst_id,p.notional_usd,p.settled_amount,
		COALESCE(p.bare_bag_qty,0),COALESCE(p.bare_bag_usd,0),
		p.opened_at,p.closed_at,COALESCE(g.trigger_kind,''),COALESCE(g.attempts,0),
		COALESCE(g.last_error,'')
		FROM fin_position p
		LEFT JOIN fin_guard_close g ON g.position_id=p.id AND g.state='closed'
		WHERE p.id=$1`, posID).Scan(&execID, &stratRef, &inst, &notional, &settled,
		&bagQty, &bagUSD, &opened, &closed, &guardTrig, &guardAttempts, &guardLastErr); err != nil {
		r.rej.Inc("write_err")
		log.Printf("WARN [ledger] 仓 #%d 结案事实读败——本仓跳判: %v", posID, err)
		return
	}
	// ops_decision 本仓最近 close 理由行（payload 携 position_id 机读键——
	// ref_id=strategy 维不可当仓键）。
	var closeReason, closeTrig string
	if err := r.pool.QueryRow(ctx, `SELECT reason,COALESCE(payload->>'trigger','')
		FROM ops_decision WHERE domain='fin' AND action='close'
			AND payload->>'position_id'=$1
		ORDER BY id DESC LIMIT 1`, strconv.FormatInt(posID, 10)).
		Scan(&closeReason, &closeTrig); err != nil {
		closeReason, closeTrig = "", "" // 行缺/读败同径——手平无异写径留痕
	}
	trig := guardTrig
	if trig == "" {
		trig = closeTrig
	}
	if trig == "" {
		trig = "manual" // 两源皆缺=手平（票面触发类谱含手平）
	}
	holdMin := closed.Sub(opened).Minutes()
	settledPct := 0.0
	if notional > 0 {
		settledPct = settled / notional * 100
	}
	var codes, detail []string
	if holdMin < minHold {
		codes = append(codes, "fast_close")
		detail = append(detail, fmt.Sprintf("持仓 %.1fmin<阈 %.0fmin", holdMin, minHold))
	}
	if math.Abs(settledPct) > pct {
		codes = append(codes, "settled_over")
		detail = append(detail,
			fmt.Sprintf("|settled|=%.1f%%>阈 %.0f%%（%.2f/名义 %.2f）", math.Abs(settledPct), pct, settled, notional))
	}
	knownSet := map[string]bool{}
	for _, k := range known {
		knownSet[k] = true
	}
	if !knownSet[trig] {
		codes = append(codes, "trigger_unknown")
		detail = append(detail, fmt.Sprintf("触发类 %q 不在认知集", trig))
	}
	if bagQty > 0 {
		codes = append(codes, "bare_bag")
		detail = append(detail, fmt.Sprintf("残袋 %.8f≈$%.2f", bagQty, bagUSD))
	}
	// GW-001 段D1 fast_close 复发实断：仅 fast_close 单码案需要查询
	// （他码在案=三源已入闸，复发与否不改变判）；7d 窗同 strategy_ref∨
	// inst_id 前序 fast_close 计数，含本仓共 ≥N 次=复发。计数读败=复发
	// 不可判按不复发（宁少烧诊断额度，与判据读败同向）。
	priorFC := -1
	if len(codes) == 1 && codes[0] == "fast_close" && repeatN > 0 {
		if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_position
			WHERE id<>$1 AND trigger_verdict LIKE '%fast_close%'
			  AND closed_at>=now()-interval '7 days'
			  AND (($2::bigint>0 AND strategy_ref=$2) OR inst_id=$3)`,
			posID, stratRef, inst).Scan(&priorFC); err != nil {
			priorFC = -1
			log.Printf("WARN [ledger] 仓 #%d fast_close 复发计数读败——按单次案处置: %v", posID, err)
		}
	}
	refID := fmt.Sprintf("position#%d", posID)
	if stratRef > 0 {
		refID = strconv.FormatInt(stratRef, 10)
	}
	facts := diagCaseFacts{PosID: posID, ExecID: execID, InstID: inst,
		Anomalies: codes, BareBagUSD: bagUSD, BareBagQty: bagQty,
		Attempts: guardAttempts, Trigger: trig, SettledPct: settledPct,
		SettledUSD:      settled,
		FastCloseRepeat: repeatN > 0 && priorFC >= 0 && int64(priorFC+1) >= repeatN}
	// T-0464 盈亏归因三分——机判序 code→ops→market→unattributed（判词面
	// fin_position.pnl_attribution 与 trigger_verdict 怎么坏轴并存两轴；
	// 人裁可改判，机判原值+锚命中留 outcome payload/案卷审计链）。
	attrF := r.attrAnchors(ctx, posID, execID, opened, closed, settled, trig,
		diagnosisWorthy(facts), strings.HasPrefix(guardLastErr, "venue:"))
	attr := attributePnl(attrF)
	// T-0428 判词回写（trigger_verdict——trigger_kind 原貌不动为审计链）：
	// 正常件=复核确认原判；异常件=机械修正码集（裁定席语义修正可后覆写）。
	// 记忆写不阻钱径：写败计数留痕不回滚结算事实（与监察案写同律）。
	verdict := strings.Join(codes, ",")
	if len(codes) == 0 {
		verdict = "confirmed"
	}
	if _, err := r.pool.Exec(ctx,
		`UPDATE fin_position SET trigger_verdict=$2, pnl_attribution=$3 WHERE id=$1`,
		posID, verdict, attr); err != nil {
		r.rej.Inc("write_err")
		log.Printf("WARN [ledger] 仓 #%d 判词回写败（trigger_verdict=%s pnl_attribution=%s）: %v",
			posID, verdict, attr, err)
	}
	if len(codes) == 0 {
		// 正常件：expected 判词落 ops_decision——记忆×结果对照案闭环
		// （WriteEsc 写败升件自报不阻钱径）。
		decision.WriteEsc(ctx, r.pool, decision.Row{
			Domain: "fin", Actor: "postmortem", RefID: refID, Action: "outcome",
			Reason: fmt.Sprintf("expected：结案判据内——trig=%s hold=%.1fmin settled=%.2f（%.1f%%） 归因=%s",
				trig, holdMin, settled, settledPct, attr),
			Payload: map[string]any{
				"position_id": posID, "verdict": "expected", "trigger": trig,
				"hold_minutes": holdMin, "settled_usd": settled, "settled_pct": settledPct,
				"notional_usd": notional, "bare_bag_qty": bagQty,
				"pnl_attribution": attr, "attr_hits": attrF.hits()}})
		return
	}
	// 异常件：监察案入裁定席 pending 集（UNIQUE(kind,entity_ref) 幂等一仓一案）。
	r.rej.Inc("postmortem_anomaly")
	numMap := map[string]any{
		"position_id": posID, "strategy_ref": stratRef, "inst_id": inst,
		"execution_id": execID, "trigger": trig, "hold_minutes": holdMin,
		"settled_usd": settled, "settled_pct": settledPct, "notional_usd": notional,
		"bare_bag_qty": bagQty, "bare_bag_usd": bagUSD,
		"anomalies": codes, "detail": strings.Join(detail, "；"),
		"close_reason": closeReason, "decision_row_found": closeReason != "",
		"pnl_attribution": attr, "attr_hits": attrF.hits(),
		"summary": fmt.Sprintf("仓#%d %s 结案异常（%s）——trigger=%s hold=%.1fmin settled=%.2f(%.1f%%) 归因=%s",
			posID, inst, strings.Join(codes, ","), trig, holdMin, settled, settledPct, attr)}
	// GW-001 段D1 半成品窗去向写明：fast_close 单码案未入诊断闸=单次秒平
	// （复发键未设∨复发未达且盈利）——案不丢：监察案本行照落，裁定席
	// 平仓监察案分支照消化；diag_gate 字段机读留证去向与未入闸因。
	if len(codes) == 1 && codes[0] == "fast_close" && !diagnosisWorthy(facts) {
		if repeatN <= 0 {
			numMap["diag_gate"] = "fast_close 单次未入诊断闸——复发键 guard.postmortem.fast_close_repeat_n 未设（复发支不激活）；" +
				"去向=监察案 trade_postmortem（本行）裁定席平仓监察案分支消化"
		} else {
			numMap["diag_gate"] = fmt.Sprintf("fast_close 单次未入诊断闸——复发未达（N=%d，7d 窗同 strategy_ref/inst_id 前序 %d 次+本仓共 %d<N）且 settled=%.2f≥0；"+
				"去向=监察案 trade_postmortem（本行）裁定席平仓监察案分支消化", repeatN, priorFC, priorFC+1, settled)
		}
	}
	numbers, _ := json.Marshal(numMap)
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_econ_lesson(kind,entity_ref,numbers)
		VALUES('trade_postmortem',$1,$2::jsonb) ON CONFLICT(kind,entity_ref) DO NOTHING`,
		fmt.Sprintf("position#%d", posID), string(numbers)); err != nil {
		r.rej.Inc("write_err")
		log.Printf("WARN [ledger] 仓 #%d 监察案落账败——结案异常失显形: %v", posID, err)
		return
	}
	log.Printf("[ledger] 仓 #%d 结案异常（%s）归因=%s——监察案 trade_postmortem 入裁定席 pending",
		posID, strings.Join(codes, ","), attr)
	// T-0431 根因诊断案同拍生成：bug 形异常（账实差未定性/认知集外触发/
	// 守卫环——三案卷源任一）自动生成诊断任务（根因案型）——纯市况案不生成
	// （裁定席既有三分支消化，诊断额度不烧市况案）。
	if diagnosisWorthy(facts) {
		emitDiagTask(ctx, r, facts, numbers)
	}
}

// attrAnchors 归因锚取证（T-0464——机判序输入面；读败锚=零值回落
// unattributed 不虚构归因——判词宁缺毋假与判据链同律）。
func (r *Runner) attrAnchors(ctx context.Context, posID, execID int64,
	opened, closed time.Time, settled float64, trig string,
	bugShaped, guardVenueEr bool) attrFacts {
	f := attrFacts{BugShaped: bugShaped, GuardVenueEr: guardVenueEr,
		MktTrigger: marketTriggers[trig], Settled: settled}
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_order_leg
		WHERE execution_id=$1 AND (state IN ('failed','ambiguous')
			OR venue_err_code IS NOT NULL OR venue_err_msg IS NOT NULL)`,
		execID).Scan(&f.LegErrs); err != nil {
		log.Printf("WARN [ledger] 仓 #%d 归因锚腿错因读败（按零锚）: %v", posID, err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE ts>=$1 AND ts<=$2 AND (note LIKE '%断链%' OR note LIKE '%断供%'
			OR note LIKE '%凭据%' OR note LIKE '%部署停滞%' OR note LIKE '%ENOSPC%'
			OR note LIKE '%网络%')`, opened, closed).Scan(&f.OpsDisrupt); err != nil {
		log.Printf("WARN [ledger] 仓 #%d 归因锚 ops 断供读败（按零锚）: %v", posID, err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE position_ref=$1 AND kind='funding_settle'`, posID).Scan(&f.FundingNet); err != nil {
		log.Printf("WARN [ledger] 仓 #%d 归因锚 funding 净额读败（按零锚）: %v", posID, err)
	}
	return f
}
