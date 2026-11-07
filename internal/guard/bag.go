// bag.go —— 平仓残袋显形对拍（T-0419 ②③，钱径实证 2026-11-07 仓#33：
// 平腿单位错配只清 1.3%，1890 CORE 裸袋留 venue 被 Earn 自动收息、仓标
// closed 记全损——「closed 了但货还在」静默背离=delta 中性章程失守）。
//
// 时点钦定：ExecuteClose 成交后、markClosed 前必走——settle 读挂账列须先立。
// 逐平仓腿两真源对拍：
//
//	spot 腿  venue 实余（sizer 现读）×腿账差（开腿实得 closeWant−平腿实卖
//	         qty_base，平腿行链自 close exec）→残袋=min(实余,腿账差)；
//	         估值≥guard.bare_bag_min_usd 阈→fin_position.bare_bag_* 挂账
//	         +ops_event「裸袋」行+fin_holding 显形（未回资产=挂账非实现损）。
//	SWAP 腿  合约残仓（ContractPos>0）=敞口事故非资产——ops_event+Alert 升件，
//	         不挂账（衍生品残仓非可回资产）。
//
// 平腿行缺=facts 异常本腿跳判（实卖无真源不猜——禁推断处置）；sizer 读败=
// 本腿让位计数（平仓事实不动，显形下拍 venue 巡检/托管标兜底）。
package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// bareBags 平仓残袋对拍一拍（非阻塞——写败计数留痕不回滚已平事实）。
func (w *Warden) bareBags(ctx context.Context, p posRow, legs legSet, cls [2]CloseLeg, closeExecID int64) {
	minUSD, err := w.getF("guard.bare_bag_min_usd")
	if err != nil {
		w.rej.Inc("missing_param")
		w.note("missing_param:barebag", fmt.Sprintf("仓 %d guard.bare_bag_min_usd 读败——残袋对拍跳过: %v", p.id, err))
		return
	}
	// 平腿真源行（execution_id=closeExecID）——实卖量/在否双判据；腿行缺=异常本腿跳判。
	sold := map[string]float64{}
	filled := map[string]bool{}
	rows, err := w.pool.Query(ctx,
		`SELECT inst_id,COALESCE(qty_base,0),state FROM fin_order_leg WHERE execution_id=$1`, closeExecID)
	if err != nil {
		w.rej.Inc("facts_missing")
		return
	}
	for rows.Next() {
		var inst, st string
		var qb float64
		if err := rows.Scan(&inst, &qb, &st); err != nil {
			rows.Close()
			w.rej.Inc("facts_missing")
			return
		}
		if st == "filled" {
			filled[inst] = true
			sold[inst] = qb
		}
	}
	rows.Close()
	for _, cl := range cls {
		if !filled[cl.InstID] {
			w.rej.Inc("facts_missing")
			log.Printf("WARN [guard] 仓 %d 平腿 %s 行缺——残袋对拍本腿跳过", p.id, cl.InstID)
			continue
		}
		if cl.Venue == "mt5" || strings.HasSuffix(cl.InstID, "-SWAP") {
			// T-0145：mt5 CFD 残仓同 SWAP 律——合约残仓=敞口事故升件非资产挂账
			w.bareBagSwap(ctx, p, cl)
			continue
		}
		w.bareBagSpot(ctx, p, legs, cl, sold[cl.InstID], minUSD)
	}
}

// bareBagSwap 合约残仓显形——venue 净仓>0=平未净敞口事故（非资产不挂账）。
// T-0145：sizer 按腿 venue 路由（mt5 腿=Mt5Sizer；路由缺=显形让位计数）。
func (w *Warden) bareBagSwap(ctx context.Context, p posRow, cl CloseLeg) {
	szr := w.sizerFor(cl.Venue)
	if szr == nil {
		w.rej.Inc("facts_missing")
		log.Printf("WARN [guard] 仓 %d %s venue=%s 现读面缺——残仓显形让位", p.id, cl.InstID, cl.Venue)
		return
	}
	resid, err := szr.ContractPos(ctx, cl.InstID)
	if err != nil {
		w.rej.Inc("facts_missing")
		log.Printf("WARN [guard] 仓 %d %s 合约残仓现读败——本腿显形让位: %v", p.id, cl.InstID, err)
		return
	}
	if resid <= 0 {
		return
	}
	w.rej.Inc("bare_bag")
	w.al.Alert(ctx, "close_residual_perp",
		fmt.Sprintf("仓 %d %s 合约残仓 %.4f 张——平未净敞口人工介入", p.id, cl.InstID, resid))
	w.bagEvent(ctx, p, cl.InstID, fmt.Sprintf("合约残仓 %.4f 张——敞口升件非资产挂账", resid))
}

