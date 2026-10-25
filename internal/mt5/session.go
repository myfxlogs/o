// session.go —— OpenSession 有界化件（T-0053）：懒拨号首 RPC 无死线=OpenSession
// 静默悬挂（SG 生产实证）——全程死线+分阶段 log，悬挂态在日志面显形。
package mt5

import (
	"context"
	"fmt"
	"log"
	"time"

	"o/internal/platform/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// openSessionBudget OpenSession 全程死线（T-0053：懒拨号首 RPC 无死线=静默悬挂；
// 调用方免操心，超死线必返错显形禁静默占岗）。
const openSessionBudget = 30 * time.Second

// OpenSession 独立一次性会话（开连+登录——用毕调用方 Disconnect；facts 采面/
// 探针面用）。分阶段 log（dial→connect→authed 起讫；失败含阶段名+err）。
func OpenSession(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client) (*Client, error) {
	addr, cr, err := readCreds(ctx, pool, cfg)
	if err != nil {
		return nil, fmt.Errorf("阶段=creds: %w", err)
	}
	sctx, cancel := context.WithTimeout(ctx, openSessionBudget)
	defer cancel()
	log.Printf("[mt5] OpenSession dial 起 addr=%s", addr)
	cl, err := Dial(addr)
	if err != nil {
		log.Printf("WARN [mt5] OpenSession 阶段=dial 败 addr=%s: %v", addr, err)
		return nil, fmt.Errorf("阶段=dial: %w", err)
	}
	log.Printf("[mt5] OpenSession connect 起（budget %s）", openSessionBudget)
	if err := cl.Connect(sctx, cr); err != nil {
		cl.Close()
		log.Printf("WARN [mt5] OpenSession 阶段=connect 败 addr=%s（超死线/登录拒非悬挂）: %v", addr, err)
		return nil, fmt.Errorf("阶段=connect: %w", err)
	}
	log.Printf("[mt5] OpenSession authed（会话建，用毕 Disconnect）")
	return cl, nil
}
