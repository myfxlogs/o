// Package ledger —— 账本域（G9 单源货币真相）。
// 铁律：fin_cashflow 每行强制 source_ref+source_table（禁推断记账）；kind/ccy
// 闭集（OKX=USDT/MT5=USD 各记各的零换算，杂币不入账拒写）；净额=Σcashflow；
// 对账拿 venue 私有流快照交叉，差即告警计数；账本不触发钱动作。
package ledger

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
)

var kinds = map[string]bool{
	"fee": true, "funding_settle": true, "trade_pnl": true,
	"rollback_cost": true, "deposit": true, "withdraw": true,
}
var ccys = map[string]bool{"USDT": true, "USD": true}

// Entry 入账入参。
type Entry struct {
	Kind        string
	Venue       string
	Ccy         string
	Amount      float64
	PositionRef *int64
	SourceRef   int64
	SourceTable string
	OccurredAt  string // timestamptz 文本（测试直控）
}

// Post 入账：闭集+来源行校验（source_ref<=0 或杂币/kind 非法=拒写上抛）。
func Post(ctx context.Context, pool *pgxpool.Pool, e Entry) (int64, error) {
	if !kinds[e.Kind] {
		return 0, fmt.Errorf("kind %q 非法闭集外", e.Kind)
	}
	if !ccys[e.Ccy] {
		return 0, fmt.Errorf("ccy %q 非法——杂币不入账", e.Ccy)
	}
	if e.SourceRef <= 0 || e.SourceTable == "" {
		return 0, fmt.Errorf("缺 source_ref/source_table——禁推断记账")
	}
	var id int64
	var ts any
	if e.OccurredAt != "" {
		ts = e.OccurredAt
	}
	err := pool.QueryRow(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,COALESCE($8,now())) RETURNING id`,
		e.Kind, e.Venue, e.Ccy, e.Amount, e.PositionRef, e.SourceRef, e.SourceTable, ts).Scan(&id)
	return id, err
}

// NetAmount venue×ccy 净额=Σcashflow（一仓一判决书=position_ref 过滤同式）。
func NetAmount(ctx context.Context, pool *pgxpool.Pool, venue, ccy string, positionRef *int64) (float64, error) {
	q := "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE venue=$1 AND ccy=$2"
	args := []any{venue, ccy}
	if positionRef != nil {
		q += " AND position_ref=$3"
		args = append(args, *positionRef)
	}
	var net float64
	err := pool.QueryRow(ctx, q, args...).Scan(&net)
	return net, err
}

// Breakdown 构成明细（判决书=净额+kind 分桶构成可查）。
func Breakdown(ctx context.Context, pool *pgxpool.Pool, positionRef int64) (map[string]float64, error) {
	rows, err := pool.Query(ctx,
		"SELECT kind, sum(amount) FROM fin_cashflow WHERE position_ref=$1 GROUP BY kind", positionRef)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var k string
		var v float64
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// Diff 对账差（venue 快照 equity − 账本净额绝对差）。
func Diff(ledgerNet, venueEquity float64) float64 {
	return math.Abs(ledgerNet - venueEquity)
}
