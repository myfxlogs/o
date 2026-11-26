// intent.go —— 停泊环意向豁免（O-40829 P0 钱径正确性缺陷）：
// 「闲钱去 Earn」被实现成「非白名单内钱的都闲」——赎回款/负债担保存钱/
// 在途单预备金与天然闲钱同待遇被扫走（实证：Earn 赎 1000→1s 被再申购；
// margin 借币仓抵押物被扫→adjEq 崩→venue 强平，margin-short 活不过几分钟）。
//
// 判据源（复合，单判据函数一源——funding/trading 双段同罩）：
//
//	① 在途赎回款——fin_risk_decision op='earn_redeem' <15min（venue 域限定）：
//	  RedeemToTrading 摆渡中/开仓赎回窗，钱已出 Earn 未成仓=有用途（覆盖簿记未落行的窗口，
//	  margin 开仓 redeem→book 序间隙同罩——spec「能读已有数据别建第二套状态」）；
//	② 本 venue 在持仓——fin_position 非 closed 且执行腿 venue=本户（MT5/DCD
//	  仓抵押物不在 okx 户不连坐；无腿行不计=待腿在途由①③罩）；
//	③ 本 venue margin 借币负债在册——fin_margin_loan open/closing（簿记直证，
//	  venue 读面滞后窗的互补）；
//	④ venue 负债真值——/account/balance details liab 系>0∨cashBal<0（spec 禁线
//	  钦定源：venue 有负债账上没有=担保钱，手动借/他径借全覆盖——权威判据）；
//	⑤ fin_execution state='open' <1h 在途单——sweepIdle 既有 T-0462 闸同段
//	  前置，本判据不重复计（让位原因归并 sweep_skip_inflight 旧门）；
//	⑥ margin 抵押保底（O-41364）——margin.exec_enabled=true ∧ trading 可用
//	  <reserve：venue 借币额度唯抵押物在场可证，扫光=eval 恒拒额度不足
//	  死锁环；reserve 键缺=fail-closed 豁免（同缺省律），exec 关=失效。
//
// 白名单闸（spec 候选 c）不入：sweepIdle 只动计价币 USDT 本位——USDT 恒在
// FundCcy 三域在册，查了=永豁免=停泊环死；币级白名单归 sweepResidual 既有径。
//
// fail-closed 律：任一判据读败=按「有用途」豁免（不虚构没有用途——
// 宁留闲钱不误动担保钱）；venue 读件未接线（nil）=该源缺席让位降级，
// 产线 main.go 必挂（读败在 fn 内上抛由本面转豁免）。
package earn

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

const (
	// keyIntentExempt 意向豁免总闸（缺省 true——缺键按启用安全向；
	// 置 false=回滚旧行为全复，钱径机制零改）。
	keyIntentExempt = "earn.sweep_intent_exempt"
	// keyLiabCheck venue 负债查径开关（缺省 true；false=降级跳④源——
	// 余判据照走，供 venue 读面故障期人工降级）。
	keyLiabCheck = "earn.sweep_liab_check"
	// redeemInFlightWin 赎回在途窗——earn_redeem 留痕后钱未成仓的合理上限
	// （开仓全链秒级；15min 宽限覆盖借币/成交回报滞后，超时=钱真闲照扫）。
	redeemInFlightWin = "15 minutes"
)

// AttachLiabCheck 挂 venue 负债现读件（产线=okxexec.Client.AccountHasLiability；
// fn 返 (有负债, err)——err=读败由本面转 fail-closed 豁免；nil=源缺席降级）。
func (w *Worker) AttachLiabCheck(fn func(ctx context.Context) (bool, error)) *Worker {
	w.liab = fn
	return w
}

// liabCheckOn 负债查径开关热读（缺键/读败=缺省 true——安全向同上）。
func (w *Worker) liabCheckOn() bool {
	on, err := w.cfg.GetBool(keyLiabCheck)
	return err != nil || on
}

// sweepIntentExempt 意向豁免判（sweepIdle 入口单源）。返让位原因+是否豁免；
// 原因串落 log 让位显形（让位是判词不是静默吞——ops 读日志可追）。
func (w *Worker) sweepIntentExempt(ctx context.Context) (string, bool) {
	on, err := w.cfg.GetBool(keyIntentExempt)
	if err != nil {
		on = true // 缺键=缺省 true（fail-closed 安全向——宁停泊不误扫）
	}
	if !on {
		return "", false // 总闸关=旧行为全复（回滚径）
	}
	// ① 在途赎回款（簿记未落行窗的唯一信号——decision 行先于仓位行落账）
	var n int
	if err := w.pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_risk_decision
		 WHERE intent->>'kind'='transfer' AND intent->>'op'='earn_redeem'
		   AND intent->>'venue'=$1
		   AND ts>now()-interval '`+redeemInFlightWin+`'`, w.venue).Scan(&n); err != nil {
		return "intent_read_fail", true
	}
	if n > 0 {
		return fmt.Sprintf("redeem_inflight:%d", n), true
	}
	// ② 本 venue 在持仓（腿 venue 域限定——MT5/DCD 仓不连坐 okx 扫面）
	var posOpen bool
	if err := w.pool.QueryRow(ctx,
		`SELECT EXISTS(
		   SELECT 1 FROM fin_position p JOIN fin_execution e ON e.id=p.execution_id
		   WHERE p.state<>'closed' AND EXISTS(
		     SELECT 1 FROM fin_order_leg l WHERE l.execution_id=e.id AND l.venue=$1))`,
		w.venue).Scan(&posOpen); err != nil {
		return "intent_read_fail", true
	}
	if posOpen {
		return "position_open", true
	}
	// ③ 本 venue margin 借币负债在册（簿记直证——venue 滞后窗互补）
	var loanOpen bool
	if err := w.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM fin_margin_loan
		 WHERE venue=$1 AND status IN ('open','closing'))`, w.venue).Scan(&loanOpen); err != nil {
		return "intent_read_fail", true
	}
	if loanOpen {
		return "margin_loan_open", true
	}
	// ④ venue 负债真值（权威源——手动借/他径借账上无行全覆盖；读败=fail-closed）
	//    源未接线=缺席降级跳过本判据（产线必挂——测试面零网可缺；O-41364
	//    起本判据非末位——早退会让后续判据整段死码，故降级=跳过非返回）。
	if w.liabCheckOn() && w.liab != nil {
		has, err := w.liab(ctx)
		if err != nil {
			return "liab_read_fail", true
		}
		if has {
			return "venue_liability", true
		}
	}
	// ⑥ margin 抵押保底（O-41364——借币额度唯抵押物在场可证：trading 可用
	//   <reserve 时扫走=eval 恒拒额度不足死锁环；泊资径见 collateral.go）。
	//   源缺席（w.idle nil）=降级跳过；读败=fail-closed 豁免。
	if floor, hold := w.reserveState(); hold {
		return "collateral_reserve_key_missing", true
	} else if floor > 0 && w.idle != nil {
		avail, err := w.idle(ctx)
		if err != nil {
			return "collateral_read_fail", true
		}
		if avail < floor {
			return fmt.Sprintf("collateral_reserve:avail=%.2f<floor=%.2f", avail, floor), true
		}
	}
	return "", false
}

