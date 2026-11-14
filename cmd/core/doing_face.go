// doing_face.go —— 「每场各自在办什么」unit粒度开检查点面（在办契读侧件）。
// 在办行 ticket=unit；闭=同unit 会话收口/唤起果/唤起败（ticket∨note unit= 双配——
// 收口系事件 unit 在 note 不在 ticket）∨ 处置行引 在办#<id> 弃置销。
// 态=服务端派生（前端零判据律）：同unit 2h 内有事件=在干，否则=悬账
// （TTL 同 selfaware 办配对律；猝死残态正是悬账——检查点契的存在理由）。
package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/gen/dashboardv1"
)

// doingRowT 在办行件（queryDoingRows 出参——cockpit RPC 面与 sys_push 富卡
// 面共用正源）。
type doingRowT struct {
	unit, note string
	ts         time.Time
	fresh      bool // 同unit 2h活性=在干，否则悬账
}

// queryDoingRows 取 48h 窗内未闭在办（每 unit 取最新一行——相位变更多行同
// unit 时末行=当前声明）。
func queryDoingRows(ctx context.Context, pool *pgxpool.Pool) ([]doingRowT, error) {
	rows, err := pool.Query(ctx, `
		SELECT t.unit, t.note, t.ts, t.fresh FROM (
		  SELECT DISTINCT ON (e.ticket) e.ticket AS unit, e.id, e.note, e.ts,
		    EXISTS (SELECT 1 FROM ops_event a
		            WHERE a.ts > now() - interval '2 hours'
		              AND (a.ticket = e.ticket
		                   OR a.note ~ ('unit='||e.ticket||'([^0-9]|$)'))) AS fresh
		  FROM ops_event e
		  WHERE e.event = '在办'
		    AND e.ts > now() - interval '48 hours'
		    AND NOT EXISTS (
		      SELECT 1 FROM ops_event x
		      WHERE x.id > e.id
		        AND x.event IN ('会话收口','唤起果','唤起败')
		        AND (x.ticket = e.ticket
		             OR x.note ~ ('unit='||e.ticket||'([^0-9]|$)')))
		    AND NOT EXISTS (
		      SELECT 1 FROM ops_event d
		      WHERE d.id > e.id AND d.event = '处置'
		        AND d.note ~ ('在办#'||e.id::text||'([^0-9]|$)'))
		  ORDER BY e.ticket, e.id DESC
		) t ORDER BY t.ts DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []doingRowT
	for rows.Next() {
		var d doingRowT
		if err := rows.Scan(&d.unit, &d.note, &d.ts, &d.fresh); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// doingRows cockpit 面装配；败=missing "doing" 不上抛不炸面（同 cockpit 各段律）。
func (c coreService) doingRows(ctx context.Context, r *dashboardv1.GetCockpitResponse) {
	ds, err := queryDoingRows(ctx, c.pool)
	if err != nil {
		r.Missing = append(r.Missing, "doing")
		return
	}
	for _, d := range ds {
		row := &dashboardv1.CogDoingRow{
			Unit: d.unit, Doing: d.note,
			Ts: tsProto(d.ts), AgeS: int64(time.Since(d.ts).Seconds()),
			State: "悬账",
		}
		if d.fresh {
			row.State = "在干"
		}
		r.Doing = append(r.Doing, row)
	}
}
