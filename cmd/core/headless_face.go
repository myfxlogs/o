// headless_face.go —— T-0263 无头单状态面 GetHeadless：unary 全量快照恢复面。
// ops_headless 有界态（dev 泵自报）LEFT JOIN ops_event 最新自报→相位派生；
// 五态显示标服务端判态，前端零判据纯渲染。seq=行版水印（流帧/快照去重锚）。
package main

import (
	"context"
	"time"

	"o/internal/gen/dashboardv1"
)

// headlessStateLabel 五态机键→显示标（服务端判态——前端纯渲染零判据）。
func headlessStateLabel(state string) string {
	switch state {
	case "alive":
		return "在飞"
	case "stalled":
		return "停滞"
	case "dead":
		return "死亡"
	case "overdue":
		return "逾期"
	case "done":
		return "完工"
	case "exit":
		return "已退"
	default:
		return state
	}
}

func (c coreService) GetHeadless(ctx context.Context,
	_ *dashboardv1.GetHeadlessRequest) (*dashboardv1.GetHeadlessResponse, error) {
	r := &dashboardv1.GetHeadlessResponse{}
	// 完工行同工单板口径 7d 出局（历史面=ops_headless 表自持不在读径）。
	rows, err := c.pool.Query(ctx,
		`SELECT h.ticket,h.seat,h.model,h.state,h.last_activity,h.seq,
			COALESCE(e.event,'')
		 FROM ops_headless h
		 LEFT JOIN LATERAL (
			SELECT event FROM ops_event WHERE ticket=h.ticket ORDER BY id DESC LIMIT 1
		 ) e ON true
		 WHERE h.state NOT IN ('done','exit') OR h.updated_at>now()-interval '7 days'
		 ORDER BY h.seq DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var row dashboardv1.HeadlessRow
		var la *time.Time
		var ev string
		if err := rows.Scan(&row.Ticket, &row.Seat, &row.Model, &row.State,
			&la, &row.Seq, &ev); err != nil {
			return nil, err
		}
		row.State = headlessStateLabel(row.State)
		row.Phase = stageOf(ev) // 无自报=空相位显形（不猜）
		if la != nil {
			row.LastActivity = tsProto(*la)
		}
		if row.Seq > r.LastSeq {
			r.LastSeq = row.Seq
		}
		r.Rows = append(r.Rows, &row)
	}
	return r, rows.Err()
}
