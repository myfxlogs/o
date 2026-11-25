package main

// margin_open.go —— O-40707 margin-short 开仓编排（spec②执行径）：
// margin.exec_enabled 总闸（缺省 false fail-closed——前置三件业主面实证
// 全通后仍待命，开闸=裁定/业主面动作）→名义帽→证据三闸复核（评估侧已
// 判，执行时点重验——率/额度可变）→JIT 赎回（Earn→trading 摆渡——
// 停泊环打架钉：adjEq≈0 时 maxLoan≈0，必须先赎再借）→借
// （spot-manual-borrow-repay=spec 钦定端点——borrow-repay=VIP 通道/
// quick-margin=404/isolated=54117 三条死径已实证排除）→卖现货→
// fin_execution+fin_position(delta_neutral=false)+fin_margin_loan+
// spot_sell 现金流簿记。
// 歧态势律：借贷 venue 无 clOrdId——借前查 open loan+borrow 历史对账，
// 借后必经历史行确认才卖（确认不了=abort 不卖——借了不卖=负债裸持）。

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/okxexec"
)

// marginBaseCcy funding instId→基币（SOL-USDT-SWAP→SOL；现货对=ccy-USDT）。
func marginBaseCcy(instID string) string {
	return strings.TrimSuffix(strings.TrimSuffix(instID, "-SWAP"), "-USDT")
}

