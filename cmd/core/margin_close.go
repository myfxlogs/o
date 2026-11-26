package main

// margin_close.go —— O-40707 平仓序族（margin_loan.go 拍面姊妹件，
// O-41350 棘轮拆分）：flatten_failed 重试闸+封顶升件+幂等平仓序+
// repay 额 venue 负债面。

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/trading/okxexec"
)

// retryFlattenFailed flatten_failed 重试闸（O-41350）：
// 连击<max_fails∧距上试≥retry_ivl→marginCloseLoan 重入（幂等序——
// 先 ResolveOrderDetail 同 clOrdID 定 flatten 前序态，filled=直进 repay
// 不双买）；键缺/非正=maxFails 0=不自动重试，直接补落封顶升件
// （fail-closed——重试权无权威值不裸跑）。连击达顶=升件一次
// （flatten_escalated 幂等标）后永久停自动重试=人工处置面。
func retryFlattenFailed(ctx context.Context, pool *pgxpool.Pool,
	cfg *config.Client, venue marginVenue, l marginLoan) error {
	maxFails, err1 := cfg.GetFloat("margin.flatten_max_fails")
	ivlS, err2 := cfg.GetFloat("margin.flatten_retry_ivl_s")
	if err1 != nil || err2 != nil || maxFails < 0 || ivlS <= 0 {
		maxFails, ivlS = 0, 0 // 判据缺=重试权关断（下方封顶升件径仍走）
	}
	if float64(l.FlattenFails) >= maxFails {
		return marginFlattenEscalate(ctx, pool, l) // 封顶态——升件幂等补落即返
	}
	if l.LastFlattenTryAt != nil &&
		time.Since(*l.LastFlattenTryAt) < time.Duration(ivlS*float64(time.Second)) {
		return nil // 防抖窗内——下拍续
	}
	return marginCloseLoan(ctx, pool, venue, l, "flatten_retry")
}

// marginFlattenEscalate 连击封顶升件（flatten_escalated 一次性标——
// 重复拍不重复刷件；负债本体仍在枚举窗=息腿/健康闸不受升件影响）。
func marginFlattenEscalate(ctx context.Context, pool *pgxpool.Pool, l marginLoan) error {
	if l.FlattenEscalated {
		return nil
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_margin_loan SET flatten_escalated=true
		WHERE id=$1`, l.ID); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','O-41350',$1,'margin_flatten_fail','okx')`,
		fmt.Sprintf("margin-short flatten 序连击封顶 loan#%d %s（fails=%d）——负债仍在账计息，自动平仓序停待人工处置", l.ID, l.Ccy, l.FlattenFails))
	return err
}

