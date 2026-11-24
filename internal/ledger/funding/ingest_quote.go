// ingest_quote.go —— O-39310 计价币腿入账件（自 ingest.go 拆出守 300 行
// 棘轮）。SPOT/MARGIN 双币账单=对称两腿之计价币侧——venue 侧 bot/手动单
// 无对应簿记径永久漏记=守恒破口（fin_venue_bill 实证 410 行 USDT 腿零
// cashflow）；O 自执行单两径已记——legCash(fin_order_leg)∨
// residual_clear(fin_risk_decision.cl_ord_id)，判别防双计。
package funding

import (
	"context"
	"time"
)

// spotQuote SPOT/MARGIN 计价币（USDT/USD）成交腿入账，分量级判别：
//   - gross 域（spot_*/unmapped_bill）：clOrdId 命中 fin_order_leg 已记
//     腿（legCash）∨ fin_risk_decision cl_ord_id（residual_clear）=已覆盖
//     跳过；不命中=venue 侧活动落 spotNative 同式对称入账。
//   - fee 域：bill 费腿独立判别——legCash 仅 feeCcy=USDT 实记 fee 行，
//     residual_clear 不记 fee；同 clOrdId 无 fee 现金行时 bill fee 腿
//     是唯一账源照记（覆盖判定不连带 fee——gross 已记≠fee 已记）。
//
// clOrdId 缺载=无判别键保守跳过+rej 显形——漏记>双计（禁线：判别失败
// 侧宁可漏记不可裸放）。判别读败上抛（与全 ingest 同律——源行已落档，
// 重放 billId 幂等切面不重复写 cashflow=漏记方向安全）。
func (i *Ingester) spotQuote(ctx context.Context, b Bill, ccy string, srcID int64, occurred time.Time) (int, error) {
	if b.ClOrdID == "" {
		i.rej.Inc("clordid_missing")
		return 0, nil
	}
	var covered bool
	if err := i.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM fin_order_leg l
		JOIN fin_cashflow c ON c.source_table='fin_order_leg' AND c.source_ref=l.id
		WHERE l.venue='okx' AND l.cl_ord_id=$1)
		OR EXISTS(SELECT 1 FROM fin_risk_decision d
			WHERE d.intent->>'cl_ord_id'=$1)`, b.ClOrdID).Scan(&covered); err != nil {
		return 0, err
	}
	feeCovered := false
	if b.Fee != 0 {
		if err := i.pool.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM fin_order_leg l
			JOIN fin_cashflow c ON c.source_table='fin_order_leg' AND c.source_ref=l.id
				AND c.kind='fee'
			WHERE l.venue='okx' AND l.cl_ord_id=$1)`, b.ClOrdID).Scan(&feeCovered); err != nil {
			return 0, err
		}
	}
	if covered && (b.Fee == 0 || feeCovered) {
		return 0, nil
	}
	gross := b.BalChg - b.Fee
	kind := spotLegKind(b.InstID, ccy, gross)
	n := 0
	if !covered {
		if kind != "" && gross != 0 {
			if err := i.post(ctx, kind, gross, b, ccy, srcID, occurred); err != nil {
				return n, err
			}
			n++
		}
		if kind == "" { // 币对解不出=残差守恒兜底（净额真值不丢）
			if err := i.post(ctx, "unmapped_bill", b.BalChg, b, ccy, srcID, occurred); err != nil {
				return n, err
			}
			n++
		}
	}
	if b.Fee != 0 && !feeCovered { // 费腿独立判别——扣款币原生口径同 spotNative
		if err := i.post(ctx, "fee", b.Fee, b, ccy, srcID, occurred); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
