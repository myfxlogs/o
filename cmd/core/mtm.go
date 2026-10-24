package main

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mtmReader 持仓盯市偏移读件（T-0076：venue×ccy——腿现价 vs 入价差×方向号；
// 现货腿+perp 腿同式，对账 net+mtm=NAV 同口径；价缺=该腿跳过 fail-closed）。
func mtmReader(pool *pgxpool.Pool,
	ctv func(ctx context.Context, venue, inst string) (float64, bool),
) func(ctx context.Context) (map[[2]string]float64, error) {
	return func(ctx context.Context) (map[[2]string]float64, error) {
		out := map[[2]string]float64{}
		rows, err := pool.Query(ctx, `SELECT l.venue,l.inst_id,l.side,l.sz,l.px
			FROM fin_order_leg l JOIN fin_position p ON p.execution_id=l.execution_id
			WHERE p.closed_at IS NULL AND l.state='filled' AND l.px IS NOT NULL`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var venue, inst, side string
			var sz, entry float64
			if err := rows.Scan(&venue, &inst, &side, &sz, &entry); err != nil {
				continue
			}
			var now float64
			if err := pool.QueryRow(ctx,
				"SELECT last FROM md_ticks WHERE inst_id=$1 ORDER BY ts DESC LIMIT 1",
				inst).Scan(&now); err != nil || now <= 0 {
				continue
			}
			mult := 1.0
			if strings.HasSuffix(inst, "-SWAP") {
				if cv, ok := ctv(ctx, venue, inst); ok {
					mult = cv
				}
			}
			sign := 1.0
			if side == "sell" {
				sign = -1.0
			}
			out[[2]string{venue, "USDT"}] += sign * sz * mult * (now - entry)
		}
		return out, nil
	}
}
