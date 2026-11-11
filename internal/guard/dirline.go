// dirline.go —— 方向族亏损预算熔断（工单#10413——业主令⑧ 2026-11-11：
// 族级累计亏损=实现+浮亏合并 ≥本金总额×预算比 → 族级熔断：模板禁（冻结新开）
// +族策略降级（停 reopen）+族仓全平+升件待业主复审；界内收益不封顶）。
// 族判别=机械可证不凭名册：在仓行腿集全同边（净裸 delta）即方向位——hedged
// 结构腿集双边对称自然排除；DCD 类无腿产品不在测域（其险由 dd_30d 账本线
// 与族级退役闸分守）。方向族=template_ref 粒度（fin_opp_template↔grammar 1:1）。
// 亏损口径（ticket 钦定先行件）：实现=fin_cashflow 六损益 kind（venue<>'sim'，
// position_ref→strategy→template_ref 全史累计）；浮亏=族在仓方向位逐 inst 现读
// mark 盯市（net_qty×last−net_cost，取 filled 腿 qty_base×px——mark 缺=该 inst
// 让位+dir_mark_missing 计数,实现口径照判不虚构）。本金=md_equity_curve
// ccy='NAV' 末行（<48h 鲜,同 idle_capital 判据;缺/陈=fail-closed 本拍不判）。
// 触线动作序：latch 键 guard.dir_broken.<T>=true 先落（防重复熔断+重启后续平
// 定向）→模板 enabled=false→族策略（live/probe/paper 合法迁移集）降级→升件
// →逐仓 dir_loss_breaker 强制平（复用 tend 径:过闸/反向原子平/节拍重试）。
// 复位语义：latch 永不自清——解封=显式热改 guard.dir_broken.<T>=false+模板
// enabled 回填+策略回迁（业主复审件,机面不复位,dd_30d 同律）。
package guard

import (
	"context"
	"fmt"
	"log"
)

const dirTrigKind = "dir_loss_breaker" // 熔断触发名（fin_guard_close.trigger 留痕）

const dirPnlKinds = "'fee','funding_settle','trade_pnl','rollback_cost','spot_buy','spot_sell'"

// dirScan 一拍：latch 续平→方向族测量→触线动作链。
func (w *Warden) dirScan(ctx context.Context) {
	pct, err := w.getF("guard.dir_loss_budget_pct")
	if err != nil || pct <= 0 {
		w.rej.Inc("dir_missing_param")
		return
	}
	broken := w.dirBroken(ctx) // latch 集续平——重启失忆不影响,latch 在库定向
	for fam := range broken {
		w.dirFlatten(ctx, fam)
	}
	var nav float64 // 本金基线=NAV 聚合读面（同 idle_capital/profit 判据源）
	if err := w.pool.QueryRow(ctx, `SELECT equity FROM md_equity_curve
		WHERE ccy='NAV' AND ts>now()-interval '48 hours'
		ORDER BY ts DESC LIMIT 1`).Scan(&nav); err != nil || nav <= 0 {
		w.rej.Inc("dir_nav_missing")
		return
	}
	budget := pct * nav
	for _, fam := range w.dirWatch(ctx, broken) { // 方向族现值集（latch 已断族不重测）
		realized, err := w.dirRealized(ctx, fam)
		if err != nil {
			w.rej.Inc("dir_measure_err")
			continue
		}
		floating, err := w.dirFloating(ctx, fam)
		if err != nil {
			w.rej.Inc("dir_measure_err")
			continue
		}
		loss := realized + floating
		if loss >= 0 {
			continue // 未亏零动作（收益界内不封顶——本件只管亏损侧）
		}
		if -loss < budget {
			continue // 阈下不误触（测试锚）
		}
		w.dirTrip(ctx, fam, -loss, realized, floating, budget, nav)
	}
}

// dirBroken latch 集读（app_config guard.dir_broken.<T>=true——熔断态唯一真源；
// 读败=空集让位不虚构断态）。
func (w *Warden) dirBroken(ctx context.Context) map[string]bool {
	rows, err := w.pool.Query(ctx,
		`SELECT substring(key from 'guard\.dir_broken\.(.+)$') FROM app_config
		WHERE key LIKE 'guard.dir_broken.%' AND value='true'::jsonb`)
	if err != nil {
		w.rej.Inc("write_err")
		return nil
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil && k != "" {
			out[k] = true
		}
	}
	return out
}

