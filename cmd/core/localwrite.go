// localwrite.go —— T-0209 SG 开仓闸行情自供谓词（main.go 棘轮瘦身件）。
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/data/okxws"
)

// attachLocalWritePred 活跃集谓词接件：inst∈fin_position 未平仓∪
// fin_strategy 非 retired（projector serve 谓词同式自算不赖种子表；
// tickers→md_ticks、funding→md_funding 同键幂等共写——dev 桥恢复后
// 最新赢自动让位，行数不膨胀撕裂对拍）。读败=不落行 fail-closed。
func attachLocalWritePred(ctx context.Context, c *okxws.Collector, pool *pgxpool.Pool) *okxws.Collector {
	c.LocalWritePred = func(kind, instID string) bool {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM (
			SELECT inst_id FROM fin_position WHERE closed_at IS NULL
			UNION SELECT inst_id FROM fin_strategy WHERE state<>'retired') t
			WHERE t.inst_id=$1`, instID).Scan(&n); err != nil {
			return false // 读败=不落行（fail-closed 不写僵尸）
		}
		return n > 0
	}
	return c
}