// marginCloseLoan 平仓序（spec 钉死先 flatten 后 repay；O-41350 幂等化
// 重写——同 clOrdID=mgc<id> 前序态先定，retry/closing 重入不双买）：
// ①ResolveOrderDetail 定 flatten 前序态：filled=直进 repay（前单已成）、
//
//	live/partial=在途 flatten_failed 留账下拍续、absent/终态=发新单；
//	读败=状态还原上抛不动不猜。
//
// ②市价买回 amt 基币（tgtCcy=base_ccy——sz=币量）③定态 filled 才进
// ④SpotManualBorrowRepay repay——flatten 败/歧态=flatten_failed 连击计数
//
//	+防抖时戳不 repay（币没买回就还=裸空事故）；repay 败=留 closing 态
//	下拍续（幂等：venue 侧 repay 重复=重复还——先查历史对账，见
//	marginRepayOnce）。
func marginCloseLoan(ctx context.Context, pool *pgxpool.Pool, venue marginVenue,
	l marginLoan, reason string) error {
	prev := l.Status
	if _, err := pool.Exec(ctx, `UPDATE fin_margin_loan SET status='closing',
		note=$2 WHERE id=$1 AND status IN ('open','closing','flatten_failed')`,
		l.ID, reason); err != nil {
		return err
	}
	clOrdID := fmt.Sprintf("mgc%d", l.ID)
	// ①flatten 前序态定真——同 clOrdID 幂等：filled=前单已成直进 repay
	// 不双买（flatten_failed/closing 重入核心防线）；live/partial=在途
	// 不动；absent=发新单。读败=还原前态上抛（不动不猜——歧态势律）。
	st, _, fillSz, _, _, rerr := venue.ResolveOrderDetail(ctx, l.InstID, clOrdID)
	if rerr != nil {
		_, _ = pool.Exec(ctx, `UPDATE fin_margin_loan SET status=$2 WHERE id=$1`, l.ID, prev)
		return fmt.Errorf("flatten 前序态读败（还原 %s 不动不猜）: %w", prev, rerr)
	}
	switch {
	case st == okxexec.StateFilled && fillSz > 0:
		// 前序 flatten 已 filled——跳过下单直进 repay。
	case st == okxexec.StateLive || st == okxexec.StatePartial:
		return marginFlattenFail(ctx, pool, l,
			fmt.Sprintf("flatten_inflight:%s fill=%.8g", st, fillSz))
	default: // absent/终态=可发新 flatten 单
		if _, err := venue.PlaceOrder(ctx, okxexec.OrderParams{
			InstID: l.InstID, TdMode: "cash", Side: "buy", OrdType: "market",
			Sz: fmt.Sprintf("%g", l.Amt), ClOrdID: clOrdID, TgtCcy: "base_ccy"}); err != nil {
			return marginFlattenFail(ctx, pool, l, "flatten_place:"+err.Error())
		}
		st, _, fillSz, _, _, rerr = venue.ResolveOrderDetail(ctx, l.InstID, clOrdID)
		if rerr != nil || st != okxexec.StateFilled || fillSz <= 0 {
			return marginFlattenFail(ctx, pool, l,
				fmt.Sprintf("flatten_state:%v fill=%.8g err=%v", st, fillSz, rerr))
		}
	}
	// ②repay——先历史对账防重复还（venue 无幂等键：同 ccy 近 10min 已有
	// repay 行且本仓未标 repaid=疑似在途/已还——跳过本次 repay 直接核）。
	if err := marginRepayOnce(ctx, pool, venue, l); err != nil {
		return err // 留 closing——下拍续（repay 对账幂等不双还）
	}
	// ③终态：仓+执行+头寸同事务收口。
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE fin_margin_loan SET status='repaid',
		repaid_at=now(), note=$2 WHERE id=$1`, l.ID, "closed:"+reason); err != nil {
		return err
	}
	if l.PositionRef != nil {
		if _, err := tx.Exec(ctx, `UPDATE fin_position SET state='closed',
			closed_at=now() WHERE id=$1 AND closed_at IS NULL`, *l.PositionRef); err != nil {
			return err
		}
	}
	if l.ExecutionID != nil {
		if _, err := tx.Exec(ctx, `UPDATE fin_execution SET state='closed'
			WHERE id=$1 AND state<>'closed'`, *l.ExecutionID); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	log.Printf("[margin] 平仓序成 %s #%d（%s）——flatten filled=%.8g→repaid", l.Ccy, l.ID, reason, fillSz)
	return nil
}

// marginFlattenFail flatten 败统一落态（O-41350）：status=flatten_failed
// +连击计数+防抖时戳+note 留因——负债仍在枚举窗=息腿/健康闸不受影响；
// 连击封顶判/升件归 retryFlattenFailed 拍面（本函数只计数不判顶——
// dead/kill 直调径同计连击，封顶闸单点在枚举消费面）。
func marginFlattenFail(ctx context.Context, pool *pgxpool.Pool, l marginLoan, note string) error {
	if _, err := pool.Exec(ctx, `UPDATE fin_margin_loan SET status='flatten_failed',
		flatten_fails=flatten_fails+1, last_flatten_try_at=now(), note=$2 WHERE id=$1`,
		l.ID, note); err != nil {
		return err
	}
	return fmt.Errorf("flatten 败留账 flatten_failed（fails+1 防抖起算）: %s", note)
}

// marginRepayOnce repay+历史对账（venue 无幂等键——先查 spot-borrow-
// repay-history：近 10min 已有本币 repay 行=疑似在途/已还→跳过发单
// 直接由调用面收口；无行才发 repay。发单败=err 上抛留 closing 下拍续）。
// O-41350④repay 额=venue 当前负债额（SpotBorrowLiab=liab+interest 含息渣
// ——还本金后息渣续计息根修）；负债面缺席（现货模式字段空）/读败→回退
// 簿记估计 l.Amt+accrued_interest（已入账息=venue 计息残部最优估计，
// 率漂残差如实留 note 不虚构）；venue 报零负债=债已清不发 repay 直接收口。
func marginRepayOnce(ctx context.Context, pool *pgxpool.Pool, venue marginVenue,
	l marginLoan) error {
	hist, err := venue.SpotBorrowRepayHistory(ctx, l.Ccy, "", 10)
	if err != nil {
		return fmt.Errorf("repay 前历史对账读败（禁盲发）: %w", err)
	}
	for _, h := range hist {
		if h.Side != "repay" {
			continue
		}
		if ts, perr := strconv.ParseInt(h.TS, 10, 64); perr == nil &&
			time.Since(time.UnixMilli(ts)) < 10*time.Minute {
			return nil // 近 10min 已有 repay 在案——不重发（歧态势律）
		}
	}
	amt := l.Amt + l.AccruedInterest // 簿记估计兜底（本金+已入账息残部）
	liab, present, lerr := venue.SpotBorrowLiab(ctx, l.Ccy)
	switch {
	case lerr != nil:
		log.Printf("[margin] %s 负债面读败——repay 回退簿记估计 %.8g: %v", l.Ccy, amt, lerr)
	case present && liab <= 1e-8:
		log.Printf("[margin] %s venue 负债面=0（债已清）——跳 repay 直接收口", l.Ccy)
		return nil
	case present:
		amt = liab
	default: // 现货模式字段空——簿记估计兜底（上方 amt 已就位）
	}
	if _, err := venue.SpotManualBorrowRepay(ctx, l.Ccy, amt, "repay"); err != nil {
		return fmt.Errorf("repay 发单败（留 closing 下拍续）: %w", err)
	}
	return nil
}
