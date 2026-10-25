package ledger

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LedgeredCcy 在册口径集（T-0090 唯一真源）：fin_cashflow.(venue,ccy) ∪
// 已成交 fin_order_leg.inst_id 基础币（ETH-USDT→ETH，-USDT/-SWAP 后缀
// split_part 取首段天然剥除）。「钱在哪」legacy 标位与对拍白名单同源消费
// ——现货腿底层币是活仓腿非遗留资产。
func LedgeredCcy(ctx context.Context, pool *pgxpool.Pool) (map[[2]string]bool, error) {
	out := map[[2]string]bool{}
	rows, err := pool.Query(ctx, `SELECT DISTINCT venue,ccy FROM fin_cashflow
		UNION SELECT venue,split_part(inst_id,'-',1) FROM fin_order_leg WHERE state='filled'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v, c string
		if err := rows.Scan(&v, &c); err != nil {
			return nil, err
		}
		out[[2]string{v, c}] = true
	}
	return out, rows.Err()
}
