package execution

import (
	"context"
	"fmt"
	"time"

	"phase/internal/platform/roster"
)

// Run 员工「执事」在岗登记+心跳（执行面=调用驱动——令牌到手即 Execute）。
func Run(ctx context.Context, errc chan<- error, r *Runner) {
	if err := roster.Register(ctx, r.pool, "执事", "trading", "执行岗——双腿原子+回滚", nil); err != nil {
		errc <- fmt.Errorf("执事 register: %w", err)
		return
	}
	roster.BeatLoop(ctx, r.pool, "执事", 30*time.Second, errc)
}
