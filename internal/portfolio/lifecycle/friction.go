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

// frictionCalibrate 实绩回灌校准（6h 拍）：近 30d 已平 live 仓腿费聚合
// →往返费率=Σ|fee|/Σnotional×1e4+滑点缓冲→DMSet lifecycle.friction_bps.<venue>。
// 样本<minSamples 不覆写（薄样测率噪声>信号——持旧值不虚构）；venue='sim'
// 演习域剔除（演习费非真钱）。
func (r *Runner) frictionCalibrate(ctx context.Context) error {
	// 工单#21129 根修：分子从 cashflow JOIN 换腿级测量列——cashflow 口径
	// 三洞（基币费不入账/bill 挂账跳过/position_ref NULL 排除）漏 ~80%。
	// leg 存 venue 返费原值（fee+fee_ccy，fin_0204 起含回填）——按仓归集
	// 精确（execution_id 1:1）。口径=全腿已测仓（unmeasured=0：任一腿缺
	// 费=该仓测量不全禁入分子分母——宁缺不混），非 USDT 费按腿 px 折
	// USDT；分子分母同集配对。taker/maker 腿数比注入口径（maker=post_only）。
	rows, err := r.pool.Query(ctx, `WITH pos AS (
		SELECT p.id, p.execution_id, p.notional_usd
		FROM fin_position p
		WHERE p.closed_at IS NOT NULL AND p.opened_at > now() - interval '30 days'
	), legf AS (
		SELECT p.id AS pid, max(l.venue) AS venue,
			sum(CASE WHEN COALESCE(l.fee_ccy,'USDT')='USDT' THEN -l.fee
			         ELSE -l.fee * l.px END) AS fee_usd,
			count(*) FILTER (WHERE l.fee IS NULL) AS unmeasured,
			count(*) FILTER (WHERE l.fee IS NOT NULL
			                 AND COALESCE(l.fee_ccy,'USDT')<>'USDT'
			                 AND (l.px IS NULL OR l.px <= 0)) AS nopx,
			count(*) FILTER (WHERE COALESCE(l.ord_type,'market')='market') AS taker,
			count(*) AS legs
		FROM pos p JOIN fin_order_leg l ON l.execution_id = p.execution_id AND l.state='filled'
		GROUP BY p.id
	)
	SELECT l.venue, sum(l.fee_usd), sum(p.notional_usd), count(*),
		sum(l.taker)::bigint, sum(l.legs)::bigint
	FROM pos p JOIN legf l ON l.pid = p.id
	WHERE l.unmeasured = 0 AND l.nopx = 0 AND l.venue <> 'sim' AND l.venue <> ''
	GROUP BY l.venue`)
	if err != nil {
		return err
	}
	type agg struct {
		venue string
		fee   float64
		ntl   float64
		n     int64
		taker int64
		legs  int64
	}
	var as []agg
	for rows.Next() {
		var a agg
		if err := rows.Scan(&a.venue, &a.fee, &a.ntl, &a.n, &a.taker, &a.legs); err != nil {
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
		ev := fmt.Sprintf("30d 实绩回灌: n=%d仓 fee=$%.4f ntl=$%.2f→fee=%.2fbps+slip=%.2fbps(%s) 口径=taker %d/%d腿",
			a.n, a.fee, a.ntl, feeBps, slipBps, slipSrc, a.taker, a.legs)
		// T-0521 读侧合流分量键：fee/slip 分记+at 鲜度戳——breakevenOK
		// 判词实测分量替换声明参数的同源消费面（判据读事实非机器改判据）。
		for _, kv := range [][2]any{
			{"lifecycle.friction_bps." + a.venue, bps},
			{"lifecycle.friction_fee_bps." + a.venue, feeBps},
			{"lifecycle.friction_slip_bps." + a.venue, slipBps},
			{"lifecycle.friction_at." + a.venue, float64(time.Now().Unix())},
		} {
			key, v := kv[0].(string), kv[1].(float64)
			val, _ := json.Marshal(v)
			res, err := r.cfg.DMSet(ctx, key, val, ev, "T-0521")
			if err != nil {
				log.Printf("[lifecycle] %s 校准写败: %v", key, err)
				continue
			}
			if res.Changed {
				log.Printf("[lifecycle] %s 校准 %s→%.2f（%s）", key, string(res.Old), v, ev)
			}
		}
	}
	return nil
}
