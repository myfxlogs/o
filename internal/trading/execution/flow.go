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
		// T-0145：leg1 歧态势（传输败/live/部分成→Resolve 定不了真态）——
		// venue 可能已成仓（裸腿残留）=告警升件人查，禁静默当败。
		if res1.State == "unknown" || res1.State == "live" || res1.State == "partially_filled" {
			r.alert(ctx, "leg1_ambiguous",
				fmt.Sprintf("exec#%d leg1 %s@%s 歧态势（state=%s）——venue 可能已成仓，残留敞口人查", execID, l1.InstID, l1.Venue, res1.State))
		}
		r.setState(ctx, execID, "failed")
		return execID, fmt.Errorf("leg1: %w", err)
	}
	// ⑤leg2——败/歧义=LEGGED→自动回滚 leg1（不留半成品敞口）。
	res2, err := r.placeAndResolve(ctx, execID, 2, l2, posRef)
	if err != nil || res2.State == "ambiguous" {
		// T-0145：leg2 歧义（回滚 leg1 时 leg2 真态未定——两腿可能同存=
		// 双敞口事故）=告警升件人查定真态。
		if res2.State == "unknown" || res2.State == "live" || res2.State == "partially_filled" {
			r.alert(ctx, "leg2_ambiguous",
				fmt.Sprintf("exec#%d leg2 %s@%s 歧态势（state=%s）——回滚 leg1 后 leg2 实存须人查", execID, l2.InstID, l2.Venue, res2.State))
		}
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
	// T-0238 maker 优先径：exec.maker_enabled 且 okx 腿且盘口鲜活→post_only
	// 挂价等时；时限/未成/拒→取消定态降级 market（成=低费，败=原径无损）。
	if mres, ok := r.makerAttempt(ctx, execID, no, leg, posRef); ok {
		if mres.State != "partial" {
			return mres, nil
		}
		leg = scaleLegRemainder(leg, mres) // 碎量余腿缩放（maker.go——行棘轮拆位）
	}
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
		legID := r.legPx(ctx, execID, no, leg, clOrdID, "filled", vc, vm, res.FillPx, res.FillSz, "market", res.FillTicket)
		r.legCash(ctx, legID, leg, res.FillPx, res.FillSz, posRef)
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

func (r *Runner) leg(ctx context.Context, execID int64, no int, leg Leg, clOrdID, state, venueCode, venueMsg string) {
	r.legPx(ctx, execID, no, leg, clOrdID, state, venueCode, venueMsg, 0, 0, "market", 0)
}

// legPx 腿行落库携成交价+量纲消歧（T-0107：sz_unit=腿 sz 语义位——SWAP=contract/
// mt5 cfd=lot/spot quote_ccy 委托=quote/余=base；qty_base=venue 实填币量
// （accFillSz）唯一真源——请求量禁当成交记，fill 缺=NULL 消费面 fail-closed
// 禁猜；mt5 腿 qty_base=fillSz×CtVal 面值换算）。px=NULL 旧调用面保值守；
// ticket=venue 仓凭据（MT5——平仓必携；okx=0 落 NULL 略）。返腿行 id。
func (r *Runner) legPx(ctx context.Context, execID int64, no int, leg Leg, clOrdID, state, venueCode, venueMsg string, px, fillSz float64, ordType string, ticket int64) int64 {
	var pxArg *float64
	if px > 0 {
		pxArg = &px
	}
	var tkArg *int64
	if ticket > 0 {
		tkArg = &ticket
	}
	szUnit := "base"
	var qtyBase *float64
	switch {
	case leg.Venue == "mt5":
		szUnit = "lot"
		if fillSz > 0 && leg.CtVal > 0 {
			qb := fillSz * leg.CtVal
			qtyBase = &qb
		}
	case strings.HasSuffix(leg.InstID, "-SWAP"):
		szUnit = "contract"
	case leg.TgtQuote:
		szUnit = "quote"
		if fillSz > 0 {
			qtyBase = &fillSz
		}
	default:
		if fillSz > 0 {
			qtyBase = &fillSz
		}
	}
	var id int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state,venue_err_code,venue_err_msg,px,sz_unit,qty_base,ord_type,venue_ticket)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),NULLIF($10,''),$11,$12,$13,$14,$15) RETURNING id`,
		execID, no, clOrdID, leg.Venue, leg.Side, leg.InstID, leg.Sz, state, venueCode, venueMsg, pxArg, szUnit, qtyBase, ordType, tkArg).Scan(&id); err != nil {
		r.rej.Inc("write_err")
	}
	return id
}

// legCash 腿级现金行（T-0083 对称记账）：okx 现货腿（非 SWAP）买=流出毛额负、
// 卖=流入毛额正——两腿各记净额自然=真价差；合约腿无名义现金（结算走 bills/fee）。
// source_ref=腿行 id（幂等溯源锚）；perp/CFD/杂 venue 不触现径。
func (r *Runner) legCash(ctx context.Context, legID int64, leg Leg, px, fillSz float64, posRef *int64) {
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
		kind, leg.Venue, sign*r.legGross(leg, px, fillSz), posRef, legID); err != nil {
		r.rej.Inc("write_err")
	}
}

// legGross 腿毛额（T-0116 根修：quote_ccy 买=成交额 accFillSz×avgPx 非委托
// 额——OKX 市价 quote 买实扣<委托，差额回退交易户现金账本看不见；成交额
// 读不到=退委托额并计 rej.gross_fallback 显形）。base 委托=sz×px；px/sz 缺=
// 名义兜底——毛额锚不空。回滚卖腿（rev）走名义锚（quote 额不可当 base 量卖）。
func (r *Runner) legGross(leg Leg, px, fillSz float64) float64 {
	sz, _ := strconv.ParseFloat(leg.Sz, 64)
	if leg.TgtQuote {
		if leg.Side == "buy" && sz > 0 {
			if fillSz > 0 && px > 0 {
				return fillSz * px // T-0116：成交额锚（非委托额）
			}
			r.rej.Inc("gross_fallback")
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