// sweepIdle 闲余>阈值（热启）→回存停泊。判据源：unified 快照行 avail（WS 事件源，
// 鲜<5min fail-closed 不动钱）；执行在途（state=open ∧ <1h）让位防饿腿——
// 超龄 open=执行行漏关账烂非在途（在途单秒级态迁），放行+探针升件（工单#25693：
// #1117/#1134 挂 14 天把闸堵死实证——账烂堵钱比饿腿更伤）。
func (w *Worker) sweepIdle(ctx context.Context) error {
	if !w.autoPurchaseOn() {
		return nil // T-0342：自动申购停用中——本拍全段让位（钱留原账户）
	}
	thr, err := w.getF("earn.sweep_min_usd")
	if err != nil {
		w.rej.Inc("missing_param")
		return fmt.Errorf("earn.sweep_min_usd: %w", err)
	}
	// O-40829 意向豁免（入口单源——funding/trading 双段同罩）：有用途的钱
	// （赎回在途/本 venue 在持仓/margin 借币负债/venue 负债真值）本拍全段
	// 让位；读败=fail-closed 豁免。让位原因落 log 显形非静默吞。
	if reason, exempt := w.sweepIntentExempt(ctx); exempt {
		w.rej.Inc("sweep_intent_exempt")
		log.Printf("[earn] sweep 让位（意向豁免）: %s", reason)
		return nil
	}
	// 在途让位先判（T-0462——dcd 摆渡款落 funding 即属在途单预备金；
	// 先扫 funding 段会把渡款抢去 Earn=ping-pong 病灶。原序=funding 段
	// 裸奔无此闸——交易户段同款闸上移罩全域）。
	var inFlight, stale int
	if err := w.pool.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE created_at>now()-interval '1 hour'),
			count(*) FILTER (WHERE created_at<=now()-interval '1 hour')
		 FROM fin_execution WHERE state='open'`).Scan(&inFlight, &stale); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	if stale > 0 {
		w.staleExecEsc(ctx)
	}
	if inFlight > 0 {
		w.rej.Inc("sweep_skip_inflight")
		return nil
	}
	// T-0149 ②funding 户闲资先扫（入金落 funding 天然闲——直购 Earn 免
	// 摆渡单跳；读件缺=让位不误报非阻断径）。
	if w.idleFund != nil {
		if favail, err := w.idleFund(ctx); err == nil && favail > thr {
			if w.depositIdleFunding(ctx, favail) {
				return nil // funding 段已动——下一拍再扫交易户段
			}
		} else if err != nil {
			w.rej.Inc("idle_fund_missing")
		}
	}
	idle, ok, err := w.tradingIdleUSDT(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return nil // 计数门已挂（snapshot_stale）——陈快照 fail-closed 不动钱
	}
	// O-41364 抵押保底截量——exec_enabled∧reserve 在案时只扫超部
	// （sweepable=idle−floor：保底内=抵押物不扫；禁无限豁免——avail≥floor
	// 时判据⑥放行到本段，超部照扫不误钉死在此）。
	floor, hold := w.reserveState()
	if hold {
		w.rej.Inc("sweep_collateral_reserve")
		log.Printf("[earn] sweep 让位（抵押保底键缺 fail-closed）")
		return nil
	}
	sweepable := idle - floor
	if sweepable <= thr {
		if floor > 0 && idle > thr {
			w.rej.Inc("sweep_collateral_reserve")
			log.Printf("[earn] sweep 让位（抵押保底 collateral_reserve）: idle=%.2f floor=%.2f 超部≤阈", idle, floor)
		}
		return nil
	}
	if time.Since(w.lastSweepFailAt) < 5*time.Minute && math.Abs(sweepable-w.lastSweepFailAmt) < 0.01 {
		w.rej.Inc("sweep_backoff") // 同因退避中——禁同参重发（死锁环根修件）
		return nil
	}
	if !w.mu.TryLock() {
		w.rej.Inc("sweep_busy")
		return nil
	}
	defer w.mu.Unlock()
	if moved := w.depositIdle(ctx, sweepable); !moved {
		w.lastSweepFailAmt, w.lastSweepFailAt = sweepable, time.Now()
	} else {
		w.lastSweepFailAmt, w.lastSweepFailAt = 0, time.Time{}
	}
	return nil
}
