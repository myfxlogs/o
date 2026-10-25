package main

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mtmReader 对账净值偏移读件（venue×ccy——T-0076 合约增量+T-0087 现货全值并轨）：
// 合约腿只有保证金+浮盈=加增量 sign×sz×ctv×(现价−入价)；现货腿持真币=加全值
// sign×sz×现价（买入现金流已计流出，全值项补资产本金，增量已含在现价里）。
// 现价=md_ticks 最新 last；价缺=该腿按 0 加不减（不产假偏移）。
func mtmReader(pool *pgxpool.Pool,
	ctv func(ctx context.Context, venue, inst string) (float64, bool),
) func(ctx context.Context) (map[[2]string]float64, error) {
	return func(ctx context.Context) (map[[2]string]float64, error) {
		out := map[[2]string]float64{}
		rows, err := pool.Query(ctx, `SELECT l.venue,l.inst_id,l.side,l.sz,l.px,l.sz_unit,l.qty_base
			FROM fin_order_leg l JOIN fin_position p ON p.execution_id=l.execution_id
			WHERE p.closed_at IS NULL AND l.state='filled' AND l.px IS NOT NULL`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var venue, inst, side string
			var sz, entry float64
			var su *string
			var qb *float64
			if err := rows.Scan(&venue, &inst, &side, &sz, &entry, &su, &qb); err != nil {
				continue
			}
			qtyBase, szUnit := 0.0, ""
			if qb != nil {
				qtyBase = *qb
			}
			if su != nil {
				szUnit = *su
			}
			var now float64
			if err := pool.QueryRow(ctx,
				"SELECT last FROM md_ticks WHERE inst_id=$1 ORDER BY ts DESC LIMIT 1",
				inst).Scan(&now); err != nil || now <= 0 {
				continue
			}
			sign := 1.0
			if side == "sell" {
				sign = -1.0
			}
			if !strings.HasSuffix(inst, "-SWAP") {
				// T-0107 量纲消歧：实填币量（qty_base）×现价；base 回填行=sz×现价；
				// quote 额腿缺 base 量/未消歧=跳过不产假偏移（24×80460=$1.9M 病灶）。
				switch {
				case qtyBase > 0:
					out[[2]string{venue, "USDT"}] += sign * qtyBase * now
				case szUnit == "base":
					out[[2]string{venue, "USDT"}] += sign * sz * now
				}
				continue
			}
			mult := 1.0
			if cv, ok := ctv(ctx, venue, inst); ok {
				mult = cv
			}
			out[[2]string{venue, "USDT"}] += sign * sz * mult * (now - entry)
		}
		return out, nil
	}
}
