package execution

import (
	"context"
	"fmt"
	"log"
	"time"

	"o/internal/platform/roster"
)

// Run 员工「执事」在岗登记+心跳（执行面=调用驱动——令牌到手即 Execute）。
func Run(ctx context.Context, errc chan<- error, r *Runner) {
	if err := roster.Register(ctx, r.pool, "执事", "trading", "执行岗——双腿原子+回滚", nil); err != nil {
		errc <- fmt.Errorf("执事 register: %w", err)
		return
	}
	// saga 断点恢复臂（工单#25807）——启动扫陈 active 转 compensating
	// 按 done 步反序补偿（选型：转补偿非续行，选型注 saga.go 头）。
	if n := r.ResumeStaleSagas(ctx); n > 0 {
		log.Printf("[execution] saga 断点恢复：%d 陈 active 已处置", n)
	}
	roster.BeatLoop(ctx, r.pool, "执事", 30*time.Second, errc)
}
