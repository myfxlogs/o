// settle.go —— closed 仓结算岗（T-0419 自 run.go 拆件——行数棘轮）：
// 实现额=Σcashflow+裸袋挂账落 settled_*。
package ledger

import (
	"context"
	"log"
)

// settle closed 未结算仓：实现额=Σcashflow+裸袋挂账（position_ref 过滤）落
// settled_*——T-0419 账实根修：平仓未回资产=bare_bag_usd 挂账非实现损
// （「closed 了但货还在」静默背离禁线；挂账 NULL=无袋按纯现金净）。
func (r *Runner) settle(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT id,COALESCE(bare_bag_usd,0) FROM fin_position
		WHERE closed_at IS NOT NULL AND settled_at IS NULL`)
	if err != nil {
		return err
	}
	var ids []struct {
		id  int64
		bag float64
	}
	for rows.Next() {
		var x struct {
			id  int64
			bag float64
		}
		if err := rows.Scan(&x.id, &x.bag); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, x)
	}
	rows.Close()
	for _, x := range ids {
		var net float64
		if err := r.pool.QueryRow(ctx,
			"SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE position_ref=$1 AND kind NOT IN ('earn_in','earn_out')", x.id).Scan(&net); err != nil {
			return err
		}
		net += x.bag
		if _, err := r.pool.Exec(ctx,
			"UPDATE fin_position SET settled_amount=$2,settled_ccy='USDT',settled_at=now() WHERE id=$1", x.id, net); err != nil {
			r.rej.Inc("write_err")
			return err
		}
		if x.bag != 0 {
			log.Printf("[ledger] 仓 #%d 结算落位 net=%.4f USDT（含裸袋挂账 %.4f——未回资产非实现损）", x.id, net, x.bag)
		} else {
			log.Printf("[ledger] 仓 #%d 结算落位 net=%.4f USDT", x.id, net)
		}
		r.postmortem(ctx, x.id) // T-0420 结案判定机——settled 落位同槽每平必审
	}
	return nil
}
