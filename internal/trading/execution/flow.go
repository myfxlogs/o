package execution

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// Execute 双腿原子开仓：令牌→live 闸→对冲带→leg1→leg2→（败则 LEGGED 回滚）。
// 返回值只表执行结局（该不该开已过 G6——本件只保证「开成/全收回」）。
func (r *Runner) Execute(ctx context.Context, in Intent) (int64, error) {
	// ①令牌验证为第一件事（四径拒+单次消费原子）。
	tok, err := r.verifyToken(ctx, in.DecisionID)
	if err != nil {
		return 0, err
	}
	// ②live 闸（委员会开关位——开仓闸非退仓闸：close intent 免检，
	// 缺键/非 true fail-closed 只对 open）。
	if in.Kind != "close" {
		live, err := r.getB("invest.lifecycle.live_switch")
		if err != nil || !live {
			r.rej.Inc("live_switch_off")
			return 0, fmt.Errorf("live_switch_off")
		}
	}
	// ③对冲带：双腿名义额差容忍 exec.hedge_band_bps（缺键 fail-closed）。
	band, err := r.getF("exec.hedge_band_bps")
	if err != nil {
		r.rej.Inc("missing_param")
		return 0, fmt.Errorf("missing_param exec.hedge_band_bps")
	}
	l1, l2 := in.Legs[0], in.Legs[1]
	if diff := math.Abs(l1.Notional-l2.Notional) / max(l1.Notional, l2.Notional); diff > band/1e4 {
		r.rej.Inc("hedge_band")
		return 0, fmt.Errorf("hedge_band: 双腿名义差 %.4f > %.4f", diff, band/1e4)
	}
	var execID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES($1,$2,$3,$4,$5,'open') RETURNING id`,
		in.StrategyRef, in.DecisionID, tok, in.InstID, in.NotionalUSD).Scan(&execID); err != nil {
		r.rej.Inc("write_err")
		return 0, err
	}
	// ③.5 前置赎回（D-0006 停泊态：闲资默认停 Earn——开仓先赎回 notional+buffer 回拢；
	// 败=fail-closed 不开仓，留痕=exec 行翻 failed+计数门。close 免赎——退仓不进钱径）。
	if r.earn != nil && in.Kind != "close" {
		if err := r.earn.RedeemToTrading(ctx, in.NotionalUSD); err != nil {
			r.setState(ctx, execID, "failed")
			r.rej.Inc("earn_redeem_failed")
			return execID, fmt.Errorf("前置赎回: %w", err)
		}
	}
	// ④leg1
	res1, err := r.placeAndResolve(ctx, execID, 1, l1)
	if err != nil {
		r.setState(ctx, execID, "failed")
		return execID, fmt.Errorf("leg1: %w", err)
	}
	// ⑤leg2——败/歧义=LEGGED→自动回滚 leg1（不留半成品敞口）。
	res2, err := r.placeAndResolve(ctx, execID, 2, l2)
	if err != nil || res2.State == "ambiguous" {
		r.setState(ctx, execID, "legged")
		r.rollback(ctx, execID, l1, res1)
		return execID, fmt.Errorf("leg2 败/歧义→LEGGED 回滚 leg1")
	}
	// ⑥仓行（delta 中性标——funding 对冲天然）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES($1,$2,$3,$4)`, execID, in.StrategyRef, in.InstID, in.NotionalUSD); err != nil {
		r.rej.Inc("write_err")
	}
	log.Printf("[execution] #%d %s 双腿成交（clOrdId %s/%s）", execID, in.InstID, res1.ClOrdID, res2.ClOrdID)
	return execID, nil
}

// placeAndResolve 下单+歧义定态（okx-exec 契约：传输败→Resolve 定真态；
// unknown=ambiguous 禁补单）。腿行每拍落 fin_order_leg。
func (r *Runner) placeAndResolve(ctx context.Context, execID int64, no int, leg Leg) (OrderResult, error) {
	clOrdID := fmt.Sprintf("ph-%d-%d-%d", execID, no, time.Now().UnixNano()%1e6)
	res, err := r.exec.Place(ctx, leg, clOrdID)
	if err != nil {
		// 传输败→clOrdId 定态三分支；查询亦败=ambiguous 禁补单。
		res, err = r.exec.Resolve(ctx, leg, clOrdID)
		if err != nil || res.State == "unknown" {
			r.leg(ctx, execID, no, leg, clOrdID, "ambiguous")
			return res, fmt.Errorf("ambiguous")
		}
	}
	switch res.State {
	case "filled":
		r.leg(ctx, execID, no, leg, clOrdID, "filled")
		return res, nil
	case "live", "partially_filled":
		r.leg(ctx, execID, no, leg, clOrdID, "ack")
		return res, fmt.Errorf("未即时成交（live 确认轨未落）")
	default:
		r.leg(ctx, execID, no, leg, clOrdID, "failed")
		return res, fmt.Errorf("state=%s", res.State)
	}
}

// rollback leg1 对向平仓腿（LEGGED 处置）；回滚败=rollback_failed 升级留痕。
func (r *Runner) rollback(ctx context.Context, execID int64, leg Leg, res OrderResult) {
	rev := leg
	if leg.Side == "buy" {
		rev.Side = "sell"
	} else {
		rev.Side = "buy"
	}
	rev.Close = true // hedge 平仓语义根修（T-0036：反向新开非平仓——executor 按位走平径）
	rev.Ticket = res.FillTicket
	clOrdID := fmt.Sprintf("ph-%d-rb-%d", execID, time.Now().UnixNano()%1e6)
	rb, err := r.exec.Place(ctx, rev, clOrdID)
	if err != nil {
		rb, err = r.exec.Resolve(ctx, rev, clOrdID)
	}
	if err != nil || (rb.State != "filled" && rb.State != "live" && rb.State != "partially_filled") {
		r.setState(ctx, execID, "rollback_failed")
		r.rej.Inc("rollback_failed")
		r.al.Alert(ctx, "rollback_failed", "LEGGED 回滚败——人工介入")
		log.Printf("ALERT [execution] #%d 回滚败——残留 leg1 敞口须人工升级处置（err=%v）", execID, err)
		r.leg(ctx, execID, 1, leg, clOrdID, "ambiguous")
		return
	}
	r.leg(ctx, execID, 1, leg, clOrdID, "rolled_back")
	r.setState(ctx, execID, "rolled_back")
	log.Printf("[execution] #%d LEGGED→leg1 回滚成", execID)
}

func (r *Runner) leg(ctx context.Context, execID int64, no int, leg Leg, clOrdID, state string) {
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		execID, no, clOrdID, leg.Venue, leg.Side, leg.InstID, leg.Sz, state); err != nil {
		r.rej.Inc("write_err")
	}
}
func (r *Runner) setState(ctx context.Context, execID int64, state string) {
	if _, err := r.pool.Exec(ctx, "UPDATE fin_execution SET state=$2 WHERE id=$1", execID, state); err != nil {
		r.rej.Inc("write_err")
	}
}