// dirWatch 方向族现值集：在仓行腿集全同边（HAVING count(DISTINCT side)=1
// 有腿且单边）的 template_ref——latch 族除外。simdrill% 演习族不入闸域。
func (w *Warden) dirWatch(ctx context.Context, broken map[string]bool) []string {
	rows, err := w.pool.Query(ctx, `SELECT DISTINCT s.template_ref
		FROM fin_position p
		JOIN fin_strategy s ON s.id=p.strategy_ref
		WHERE p.closed_at IS NULL AND s.template_ref NOT LIKE 'simdrill%'
		AND EXISTS (
			SELECT 1 FROM fin_order_leg l WHERE l.execution_id=p.execution_id
			  AND l.state='filled'
			GROUP BY l.execution_id HAVING count(DISTINCT l.side)=1)`)
	if err != nil {
		w.rej.Inc("write_err")
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil && !broken[t] {
			out = append(out, t)
		}
	}
	return out
}

// dirRealized 族实现盈亏：六损益 kind 族全史累计（venue<>'sim'——同盈利报告口径）。
func (w *Warden) dirRealized(ctx context.Context, fam string) (float64, error) {
	var v float64
	err := w.pool.QueryRow(ctx, `SELECT COALESCE(sum(cf.amount),0) FROM fin_cashflow cf
		JOIN fin_position p ON p.id=cf.position_ref
		JOIN fin_strategy s ON s.id=p.strategy_ref
		WHERE s.template_ref=$1 AND cf.venue<>'sim'
		AND cf.kind IN (`+dirPnlKinds+`)`, fam).Scan(&v)
	return v, err
}

// dirFloating 族浮亏：在仓方向位逐 inst mark 盯市（net_qty×last−net_cost；
// mark 缺=该 inst 让位+计数——缺据不虚零也不放大）。
func (w *Warden) dirFloating(ctx context.Context, fam string) (float64, error) {
	rows, err := w.pool.Query(ctx, `SELECT l.inst_id, l.venue,
		sum(CASE WHEN l.side='buy' THEN l.qty_base ELSE -l.qty_base END) AS net_qty,
		sum(CASE WHEN l.side='buy' THEN l.qty_base*l.px ELSE -l.qty_base*l.px END) AS net_cost
		FROM fin_order_leg l
		JOIN fin_position p ON p.execution_id=l.execution_id
		JOIN fin_strategy s ON s.id=p.strategy_ref
		WHERE s.template_ref=$1 AND p.closed_at IS NULL AND l.state='filled'
		AND p.execution_id IN (
			SELECT execution_id FROM fin_order_leg WHERE state='filled'
			GROUP BY execution_id HAVING count(DISTINCT side)=1)
		GROUP BY l.inst_id, l.venue`, fam)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var sum float64
	for rows.Next() {
		var inst, ven string
		var netQty, netCost *float64
		if err := rows.Scan(&inst, &ven, &netQty, &netCost); err != nil {
			return 0, err
		}
		if netQty == nil || netCost == nil {
			w.rej.Inc("dir_leg_px_missing")
			continue // px/qty 缺=该腿不可测——让位不虚构
		}
		var mark float64
		if err := w.pool.QueryRow(ctx, `SELECT last FROM md_ticks
			WHERE venue=$1 AND inst_id=$2`, ven, inst).Scan(&mark); err != nil {
			w.rej.Inc("dir_mark_missing")
			continue // mark 缺=该 inst 浮亏让位（实现口径照判）
		}
		sum += *netQty*mark - *netCost
	}
	return sum, rows.Err()
}

