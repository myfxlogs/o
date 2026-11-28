// reverse_valuation.go —— O-41395 件② fin_valuation_ccy 反向桥（估值需求币
// 发布面：SG o-core 发布器正本→PG-md 只读副本；collector activeSet/
// localWritePred 经 legset.ValuationCcyArm 消费，BTC-USDT tick 死订根修）。
// 集合同步语义——want=SG 正本全量、have=md 副本：差行 upsert + have 独有
// 行 DELETE（出集币退订——状态集非流水集，revTableDiff/reverseSubRequests
// 均无删径本件补；键=ccy text 非 int64 id 不套 revTableDiff 泛型）。
package projector

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const qValuationCcyRev = `SELECT ccy,src,updated_at FROM fin_valuation_ccy`

const uValuationCcyMirror = `INSERT INTO fin_valuation_ccy(ccy,src,updated_at)
	VALUES($1,$2,$3::timestamptz)
	ON CONFLICT (ccy) DO UPDATE SET src=EXCLUDED.src,updated_at=EXCLUDED.updated_at`

// reverseValuationCcy 估值币集正本→副本全表 diff+出集删（O-41395——
// ev_valuation 帧（fin_0279 触发器）即发+Reconcile 期界兜底同口调用）。
func (r *Runner) reverseValuationCcy(ctx context.Context) error {
	type row struct {
		src string
		at  time.Time
	}
	fetch := func(pool *pgxpool.Pool) (map[string]row, error) {
		rows, err := pool.Query(ctx, qValuationCcyRev)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := map[string]row{}
		for rows.Next() {
			var c string
			var x row
			if err := rows.Scan(&c, &x.src, &x.at); err != nil {
				return nil, err
			}
			out[c] = x
		}
		return out, rows.Err()
	}
	want, err := fetch(r.sg)
	if err != nil {
		return fmt.Errorf("valuation_ccy 正本读: %w", err)
	}
	have, err := fetch(r.md)
	if err != nil {
		return fmt.Errorf("valuation_ccy 副本读: %w", err)
	}
	var batch [][]any
	for ccy, w := range want {
		if h, ok := have[ccy]; !ok || h.src != w.src || !h.at.Equal(w.at) {
			batch = append(batch, []any{ccy, w.src, w.at})
		}
	}
	if err := batchExec(ctx, r.md, uValuationCcyMirror, batch); err != nil {
		return fmt.Errorf("valuation_ccy 副本 upsert: %w", err)
	}
	var stale []string
	for ccy := range have {
		if _, ok := want[ccy]; !ok {
			stale = append(stale, ccy)
		}
	}
	if len(stale) > 0 {
		if _, err := r.md.Exec(ctx,
			`DELETE FROM fin_valuation_ccy WHERE ccy = ANY($1)`, stale); err != nil {
			return fmt.Errorf("valuation_ccy 副本出集删: %w", err)
		}
	}
	return nil
}
