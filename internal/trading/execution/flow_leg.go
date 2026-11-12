package execution

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// 腿行落库一族——fin_order_leg 腿行+滑点采样+现金对称记账+position_ref 补链
// （T-0521 自 flow.go 拆出——棘轮基线 300 行闸）。

func (r *Runner) leg(ctx context.Context, execID int64, no int, leg Leg, clOrdID, state, venueCode, venueMsg string) {
	r.legPx(ctx, execID, no, leg, clOrdID, state, venueCode, venueMsg, 0, 0, "market", 0)
}

// legPx 腿行落库携成交价+量纲消歧（T-0107：sz_unit=腿 sz 语义位——合约域
// (SWAP/datedFUT,T-0483)=contract/mt5 cfd=lot/spot quote_ccy 委托=quote/余=base；qty_base=venue 实填量
// （accFillSz）唯一真源——请求量禁当成交记，fill 缺=NULL 消费面 fail-closed
// 禁猜；datedFUT=实填张数（T-0483）；mt5 腿 qty_base=fillSz×CtVal 面值换算）。px=NULL 旧调用面保值守；
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
	case datedFUTRe.MatchString(leg.InstID):
		szUnit = "contract"
		if fillSz > 0 {
			qtyBase = &fillSz // T-0483 钦定：datedFUT 实填张数落 qty_base（消费锚=guard closeWant qtyBase>0 径——SWAP 域 NULL 不同形）
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
		return id
	}
	if state == "filled" && px > 0 {
		r.slipSample(ctx, id, leg, px)
	}
	return id
}

// slipSample 成交腿滑点采样（T-0521 FinRL 镜照③——mark 当时锁存：
// md_ticks 状态表无史可回溯，历史滑点自此自产累积）。mark=(bid+ask)/2
// 缺一腿退 last；slip_bps 侧向化（buy=(px-mark)/mark，sell 反向——正=成本）。
// mark 行缺/价非正=样本仍落 mark NULL（覆盖缺口显形非虚构）；写败仅 rej
// 计数不阻腿径（采样非钱径）。mark_ts 携行——校准面按鲜度过滤。
func (r *Runner) slipSample(ctx context.Context, legID int64, leg Leg, px float64) {
	var mark, bid, ask, last float64
	var markTs time.Time
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(NULLIF((bid+ask)/2,0),last),COALESCE(bid,0),COALESCE(ask,0),COALESCE(last,0),ts
		FROM md_ticks WHERE venue=$1 AND inst_id=$2`, leg.Venue, leg.InstID).
		Scan(&mark, &bid, &ask, &last, &markTs)
	var slip *float64
	var spread *float64
	if err == nil && mark > 0 {
		s := (px - mark) / mark * 1e4
		if leg.Side == "sell" {
			s = -s
		}
		slip = &s
		if bid > 0 && ask > bid {
			sp := (ask - bid) / mark * 1e4
			spread = &sp
		}
	}
	var markTsArg *time.Time
	if !markTs.IsZero() {
		markTsArg = &markTs
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_slip_sample(leg_id,venue,inst_id,side,px,mark_px,mark_ts,slip_bps,spread_bps)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,0),$7,$8,$9)`,
		legID, leg.Venue, leg.InstID, leg.Side, px, mark, markTsArg, slip, spread); err != nil {
		r.rej.Inc("write_err")
	}
}

// legCash 腿级现金行（T-0083 对称记账）：okx 现货腿（非合约域 SWAP/datedFUT——
// T-0483）买=流出毛额负、卖=流入毛额正——两腿各记净额自然=真价差；合约腿无名义现金（结算走 bills/fee）。
// source_ref=腿行 id（幂等溯源锚）；perp/CFD/杂 venue 不触现径。
func (r *Runner) legCash(ctx context.Context, legID int64, leg Leg, px, fillSz float64, posRef *int64, fee float64, feeCcy string) {
	if legID == 0 || leg.Venue != "okx" || contractInst(leg.InstID) {
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
	// 工单#11438 费净额锚：venue 实返 fee 仅 feeCcy=USDT 触本账——卖腿净收=
	// 毛额+fee(负)，买腿费在 base 不触 USDT。kind='fee' 独立行=费归费账
	// （review/caliber 已汇此 kind；藏进 spot_* 毛额=账实差逐笔增生）。
	if fee != 0 && feeCcy == "USDT" {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_cashflow
			(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
			VALUES('fee',$1,'USDT',$2,$3,$4,'fin_order_leg',now())`,
			leg.Venue, fee, posRef, legID); err != nil {
			r.rej.Inc("write_err")
		}
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