// dirTrip 触线动作链（顺序=latch→模板禁→族策略降级→升件→全平——latch 先落
// 保败序下轮仍定向，升件在动作前=复审件先见后动同拍）。
func (w *Warden) dirTrip(ctx context.Context, fam string, loss, realized, floating, budget, nav float64) {
	key := dirLatchKey(fam)
	if _, err := w.pool.Exec(ctx, `INSERT INTO app_config(key,value)
		VALUES($1,'true'::jsonb) ON CONFLICT (key) DO UPDATE SET value='true'`, key); err != nil {
		w.rej.Inc("write_err")
		log.Printf("ALERT [guard] ERROR 方向熔断 latch 落败（%v）——本拍不动作防半熔断", err)
		return
	}
	tx, err := w.pool.Begin(ctx) // 模板禁+族策略降级同事务（冻新提案+停 reopen 双翼）
	if err != nil {
		w.rej.Inc("write_err")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE fin_opp_template SET enabled=false WHERE template=$1`, fam); err != nil {
		w.rej.Inc("dir_freeze_err") // 模板行缺/面异=计数续行（停仓不阻）
	}
	// 合法迁移集=live/probe/paper→degraded（proposed 无仓不入——
	// 模板禁已冻其新提案径；lifecycle.legal 闭集同律）。
	if _, err := tx.Exec(ctx, `INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
		SELECT id,state,'degraded',$2 FROM fin_strategy
		WHERE template_ref=$1 AND state IN ('live','probe','paper')`,
		fam, "方向族亏损熔断——族级降级待业主复审（"+dirTrigKind+"）"); err != nil {
		tx.Rollback(ctx)
		w.rej.Inc("write_err")
		return
	}
	res, err := tx.Exec(ctx, `UPDATE fin_strategy SET state='degraded',entered_at=now()
		WHERE template_ref=$1 AND state IN ('live','probe','paper')`, fam)
	if err != nil {
		tx.Rollback(ctx)
		w.rej.Inc("write_err")
		return
	}
	degraded := res.RowsAffected()
	if err := tx.Commit(ctx); err != nil {
		w.rej.Inc("write_err")
		return
	}
	anomaly := w.dirAnomaly(ctx, fam)
	anomalyNote := ""
	if len(anomaly) > 0 {
		anomalyNote = fmt.Sprintf(";畸形/单腿仓 %d 件待人平(ids=%v——自动平仓径只接双腿契约)", len(anomaly), anomaly)
	}
	note := fmt.Sprintf("方向族亏损熔断:族=%s 累计亏损 %.2f ≥预算 %.2f（=本金NAV %.2f×预算比;实现 %.2f+浮亏 %.2f）——模板禁+%d 策略降级+族仓全平,待业主复审%s(解封=显式 guard.dir_broken.%s=false+模板enabled+策略回迁)",
		fam, loss, budget, nav, realized, floating, degraded, anomalyNote, fam)
	if _, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('仓巡','升件','',$1,'守卫环','dir_loss_breaker')`, note); err != nil {
		w.rej.Inc("write_err")
	}
	w.al.Alert(ctx, dirTrigKind, note)
	w.rej.Inc("dir_loss_breaker_tripped")
	log.Printf("ALERT [guard] ERROR %s", note)
	w.dirFlatten(ctx, fam)
}

// dirAnomaly 族在仓畸形位（无腿/单腿/>2 腿=自动平仓径契约外）——id 集,
// 触线升件载明待人平（tend 同 anomaly 语义:不盲发不静默）。
func (w *Warden) dirAnomaly(ctx context.Context, fam string) []int64 {
	var out []int64
	rows, err := w.pool.Query(ctx, `SELECT p.id FROM fin_position p
		JOIN fin_strategy s ON s.id=p.strategy_ref
		WHERE s.template_ref=$1 AND p.closed_at IS NULL
		AND (SELECT count(*) FROM fin_order_leg l
		     WHERE l.execution_id=p.execution_id AND l.state='filled') <> 2`, fam)
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
		WHERE s.template_ref=$1 AND p.closed_at IS NULL
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
		WHERE s.template_ref=$1 AND p.closed_at IS NULL AND COALESCE(g.attempts,0)>=$2`,
		fam, maxAtt).Scan(&stuck); err == nil && stuck > 0 {
		w.rej.Inc("dir_close_exhausted")
		w.al.Alert(ctx, "dir_close_exhausted", fmt.Sprintf("方向族 %s %d 仓耗尽敞口仍开——人工介入", fam, stuck))
	}
}

// dirLatchKey latch 键拼（trip/测试同径——键形单源防漂）。
func dirLatchKey(fam string) string { return "guard.dir_broken." + fam }
