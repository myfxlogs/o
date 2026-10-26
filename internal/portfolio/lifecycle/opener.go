// opener.go —— live 策略开仓消费面（T-0050 首仓序④）：fin_strategy state='live'
// 且无未平仓行 → Opener 缝开仓（cmd 装配=gate.Check→BuildLegs→Runner.Execute
// 与 openfirst 同码径——投资域不反向依赖交易/风控域，guard.Executor 先例）。
// 幂等=fin_position 行（开成即有行=不再开）；事件驱（ev_strategy/ev_eval NOTIFY，
// T-0048 脊）零轮询。
package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// OpenSpec live 开仓规格（funding-hedge 模板：direction=short_perp_long_spot）。
type OpenSpec struct {
	StrategyRef int64
	Venue       string
	InstID      string // SWAP 键（BTC-USDT-SWAP）——现货腿装配层派生
	Direction   string
	NotionalUSD float64
}

// Opener 开仓执行缝（未接线=nil fail-closed 不开——禁无闸直开）。
type Opener interface {
	Open(ctx context.Context, spec OpenSpec) (execID int64, err error)
}

// AttachOpener 挂开仓缝（cmd 装配层接线）。
func (r *Runner) AttachOpener(o Opener) *Runner { r.op = o; return r }

// openLive live 策略开仓拍：无未平仓行的 live 行逐个过缝。败=计数+log 不炸
// 事件主径（下帧事件续试——ev_strategy 行变才触发，无热环）。
func (r *Runner) openLive(ctx context.Context) {
	if r.op == nil {
		return // 未接线=无消费面（cmd 未挂——禁开）
	}
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.direction, s.notional_usd, COALESCE(s.venue,'okx'), s.template_ref
		FROM fin_strategy s
		WHERE s.state='live' AND NOT EXISTS (
			SELECT 1 FROM fin_position p WHERE p.strategy_ref=s.id AND p.closed_at IS NULL)`)
	if err != nil {
		r.rej.Inc("write_err")
		return
	}
	type cand struct {
		spec OpenSpec
		tmpl string
	}
	var cs []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.spec.StrategyRef, &c.spec.InstID, &c.spec.Direction,
			&c.spec.NotionalUSD, &c.spec.Venue, &c.tmpl); err != nil {
			rows.Close()
			r.rej.Inc("write_err")
			return
		}
		cs = append(cs, c)
	}
	rows.Close()
	for _, c := range cs {
		// T-0144 armed→open 显式禁：xvenue_carry 执行原子性语义未建（两腿序+
		// 败一腿处置缺——T-0145 另票），抢跑=裸腿风险。永拒非瞬态（钉死）。
		if c.tmpl == "xvenue-carry" {
			r.rej.Inc("xvenue_open_forbidden")
			log.Printf("ALERT [lifecycle] 策略 %d %s armed→open 显式禁（xvenue-carry 双腿执行径未建——T-0145 另票）",
				c.spec.StrategyRef, c.spec.InstID)
			continue
		}
		// T-0103 ③同键振荡保险丝：1h 内开平轮次 ≥3=判据分裂振荡兜底——
		// 降级+告警，不再喂下一轮（#1662 每分钟开平刷费实证）。
		var rounds int
		if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_position
			WHERE strategy_ref=$1 AND opened_at>now()-interval '1 hour'`,
			c.spec.StrategyRef).Scan(&rounds); err != nil {
			r.rej.Inc("write_err")
			continue
		}
		cd := 300.0 // T-0136 ③拒放冷窗（缺省秒，lifecycle.reject_cooldown_s 热启）
		if v, err := r.cfg.GetFloat("lifecycle.reject_cooldown_s"); err == nil && v > 0 {
			cd = v
		}
		if r.denyAt != nil && time.Since(r.denyAt[c.spec.StrategyRef]) < time.Duration(cd*float64(time.Second)) {
			continue // T-0136 ③冷窗（窗内拒放出口前移——拒因出口全位写戳）
		}
		if rounds >= 3 {
			r.rej.Inc("oscillation_fuse")
			log.Printf("ALERT [lifecycle] 策略 %d %s 1h 内 %d 轮开平——振荡保险丝熔断降级", c.spec.StrategyRef, c.spec.InstID, rounds)
			if terr := Transition(ctx, r.pool, c.spec.StrategyRef, "degraded",
				"同键开平振荡保险丝——1h 内 ≥3 轮开平降级待复盘", false); terr != nil {
				r.rej.Inc("write_err")
				log.Printf("ALERT [lifecycle] 策略 %d 振荡降级败: %v", c.spec.StrategyRef, terr)
			}
			continue
		}
		// T-0103 ①开平判据对偶化：开仓评估含「预测持仓寿命 ≥ 最短窗」——
		// 平闸 funding 翻面判据（latest+next 双口径）开时同测：开即翻=
		// 寿命低于一个结算窗=开闸拒，fin_opp_eval 落 rejected/lifetime_below_min。
		if ok, why := r.lifetimeOK(ctx, c.spec); !ok {
			r.denyAt[c.spec.StrategyRef] = time.Now()
			log.Printf("[lifecycle] 策略 %d %s 开闸拒——%s", c.spec.StrategyRef, c.spec.InstID, why)
			continue
		}
		// T-0111 ①回本窗判据：预期 carry（持有上限内）≥ 往返摩擦×1.5 不满足
		// =拒开（fin_opp_eval rejected/breakeven_beyond_hold——「72h 开闸制
		// 度性亏损」根修；T-0103 对偶同位判）。
		if ok, why := r.breakevenOK(ctx, c.spec); !ok {
			r.denyAt[c.spec.StrategyRef] = time.Now()
			log.Printf("[lifecycle] 策略 %d %s 开闸拒——%s", c.spec.StrategyRef, c.spec.InstID, why)
			continue
		}
		eid, err := r.op.Open(ctx, c.spec)
		if err != nil {
			// T-0147 拦截前周期重试闸（防御纵深第二层）：开仓败命中黑名单
			// inst→直接退役不重试（防存量漏网形复发——黑名单在册=终局事实，
			// 周期续试=风暴复发温床）。
			var banned int
			if be := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_untradeable
				WHERE inst_id=$1`, c.spec.InstID).Scan(&banned); be == nil && banned > 0 {
				r.rej.Inc("blacklist_retire")
				if terr := Transition(ctx, r.pool, c.spec.StrategyRef, "retired",
					"venue_untradeable:blacklisted 开仓败命中黑名单——退役不重试", false); terr != nil {
					r.rej.Inc("write_err")
				}
				log.Printf("[lifecycle] 策略 %d %s 命中黑名单——退役不重试（终局）", c.spec.StrategyRef, c.spec.InstID)
				continue
			}
			r.rej.Inc("open_failed")
			log.Printf("[lifecycle] WARN 策略 %d 开仓败（%v exec_id=%d）——事件续试", c.spec.StrategyRef, err, eid)
			continue
		}
		log.Printf("[lifecycle] 策略 %d %s live 开仓成 exec_id=%d notional=%.2f",
			c.spec.StrategyRef, c.spec.InstID, eid, c.spec.NotionalUSD)
	}
}

// lifetimeOK 预测持仓寿命判（T-0103 ①）：funding-hedge=perp 空+现货多——
// perp 空收正 funding（sign=+1，long_perp_short_spot=-1）。平闸
// funding_flip/funding_flip_preempt 同口径（latest+next×sign<0 且 |率|>
// flip_bps）开时先测：开即注定被平=寿命<最短窗=拒。判据键缺/facts 缺
// =fail-closed 拒开（不批放振荡窗）。
func (r *Runner) lifetimeOK(ctx context.Context, spec OpenSpec) (bool, string) {
	flipBps, err := r.cfg.GetFloat("guard.flip_bps")
	if err != nil {
		r.rej.Inc("missing_param")
		return false, "guard.flip_bps 判据键缺 fail-closed"
	}
	sign := 1.0
	if spec.Direction == "long_perp_short_spot" {
		sign = -1
	}
	col := func(c string) (float64, bool) {
		var v *float64
		if err := r.pool.QueryRow(ctx, `SELECT `+c+` FROM md_funding
			WHERE inst_id=$1 AND `+c+` IS NOT NULL ORDER BY ts DESC LIMIT 1`,
			spec.InstID).Scan(&v); err != nil || v == nil {
			return 0, false
		}
		return *v, true
	}
	latest, hasL := col("funding_rate")
	next, hasN := col("next_funding_rate")
	if !hasL && !hasN {
		r.rej.Inc("facts_missing")
		return false, "funding facts 行缺 fail-closed"
	}
	if hasL && latest*sign < 0 && -latest*sign*10000 > flipBps {
		return r.rejectEval(ctx, spec, latest, next), fmt.Sprintf(
			"lifetime_below_min:latest funding %.6f 翻面（平闸同口径）", latest)
	}
	if hasN && next*sign < 0 && -next*sign*10000 > flipBps {
		return r.rejectEval(ctx, spec, latest, next), fmt.Sprintf(
			"lifetime_below_min:next funding %.6f 预测翻（平闸同口径）", next)
	}
	return true, ""
}

// breakevenOK 回本窗判据（T-0111 ①）：预期 carry=|funding 率|×结算次数
// （max_hold 窗内 8h 一结）×名义 ≥ 往返摩擦×安全系数——摩擦=4 腿×
// （taker+slippage）bps（research.funding.* 键组），安全系数
// guard.breakeven_safety（缺省 1.5）。键缺/率缺=fail-closed 拒开。
func (r *Runner) breakevenOK(ctx context.Context, spec OpenSpec) (bool, string) {
	maxH, err := r.cfg.GetFloat("guard.max_hold_hours")
	if err != nil {
		r.rej.Inc("missing_param")
		return false, "guard.max_hold_hours 判据键缺 fail-closed"
	}
	taker, err := r.cfg.GetFloat("research.funding.taker_fee_bps")
	if err != nil {
		r.rej.Inc("missing_param")
		return false, "research.funding.taker_fee_bps 判据键缺 fail-closed"
	}
	slip, err := r.cfg.GetFloat("research.funding.slippage_bps")
	if err != nil {
		r.rej.Inc("missing_param")
		return false, "research.funding.slippage_bps 判据键缺 fail-closed"
	}
	safety := 1.5
	if v, err := r.cfg.GetFloat("guard.breakeven_safety"); err == nil && v > 0 {
		safety = v
	}
	var rate *float64
	if err := r.pool.QueryRow(ctx, `SELECT funding_rate FROM md_funding
		WHERE inst_id=$1 AND funding_rate IS NOT NULL ORDER BY ts DESC LIMIT 1`,
		spec.InstID).Scan(&rate); err != nil || rate == nil {
		r.rej.Inc("facts_missing")
		return false, "funding 率行缺 fail-closed"
	}
	settles := maxH / 8
	if settles < 1 {
		settles = 1 // 不足一窗仍按一次结算计下限
	}
	carryBps := *rate * 10000 * settles // 预期 carry（bps）
	frictionBps := 4 * (taker + slip)   // 4 腿往返摩擦（bps）
	if carryBps < frictionBps*safety {
		if n := r.rejectEvalReason(ctx, spec, "breakeven_beyond_hold", map[string]any{
			"carry_bps": carryBps, "friction_bps": frictionBps, "safety": safety,
			"funding_rate": *rate, "settles": settles}); !n {
			return false, fmt.Sprintf("breakeven_beyond_hold:预期 carry %.2fbps<摩擦 %.2fbps×%.1f",
				carryBps, frictionBps, safety)
		}
		return false, "breakeven_beyond_hold"
	}
	return true, ""
}

// rejectEvalReason 通用拒卡落件（T-0111：reason 参数化——lifetime/回本窗同径
// 落卡幂等窗共位；rejectEval 保留原名签名收编到本件）。
func (r *Runner) rejectEvalReason(ctx context.Context, spec OpenSpec, reason string, ev map[string]any) bool {
	ev["reason"] = reason
	b, _ := json.Marshal(ev)
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE inst_id=$1 AND verdict='rejected'
		AND evidence->>'reason'=$2 AND ts>now()-interval '1 hour'`,
		spec.InstID, reason).Scan(&n); err != nil || n > 0 {
		return false
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES($1,'funding-hedge',$2,'rejected','[]',$3)
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		verdict=EXCLUDED.verdict,legs=EXCLUDED.legs,evidence=EXCLUDED.evidence`,
		spec.InstID, spec.Direction, b); err != nil {
		r.rej.Inc("write_err")
	}
	return false
}

// rejectEval 寿命拒落卡（T-0103：fin_opp_eval rejected+reason 机读面——
// 1h 幂等窗防振荡期行洪）。
func (r *Runner) rejectEval(ctx context.Context, spec OpenSpec, latest, next float64) bool {
	return r.rejectEvalReason(ctx, spec, "lifetime_below_min", map[string]any{
		"latest_funding": latest, "next_funding": next})
}
