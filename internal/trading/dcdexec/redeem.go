// redeem.go —— O-CAP1 开仓前置赎回缝（工单#20484：dcdexec 摆渡前
// trading 不足 deficit 时先赎 USDT 储蓄回拢——Earn→funding→trading
// 多一跳=pipeline v1 可忍；worker 自带 buffer bps/单飞锁/歧态势
// 仲裁+台账行，调用侧传 deficit 原值勿自乘）。赎回败 fail-closed：
// 钱径半成不摆，exec 行如实 failed。nil=旧径不变，dcdSim 缝/未装配
// 零影响。独立成件=dcdexec.go 行棘轮基线（check-lines >300 硬阻断）。
package dcdexec

import (
	"context"
	"fmt"
)

// preRedeem deficit>0 前置赎回——nil 缝/赎回成=放行摆渡；赎回败
// Mark failed+上抛（fail-closed——不摆不盲动）。
func (d Deps) preRedeem(ctx context.Context, execID int64, deficit float64) error {
	if d.Redeem == nil {
		return nil
	}
	if rerr := d.Redeem(ctx, deficit); rerr != nil {
		Mark(ctx, d.Pool, execID, "failed", rerr)
		return fmt.Errorf("开仓前赎回败 fail-closed: %w", rerr)
	}
	return nil
}
