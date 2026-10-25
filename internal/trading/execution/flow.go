package execution

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"o/internal/trading/okxexec"
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
	// T-0083 腿级现金流对称：现货腿现金行 source_ref=腿行 id+position_ref 归仓
	// ——close 径仓已存按 inst 现读归仓；open 径仓行在双腿成交后建（linkLegCash 补挂）。
	var posRef *int64
	if in.Kind == "close" {
		var pid int64
		if err := r.pool.QueryRow(ctx, `SELECT id FROM fin_position
			WHERE inst_id=$1 AND closed_at IS NULL ORDER BY id DESC LIMIT 1`,
			in.InstID).Scan(&pid); err == nil {
			posRef = &pid
		}
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
	res1, err := r.placeAndResolve(ctx, execID, 1, l1, posRef)
	if err != nil {
		r.setState(ctx, execID, "failed")
		return execID, fmt.Errorf("leg1: %w", err)
	}
	// ⑤leg2——败/歧义=LEGGED→自动回滚 leg1（不留半成品敞口）。
	res2, err := r.placeAndResolve(ctx, execID, 2, l2, posRef)
	if err != nil || res2.State == "ambiguous" {
		r.setState(ctx, execID, "legged")
		r.rollback(ctx, execID, l1, res1)
		return execID, fmt.Errorf("leg2 败/歧义→LEGGED 回滚 leg1")
	}
	// ⑥仓行（delta 中性标——funding 对冲天然）——仅 open 径建（T-0083 返修①：
	// close 径禁建新仓行——被平的是既有仓（guard markClosed 关原行），close 建
	// 行=空壳壳行+cash 错绑病灶源）；linkLegCash 同律只挂 open 径（返修②：
	// close 腿 posRef 已按开仓仓现读对，execution_id 回溯会错绑本 exec 壳行）。
	if in.Kind != "close" {
		var posID int64
		if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
			VALUES($1,$2,$3,$4) RETURNING id`,
			execID, in.StrategyRef, in.InstID, in.NotionalUSD).Scan(&posID); err != nil {
			r.rej.Inc("write_err")
		} else {
			r.linkLegCash(ctx, execID, posID)
		}
	}
	log.Printf("[execution] #%d %s 双腿成交（clOrdId %s/%s）", execID, in.InstID, res1.ClOrdID, res2.ClOrdID)
	return execID, nil
}

// placeAndResolve 下单+歧义定态（okx-exec 契约：传输败→Resolve 定真态；
// unknown=ambiguous 禁补单）。腿行每拍落 fin_order_leg。
func (r *Runner) placeAndResolve(ctx context.Context, execID int64, no int, leg Leg, posRef *int64) (OrderResult, error) {
	// clOrdId=大小写英数 only（OKX 实拍 51000——T-0038：连字符非法；字母位分隔）。
	clOrdID := fmt.Sprintf("ph%dx%dx%d", execID, no, time.Now().UnixNano()%1e6)
	res, err := r.exec.Place(ctx, leg, clOrdID)
	// envelope 业务拒（*APIError）分码族定态（T-0042 返修）：
	// 51xxx=参数/业务拒族——确定单未建，absent+拒码进 leg 行 venue_err_*；
	// 非 51xxx（50004 超时/50011 限速/50013 系统忙等）——单或已建=歧态势，
	// 留痕后仍走 Resolve 定真态（禁直接 absent）。
	var vc, vm string
	var ae *okxexec.APIError
	if errors.As(err, &ae) {
		vc, vm = ae.Code, ae.Msg
		if strings.HasPrefix(ae.Code, "51") {
			r.leg(ctx, execID, no, leg, clOrdID, "failed", vc, vm)
			return OrderResult{ClOrdID: clOrdID, State: "absent"}, fmt.Errorf("venue 拒: %w", err)
		}
	}
	if err != nil {
		// 传输败/非族 envelope 拒→clOrdId 定态三分支；查询亦败=ambiguous 禁补单
		//（拒码 vc/vm 透传留痕——禁走 err 径丢码）。
		res, err = r.exec.Resolve(ctx, leg, clOrdID)
		if err != nil || res.State == "unknown" {
			r.leg(ctx, execID, no, leg, clOrdID, "ambiguous", vc, vm)
			return res, fmt.Errorf("ambiguous")
		}
	}
	switch res.State {
	case "filled":
		legID := r.legPx(ctx, execID, no, leg, clOrdID, "filled", vc, vm, res.FillPx)
		r.legCash(ctx, legID, leg, res.FillPx, posRef)
		return res, nil
	case "live", "partially_filled":
		r.leg(ctx, execID, no, leg, clOrdID, "ack", vc, vm)
		return res, fmt.Errorf("未即时成交（live 确认轨未落）")
	default:
		// T-0046 ⑤：envelope 拒码兜底——Resolve 后 absent 态 res 无码（Resolve 不产
		// 拒码），先 Place 已抽的 vc/vm 顶上，禁留空吞真拒因。
		r.leg(ctx, execID, no, leg, clOrdID, "failed", orVal(res.VenueCode, vc), orVal(res.VenueMsg, vm))
		return res, fmt.Errorf("state=%s", res.State)
	}
}

// orVal 首非空值（venue_err 兜底装配）。
func orVal(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// rollback leg1 对向平仓腿（LEGGED 处置）；回滚败=rollback_failed 升级留痕。
func (r *Runner) rollback(ctx context.Context, execID int64, leg Leg, res OrderResult) {
	rev := leg
	if leg.Side == "buy" {
		rev.Side = "sell"
	} else {
		rev.Side = "buy"
	}
	// 回滚方向律（T-0046 ③根修，canon rollback-close-direction）：rev.Close 随原腿
	// 反置——开腿回滚=平（T-0036 语义）；平腿回滚=重开（恒 true=reduceOnly 打空仓
	// 拒单→歧态势，exec5 实证）。语义位按 Intent 上下文派生非恒定值。
	rev.Close = !leg.Close
	rev.Ticket = res.FillTicket
	clOrdID := fmt.Sprintf("ph%drb%d", execID, time.Now().UnixNano()%1e6)
	rb, err := r.exec.Place(ctx, rev, clOrdID)
	// 回滚径拒码留痕（T-0046 ⑤：任何 Place/Resolve 败径都带拒码——envelope 业务拒
	// 先抽码，透传失败面 leg 行）。
	var rvc, rvm string
	var rae *okxexec.APIError
	if errors.As(err, &rae) {
		rvc, rvm = rae.Code, rae.Msg
	}
	if err != nil {
		rb, err = r.exec.Resolve(ctx, rev, clOrdID)
	}
	if err != nil || (rb.State != "filled" && rb.State != "live" && rb.State != "partially_filled") {
		r.setState(ctx, execID, "rollback_failed")
		r.rej.Inc("rollback_failed")
		r.al.Alert(ctx, "rollback_failed", "LEGGED 回滚败——人工介入")
		log.Printf("ALERT [execution] #%d 回滚败——残留 leg1 敞口须人工升级处置（err=%v）", execID, err)
		r.leg(ctx, execID, 1, leg, clOrdID, "ambiguous", rvc, rvm)
		return
	}
	legID := r.legPx(ctx, execID, 1, leg, clOrdID, "rolled_back", "", "", 0)
	// 回滚腿=真实现金事件（买回即卖出）——spot 腿同径记账（rev.Side 定向，px 缺=名义锚）。
	r.legCash(ctx, legID, rev, rb.FillPx, nil)
	r.setState(ctx, execID, "rolled_back")
	log.Printf("[execution] #%d LEGGED→leg1 回滚成", execID)
}

func (r *Runner) leg(ctx context.Context, execID int64, no int, leg Leg, clOrdID, state, venueCode, venueMsg string) {
	r.legPx(ctx, execID, no, leg, clOrdID, state, venueCode, venueMsg, 0)
}

// legPx 腿行落库携成交价（T-0077：avgPx>0 才落——filled 态 anchor；
// px=NULL 旧调用面保值守）。返腿行 id（cashflow source_ref 锚用）。
func (r *Runner) legPx(ctx context.Context, execID int64, no int, leg Leg, clOrdID, state, venueCode, venueMsg string, px float64) int64 {
	var pxArg *float64
	if px > 0 {
		pxArg = &px
	}
	var id int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state,venue_err_code,venue_err_msg,px)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),NULLIF($10,''),$11) RETURNING id`,
		execID, no, clOrdID, leg.Venue, leg.Side, leg.InstID, leg.Sz, state, venueCode, venueMsg, pxArg).Scan(&id); err != nil {
		r.rej.Inc("write_err")
	}
	return id
}

