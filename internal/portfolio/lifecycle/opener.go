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
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.direction, s.notional_usd, COALESCE(s.venue,'okx')
		FROM fin_strategy s
		WHERE s.state='live' AND NOT EXISTS (
			SELECT 1 FROM fin_position p WHERE p.strategy_ref=s.id AND p.closed_at IS NULL)`)
	if err != nil {
		r.rej.Inc("write_err")
		return
	}
	type cand struct {
		spec OpenSpec
	}
	var cs []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.spec.StrategyRef, &c.spec.InstID, &c.spec.Direction,
			&c.spec.NotionalUSD, &c.spec.Venue); err != nil {
			rows.Close()
			r.rej.Inc("write_err")
			return
		}
		cs = append(cs, c)
	}
	rows.Close()
	for _, c := range cs {
		// T-0103 ③同键振荡保险丝：1h 内开平轮次 ≥3=判据分裂振荡兜底——
		// 降级+告警，不再喂下一轮（#1662 每分钟开平刷费实证）。
		var rounds int
		if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_position
			WHERE strategy_ref=$1 AND opened_at>now()-interval '1 hour'`,
			c.spec.StrategyRef).Scan(&rounds); err != nil {
			r.rej.Inc("write_err")
			continue
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
			log.Printf("[lifecycle] 策略 %d %s 开闸拒——%s", c.spec.StrategyRef, c.spec.InstID, why)
			continue
		}
		eid, err := r.op.Open(ctx, c.spec)
		if err != nil {
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

// rejectEval 寿命拒落卡（T-0103：fin_opp_eval rejected+reason 机读面——
// 1h 幂等窗防振荡期行洪）。
func (r *Runner) rejectEval(ctx context.Context, spec OpenSpec, latest, next float64) bool {
	ev, _ := json.Marshal(map[string]any{
		"reason": "lifetime_below_min", "latest_funding": latest, "next_funding": next})
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE inst_id=$1 AND verdict='rejected'
		AND evidence->>'reason'='lifetime_below_min' AND ts>now()-interval '1 hour'`,
		spec.InstID).Scan(&n); err != nil || n > 0 {
		return false
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES($1,'funding-hedge',$2,'rejected','[]',$3)`,
		spec.InstID, spec.Direction, ev); err != nil {
		r.rej.Inc("write_err")
	}
	return false
}
