// residual_clear.go —— 杂币清仓执行面（T-0439 ②③④ 落点，自 residual.go
// 拆立——行数棘轮 300 线纪律）：
//
//	升件两键：「杂币敞口」=白名单外非粉尘检出即案（首检升件幂等）；
//	「杂币清仓」=清仓败/歧态势定性上抛（同键节流=清仓径让位不补单）。
//	豁免册：venue 终拒不可卖残存落 fin_known_residual——审计锚+双径跳过。
//	清仓径：ClearLeg 复用（Close=true，spot cash 不携 reduceOnly——装配层
//	翻译 execution.Leg）——clOrdId 定真态三分支同 okx-exec 律：filled=
//	留痕记账；absent+终拒码=豁免册；余态（absent 他码/live/partial/
//	unknown/传输歧）=升件禁补单。
package earn

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"o/internal/ledger"
)

// exemptResidual 豁免册落行（幂等——同 venue+ccy 未过期行在不重落）：
// venue 终拒（51001/51003=标的不可交易）的杂币残存入册，30 天窗满重估
// （标的可交易状态可能变——窗满自然复检非永久盖布）；reason 携拒码=审计锚。
func (w *Worker) exemptResidual(ctx context.Context, ccy string, amt, usd float64, reason string) {
	res, err := w.pool.Exec(ctx, `INSERT INTO fin_known_residual(venue,ccy,amount_usd,reason,ticket_ref,expires_at)
		SELECT $1,$2,$3,$4,'T-0439',now()+interval '30 days'
		WHERE NOT EXISTS (SELECT 1 FROM fin_known_residual
			WHERE venue=$1 AND ccy=$2 AND expires_at>now())`,
		w.venue, ccy, usd, reason)
	if err != nil {
		w.rej.Inc("write_err")
		return
	}
	if res.RowsAffected() > 0 {
		w.rej.Inc("residual_exempted")
		log.Printf("[earn] 杂币豁免册落行 %.8f %s（≈$%.2f）——%s", amt, ccy, usd, reason)
	}
}

// exposureEscalated 「杂币敞口」首检升件（T-0439 ④——检出即案，幂等同键节流：
// note 前缀 venue+ccy 为键，已有行不重落=周期复扫不连鸣）。
func (w *Worker) exposureEscalated(ctx context.Context, ccy string, amt, usd float64) {
	res, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		SELECT '配管','杂币敞口','T-0439',$1
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='杂币敞口' AND note LIKE $2)`,
		fmt.Sprintf("%s %s 杂币敞口 %.8f≈$%.2f——白名单外未授权持仓首检升件（清仓径处置中）",
			w.venue, ccy, amt, usd),
		w.venue+" "+ccy+" %")
	if err != nil {
		w.rej.Inc("write_err")
		return
	}
	if res.RowsAffected() > 0 {
		w.rej.Inc("residual_exposure")
		w.al.Alert(ctx, "residual_exposure",
			fmt.Sprintf("%s 杂币敞口 %.8f≈$%.2f——白名单外未授权持仓，清仓径处置中", ccy, amt, usd))
		log.Printf("[earn] 杂币敞口升件 %.8f %s（≈$%.2f）", amt, ccy, usd)
	}
}

// escalateClearFail 「杂币清仓」升件行（幂等同键节流）+Alert——清仓败/歧态势
// 定性上抛人工；歧态势（unknown/live/partial）尤其禁补单（双卖险）。
func (w *Worker) escalateClearFail(ctx context.Context, ccy string, amt float64, state, detail string) {
	w.rej.Inc("residual_clear_failed")
	w.al.Alert(ctx, "residual_clear_failed",
		fmt.Sprintf("%s 清仓 %s（%.8f）——升件不补单，%s", ccy, state, amt, detail))
	if _, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		SELECT '配管','杂币清仓','T-0439',$1
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='杂币清仓' AND note LIKE $2)`,
		fmt.Sprintf("%s %s 清仓 %s %.8f——升件不补单（%s），同键节流待人工",
			w.venue, ccy, state, amt, detail),
		w.venue+" "+ccy+" %"); err != nil {
		w.rej.Inc("write_err")
	}
	log.Printf("ALERT [earn] %s %s 清仓升件 %s %.8f——不补单待人工（%s）", w.venue, ccy, state, amt, detail)
}

