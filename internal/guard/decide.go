// decide.go —— 平仓触发判序+决策记忆写件（T-0415 自 guard.go/facts.go 拆立：
// evaluate 携判据+实况（reason 人读串+facts 机读载荷）供 tend/markClosed；
// 平仓决策行=红桩①钦定件——「为什么平」从此可问）。
package guard

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"o/internal/platform/decision"
)

// closeTrig 平仓触发件（kind=触发词留痕沿用；reason=触发判据+实况人读串；
// facts=机读载荷并入 ops_decision.payload）。
type closeTrig struct {
	kind   string
	reason string
	facts  map[string]any
}

// evaluate 触发判序（单仓单触发，先到先平）：funding 翻面→delta 漂移→持仓超龄
// →策略降级/退役联动。判据键缺=fail-closed 跳过+计数；事实行缺=计数不误判。
func (w *Warden) evaluate(ctx context.Context, p posRow, legs legSet) closeTrig {
	// ① funding 翻面：最新 funding 与开仓方向异号且超 guard.flip_bps。
	if flipBps, err := w.getF("guard.flip_bps"); err != nil {
		w.rej.Inc("missing_param")
		w.note("missing_param:flip", fmt.Sprintf("仓 %d guard.flip_bps 读败——判据链跳过: %v", p.id, err))
	} else if rate, ok := w.latestFunding(ctx, p.inst); !ok {
		w.rej.Inc("facts_missing")
		w.note("facts_missing:funding", fmt.Sprintf("仓 %d 无 funding 事实行——翻面判据跳过", p.id))
	} else if sign, ok := perpSign(legs, p.inst); !ok {
		w.rej.Inc("leg_anomaly")
		w.note("leg_anomaly:perpSign", fmt.Sprintf("仓 %d perp 侧不可辨——翻面判据跳过", p.id))
	} else if rate*sign < 0 && math.Abs(rate)*10000 > flipBps {
		return closeTrig{kind: "funding_flip",
			reason: fmt.Sprintf("funding_flip：latest=%.6f×sign=%.0f |率|%.1fbps>flip_bps=%.0f",
				rate, sign, math.Abs(rate)*10000, flipBps),
			facts: map[string]any{"funding_rate": rate, "perp_sign": sign, "flip_bps": flipBps}}
	} else if nr, ok := w.nextFunding(ctx, p.inst); ok && nr*sign < 0 && math.Abs(nr)*10000 > flipBps {
		return closeTrig{kind: "funding_flip_preempt", // T-0084：预测负值前置平——不等实收先交一刀
			reason: fmt.Sprintf("funding_flip_preempt：next=%.6f×sign=%.0f |率|%.1fbps>flip_bps=%.0f",
				nr, sign, math.Abs(nr)*10000, flipBps),
			facts: map[string]any{"next_funding_rate": nr, "perp_sign": sign, "flip_bps": flipBps}}
	}
	// ② delta 漂移（T-0064 语义正名）：漂移=对**开时基线**的偏移非绝对失衡——
	// |imb_now − imb_open| > band（出生粒度残差不杀，canon
	// lotsz-granularity-vs-hedge-band 裁决案）；双闸：imb_now>500bps 绝对
	// 兜底（防 sizer 失守放裸腿级失衡）。基线=NULL=首拍实算落列（0033 列）。
	if bandBps, err := w.getF("exec.hedge_band_bps"); err != nil {
		w.rej.Inc("missing_param")
		w.note("missing_param:band", fmt.Sprintf("仓 %d exec.hedge_band_bps 读败——漂移判据链跳过: %v", p.id, err))
	} else if legs.one {
		w.rej.Inc("leg_anomaly")
		w.note("leg_anomaly:legs", fmt.Sprintf("仓 %d 腿形态异常——漂移判据跳过", p.id))
	} else if n1, ok1 := w.notionalNow(ctx, legs.a); !ok1 {
		w.rej.Inc("facts_missing")
		w.note("facts_missing:notional", fmt.Sprintf("仓 %d 腿 %s 名义读败（tick/ctVal 缺）——漂移判据跳过", p.id, legs.a.inst))
	} else if n2, ok2 := w.notionalNow(ctx, legs.b); !ok2 {
		w.rej.Inc("facts_missing")
		w.note("facts_missing:notional", fmt.Sprintf("仓 %d 腿 %s 名义读败（tick/ctVal 缺）——漂移判据跳过", p.id, legs.b.inst))
	} else if avg := (n1 + n2) / 2; avg > 0 {
		if imb := (n1 - n2) / avg * 10000; w.driftTriggered(ctx, p.id, imb, bandBps) {
			return closeTrig{kind: "delta_drift",
				reason: fmt.Sprintf("delta_drift：imb_now=%.1fbps band=%.0fbps（对开时基线偏移；|imb|>500 绝对兜底同闸）",
					imb, bandBps),
				facts: map[string]any{"imb_now_bps": imb, "band_bps": bandBps, "abs_cap_bps": driftAbsCapBps}}
		}
	}
	// ③ 持仓超龄：age > guard.max_hold_hours（T-0111：收益失效判——live 且
	// trailing 24h 实收 funding>0 的仓免超龄闸；无收入仓保留年龄闸——僵尸
	// 杀手机制不丢，盈利 carry 仓不再制度性误杀）。
	if maxH, err := w.getF("guard.max_hold_hours"); err != nil {
		w.rej.Inc("missing_param")
		w.note("missing_param:maxhold", fmt.Sprintf("仓 %d guard.max_hold_hours 读败——超龄判据跳过: %v", p.id, err))
	} else if age := time.Since(p.opened).Hours(); age > maxH {
		if w.trailingIncomePositive(ctx, p.id) {
			w.note("max_hold_exempt", fmt.Sprintf("仓 %d 超龄但 trailing 24h 实收>0——收益失效判豁免", p.id))
		} else {
			return closeTrig{kind: "max_hold",
				reason: fmt.Sprintf("max_hold：age=%.1fh>max_hold_hours=%.0f（trailing 24h 实收≤0 无豁免）", age, maxH),
				facts:  map[string]any{"age_hours": age, "max_hold_hours": maxH}}
		}
	}
	// ④ 策略降级/退役联动（状态机事实直读，无键面——degraded/retired 必平）。
	switch p.stratState {
	case "degraded":
		return closeTrig{kind: "strategy_degraded",
			reason: "strategy_degraded：策略降级联动必平",
			facts:  map[string]any{"strategy_state": p.stratState, "strategy_ref": p.stratRef}}
	case "retired":
		return closeTrig{kind: "strategy_retired",
			reason: "strategy_retired：策略退役联动必平",
			facts:  map[string]any{"strategy_state": p.stratState, "strategy_ref": p.stratRef}}
	}
	w.note("no_trigger", fmt.Sprintf("仓 %d 全判据无触发——无触发也是决策（可观测）", p.id))
	return closeTrig{}
}

