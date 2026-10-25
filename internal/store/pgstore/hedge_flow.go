// hedge_flow.go —— 对冲流水 UNION 读面（T-0110 nox 移植——trx/cmc 双源归一，
// service_hedge_flow 对拍；cheat 剔除+vote 行剔除+买卖成交剔除照搬）。
package pgstore

import (
	"context"
	"fmt"
	"time"
)

// HedgeFlowItem 统一流水行（trx=TRX 单位折值/cmc=USD 原值）。
type HedgeFlowItem struct {
	Source       string
	Ts           time.Time
	Kind         string
	AmountValue  float64
	AmountUnit   string
	Counterparty string
	TxID         string
}

// ListHedgeFlowPage 双源 UNION 归一分页（sourceFilter ”|trx|cmc）。
func (s *Store) ListHedgeFlowPage(ctx context.Context, sourceFilter string, limit, offset int) ([]HedgeFlowItem, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT source, ts, kind, amount_value, amount_unit, counterparty, tx_id FROM (
			SELECT 'trx' AS source, ts, kind,
			       amount_sun / 1e6 AS amount_value, 'TRX' AS amount_unit,
			       COALESCE(counterparty,'') AS counterparty, tx_id
			FROM fin_trx_txs
			WHERE ($1 = '' OR $1 = 'trx')
			  AND NOT cheat_status
			  AND kind <> 'vote'
			UNION ALL
			SELECT 'cmc' AS source, transaction_time AS ts,
			       CASE transaction_type
			         WHEN 'OVERNIGHT_SETTLEMENT' THEN 'overnight'
			         WHEN 'PAYMENTS_IN' THEN 'deposit'
			         WHEN 'CLOSE_TRADE' THEN 'close_trade'
			         WHEN 'SELL_ORDER' THEN 'sell_order'
			         WHEN 'ORDER_CANCELLED' THEN 'order_cancelled'
			         ELSE 'other'
			       END AS kind,
			       amount AS amount_value, 'USD' AS amount_unit,
			       '' AS counterparty, transaction_id AS tx_id
			FROM fin_cmc_history
			WHERE ($1 = '' OR $1 = 'cmc')
			  AND transaction_type NOT IN ('SELL_TRADE','BUY_TRADE')
		) AS u
		ORDER BY ts DESC, tx_id DESC
		LIMIT $2 OFFSET $3
	`, sourceFilter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list hedge flow page: %w", err)
	}
	defer rows.Close()
	var out []HedgeFlowItem
	for rows.Next() {
		var it HedgeFlowItem
		if err := rows.Scan(&it.Source, &it.Ts, &it.Kind, &it.AmountValue, &it.AmountUnit, &it.Counterparty, &it.TxID); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ListHedgeFlowCount 流水总数（读面容错径——计数败由调用方如实零处理）。
func (s *Store) ListHedgeFlowCount(ctx context.Context, sourceFilter string) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT tx_id FROM fin_trx_txs
			WHERE ($1 = '' OR $1 = 'trx') AND NOT cheat_status AND kind <> 'vote'
			UNION ALL
			SELECT transaction_id FROM fin_cmc_history
			WHERE ($1 = '' OR $1 = 'cmc')
			  AND transaction_type NOT IN ('SELL_TRADE','BUY_TRADE')
		) AS u
	`, sourceFilter).Scan(&n)
	return n, err
}
