package risk

import (
	"context"
	"fmt"
	"time"

	"o/internal/platform/roster"
)

// Run 员工「门官」在岗登记+心跳（闸=被动面——在岗行/心跳供产出窗与活性判据）。
func Run(ctx context.Context, errc chan<- error, g *Gate) {
	if err := roster.Register(ctx, g.pool, "门官", "risk", "风控闸官——唯一交易前闸", nil); err != nil {
		errc <- fmt.Errorf("门官 register: %w", err)
		return
	}
	roster.BeatLoop(ctx, g.pool, "门官", 30*time.Second, errc)
}