// bareBagSpot 现货残袋显形+挂账：残袋=min(venue 实余,max(0,开腿实得−平腿实卖))；
// 腿账不可定（SzKnown 缺）=实余全额计袋（保守显形不藏）。超阈→挂账三列+
// 事件行+台账标；碎尘（估<阈）不挂账但同式查——lotSz 粒度残量免噪。
func (w *Warden) bareBagSpot(ctx context.Context, p posRow, legs legSet, cl CloseLeg, soldQty, minUSD float64) {
	ccy := strings.SplitN(cl.InstID, "-", 2)[0]
	szr := w.sizerFor(cl.Venue)
	if szr == nil {
		w.rej.Inc("facts_missing")
		log.Printf("WARN [guard] 仓 %d %s venue=%s 现读面缺——残袋显形让位", p.id, ccy, cl.Venue)
		return
	}
	avail, err := szr.SpotAvailBal(ctx, cl.InstID)
	if err != nil {
		w.rej.Inc("facts_missing")
		log.Printf("WARN [guard] 仓 %d %s 现货实余现读败——本腿显形让位: %v", p.id, ccy, err)
		return
	}
	var bag float64
	if orig, ok := legs.of(cl.InstID); ok {
		if want, known := orig.closeWant(); known {
			if claim := want - soldQty; claim > 0 {
				bag = claim
			}
		} else {
			bag = avail // 腿账不可定=实余全额计袋（保守显形）
		}
	} else {
		bag = avail // 原腿集对不上平腿=契约外形态——实余全额计袋
	}
	if bag > avail {
		bag = avail
	}
	if bag <= 0 {
		return
	}
	usd := w.bagUSD(ctx, ccy, bag)
	if usd > 0 && usd < minUSD {
		return // 碎尘——lotSz 粒度残量免噪
	}
	w.rej.Inc("bare_bag")
	var usdArg *float64
	if usd > 0 {
		usdArg = &usd
	}
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_position SET bare_bag_ccy=$2,bare_bag_qty=$3,bare_bag_usd=$4 WHERE id=$1`,
		p.id, ccy, bag, usdArg); err != nil {
		w.rej.Inc("write_err")
	}
	w.bagEvent(ctx, p, cl.InstID,
		fmt.Sprintf("%s 残袋 %.8f≈$%.2f——挂账待处置（settled=现金净+挂账=实现额）", ccy, bag, usd))
	if blob, err := json.Marshal(map[string]any{"bare_bag_position": p.id, "detected": "close_recon"}); err == nil {
		if _, err := w.pool.Exec(ctx, `INSERT INTO fin_holding(venue,kind,ccy,amount,venue_ref,val_ref)
			VALUES($1,'spot_dust',$2,$3,'',$4)
			ON CONFLICT(venue,kind,ccy,venue_ref) DO UPDATE SET amount=EXCLUDED.amount`,
			cl.Venue, ccy, bag, blob); err != nil {
			w.rej.Inc("write_err")
		}
	}
	w.al.Alert(ctx, "bare_bag",
		fmt.Sprintf("仓 %d %s 残袋 %.8f≈$%.2f——平仓未回资产已挂账", p.id, ccy, bag, usd))
	log.Printf("[guard] 仓 %d %s 残袋 %.8f（≈$%.2f）——裸袋挂账显形", p.id, ccy, bag, usd)
}

// bagUSD 裸袋 USD 估值（md_ticks/mark_index 最新价——无价=0 不挂虚值，
// 裸袋仍显形挂币量）。
func (w *Warden) bagUSD(ctx context.Context, ccy string, qty float64) float64 {
	var px *float64
	_ = w.pool.QueryRow(ctx, `SELECT v FROM (SELECT last v,ts FROM md_ticks WHERE inst_id=$1
		UNION ALL SELECT px,ts FROM md_mark_index WHERE inst_id=$1) q
		ORDER BY ts DESC LIMIT 1`, ccy+"-USDT").Scan(&px)
	if px == nil || *px <= 0 {
		return 0
	}
	return qty * *px
}

// bagEvent 「裸袋」事件行（同仓同票同腿去重——重试拍不双落；append-only 簿）。
func (w *Warden) bagEvent(ctx context.Context, p posRow, inst, detail string) {
	if _, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		SELECT '仓巡','裸袋','T-0419',$1
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='裸袋' AND note LIKE $2)`,
		fmt.Sprintf("仓#%d %s %s", p.id, inst, detail),
		fmt.Sprintf("仓#%d %s %%", p.id, inst)); err != nil {
		w.rej.Inc("write_err")
	}
}
