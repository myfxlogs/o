// mover.go —— Earn 钱动执行面（回存/赎回）+留痕。律：传输败≠业务拒——transfer
// 按 clTId 查 transfer-state 定真态（同 okx-exec clOrdId 定态三分支，禁盲重发）；
// 补偿只对「确认未动」态做（在途/未知禁补偿——双动险）；钱动成=decision+cashflow
// 双行留痕（source_ref=decision id，禁推断记账）。
package earn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	"phase/internal/ledger"
	"phase/internal/trading/okxexec"
)

// transferOutcome 划转定态三值（okx-exec 定态律的 transfer 版）。
type transferOutcome int

const (
	txMoved     transferOutcome = iota // 确认已动（含传输名败实成——state=success 定真）
	txFailed                           // 确认未动（state=failed——钱在本端原位）
	txAmbiguous                        // 在途/态不可知——禁补偿禁重发
)

const (
	transferTxSuccess = "success"
	transferTxFailed  = "failed"
)

// RedeemToTrading 开仓前置赎回：Earn 灵活赎回 usdAmt*(1+buffer bps) 回拢交易户。
// 赎回/划转败=fail-closed 上抛（执行链不开仓）；buffer 键缺=fail-closed。
func (w *Worker) RedeemToTrading(ctx context.Context, usdAmt float64) error {
	if w.api == nil {
		w.rej.Inc("venue_absent")
		return fmt.Errorf("earn venue 未挂接（凭据缺席）——fail-closed 不开仓")
	}
	bps, err := w.getF("earn.redeem_buffer_bps")
	if err != nil {
		w.rej.Inc("missing_param")
		return fmt.Errorf("earn.redeem_buffer_bps: %w", err)
	}
	amt := usdAmt * (1 + bps/1e4)
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.api.SavingsRedeem(ctx, w.ccy, amt); err != nil {
		return fmt.Errorf("earn redeem %.4f: %w", amt, err) // 钱未动——fail-closed
	}
	out, terr := w.transfer(ctx, w.ccy, amt, okxexec.AcctFunding, okxexec.AcctTrading)
	if out == txMoved {
		w.afterMove(ctx, "earn_redeem", amt, "earn_in", amt,
			fmt.Sprintf("开仓前赎回 %.4f USDT 回拢交易户（notional %.4f+buffer %.1f bps）", amt, usdAmt, bps))
		return nil
	}
	if out == txFailed {
		// 赎回款滞 funding 户——补偿回停恢复停泊态（确认未动才补偿）
		if perr := w.api.SavingsPurchase(ctx, w.ccy, amt); perr != nil {
			w.rej.Inc("compensation_failed")
			log.Printf("ALERT [earn] 赎回后划转败且回停补偿败——%.4f USDT 滞 funding 户: %v", amt, perr)
		}
		return fmt.Errorf("earn redeem 划转败: %w", terr)
	}
	w.rej.Inc("transfer_ambiguous")
	w.al.Alert(ctx, "transfer_ambiguous", "划转歧态势——禁补偿中止")
	log.Printf("ALERT [earn] 赎回款 %.4f 在途/态不可知——禁补偿（双动险），开仓 fail-closed 中止", amt)
	return fmt.Errorf("earn redeem 划转在途: %w", terr)
}

// depositIdle 闲余回存停泊：摆渡 funding 户→申购 Earn。败=计数+ALERT+同因 5min 退避（T-0059）。
// 调用方持单飞锁。
func (w *Worker) depositIdle(ctx context.Context, amt float64) (moved bool) {
	out, err := w.transfer(ctx, w.ccy, amt, okxexec.AcctTrading, okxexec.AcctFunding)
	if out != txMoved {
		w.rej.Inc("deposit_failed")
		log.Printf("ALERT [earn] 回存摆渡败 %.8f USDT（%v）——钱留交易户，同因 5min 退避", amt, err)
		return false
	}
	if err := w.api.SavingsPurchase(ctx, w.ccy, amt); err != nil {
		w.rej.Inc("deposit_failed")
		log.Printf("ALERT [earn] Earn 申购败 %.8f USDT（%v）——补偿摆回交易户", amt, err)
		if cout, cerr := w.transfer(ctx, w.ccy, amt, okxexec.AcctFunding, okxexec.AcctTrading); cout != txMoved {
			w.rej.Inc("compensation_failed")
			log.Printf("ALERT [earn] 回存补偿摆回亦败——%.8f USDT 滞 funding 户（人工升级）: %v", amt, cerr)
		}
		return false
	}
	w.afterMove(ctx, "earn_deposit", amt, "earn_out", -amt,
		fmt.Sprintf("闲资 %.8f USDT 回存 Earn 停泊（sweep_min_usd 阈上）", amt))
	return true
}

