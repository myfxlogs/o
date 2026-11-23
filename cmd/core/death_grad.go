// death_grad.go —— 死刑梯度（业主令 2026-11-15「死刑真实性归医生裁」件）。
// hunger 探报饿不疼——差距=数字无后果，饿探火过而行为面照旧；
// 自然人饿=力气真在缩（饿到走不动），O 的「力气」=开仓能力。
// 本拍把真账年化翻成能力因子 lifecycle.death_factor（DMSet 正径
// ——参数调整账行在案，键域=dm 册 lifecycle.* 已登记）：
//
//	30d年化≥8%→1.0（饱=满力）
//	30d年化 0~8%→0.5（饿=半力）
//	30d年化<0→0.25（病——短窗回正=回暖留级）
//	7d&14d&30d 三窗皆负→0.10 临床底（非零——留一口气：小开仓
//	  复愈通道+paper/probe 学练径不受冻，归零=死锁无回头路）
//
// 因子落闸面 open 径总敞口阈（×risk.max_exposure_usd）——能力收缩
// 只减不增：无强平、平仓径不动（同 kill_switch 契），敞口自然衰减
// 向新上限。年化解负=同一公式对称自愈（0.10→0.25→0.5→1.0 不升件）。
// 触底另升「死刑/death_clinical」：机械层到此为止——本金收缩/拔管
// =业主物理件（越线三径归业主，医生不替病人拔管），本件只宣告不执行。
// NAV 缺/陈>48h 不判（同 hunger 契——权益盲时梯度无据）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const deathFloorFactor = 0.10 // 临床底——非零留复愈通道

// deathFactor 年化三窗→能力因子（纯函数——判据同 hunger 8% 成年线）。
func deathFactor(ann7, ann14, ann30 float64) float64 {
	switch {
	case ann30 >= hungerTargetPct:
		return 1.0
	case ann30 >= 0:
		return 0.5
	case ann7 < 0 && ann14 < 0:
		return deathFloorFactor
	default:
		return 0.25
	}
}

// deathGradCheck 死刑梯度拍——年化→因子→DMSet 闸键+触底宣告。
func (c coreService) deathGradCheck(ctx context.Context, deadSince map[string]time.Time) {
	var nav *float64
	var n7, n14, n30 float64
	if err := c.pool.QueryRow(ctx, `SELECT
		(SELECT equity FROM md_equity_curve WHERE ccy='NAV'
			AND ts>now()-interval '48 hours' ORDER BY ts DESC LIMIT 1),
		(SELECT COALESCE(sum(amount),0) FROM fin_pnl_net
			WHERE occurred_at>now()-interval '7 days'),
		(SELECT COALESCE(sum(amount),0) FROM fin_pnl_net
			WHERE occurred_at>now()-interval '14 days'),
		(SELECT COALESCE(sum(amount),0) FROM fin_pnl_net
			WHERE occurred_at>now()-interval '30 days')`).
		Scan(&nav, &n7, &n14, &n30); err != nil {
		return // 读件败本拍让位
	}
	if nav == nil || *nav <= 0 {
		return // 权益盲=梯度无据（nav_stale 探另报）
	}
	ann7 := n7 * 365 / 7 / *nav * 100
	ann14 := n14 * 365 / 14 / *nav * 100
	ann30 := n30 * 365 / 30 / *nav * 100
	target := deathFactor(ann7, ann14, ann30)
	if cur, err := c.cfg.GetFloat("lifecycle.death_factor"); err != nil || cur != target {
		ev := fmt.Sprintf("三窗年化 7d=%.2f%% 14d=%.2f%% 30d=%.2f%% NAV=%.0f（死刑梯度 tier→%.2f）",
			ann7, ann14, ann30, *nav, target)
		if _, err := c.cfg.DMSet(ctx, "lifecycle.death_factor",
			json.RawMessage(strconv.FormatFloat(target, 'f', -1, 64)), ev, "death-grad"); err != nil {
			return // 写败本拍让位（域闸拒=键漂移册漏登——不静默绕闸）
		}
	}
	c.escEdgeUV(ctx, deadSince, "死刑", "death_clinical",
		fmt.Sprintf("%.2f", target), target <= deathFloorFactor, func() string {
			return fmt.Sprintf(
				"死刑/death_clinical: 能力因子触底 %.2f（三窗年化 7d=%.2f%% 14d=%.2f%% 30d=%.2f%% NAV=%.0f——全负），敞口阈已缩至一成;机械层到此为止——本金收缩/拔管=业主物理件（越线三径归业主）,本升件只宣告不执行;年化解负=因子对称自愈",
				target, ann7, ann14, ann30, *nav)
		})
}