// legCash 腿级现金行（T-0083 对称记账）：okx 现货腿（非 SWAP）买=流出毛额负、
// 卖=流入毛额正——两腿各记净额自然=真价差；合约腿无名义现金（结算走 bills/fee）。
// source_ref=腿行 id（幂等溯源锚）；perp/CFD/杂 venue 不触现径。
func (r *Runner) legCash(ctx context.Context, legID int64, leg Leg, px float64, posRef *int64) {
	if legID == 0 || leg.Venue != "okx" || strings.HasSuffix(leg.InstID, "-SWAP") {
		return
	}
	kind, sign := "spot_buy", -1.0
	if leg.Side == "sell" {
		kind, sign = "spot_sell", 1.0
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_cashflow
		(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES($1,$2,'USDT',$3,$4,$5,'fin_order_leg',now())`,
		kind, leg.Venue, sign*legGross(leg, px), posRef, legID); err != nil {
		r.rej.Inc("write_err")
	}
}

// legGross 腿毛额（quote_ccy 买委托=sz 即 USDT 额；base 委托=sz×px；px/sz 缺=
// 名义兜底——毛额锚不空）。回滚卖腿（rev）走名义锚（quote 额不可当 base 量卖）。
func legGross(leg Leg, px float64) float64 {
	sz, _ := strconv.ParseFloat(leg.Sz, 64)
	if leg.TgtQuote {
		if leg.Side == "buy" && sz > 0 {
			return sz
		}
		return leg.Notional
	}
	if sz > 0 && px > 0 {
		return sz * px
	}
	return leg.Notional
}

// linkLegCash open 径仓行建后补挂 position_ref（腿行 source_ref 锚回溯——
// 下单时仓未生 position_ref 暂 NULL，此处补链不丢归仓）。
func (r *Runner) linkLegCash(ctx context.Context, execID, posID int64) {
	if _, err := r.pool.Exec(ctx, `UPDATE fin_cashflow c SET position_ref=$2
		FROM fin_order_leg l WHERE c.source_table='fin_order_leg'
		AND c.source_ref=l.id AND l.execution_id=$1 AND c.position_ref IS NULL`,
		execID, posID); err != nil {
		r.rej.Inc("write_err")
	}
}
func (r *Runner) setState(ctx context.Context, execID int64, state string) {
	if _, err := r.pool.Exec(ctx, "UPDATE fin_execution SET state=$2 WHERE id=$1", execID, state); err != nil {
		r.rej.Inc("write_err")
	}
}
