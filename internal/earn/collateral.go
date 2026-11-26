// collateral.go —— margin 抵押保底（O-41364 鸡生蛋死锁根修）：
// venue 借币额度唯有抵押物在场才可证（max-loan=抵押物比例制实测——
// adjEq≈$0.003 时全币额度尘级），sweep 把 trading 可用扫光→额度≈0→
// eval 恒拒 borrow_quota_insufficient→提案永不 armed→无意向登记→
// sweep 继续扫——死锁环每环实证在案。
//
// 修法=「常驻抵押金豁免+泊资径」复合：
//
//	① sweepIdle 判定面——margin.exec_enabled=true ∧ trading 可用<reserve
//	  →意向豁免⑤让位（让位原因 collateral_reserve）；reserve 超部照扫
//	  不误（sweepable=idle−reserve 截量——禁无限豁免，intent.go 同段）；
//	② 泊资径——reserve 不足时 RedeemToTrading 复用件把 Earn→trading
//	  摆回短额（sweep 反向=redeem 既有臂，不另造划转件）；冷却闸防
//	  连败/读面滞后锤 venue。
//
// 反向律：exec_enabled=false→reserve 失效不豁免（休眠不留钱——回滚径
// =闸本身）；reserve 键缺=fail-closed 豁免（宁停泊不误扫，同
// intentExempt 缺省律）。
package earn

import (
	"context"
	"log"
	"time"
)

const (
	// keyMarginExec margin 执行径总闸（fin_0264 种键——false=接线待命态，
	// reserve 判据整体失效）。
	keyMarginExec = "margin.exec_enabled"
	// keyCollateralReserve 常驻抵押金保底 USD（fin_0269 种键缺省 500——
	// probe$50 级量测足矣；额度随抵押物线性，实测值低估满额能力——
	// spec 注释钉）。
	keyCollateralReserve = "margin.collateral_reserve_usd"
	// reserveTopupCooldown 泊资重试冷却——Earn 余额不足/venue 读面滞后
	// 时防连拍锤 API（15min=redeemInFlightWin 同窗——在途赎回窗内不重复
	// 发起，簿记 decision 行亦罩判据①）。
	reserveTopupCooldown = 15 * time.Minute
)

// reserveState 抵押保底三态读（热键——拍距内翻转即生效）：
//
//	floor>0=应保底额；hold=true=本拍交易户段禁动（exec 开∧reserve 键缺
//	=fail-closed 宁停泊不误扫）；exec 关/reserve≤0=无保底旧行为。
func (w *Worker) reserveState() (floor float64, hold bool) {
	on, err := w.cfg.GetBool(keyMarginExec)
	if err != nil || !on {
		return 0, false // exec 关=reserve 失效不豁免（回滚径=闸本身）
	}
	r, err := w.cfg.GetFloat(keyCollateralReserve)
	if err != nil {
		return 0, true // 键缺=fail-closed 豁免（同 intentExempt 缺省律）
	}
	if r <= 0 {
		return 0, false
	}
	return r, false
}

// collateralShortfall 泊资判定——trading 可用（w.idle=USDT availBal
// ≈USD 口径）低于保底的短额。源缺席/读败=不动作（判定面不归本件——
// sweepIdle 判据⑤另有读败 fail-closed 径，本件只服务泊资不猜）。
func (w *Worker) collateralShortfall(ctx context.Context) (float64, bool) {
	floor, hold := w.reserveState()
	if hold || floor <= 0 || w.idle == nil {
		return 0, false
	}
	avail, err := w.idle(ctx)
	if err != nil {
		return 0, false
	}
	if short := floor - avail; short > 0 {
		return short, true
	}
	return 0, false
}

// collateralReserveTopup 泊资径（scan 拍 sweepIdle 前置——钱先到判据
// 才有据）：短额>0∧过冷却窗→RedeemToTrading 摆回（earn_redeem decision
// 行落账后判据①在途窗同罩——三判据互补不叠加误伤）。败=计数+log
// 下冷却窗重试（不连拍锤 venue——Earn 余额不足是诚实败如实记）。
func (w *Worker) collateralReserveTopup(ctx context.Context) {
	short, ok := w.collateralShortfall(ctx)
	if !ok {
		return
	}
	if time.Since(w.lastReserveTopup) < reserveTopupCooldown {
		w.rej.Inc("reserve_topup_cooldown")
		return
	}
	w.lastReserveTopup = time.Now()
	if err := w.RedeemToTrading(ctx, short); err != nil {
		w.rej.Inc("reserve_topup_failed")
		log.Printf("[earn] 抵押保底泊资败 shortfall=%.2f（%s 后重试）: %v",
			short, reserveTopupCooldown, err)
		return
	}
	log.Printf("[earn] 抵押保底泊资 %.2f USDT Earn→trading（floor 补足）", short)
}
