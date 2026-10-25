package ledger

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FundCcy 基金的钱币种集（T-0095 v2 在册集定义点——替代 T-0090 LedgeredCcy）：
// USDT 本位 ∪ **open** filled 腿基币（ETH-USDT→ETH，-USDT/-SWAP 后缀 split_part
// 取首段剥除）。历史腿已平的币=今日托管尘非基金钱——不入集。「钱在哪」过滤与
// 对拍白名单同源消费唯一真源：集合外=托管（卡面不显、对拍口径内天然排除不豁免）。
// mt5=基金专属券商户全计（T-0096），USD 本位恒在册。
func FundCcy(ctx context.Context, pool *pgxpool.Pool) (map[[2]string]bool, error) {
	out := map[[2]string]bool{
		[2]string{"okx", "USDT"}: true,
		[2]string{"mt5", "USD"}:  true,
	}
	rows, err := pool.Query(ctx, `SELECT DISTINCT l.venue,split_part(l.inst_id,'-',1)
		FROM fin_order_leg l JOIN fin_position p ON p.execution_id=l.execution_id
		WHERE p.closed_at IS NULL AND l.state='filled'`)
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