// marginShortOpen direction=long_perp_short_spot&&venue=okx 专用径——
// 借币卖出现货做空（保证金借币模式，非双腿原子——执行/簿记自成序）。
func (h hedgeOpener) marginShortOpen(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	on, err := h.cfg.GetBool("margin.exec_enabled")
	if err != nil || !on {
		return 0, fmt.Errorf("margin_exec_disabled（总闸缺省 false——前置三件实证后待命态）: %v", err)
	}
	maxUSD, err := h.cfg.GetFloat("margin.max_open_usd")
	if err != nil || maxUSD <= 0 {
		return 0, fmt.Errorf("margin.max_open_usd 判据键缺 fail-closed: %v", err)
	}
	cap1, e1 := h.cfg.GetFloat("exec.probe_notional_usd")
	cap2, e2 := h.cfg.GetFloat("risk.max_notional_usd")
	if e1 != nil || e2 != nil {
		return 0, fmt.Errorf("cap 键缺 fail-closed: %v/%v", e1, e2)
	}
	notional := min(spec.NotionalUSD, maxUSD, cap1, cap2)
	if notional <= 0 {
		return 0, fmt.Errorf("notional≤0 拒（spec=%.2f margin_cap=%.2f caps=%.2f/%.2f）",
			spec.NotionalUSD, maxUSD, cap1, cap2)
	}
	mv := h.mvenue
	if mv == nil {
		if h.cli == nil {
			return 0, fmt.Errorf("okx 客户端缺席——margin 执行面无 venue 件 fail-closed")
		}
		mv = h.cli
	}
	ccy := marginBaseCcy(spec.InstID)
	spotInst := ccy + "-USDT"
	// 证据三闸执行时点重验（borrowquote 同判据——评估→执行间率/额度可漂）：
	// 新鲜度窗/息尖峰/额度名义下限全 fail-closed。
	if why := h.marginEvidenceOK(ctx, ccy, spotInst, notional); why != "" {
		return 0, fmt.Errorf("借币证据闸: %s", why)
	}
	dec := h.gate.Check(ctx, risk.Intent{StrategyRef: spec.StrategyRef, Venue: spec.Venue,
		InstID: spec.InstID, Legs: []string{"spot_short_borrowed"},
		NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return 0, fmt.Errorf("闸拒: %s", dec.Reason)
	}
	// JIT 赎回——adjEq≈0（资金停泊 Earn/策略冻结）时 maxLoan≈0；先摆渡
	// Earn→trading 再借（earn.RedeemToTrading 现成件——nil=凭据缺席 fail-closed）。
	if h.redeem == nil {
		return 0, fmt.Errorf("earn 赎回件缺席——JIT 摆渡不可行 fail-closed")
	}
	if err := h.redeem(ctx, notional); err != nil {
		return 0, fmt.Errorf("JIT 赎回败（不借不卖）: %w", err)
	}
	// 在借仓唯一性——fin_margin_loan 一币一仓（unique 部分索引兜底，
	// 前置查=友好拒因不撞约束）。
	var dup int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM fin_margin_loan
		WHERE venue='okx' AND ccy=$1 AND status IN ('open','closing')`, ccy).Scan(&dup); err != nil {
		return 0, fmt.Errorf("在借仓查败 fail-closed: %w", err)
	}
	if dup > 0 {
		return 0, fmt.Errorf("%s 在借仓已在——一币一仓拒双借", ccy)
	}
	px := marginSpotPx(ctx, h.pool, spotInst)
	if px <= 0 {
		return 0, fmt.Errorf("现货价不可得（md 两级读面空）——qty 不可算拒")
	}
	qty := notional / px
	// ①借——spot-manual-borrow-repay（spec 钦定端点）。
	if _, err := mv.SpotManualBorrowRepay(ctx, ccy, qty, "borrow"); err != nil {
		return 0, fmt.Errorf("借币发单败: %w", err)
	}
	// ②借后必经确认才卖——历史行侧=borrow-repay-history 无 clOrdId；
	// 确认窗=近 2min 新行 amt≈qty side=borrow。查败/无行=ambiguous
	// →不卖不还（借了不卖=负债裸持须人工/下拍对账——如实上抛）。
	if err := marginBorrowConfirm(ctx, mv, ccy, qty); err != nil {
		return 0, fmt.Errorf("借币确认败（已发单未卖——负债裸持待对账）: %w", err)
	}
	// ③卖现货——market sell tgtCcy=base_ccy sz=qty（借来多少卖多少）。
	clOrdID := fmt.Sprintf("mgo%d", time.Now().UnixNano()%1e12)
	if _, err := mv.PlaceOrder(ctx, okxexec.OrderParams{
		InstID: spotInst, TdMode: "cash", Side: "sell", OrdType: "market",
		Sz: fmt.Sprintf("%g", qty), ClOrdID: clOrdID, TgtCcy: "base_ccy"}); err != nil {
		return 0, fmt.Errorf("卖腿发单败（借币已成负债——待 marginLoanTick/人工对账）: %w", err)
	}
	st, _, fillSz, _, _, rerr := mv.ResolveOrderDetail(ctx, spotInst, clOrdID)
	if rerr != nil || st != okxexec.StateFilled || fillSz <= 0 {
		return 0, fmt.Errorf("卖腿未 filled（state=%s fill=%.8g err=%v）——借币负债在账待对账", st, fillSz, rerr)
	}
	// ④簿记——execution+position(delta_neutral=false 非中性裸空)+loan+
	// spot_sell 现金流（卖出毛额=fillSz×avgPx→USDT；费走 bills 域不重记）。
	return h.marginBookOpen(ctx, spec, dec.DecisionID, ccy, spotInst, qty, fillSz, px, notional)
}

// marginEvidenceOK 执行时点证据三闸（borrowquote 同判据重验——md_
// margin_borrow 最新行：缺/陈=拒、息>闸=拒、额度名义<notional=拒）。
func (h hedgeOpener) marginEvidenceOK(ctx context.Context, ccy, spotInst string, notional float64) string {
	staleH, err := h.cfg.GetFloat("funding.margin_borrow_stale_hours")
	if err != nil || staleH <= 0 {
		return "borrow_evidence_unconfigured"
	}
	maxAPR, err := h.cfg.GetFloat("funding.margin_borrow_max_apr")
	if err != nil || maxAPR <= 0 {
		return "borrow_gate_unconfigured"
	}
	var apr, maxLoan float64
	var ts time.Time
	if err := h.pool.QueryRow(ctx, `SELECT borrow_apr, max_loan, ts FROM md_margin_borrow
		WHERE ccy=$1 ORDER BY ts DESC LIMIT 1`, ccy).Scan(&apr, &maxLoan, &ts); err != nil {
		return "borrow_evidence_missing"
	}
	if time.Since(ts) > time.Duration(staleH*float64(time.Hour)) {
		return "borrow_evidence_stale"
	}
	if apr > maxAPR {
		return fmt.Sprintf("borrow_rate_spike:apr=%.4f>闸%.4f", apr, maxAPR)
	}
	if px := marginSpotPx(ctx, h.pool, spotInst); px > 0 && maxLoan*px < notional {
		return fmt.Sprintf("borrow_quota_insufficient:loan_usd=%.2f<notional %.2f", maxLoan*px, notional)
	}
	return ""
}

// marginBorrowConfirm 借后确认——spot-borrow-repay-history 近 2min 内
// 本币 side=borrow 行 amt 与请求量同量级（±1%——venue 精度截断容忍）。
// 歧态势律：无行/读败=err 上抛——调用面不卖不猜。
func marginBorrowConfirm(ctx context.Context, mv marginVenue, ccy string, qty float64) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		hist, err := mv.SpotBorrowRepayHistory(ctx, ccy, "", 10)
		if err != nil {
			return fmt.Errorf("借后历史读败（歧态势）: %w", err)
		}
		for _, r := range hist {
			if r.Side != "borrow" {
				continue
			}
			ts, perr := strconv.ParseInt(r.TS, 10, 64)
			if perr != nil || time.Since(time.UnixMilli(ts)) > 2*time.Minute {
				continue
			}
			amt := okxexec.ParseF(r.Amt)
			if amt > 0 && (amt >= qty*0.99 && amt <= qty*1.01) {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("借后 5s 窗内无匹配 borrow 历史行（qty=%.8g）——歧态势不卖", qty)
		}
		time.Sleep(500 * time.Millisecond) // sched-exempt-ok：开仓事务内 5s 有界确认窗（非周期调度——一次性等待 venue 历史行落账）
	}
}

// marginBookOpen 簿记四件同事务：fin_execution(legged)+fin_position
// (delta_neutral=false——保证金裸空非中性)+fin_margin_loan(open)+
// fin_cashflow spot_sell 毛额（position_ref 挂接——息腿同位）。
func (h hedgeOpener) marginBookOpen(ctx context.Context, spec lifecycle.OpenSpec,
	decisionID int64, ccy, spotInst string, qty, fillSz, px, notional float64) (int64, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var execID int64
	if err := tx.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,inst_id,notional_usd,state)
		VALUES($1,$2,$3,$4,'legged') RETURNING id`,
		spec.StrategyRef, decisionID, spotInst, notional).Scan(&execID); err != nil {
		return 0, fmt.Errorf("execution 簿记: %w", err)
	}
	var posID int64
	if err := tx.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,delta_neutral)
		VALUES($1,$2,$3,$4,false) RETURNING id`,
		execID, spec.StrategyRef, spotInst, notional).Scan(&posID); err != nil {
		return 0, fmt.Errorf("position 簿记: %w", err)
	}
	var apr float64
	_ = tx.QueryRow(ctx, `SELECT borrow_apr FROM md_margin_borrow
		WHERE ccy=$1 ORDER BY ts DESC LIMIT 1`, ccy).Scan(&apr)
	var loanID int64
	if err := tx.QueryRow(ctx, `INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr,
		status,strategy_ref,position_ref,execution_id)
		VALUES('okx',$1,$2,$3,$4,'open',$5,$6,$7) RETURNING id`,
		ccy, spotInst, qty, apr, spec.StrategyRef, posID, execID).Scan(&loanID); err != nil {
		return 0, fmt.Errorf("margin_loan 簿记: %w", err)
	}
	proceeds := fillSz * px
	// spot_sell 毛额同事务直插（ledger.Post 无 tx 变体——class 由
	// fin_cashflow_class_stamp 触发器落；费走 bills 域不重记）。
	if _, err := tx.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('spot_sell','okx','USDT',$1,$2,$3,'fin_execution',now())`,
		proceeds, posID, execID); err != nil {
		return 0, fmt.Errorf("spot_sell 现金流: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return execID, nil
}
