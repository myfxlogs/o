// friction.go —— paper 摩擦记账+实绩回灌校准（T-0521 FinRL 镜照③：paper 记账
// 零摩擦=系统性高估纸面收益——live 入场/出场两腿 taker 费+滑点全实付。
// 回灌径：近 30d 已平 live 仓 Σ|fee cashflow|/Σnotional 测往返费率→
// lifecycle.friction_bps.<venue>（DMSet 正径——参数调整账行在案）。
// 扣点径：paper 策略开户即扣「估计往返摩擦」kind='fee' fill 一行——
// 全族共用同表同口径=同摩擦基准（fee 测量值+滑点缓冲；md_ticks 状态表
// 无史可回溯——缓冲标注 unmeasured_buffer 不虚构测量）。
package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const (
	// frictionDefaultBps 校准缺席兜底率（2腿×taker≈5bps×往返≈20bps 量级——
	// 标注 default 非测量值）。
	frictionDefaultBps = 20.0
	// frictionBufferBps 滑点/点差缓冲（md_ticks 状态表无史——滑点不可回测，
	// 固定缓冲入率 detail 明示 unmeasured）。
	frictionBufferBps = 5.0
	// frictionMinSamples 回灌最小样本（<3 仓不覆写——样本薄测出的率
	// 噪声大于信号，持旧值）。
	frictionMinSamples = 3
	// frictionCalEvery 校准节拍（每 6h 重测一次——费率缓变不用每拍跑）。
	frictionCalEvery = 6 * time.Hour
)

// frictionBps venue 往返摩擦率（bps）——校准值在册用校准值，缺=兜底率。
// 返回 (bps, src) src=calibrated|default（detail 留痕——审计可辨测/兜）。
func (r *Runner) frictionBps(venue string) (float64, string) {
	if v, err := r.cfg.GetFloat("lifecycle.friction_bps." + venue); err == nil && v > 0 {
		return v, "calibrated"
	}
	return frictionDefaultBps, "default"
}

// chargePaperFriction paper 开户摩擦扣点——一次性 kind='fee' fill（负额）：
// 口径=单程入场成本=往返率/2——paper 沙箱窗（168h）内只发生入场；出场费
// 归未来平仓（live 平仓时真 fee 实记），全程往返预支给短窗=错配定价
// （paper 窗短于 live 持仓量级，窗内可证伪的面=「入场成本能否摊回」）。
// 写败仅记日志不阻开户（记账行缺=paper 高估同旧态——不劣化）。
func (r *Runner) chargePaperFriction(ctx context.Context, strategyID int64, venue, inst string, notional float64) {
	bps, src := r.frictionBps(venue)
	amt := -notional * bps / 2 / 1e4
	var entered time.Time
	if err := r.pool.QueryRow(ctx, `SELECT entered_at FROM fin_strategy WHERE id=$1`,
		strategyID).Scan(&entered); err != nil {
		entered = time.Now() // 行缺兜底——扣点仍落（ts 属开仓时刻语义）
	}
	detail, _ := json.Marshal(map[string]any{
		"roundtrip_bps": bps, "src": src, "buffer_bps": frictionBufferBps,
		"scope": "entry_est", "unmeasured": "slip_spread_buffer"})
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
		VALUES($1,'fee',$2,$3,$4,$5)`, strategyID, inst, amt, detail, entered); err != nil {
		log.Printf("[lifecycle] #%d paper 摩擦扣点落账败（%s bps=%.1f）: %v", strategyID, venue, bps, err)
	}
}

// frictionCalibrate 实绩回灌校准（6h 拍）：近 30d 已平 live 仓 fee 流水聚合
// →往返费率=Σ|fee|/Σnotional×1e4+滑点缓冲→DMSet lifecycle.friction_bps.<venue>。
// 样本<minSamples 不覆写（薄样测率噪声>信号——持旧值不虚构）；venue='sim'
// 演习域剔除（演习费非真钱）。
func (r *Runner) frictionCalibrate(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT c.venue, sum(-c.amount), sum(p.notional_usd), count(DISTINCT p.id)
		FROM fin_cashflow c JOIN fin_position p ON p.id=c.position_ref
		WHERE c.kind='fee' AND c.venue<>'sim' AND c.venue<>''
		  AND p.closed_at IS NOT NULL AND p.opened_at>now()-interval '30 days'
		GROUP BY c.venue`)
	if err != nil {
		return err
	}
	type agg struct {
		venue string
		fee   float64
		ntl   float64
		n     int64
	}
	var as []agg
	for rows.Next() {
		var a agg
		if err := rows.Scan(&a.venue, &a.fee, &a.ntl, &a.n); err != nil {
			rows.Close()
			return err
		}
		as = append(as, a)
	}
	rows.Close()
	for _, a := range as {
		if a.n < frictionMinSamples || a.ntl <= 0 {
			continue
		}
		feeBps := a.fee / a.ntl * 1e4
		// 滑点分量：fin_slip_sample 自产历史（样本足→实测替代缓冲；
		// 不足=unmeasured 缓冲续用——空窗期不虚构测量）。仓均腿数实测
		// （单腿 slip×腿数=仓往返滑点）。
		slipBps := frictionBufferBps
		slipSrc := "unmeasured_buffer"
		var slipAvg, legsPerPos float64
		var slipN int64
		if err := r.pool.QueryRow(ctx, `SELECT COALESCE(avg(s.slip_bps),0), count(*)
			FROM fin_slip_sample s WHERE s.venue=$1 AND s.slip_bps IS NOT NULL
			AND s.ts>now()-interval '30 days'
			AND s.mark_ts IS NOT NULL AND s.ts-s.mark_ts<interval '120 seconds'`,
			a.venue).Scan(&slipAvg, &slipN); err == nil && slipN >= frictionMinSamples {
			if err := r.pool.QueryRow(ctx, `SELECT count(l.id)::float/GREATEST(count(DISTINCT p.id),1)
				FROM fin_order_leg l JOIN fin_execution e ON e.id=l.execution_id
				JOIN fin_position p ON p.execution_id=e.id
				WHERE l.venue=$1 AND l.state='filled' AND l.px IS NOT NULL
				AND p.closed_at IS NOT NULL AND p.opened_at>now()-interval '30 days'`,
				a.venue).Scan(&legsPerPos); err == nil && legsPerPos > 0 {
				slipBps = slipAvg * legsPerPos
				slipSrc = fmt.Sprintf("measured(n=%d legs/pos=%.2f)", slipN, legsPerPos)
			}
		}
		bps := feeBps + slipBps
		if bps < 1 {
			bps = 1 // 费率近零=maker/返佣域——1bps 地板防零摩擦虚构
		}
		val, _ := json.Marshal(bps)
		ev := fmt.Sprintf("30d 实绩回灌: n=%d仓 fee=$%.4f ntl=$%.2f→fee=%.2fbps+slip=%.2fbps(%s)",
			a.n, a.fee, a.ntl, feeBps, slipBps, slipSrc)
		res, err := r.cfg.DMSet(ctx, "lifecycle.friction_bps."+a.venue, val, ev, "T-0521")
		if err != nil {
			log.Printf("[lifecycle] friction_bps.%s 校准写败: %v", a.venue, err)
			continue
		}
		if res.Changed {
			log.Printf("[lifecycle] friction_bps.%s 校准 %s→%.2f（%s）", a.venue, string(res.Old), bps, ev)
		}
	}
	return nil
}
