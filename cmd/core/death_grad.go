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
// 年化分母=部署本金窗时加权均值（令#34702③——NAV 基部署率低时
// 失真）；分子=fin_pnl_net 净产单源（O-34619）。分母缺腿/零部署
// 窗不判（缺证不判——同 NAV 盲律旧约）。
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
// 年化分母=部署本金窗时加权均值（令#34702③——NAV 基部署率低时
// 失真）；分子=fin_pnl_net 不动。任一窗分母不可得/零部署→本拍
// 让位（缺证不判——同 nav 盲律）。
func (c coreService) deathGradCheck(ctx context.Context, deadSince map[string]time.Time) {
	var n7, n14, n30 float64
	if err := c.pool.QueryRow(ctx, `SELECT
		(SELECT COALESCE(sum(amount),0) FROM fin_pnl_net
			WHERE occurred_at>now()-interval '7 days'),
		(SELECT COALESCE(sum(amount),0) FROM fin_pnl_net
			WHERE occurred_at>now()-interval '14 days'),
		(SELECT COALESCE(sum(amount),0) FROM fin_pnl_net
			WHERE occurred_at>now()-interval '30 days')`).
		Scan(&n7, &n14, &n30); err != nil {
		return // 读件败本拍让位
	}
	now := time.Now()
	dep7, ok7, snap7 := c.deployedBasis(ctx, now.Add(-7*24*time.Hour), now)
	dep14, ok14, snap14 := c.deployedBasis(ctx, now.Add(-14*24*time.Hour), now)
	dep30, ok30, snap30 := c.deployedBasis(ctx, now.Add(-30*24*time.Hour), now)
	if !ok7 || !ok14 || !ok30 {
		return // 分母缺腿=梯度无据
	}
	ann7, okA7 := annualizedPct(n7, 7, dep7)
	ann14, okA14 := annualizedPct(n14, 14, dep14)
	ann30, okA30 := annualizedPct(n30, 30, dep30)
	if !okA7 || !okA14 || !okA30 {
		return // 零部署窗=年化无义——不虚计
	}
	target := deathFactor(ann7, ann14, ann30)
	depTag := annualDepTag(dep7, ok7, snap7) + "/" +
		annualDepTag(dep14, ok14, snap14) + "/" + annualDepTag(dep30, ok30, snap30)
	if cur, err := c.cfg.GetFloat("lifecycle.death_factor"); err != nil || cur != target {
		ev := fmt.Sprintf("三窗年化 7d=%.2f%% 14d=%.2f%% 30d=%.2f%% %s（死刑梯度 tier→%.2f）",
			ann7, ann14, ann30, depTag, target)
		if _, err := c.cfg.DMSet(ctx, "lifecycle.death_factor",
			json.RawMessage(strconv.FormatFloat(target, 'f', -1, 64)), ev, "death-grad"); err != nil {
			return // 写败本拍让位（域闸拒=键漂移册漏登——不静默绕闸）
		}
	}
	c.escEdgeUV(ctx, deadSince, "死刑", "death_clinical",
		fmt.Sprintf("%.2f", target), target <= deathFloorFactor, func() string {
			return fmt.Sprintf(
				"死刑/death_clinical: 能力因子触底 %.2f（三窗年化 7d=%.2f%% 14d=%.2f%% 30d=%.2f%% %s——全负），敞口阈已缩至一成;机械层到此为止——本金收缩/拔管=业主物理件（越线三径归业主）,本升件只宣告不执行;年化解负=因子对称自愈",
				target, ann7, ann14, ann30, depTag)
		})
}
