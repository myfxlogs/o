// emergency.go —— 末路只减仓出口（工单#22247：守卫正径 plan/sizer 失效
// 后敞口无人能归零的第二出口）。与 Execute/ExecuteDir 的分界：
// 不验令牌（无 G6 决策——末路即决策）、不验 live 闸（close 径本免）、
// 不验对冲带、不赎回 Earn——reduce-only 只减仓语义全靠腿 Close=true
// （合约腿 buildOrderParams 自置 reduceOnly；现货腿卖=availBal 实余
// 只减不增）。腿量已由调用侧 venue 实读定好（不共用 sizedCloseLegs）。
// 幂等：腿独立——单腿败不挡余腿（敞口单调不增）；clOrdId 每 exec 唯一
// +传输败走 Resolve 定真态同既有契约；跨次重跑安全键=venue 实读残量
// （首轮成交后残量≈0→不再发单）。
package execution

import (
	"context"
	"fmt"
)

// EmergencyIntent 末路平仓意图（decision_id/token_id 落 NULL=无闸令牌
// 事实位——末路触发词由调用侧经 trigger 落账）。
type EmergencyIntent struct {
	PositionID  int64
	StrategyRef int64
	InstID      string
	NotionalUSD float64
	Trigger     string // auto|order——落账触发词
	Legs        []Leg  // 全部 Close=true 只减仓腿（venue 实读定量）
}

// ExecuteEmergency 末路只减仓执行：fin_execution 落账（无令牌位）→逐腿
// placeAndResolve（腿行/歧态定态/现金流同既有落账面）→全腿 filled=closed，
// 任一腿败=failed（调用侧下拍按 venue 实读残量重跑——幂等续平）。
func (r *Runner) ExecuteEmergency(ctx context.Context, in EmergencyIntent) (int64, error) {
	if len(in.Legs) == 0 {
		return 0, fmt.Errorf("emergency 无腿——拒执")
	}
	for i := range in.Legs {
		in.Legs[i].Close = true // reduce-only 语义位必置（防装配侧漏带）
	}
	var execID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES($1,NULL,NULL,$2,$3,'open') RETURNING id`,
		in.StrategyRef, in.InstID, in.NotionalUSD).Scan(&execID); err != nil {
		r.rej.Inc("write_err")
		return 0, err
	}
	posRef := in.PositionID
	var lastErr error
	for i, leg := range in.Legs {
		// clOrdId=emg<exec>x<no>——OKX 字母位方言（T-0038）；每 exec 唯一，
		// 传输败 Resolve 同键定真态（幂等契约同 placeAndResolve）。
		clOrdID := fmt.Sprintf("emg%dx%d", execID, i+1)
		if _, err := r.placeAndResolveCID(ctx, execID, i+1, leg, &posRef, clOrdID); err != nil {
			lastErr = err // 独腿独立——余腿照平（敞口单调不增优先于整单原子）
		}
	}
	if lastErr != nil {
		r.setState(ctx, execID, "failed")
		return execID, fmt.Errorf("emergency %s: %w", in.Trigger, lastErr)
	}
	r.setState(ctx, execID, "closed")
	return execID, nil
}
