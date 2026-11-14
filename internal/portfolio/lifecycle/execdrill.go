// execdrill.go —— 执行径演习凭据判拆件（paper.go 行数棘轮——T-0426 毕业
// 闸消费面独立成件）。
package lifecycle

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// execDrillPass 执行径演习凭据判（T-0426）：fin_exec_drill latest 行
// verdict='pass' 才真；drilled=false=判据件缺（无行——fail-closed 同经济
// 判据缺位待遇）；verdict='fail'=演习败。读败=err 上抛 fail-closed——
// 升件#15292：ErrNoRows=真缺件映 drilled=false，余库错一律上抛
// （断凭 vs 断库必须可辨——swallow-sql-err-dead-guard 判例同型修）。
// 哨段龄界（升件#10281）：凭据须晚于 entered_at-2d（R3 侦测判词同口径——
// 跨哨段陈凭据照收=闸与侦测面互斥根修）。
func (r *Runner) execDrillPass(ctx context.Context, strategyID int64) (pass, drilled bool, err error) {
	var verdict string
	if err := r.pool.QueryRow(ctx, `SELECT d.verdict FROM fin_exec_drill d
		JOIN fin_strategy s ON s.id=d.strategy_ref WHERE d.strategy_ref=$1
		AND d.ts >= s.entered_at - interval '2 days'
		ORDER BY d.ts DESC, d.id DESC LIMIT 1`, strategyID).Scan(&verdict); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil // 无行=判据件缺（非库错——fail-closed 不晋）
		}
		return false, false, err // 库败上抛——断库映缺件=静默死径
	}
	return verdict == "pass", true, nil
}
