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
	"strings"
	"time"

	"o/internal/platform/execfam"
	"o/internal/research/funding" // T-0297：breakevenOK 摩擦/年化同源（白名单边——评估判据一源律）
)

// OpenSpec live 开仓规格（funding-hedge 模板：direction=short_perp_long_spot）。
type OpenSpec struct {
	StrategyRef int64
	Venue       string
	InstID      string // SWAP 键（BTC-USDT-SWAP）——现货腿装配层派生
	Direction   string
	NotionalUSD float64
	Template    string // 模板族（T-0160：dcd_call 走 DCD 执行面非双腿原子径）
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
	}
	var cs []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.spec.StrategyRef, &c.spec.InstID, &c.spec.Direction,
			&c.spec.NotionalUSD, &c.spec.Venue, &c.spec.Template); err != nil {
			rows.Close()
			r.rej.Inc("write_err")
			return
		}
		cs = append(cs, c)
	}
	rows.Close()
	gate := r.loadCellGate(ctx) // T-0461 armed→open 格凭据闸（判位=r.op.Open 前）
	for i, c := range cs {
		if i > 0 { // T-0208②：venue 限速节流——quote+trade 双调用扇出 1s/策略拍（429 洪泛根修；sched-exempt-ok 执行节流非轮询）
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second): // sched-exempt-ok：venue 限速节流等待（非轮询——429 洪泛根修）
			}
		}
		// T-0145：xvenue-carry 副腿行（venue='mt5'）永不独立开仓——同仓由主行
		// 一次建成（独立开=裸腿事故源）；跳过计数静默非异常。主行落缝。
		if c.spec.Template == "xvenue-carry" && c.spec.Venue != "okx" {
			r.rej.Inc("xvenue_sub_leg")
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
		// T-0202：funding 闸按模板域生效——lifetimeOK/breakevenOK 钦定语义
		// =funding-hedge 系（perp 空+现货多家族）专属；dcd_call 等无 funding
		// 腿模板跳闸（模板域错配=全谱 live 零开仓病灶根治）。
		// T-0392：族属判据归并 execfam.FundingHedge 唯一真源（行情腿派生臂
		// /覆盖探针同册——本地清单删除防双源漂移）。
		if execfam.FundingHedge(c.spec.Template) {
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
			// T-0145 豁免：xvenue-carry 跳本闸——成本模型=okx 双腿专用，跨所
			// eval 已四件分解过阈，重复闸=双计成本误杀；funding 翻面判仍守。
			if c.spec.Template != "xvenue-carry" {
				if ok, why := r.breakevenOK(ctx, c.spec); !ok {
					r.denyAt[c.spec.StrategyRef] = time.Now()
					log.Printf("[lifecycle] 策略 %d %s 开闸拒——%s", c.spec.StrategyRef, c.spec.InstID, why)
					continue
				}
			}
		}
		if !gate.allow(r, c.spec) { // T-0461 格凭据未绿 hold
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
		r.noteLiveOpen(ctx, c.spec, eid) // T-0415：live 开仓=open 决策行（decide.go）
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
	// T-0145：向位前缀判（long_perp_* 含 cfd 变体——perp 多=收负 funding 方
	// sign=-1；等值串判=cfd 向位漏判=符号反置错拒/错放）。
	if strings.HasPrefix(spec.Direction, "long_perp") {
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

// breakevenOK 回本窗判据（T-0111 ①；T-0297 同源根修）：毛收年化=
// funding 率×日结算次数×365 ≥ 往返摩擦年化×安全系数——摩擦/年化摊销
// 调 funding 包 RoundTripBps/CostAPR 同一函数（评估卡展示值与开闸判据
// 同源——自检 I），不再本地算式。摩擦含四次成交全谱（2×双腿费+2×滑点+
// 失败成本+窗口损）；安全系数 guard.breakeven_safety（缺省 1.5）。
// 键缺/率缺=fail-closed 拒开。
func (r *Runner) breakevenOK(ctx context.Context, spec OpenSpec) (bool, string) {
	p := funding.Params{}
	var err error
	get := func(k string) (float64, error) {
		v, e := r.cfg.GetFloat(k)
		if e != nil {
			r.rej.Inc("missing_param")
			return 0, fmt.Errorf("%s 判据键缺 fail-closed: %w", k, e)
		}
		return v, nil
	}
	if p.TakerFeeBps, err = get("research.funding.taker_fee_bps"); err != nil {
		return false, err.Error()
	}
	if v, merr := r.cfg.GetFloat("research.funding.maker_fee_bps"); merr == nil { // 软键——缺=双 taker 回退（T-0247 分层）
		p.MakerFeeBps, p.MakerFeeSet = v, true
	}
	if p.SlippageBps, err = get("research.funding.slippage_bps"); err != nil {
		return false, err.Error()
	}
	if p.FailCostBps, err = get("research.funding.fail_cost_bps"); err != nil {
		return false, err.Error()
	}
	if p.WindowLossBps, err = get("research.funding.window_loss_bps"); err != nil {
		return false, err.Error()
	}
	if p.SettlesPerDay, err = get("research.funding.settles_per_day"); err != nil {
		return false, err.Error()
	}
	var maxH float64
	if maxH, err = get("guard.max_hold_hours"); err != nil {
		return false, err.Error()
	}
	p.HoldDays = maxH / 24 // 与评估器同一时间地平线一源
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
	grossAPR := *rate * p.SettlesPerDay * 365 // 毛收年化（保留号——负率仍拒）
	costAPR := funding.CostAPR(p)
	if grossAPR < costAPR*safety {
		if n := r.rejectEvalReason(ctx, spec, "breakeven_beyond_hold", map[string]any{
			"cost_apr": costAPR, "gross_apr": grossAPR, "safety": safety,
			"funding_rate": *rate, "hold_days": p.HoldDays}); !n {
			return false, fmt.Sprintf("breakeven_beyond_hold:毛收 %.4f<摩擦年化 %.4f×%.1f",
				grossAPR, costAPR, safety)
		}
		return false, "breakeven_beyond_hold"
	}
	return true, ""
}

// rejectEvalReason 通用拒卡落件（T-0111：reason 参数化——lifetime/回本窗同径
// 落卡幂等窗共位；rejectEval 保留原名签名收编到本件。T-0167/D-0032 §三-3：
// 拒件落 fin_eval_reject 拆表——ID 空间与 fin_opp_eval 正本隔离，桥镜像对
// SG 严格只读前提成立；reason 升显式列，1h 幂等窗语义原样）。
func (r *Runner) rejectEvalReason(ctx context.Context, spec OpenSpec, reason string, ev map[string]any) bool {
	ev["reason"] = reason
	b, _ := json.Marshal(ev)
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_eval_reject
		WHERE inst_id=$1 AND reason=$2 AND ts>now()-interval '1 hour'`,
		spec.InstID, reason).Scan(&n); err != nil || n > 0 {
		return false
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_eval_reject(inst_id,template,direction,reason,evidence)
		VALUES($1,'funding-hedge',$2,$3,$4)
		ON CONFLICT (inst_id,template,direction,reason) DO UPDATE SET ts=now(),
		evidence=EXCLUDED.evidence`,
		spec.InstID, spec.Direction, reason, b); err != nil {
		r.rej.Inc("write_err")
	}
	return false
}

// rejectEval 寿命拒落卡（T-0103：拒因机读面——T-0167 起落 fin_eval_reject，
// 1h 幂等窗防振荡期行洪）。
func (r *Runner) rejectEval(ctx context.Context, spec OpenSpec, latest, next float64) bool {
	return r.rejectEvalReason(ctx, spec, "lifetime_below_min", map[string]any{
		"latest_funding": latest, "next_funding": next})
}
