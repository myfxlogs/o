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
// 判据集（guard.postmortem.* 热启键三件，任一键缺=fail-closed 本仓跳判+
// missing_param 计数——判据不虚构与 guard 判据链同律）：
//
//	min_hold_minutes    秒级开平阈（T-0419 实证 15s 类）；
//	settled_anomaly_pct 结算幅阈 %（|settled|>notional×阈——-49% 实证类）；
//	known_triggers      触发认知集（guard 触发词谱+manual——集外=异常）。
//
// 异常件（任一）：秒级开平 / 结算幅超阈 / 触发类不在认知集 / 残袋>0
// →fin_econ_lesson kind='trade_postmortem' entity_ref=position#<id>
// pending 入裁（UNIQUE(kind,entity_ref) 幂等——一仓一案；写败计数+log
// 不阻钱径，与记忆写面纪律②同式）。量低期零烧——判在拍内，有案才耗
// 裁定席会话（verdict_tick 水印闸 pending>0 自醒）。
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
	var (
		execID, stratRef                  int64
		inst                              string
		notional, settled, bagQty, bagUSD float64
		opened, closed                    time.Time
		guardTrig                         string
	)
	if err := r.pool.QueryRow(ctx, `SELECT p.execution_id,COALESCE(p.strategy_ref,0),
		p.inst_id,p.notional_usd,p.settled_amount,
		COALESCE(p.bare_bag_qty,0),COALESCE(p.bare_bag_usd,0),
		p.opened_at,p.closed_at,COALESCE(g.trigger_kind,'')
		FROM fin_position p
		LEFT JOIN fin_guard_close g ON g.position_id=p.id AND g.state='closed'
		WHERE p.id=$1`, posID).Scan(&execID, &stratRef, &inst, &notional, &settled,
		&bagQty, &bagUSD, &opened, &closed, &guardTrig); err != nil {
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
	refID := fmt.Sprintf("position#%d", posID)
	if stratRef > 0 {
		refID = strconv.FormatInt(stratRef, 10)
	}
	if len(codes) == 0 {
		// 正常件：expected 判词落 ops_decision——记忆×结果对照案闭环
		// （WriteEsc 写败升件自报不阻钱径）。
		decision.WriteEsc(ctx, r.pool, decision.Row{
			Domain: "fin", Actor: "postmortem", RefID: refID, Action: "outcome",
			Reason: fmt.Sprintf("expected：结案判据内——trig=%s hold=%.1fmin settled=%.2f（%.1f%%）",
				trig, holdMin, settled, settledPct),
			Payload: map[string]any{
				"position_id": posID, "verdict": "expected", "trigger": trig,
				"hold_minutes": holdMin, "settled_usd": settled, "settled_pct": settledPct,
				"notional_usd": notional, "bare_bag_qty": bagQty}})
		return
	}
	// 异常件：监察案入裁定席 pending 集（UNIQUE(kind,entity_ref) 幂等一仓一案）。
	r.rej.Inc("postmortem_anomaly")
	numbers, _ := json.Marshal(map[string]any{
		"position_id": posID, "strategy_ref": stratRef, "inst_id": inst,
		"execution_id": execID, "trigger": trig, "hold_minutes": holdMin,
		"settled_usd": settled, "settled_pct": settledPct, "notional_usd": notional,
		"bare_bag_qty": bagQty, "bare_bag_usd": bagUSD,
		"anomalies": codes, "detail": strings.Join(detail, "；"),
		"close_reason": closeReason, "decision_row_found": closeReason != "",
		"summary": fmt.Sprintf("仓#%d %s 结案异常（%s）——trigger=%s hold=%.1fmin settled=%.2f(%.1f%%)",
			posID, inst, strings.Join(codes, ","), trig, holdMin, settled, settledPct)})
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_econ_lesson(kind,entity_ref,numbers)
		VALUES('trade_postmortem',$1,$2::jsonb) ON CONFLICT(kind,entity_ref) DO NOTHING`,
		fmt.Sprintf("position#%d", posID), string(numbers)); err != nil {
		r.rej.Inc("write_err")
		log.Printf("WARN [ledger] 仓 #%d 监察案落账败——结案异常失显形: %v", posID, err)
		return
	}
	log.Printf("[ledger] 仓 #%d 结案异常（%s）——监察案 trade_postmortem 入裁定席 pending", posID, strings.Join(codes, ","))
}
