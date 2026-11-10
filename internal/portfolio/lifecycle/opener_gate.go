package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
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
