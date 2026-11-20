package earn

// staleexec.go —— 超龄 open execution 探针（工单#25693）：state='open'
// ∧created_at>1h=执行行漏关终态账烂（在途单秒级态迁，不存在挂单>1h 的
// 正常路径——下单皆市价/IOC）。病案：#1117/#1134 挂 14 天，sweepIdle
// inFlight 闸把账烂当在途→1 万元闲资零息 14 天（解堵实证 transId=
// 3367050186）。闸已加时效放行（账烂不堵钱），本件只负责升件喊人收账。

import (
	"context"
	"fmt"
)

// staleExecEsc 超龄 open exec →「资金闸」升件。抑件双契（guard/anomaly.go
// 同款）：①同 scope 未销升件在案即抑（销而未修=真再发照火）；②同 scope
// 开工单在案即抑（票管催探针静默——escInsert 票抑同契）。查败 fail-open
// 升件优先不吞；写败计数下拍重试。
func (w *Worker) staleExecEsc(ctx context.Context) {
	var ids string
	if err := w.pool.QueryRow(ctx, `SELECT COALESCE(string_agg(id::text,',' ORDER BY id),'')
		FROM fin_execution WHERE state='open' AND created_at<now()-interval '1 hour'`).Scan(&ids); err != nil || ids == "" {
		return
	}
	var open bool
	if err := w.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM ops_event e WHERE e.event='升件' AND e.esc_scope='earn_stale_exec_open'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')))`).Scan(&open); err != nil || open {
		return
	}
	var tick int
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event t
		WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND t.note ~ ('(^|[^a-zA-Z0-9_])scope='||$1||'([^a-zA-Z0-9_]|$)')
		AND (NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
				AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)') AND d.note !~ ('工单#'||t.id||' ?接续'))
			OR (t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour' > now()
				AND EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
					AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)')
					AND d.note LIKE '%挂起条件=%')))`,
		"earn_stale_exec_open").Scan(&tick); err == nil && tick > 0 {
		return
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('o-core','升件','',$1,'资金闸','earn_stale_exec_open')`,
		fmt.Sprintf("fin_execution open 超龄(>1h)挂账 id=%s——执行行漏关终态非在途，账烂曾堵 sweepIdle 闸 14 天实证(工单#25693)。处置=核 fin_position/venue 真态后迁终态", ids)); err != nil {
		w.rej.Inc("write_err")
	}
}
