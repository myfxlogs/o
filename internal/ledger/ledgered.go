package ledger

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FundCcy 基金的钱币种集（唯一真源——T-0095 v2 起版，T-0100 账户域作用化）：
// 键={作用域,ccy}，作用域=venue:account（okx:trading/funding/earn）或裸 venue
// （mt5=基金专属券商户全计，USD 本位恒在册）。三分量：
//   - USDT 本位：okx 三账户域全在册；
//   - **open** filled 腿基币：只落 trading 域（ETH-USDT→ETH——活仓腿是基金钱；
//     Earn/资金户同币尘=托管非基金钱，T-0100 ①渗漏根修）；
//   - 停泊溯源币：fin_risk_decision op='residual_park' 成功凭据币种落
//     funding/earn 域（v3 残币活期归口=基金可动钱回集，T-0100 ②）。
//
// 「钱在哪」分区过滤、venue 侧对拍白名单、earn 残币归口判据同源消费。
func FundCcy(ctx context.Context, pool *pgxpool.Pool) (map[[2]string]bool, error) {
	out := map[[2]string]bool{
		{"okx:trading", "USDT"}: true,
		{"okx:funding", "USDT"}: true,
		{"okx:earn", "USDT"}:    true,
		{"mt5", "USD"}:          true,
	}
	rows, err := pool.Query(ctx, `SELECT DISTINCT l.venue||':trading',split_part(l.inst_id,'-',1)
		FROM fin_order_leg l JOIN fin_position p ON p.execution_id=l.execution_id
		WHERE p.closed_at IS NULL AND l.state='filled'`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var v, c string
		if err := rows.Scan(&v, &c); err != nil {
			rows.Close()
			return nil, err
		}
		out[[2]string{v, c}] = true
	}
	rows.Close()
	prows, err := pool.Query(ctx, `SELECT DISTINCT intent->>'venue',intent->>'ccy'
		FROM fin_risk_decision WHERE intent->>'op'='residual_park'`)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var v, c string
		if err := prows.Scan(&v, &c); err != nil {
			return nil, err
		}
		out[[2]string{v + ":funding", c}] = true
		out[[2]string{v + ":earn", c}] = true
	}
	return out, prows.Err()
}

// ParkedAmt 停泊溯源币台账量（T-0100 ③对拍 ledger 侧按币量建键——
// residual_park 凭据行 Σamt 逐 (venue,ccy)；赎回/出清径未开则台账只增，
// venue 侧余量为零时该键天然缺席不比对）。
func ParkedAmt(ctx context.Context, pool *pgxpool.Pool) (map[[2]string]float64, error) {
	out := map[[2]string]float64{}
	rows, err := pool.Query(ctx, `SELECT intent->>'venue',intent->>'ccy',
		sum((intent->>'amt')::numeric) FROM fin_risk_decision
		WHERE intent->>'op'='residual_park' GROUP BY 1,2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v, c string
		var a float64
		if err := rows.Scan(&v, &c, &a); err != nil {
			return nil, err
		}
		out[[2]string{v, c}] += a
	}
	return out, rows.Err()
}
