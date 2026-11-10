package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"o/internal/research/funding" // T-0297：breakevenOK 摩擦/年化同源（白名单边——评估判据一源律）
)

// opener_gate.go —— 开闸判词一族（T-0521 自 opener.go 拆出——棘轮基线）:
// lifetimeOK 寿命判+breakevenOK 回本窗判+rejectEval 拒件落卡幂等窗。

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
	// T-0521 读侧合流：fee/slip 分量实测值新鲜（≤30d——对齐测量窗）则替
	// 声明参数——判据消费产线实记事实（同 md_funding 真费率喂闸同构），
	// 非机器改判据层；fail/window 成本项实测面覆盖不到=声明值续守。
	// 换算：RoundTripBps fee 项=4×taker（双腿×开平）、slip 项=2×slip——
	// 分量除回单腿口径仍走 CostAPR 单源函数（自检 I 不破）。
	if fee, e := r.cfg.GetFloat("lifecycle.friction_fee_bps." + spec.Venue); e == nil && fee > 0 {
		if slip, e := r.cfg.GetFloat("lifecycle.friction_slip_bps." + spec.Venue); e == nil && slip >= 0 {
			if at, e := r.cfg.GetFloat("lifecycle.friction_at." + spec.Venue); e == nil &&
				time.Since(time.Unix(int64(at), 0)) < 30*24*time.Hour {
				p.TakerFeeBps, p.MakerFeeSet = fee/4, false
				p.SlippageBps = slip / 2
			}
		}
	}
	var maxH float64
	if maxH, err = get("guard.max_hold_hours"); err != nil {
		return false, err.Error()
	}
	// T-0521 ①摊销窗实测定案：funding_flip 平闸在案——真实持仓寿命=收租
	// 号连续段长（产线实测全局中位 0.7d），不是 max_hold 声明窗；72h 摊
	// 销=乐观 4 倍定价病灶。分母=min(声明上限,该 inst 收租段中位 30d)。
	// 史缺=不虚构摊销窗 fail-closed（该 inst 30d 无该号收租段=本来就摊
	// 不开——拒开是测量结论非惩罚）。右删失段（在飞段）计入=保守向偏。
	sign := 1.0
	if strings.HasPrefix(spec.Direction, "long_perp") {
		sign = -1
	}
	measH, merr := r.measuredHoldDays(ctx, spec.InstID, sign)
	if merr != nil || measH <= 0 {
		return false, fmt.Sprintf("持仓段史缺（%s 30d 无收租号段——摊销窗不虚构 fail-closed）", spec.InstID)
	}
	p.HoldDays = math.Min(maxH/24, measH)
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
			"funding_rate": *rate, "hold_days": p.HoldDays, "measured_hold": measH}); !n {
			return false, fmt.Sprintf("breakeven_beyond_hold:毛收 %.4f<摩擦年化 %.4f×%.1f",
				grossAPR, costAPR, safety)
		}
		return false, "breakeven_beyond_hold"
	}
	return true, ""
}

// measuredHoldDays 实测持仓窗（T-0521 ①）：md_funding 该 inst 近 30d
// sign 号收租连续段的天数中位——正号段=short_perp 收租期,负号段=
// long_perp 收租期。sign(funding_rate)=0 行按 0 号分段（中断两侧段）。
// 段长=段内 max−min+一个结算窗（0.334d≈8h——末行后仍收一窗租）。
// 该 inst 无 sign 号段→0=facts 缺（caller fail-closed,不虚构摊销窗）。
func (r *Runner) measuredHoldDays(ctx context.Context, inst string, sign float64) (float64, error) {
	var d *float64
	err := r.pool.QueryRow(ctx, `WITH s AS (
		SELECT ts,funding_rate,LAG(funding_rate) OVER (ORDER BY ts) prev
		FROM md_funding WHERE inst_id=$1 AND ts>now()-interval '30 days'),
	r AS (SELECT ts,funding_rate,
		SUM(CASE WHEN sign(funding_rate)<>sign(COALESCE(prev,0)) THEN 1 ELSE 0 END)
		OVER (ORDER BY ts) rid FROM s),
	st AS (SELECT EXTRACT(EPOCH FROM (max(ts)-min(ts)))/86400.0+0.334 d,
		sign(max(funding_rate)) sg FROM r GROUP BY rid)
	SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY d) FROM st WHERE sg=$2`,
		inst, sign).Scan(&d)
	if err != nil {
		return 0, err
	}
	if d == nil {
		return 0, nil
	}
	return *d, nil
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