// writeCloseDecision 平仓决策记忆行（红桩①钦定件——态迁提交后调用：
// 触发判据+实况（reason）+实现的vs期望的（payload expected_net_apr/
// realized_usd/expected_carry_usd）；记忆写不阻钱径——写败升件自报非
// 回滚已平事实）。
func (w *Warden) writeCloseDecision(ctx context.Context, p posRow, closeExecID int64, trig closeTrig) {
	refID := fmt.Sprintf("position#%d", p.id)
	if p.stratRef > 0 {
		refID = strconv.FormatInt(p.stratRef, 10)
	}
	payload := map[string]any{
		"position_id": p.id, "strategy_ref": p.stratRef, "trigger": trig.kind,
		"close_exec_id": closeExecID, "inst_id": p.inst, "notional_usd": p.notional,
		"opened_at": p.opened.UTC().Format(time.RFC3339), "age_hours": time.Since(p.opened).Hours()}
	for k, v := range trig.facts {
		payload[k] = v
	}
	for k, v := range decision.MoneyFacts(ctx, w.pool, p.stratRef, p.id) {
		payload[k] = v
	}
	if exp, ok := payload["expected_net_apr"].(*float64); ok && exp != nil {
		payload["expected_carry_usd"] = *exp * p.notional * time.Since(p.opened).Hours() / 8760
	}
	decision.WriteEsc(ctx, w.pool, decision.Row{
		Domain: "fin", Actor: "guard", RefID: refID,
		Action: "close", Reason: trig.reason, Payload: payload})
}
