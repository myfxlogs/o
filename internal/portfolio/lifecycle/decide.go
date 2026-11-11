// decide.go —— ops_decision 决策记忆写件（T-0415 自 opener.go 拆立：
// live 真钱开仓=open 决策行「为什么开」——开闸判据过+期望 carry 锚提案
// 评估链；继事实行写不阻钱径，写败升件自报禁静默）。
package lifecycle

import (
	"context"
	"fmt"
	"strconv"

	"o/internal/platform/decision"
)

// noteLiveOpen live 开仓决策行（op.Open 成交后调用——payload 携 exec_id/
// 期望 vs 实现基线件 expected_net_apr/realized_usd）。
func (r *Runner) noteLiveOpen(ctx context.Context, spec OpenSpec, execID int64) {
	payload := map[string]any{
		"exec_id": execID, "inst_id": spec.InstID, "template": spec.Template,
		"direction": spec.Direction, "notional_usd": spec.NotionalUSD,
		"venue": spec.Venue}
	for k, v := range decision.MoneyFacts(ctx, r.pool, spec.StrategyRef, 0) {
		payload[k] = v
	}
	decision.WriteEsc(ctx, r.pool, decision.Row{
		Domain: "fin", Actor: "lifecycle", RefID: strconv.FormatInt(spec.StrategyRef, 10),
		Action: "open",
		Reason: fmt.Sprintf("live 开仓成 exec_id=%d（开闸判据过——%s notional=%.0f）",
			execID, spec.InstID, spec.NotionalUSD),
		Payload: payload})
}

// ftoa 浮点紧凑格式化（原 paper.go——棘轮挪位,本包共用）。
func ftoa(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}