// transfer 划转+定态（T-0059 分型律）：
//   - venue 业务终拒（*APIError——58xxx/5xxxx 族）=确定单未建→txFailed 直返，
//     **禁送仲裁禁重试**（58350 终拒→误送仲裁→58129「态未定」ALERT→5s 重试死锁实证）。
//   - 传输名败（超时/5xx/conn 断）才进 transfer-state 仲裁；仲裁查询自身业务拒
//     （*APIError）=该 clientId 无单在案→txFailed（58129/51000 双坑即此态 venue 表达）；
//     查询亦传输败=txAmbiguous 禁重发禁补偿。
func (w *Worker) transfer(ctx context.Context, ccy string, amt float64, from, to string) (transferOutcome, error) {
	clTID, transID, err := w.api.AssetTransfer(ctx, ccy, amt, from, to)
	if err == nil {
		if transID != "" {
			log.Printf("[earn] transfer 成 clTId=%s transId=%s amt=%.8f", clTID, transID, amt)
		}
		return txMoved, nil
	}
	var ae *okxexec.APIError
	if errors.As(err, &ae) {
		w.rej.Inc("transfer_rejected")
		return txFailed, fmt.Errorf("transfer 终拒（clTId=%s code=%s）: %w", clTID, ae.Code, err)
	}
	state, serr := w.api.AssetTransferState(ctx, ccy, clTID)
	switch {
	case serr == nil && state == transferTxSuccess:
		return txMoved, nil // 传输名败实成——定真态放行
	case serr == nil && state == transferTxFailed:
		return txFailed, fmt.Errorf("transfer state=failed（clTId=%s）: %w", clTID, err)
	case serr != nil && errors.As(serr, &ae):
		return txFailed, fmt.Errorf("transfer 终拒+查态无单在案（clTId=%s qcode=%s）: %w", clTID, ae.Code, err)
	default:
		w.rej.Inc("transfer_ambiguous")
		return txAmbiguous, fmt.Errorf("transfer 态未定（clTId=%s state=%s qerr=%v）: %w", clTID, state, serr, err)
	}
}

// afterMove 钱动成：decision(transfer)+cashflow 双行留痕+停泊快照事件锚定刷。
// 留痕写败不回滚钱动（ALERT+计数；对账环差即报兜底）。
func (w *Worker) afterMove(ctx context.Context, op string, amt float64, kind string, cashAmount float64, msg string) {
	intent, _ := json.Marshal(map[string]any{
		"kind": "transfer", "op": op, "venue": w.venue, "ccy": w.ccy, "amt": amt})
	var dID int64
	if err := w.pool.QueryRow(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES($1,'allow','transfer',$2) RETURNING id`, intent, w.cfg.Generation()).Scan(&dID); err != nil {
		w.rej.Inc("write_err")
		log.Printf("ALERT [earn] %s——decision 留痕写败: %v", msg, err)
	} else if kind == "earn_in" || kind == "earn_out" {
		// T-0060 根修：停泊/赎回=venue 内部划转（trading↔funding↔earn——
		// NAV 总额零影响）——fin_cashflow 单边落行=等额假差病灶，停写；
		// 留痕径=decision 行+md_account_events earn_parking 事件行。
	} else if _, err := ledger.Post(ctx, w.pool, ledger.Entry{
		Kind: kind, Venue: w.venue, Ccy: w.ccy, Amount: cashAmount,
		SourceRef: dID, SourceTable: "fin_risk_decision",
	}); err != nil {
		w.rej.Inc("write_err")
		log.Printf("ALERT [earn] %s——cashflow 留痕写败（decision #%d 已在）: %v", msg, dID, err)
	}
	if err := w.refreshParkingSnapshot(ctx, parkingSrcMove); err != nil {
		log.Printf("[earn] %s——停泊快照刷败: %v", msg, err)
	}
	log.Printf("[earn] %s", msg)
}

// refreshParkingSnapshot 停泊余额快照（事件锚定：boot 首读/移动后刷——周期态摄入禁新增）。
// 余额=Σamt（T-0046 ⑦根修，canon earn-amt-pending-doublecount：savings 的 amt
// 已含在途 pendingAmt——loanAmt+pendingAmt=amt 实证，再求和=双计虚高），按 ccy 过滤。
func (w *Worker) refreshParkingSnapshot(ctx context.Context, src string) error {
	if w.api == nil {
		w.rej.Inc("venue_absent")
		return fmt.Errorf("earn venue 未挂接")
	}
	raw, err := w.api.SavingsBalance(ctx)
	if err != nil {
		w.rej.Inc("snapshot_refresh_failed")
		return err
	}
	var rows []struct {
		Ccy string `json:"ccy"`
		Amt string `json:"amt"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		w.rej.Inc("snapshot_refresh_failed")
		return err
	}
	var bal float64
	for _, r := range rows {
		if r.Ccy != w.ccy {
			continue
		}
		bal += parseAmt(r.Amt)
	}
	blob, _ := json.Marshal(map[string]any{"earn_bal": bal, "src": src})
	if _, err := w.pool.Exec(ctx, `INSERT INTO md_account_events(ts,venue,channel,ccy,data)
		VALUES(now(),$1,'earn_parking',$2,$3)`, w.venue, w.ccy, blob); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	return nil
}

// parseAmt venue 金额串→float（空/坏=0——快照面坏值不炸读径）。
func parseAmt(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
