// maker.go —— T-0238 maker 径：post_only 挂价优先+时限降级 market 包装
// （自 flow.go 拆立——行棘轮）。
package execution

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// makerAttempt post_only 挂价优先径（T-0238）：限价挂盘口优价等
// exec.maker_timeout_s（缺键=20s）→时限未成取消+Resolve 定真态后降级
// market（ok=假——禁盲补单）；成=post_only 腿行记 ord_type。闸位：
// exec.maker_enabled 真 + okx 腿 + 非 close 腿 + md_ticks bid/ask 鲜(<30s)。
// 时限选型：20s 判据钦定——对冲裸奔窗受窗口界（单腿已成另腿未成>时限由
// leg2 降级市价补，不比 taker 原径慢太多）；改价追单禁绝（取消重写不追价）。
func (r *Runner) makerAttempt(ctx context.Context, execID int64, no int, leg Leg, posRef *int64) (OrderResult, bool) {
	if leg.Venue != "okx" || leg.Close {
		return OrderResult{}, false
	}
	on, err := r.getB("exec.maker_enabled")
	if err != nil || !on {
		return OrderResult{}, false
	}
	timeout := 20.0
	if v, err := r.cfg.GetFloat("exec.maker_timeout_s"); err == nil && v > 0 {
		timeout = v
	}
	var bid, ask float64
	if err := r.pool.QueryRow(ctx, `SELECT bid,ask FROM md_ticks WHERE inst_id=$1
		AND ts>now()-interval '30 seconds' ORDER BY ts DESC LIMIT 1`,
		leg.InstID).Scan(&bid, &ask); err != nil {
		return OrderResult{}, false // 盘口陈/缺=不挂 maker（鲜度 fail-closed）
	}
	px := bid
	if leg.Side == "sell" {
		px = ask
	}
	if px <= 0 {
		return OrderResult{}, false
	}
	ml := leg
	ml.Maker, ml.LimitPx = true, px
	if leg.TgtQuote { // spot 买 quote 名义→base 量（post_only 无 tgtCcy 方言）
		ml.TgtQuote = false
		ml.Sz = fmt.Sprintf("%.8f", leg.Notional/px)
	}
	clOrdID := fmt.Sprintf("mk%dx%dx%d", execID, no, time.Now().UnixNano()%1e6)
	if _, err := r.exec.Place(ctx, ml, clOrdID); err != nil {
		return OrderResult{}, false // post_only 拒/传输败=直接降级 market
	}
	deadline := time.Now().Add(time.Duration(timeout * float64(time.Second)))
	for {
		res, err := r.exec.Resolve(ctx, ml, clOrdID)
		if err == nil && res.State == "filled" {
			legID := r.legPx(ctx, execID, no, ml, clOrdID, "filled", "", "", res.FillPx, res.FillSz, "post_only")
			r.legCash(ctx, legID, ml, res.FillPx, res.FillSz, posRef)
			return res, true
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(500 * time.Millisecond) // sched-exempt-ok（maker 挂价期界等待——deadline 有界非周期调度）
	}
	_ = r.exec.Cancel(ctx, ml, clOrdID) // 撤果不判——定态只看 Resolve
	res, err := r.exec.Resolve(ctx, ml, clOrdID)
	if err == nil && res.State == "filled" { // 撤窗竞态成交
		legID := r.legPx(ctx, execID, no, ml, clOrdID, "filled", "", "", res.FillPx, res.FillSz, "post_only")
		r.legCash(ctx, legID, ml, res.FillPx, res.FillSz, posRef)
		return res, true
	}
	// 碎量径（审计驳回修①）：canceled+accFillSz>0=部分成交——实量记 post_only
	// 行+legCash 入账，State="partial" 回上级做余量降级（记 0 量补全=裸腿
	// 碎仓+台账吞量实证坑）。
	if res.FillSz > 0 {
		legID := r.legPx(ctx, execID, no, ml, clOrdID, "partial", "", "", res.FillPx, res.FillSz, "post_only")
		r.legCash(ctx, legID, ml, res.FillPx, res.FillSz, posRef)
		return OrderResult{ClOrdID: clOrdID, State: "partial", FillPx: res.FillPx, FillSz: res.FillSz}, true
	}
	r.legPx(ctx, execID, no, ml, clOrdID, "cancelled", "", "", 0, 0, "post_only")
	return OrderResult{}, false
}

// scaleLegRemainder 碎量余腿缩放（T-0238 驳回修①助手——maker 腿实量已录，
// 市价仅补余量：TgtQuote 腿按名义余比，合约腿按张余；FillSz 与 Sz 同量纲）。
func scaleLegRemainder(leg Leg, mres OrderResult) Leg {
	if leg.TgtQuote {
		leg.Notional -= mres.FillPx * mres.FillSz
		leg.Sz = fmt.Sprintf("%.8f", leg.Notional)
	} else if orig, err := strconv.ParseFloat(leg.Sz, 64); err == nil && orig > mres.FillSz {
		leg.Sz = fmt.Sprintf("%.8f", orig-mres.FillSz)
		leg.Notional *= (orig - mres.FillSz) / orig
	}
	return leg
}
