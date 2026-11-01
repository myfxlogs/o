// verdict_apply.go —— T-0232 乙案判决/状态应用分离接线件：OCI 评史官只写判决
// 事实（fin_review/fin_review_verdict 追加——review_rw 收窄面），SG 判行官消费
// fin_review 增量应用钱径态迁（fin_proposal 退役/strategy degraded/transition
// ——过渡轮询件，持久游标+幂等键；outbox 判决道切换即废）。
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/review"
)

// verdictApplyArm 判行官挂臂——main.go 单行接线（300 行棘轮界内净增零）。
func verdictApplyArm(ctx context.Context, errc chan<- error, pool *pgxpool.Pool) {
	go review.NewApplier(pool).Run(ctx, errc)
}
