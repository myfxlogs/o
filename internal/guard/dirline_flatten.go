// dirline_flatten.go —— 方向族熔断平仓段（dirline.go 棘轮拆分——O-33393 名册字段落行
// 顶破 300 棘轮；dirAnomaly 畸形位集+dirFlatten 逐仓强制平，语义零改）。
package guard

import (
	"context"
	"fmt"
)

// dirAnomaly 族在仓畸形位（零腿/>2 腿=自动平仓径契约外；棒5c 单腿升一等
// 契约后出畸形集——方向族 1 腿=tend 单腿径可平）。id 集触线升件载明待人平。
func (w *Warden) dirAnomaly(ctx context.Context, fam string) []int64 {
	var out []int64
	rows, err := w.pool.Query(ctx, `SELECT p.id FROM fin_position p
		JOIN fin_strategy s ON s.id=p.strategy_ref
		WHERE s.template_ref=$1 AND p.state<>'closed'
		AND (SELECT count(*) FROM fin_order_leg l
		     WHERE l.execution_id=p.execution_id AND l.state='filled') NOT IN (1,2)`, fam)
	if err != nil {
		w.rej.Inc("write_err")
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			out = append(out, id)
		}
	}
	return out
}

// dirFlatten 族在仓逐仓强制平（ddResume 同形——节拍/上限走热启键,
// 耗尽仓持续告警不静默;畸形仓由 dirAnomaly 升件面待人平,tend 自身
// 对腿契约外位也拒发不误平——双保险同语义）。
func (w *Warden) dirFlatten(ctx context.Context, fam string) {
	backoff, maxAtt, err := w.retryKeys()
	if err != nil {
		w.rej.Inc("dir_missing_param")
		return
	}
	rows, err := w.pool.Query(ctx, `SELECT p.id, p.execution_id, p.strategy_ref, p.inst_id,
		p.notional_usd, p.opened_at, COALESCE(g.attempts,0), COALESCE(s.state,'')
		FROM fin_position p
		JOIN fin_strategy s ON s.id=p.strategy_ref
		LEFT JOIN fin_guard_close g ON g.position_id = p.id
		WHERE s.template_ref=$1 AND p.state<>'closed'
		  AND COALESCE(g.state,'open') <> 'closed'
		  AND COALESCE(g.attempts,0) < $2
		  AND (g.last_try_at IS NULL OR now() - g.last_try_at >= make_interval(secs => $3))`,
		fam, maxAtt, backoff)
	if err != nil {
		w.rej.Inc("write_err")
		return
	}
	var batch []posRow
	for rows.Next() {
		var p posRow
		if err := rows.Scan(&p.id, &p.execID, &p.stratRef, &p.inst, &p.notional, &p.opened,
			&p.attempts, &p.stratState); err != nil {
			rows.Close()
			w.rej.Inc("write_err")
			return
		}
		batch = append(batch, p)
	}
	rows.Close()
	for _, p := range batch {
		w.tend(ctx, p, maxAtt, closeTrig{kind: dirTrigKind,
			reason: "方向族亏损熔断——族=" + fam + " 触线全平待复审",
			facts:  map[string]any{"dir_family": fam}})
	}
	var stuck int // 耗尽仓声张（ddResume 同律——敞口禁无限期挂）
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM fin_position p
		JOIN fin_strategy s ON s.id=p.strategy_ref
		LEFT JOIN fin_guard_close g ON g.position_id=p.id
		WHERE s.template_ref=$1 AND p.state<>'closed' AND COALESCE(g.attempts,0)>=$2`,
		fam, maxAtt).Scan(&stuck); err == nil && stuck > 0 {
		w.rej.Inc("dir_close_exhausted")
		w.al.Alert(ctx, "dir_close_exhausted", fmt.Sprintf("方向族 %s %d 仓耗尽敞口仍开——人工介入", fam, stuck))
	}
}