// clearResidual 杂币清仓：执行径现货市价卖 ccy→USDT（ClearLeg 复用——
// Close=true 语义位，spot cash tdMode 不携 reduceOnly）。定真态三分支：
// filled=留痕记账；absent+终拒码（51001/51003）=不可卖入豁免册；余态
// （absent 他码/live/partial/unknown/传输歧）=升件不补单同键节流。
func (w *Worker) clearResidual(ctx context.Context, ccy string, amt, usd float64) {
	if w.clr == nil {
		w.rej.Inc("clearer_absent")
		return // 执行缝未挂——升件行已落，人工处置
	}
	if !w.mu.TryLock() {
		w.rej.Inc("sweep_busy")
		return
	}
	defer w.mu.Unlock()
	inst := ccy + "-USDT"
	// clOrdId=大小写英数 only≤32（OKX 实拍 51000——T-0038）；升件不补单律下
	// 单发即终，幂等键无需跨拍复用。
	clOrdID := fmt.Sprintf("ec%d", time.Now().UnixNano())
	leg := ClearLeg{Venue: w.venue, InstID: inst, Side: "sell", Close: true,
		Sz: strconv.FormatFloat(amt, 'f', -1, 64)}
	res, err := w.clr.Place(ctx, leg, clOrdID)
	if err != nil {
		// 传输/装配败→clOrdId 定真态（okx-exec 三分支同律）；查询亦败=歧态禁补单。
		if rres, rerr := w.clr.Resolve(ctx, leg, clOrdID); rerr == nil {
			res = rres
		} else {
			res = ClearResult{ClOrdID: clOrdID, State: "unknown"}
		}
	}
	switch res.State {
	case "filled":
		w.afterClear(ctx, ccy, amt, res, clOrdID)
	case "absent":
		if res.VenueCode == "51001" || res.VenueCode == "51003" {
			// 标的不可交易终局拒=暂不可清残存——入豁免册（审计锚），双径跳过。
			w.exemptResidual(ctx, ccy, amt, usd,
				fmt.Sprintf("清仓 venue 终拒 %s %s——不可卖残存豁免（T-0439）", res.VenueCode, res.VenueMsg))
			return
		}
		w.escalateClearFail(ctx, ccy, amt, "absent",
			fmt.Sprintf("venue 拒 %s %s", res.VenueCode, res.VenueMsg))
	default: // unknown/live/partially_filled/canceled——歧态族一律升件禁补单
		w.escalateClearFail(ctx, ccy, amt, res.State, "歧态禁补单")
	}
}

// afterClear 清仓成交留痕：decision(transfer,op=residual_clear)+spot_sell
// 现金流行（USDT 实得=FillSz×FillPx 成交额锚；venue 实填缺=估值兜底+
// gross_fallback 显形）。写败不回滚钱动（ALERT+计数；对账环差即报兜底）。
func (w *Worker) afterClear(ctx context.Context, ccy string, amt float64, res ClearResult, clOrdID string) {
	gross := res.FillSz * res.FillPx
	if gross <= 0 {
		w.rej.Inc("gross_fallback")
		gross = valCcyOf(ctx, w.pool, ccy, amt)
	}
	intent, _ := json.Marshal(map[string]any{
		"kind": "transfer", "op": "residual_clear", "venue": w.venue,
		"ccy": ccy, "amt": amt, "cl_ord_id": clOrdID})
	var dID int64
	if err := w.pool.QueryRow(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES($1,'allow','transfer',$2) RETURNING id`, intent, w.cfg.Generation()).Scan(&dID); err != nil {
		w.rej.Inc("write_err")
		log.Printf("ALERT [earn] 杂币 %.8f %s 清仓成但 decision 留痕写败: %v", amt, ccy, err)
		return
	}
	if _, err := ledger.Post(ctx, w.pool, ledger.Entry{
		Kind: "spot_sell", Venue: w.venue, Ccy: w.ccy, Amount: gross,
		SourceRef: dID, SourceTable: "fin_risk_decision",
	}); err != nil {
		w.rej.Inc("write_err")
		log.Printf("ALERT [earn] 杂币 %.8f %s 清仓成但 cashflow 留痕写败（decision #%d 已在）: %v", amt, ccy, dID, err)
	}
	log.Printf("[earn] 杂币 %.8f %s 清仓成——USDT 实得 %.4f（clOrdId %s）", amt, ccy, gross, clOrdID)
}
