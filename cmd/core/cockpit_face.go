// cockpit_face.go —— T-0239 业主驾驶舱 GetCockpit：五块只读聚合
// （存活+钱面+工单+待开仓+在做什么）。服务端判态派生，前端零判据。
package main

import (
	"context"
	"time"

	"o/internal/gen/dashboardv1"
)

// stageOf 事件→阶段派生（钦定映射：领单=施工/收口·移交=等审计/挂起=挂起/
// 裁决=闭环；开工=调研中——票面语「各席最新一条自报映射」）。
func stageOf(ev string) string {
	switch ev {
	case "领单", "开工":
		return "施工"
	case "收口", "移交":
		return "等审计"
	case "挂起":
		return "挂起"
	case "裁决":
		return "闭环"
	default:
		return ev
	}
}

func (c coreService) GetCockpit(ctx context.Context,
	_ *dashboardv1.GetCockpitRequest) (*dashboardv1.GetCockpitResponse, error) {
	r := &dashboardv1.GetCockpitResponse{Money: &dashboardv1.CockpitMoney{}}

	// ①⑤ 存活面+在做什么：每席最新自报→阶段派生
	rows, err := c.pool.Query(ctx,
		`SELECT seat,ticket,event,ts,note FROM ops_seat_latest`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s dashboardv1.CockpitSeat
			var ts *time.Time
			if err := rows.Scan(&s.Seat, &s.Ticket, &s.Event, &ts, &s.Note); err == nil {
				s.Stage = stageOf(s.Event)
				if ts != nil {
					s.Ts = tsProto(*ts)
				}
				r.Seats = append(r.Seats, &s)
			}
		}
	}

	// ② 钱面：D-0037 口径月净利+本金水位+近30d 日净序列
	_ = c.pool.QueryRow(ctx,
		`SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		 WHERE ts>date_trunc('month',now()) AND kind IN ('interest','funding_settle','fee','spot_buy','spot_sell')`).Scan(&r.Money.NetMonth)
	_ = c.pool.QueryRow(ctx,
		`SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='deposit'`).Scan(&r.Money.Capital)
	dr, err := c.pool.Query(ctx,
		`SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		 WHERE ts>now()-interval '30 days' AND kind IN ('interest','funding_settle','fee','spot_buy','spot_sell')
		 GROUP BY ts::date ORDER BY ts::date`)
	if err == nil {
		defer dr.Close()
		for dr.Next() {
			var v float64
			if dr.Scan(&v) == nil {
				r.Money.Daily = append(r.Money.Daily, v)
			}
		}
	}

	// ③ 工单板：ops_ticket_state 派生态（闭环只看近 7d）
	tr, err := c.pool.Query(ctx,
		`SELECT ticket,event,ts,seat FROM ops_ticket_state ORDER BY ts DESC`)
	if err == nil {
		defer tr.Close()
		for tr.Next() {
			var tk dashboardv1.CockpitTicket
			var ts *time.Time
			if tr.Scan(&tk.Ticket, &tk.State, &ts, &tk.Seat) != nil {
				continue
			}
			switch tk.State {
			case "领单", "开工", "收口", "移交":
				tk.State = "在飞"
			case "挂起":
				tk.State = "挂起"
			default:
				if ts != nil && time.Since(*ts) > 7*24*time.Hour {
					continue // 闭环超 7d 出局（近7d闭环计数面）
				}
				tk.State = "闭环"
			}
			if ts != nil {
				tk.Ts = tsProto(*ts)
			}
			r.Tickets = append(r.Tickets, &tk)
		}
	}

	// ④ 待开仓：armed 提案+TTL 倒计时（24h 默认同 lifecycle 口径）
	ar, err := c.pool.Query(ctx,
		`SELECT id,inst_id,direction,created_at,
			COALESCE((SELECT value::float FROM app_config WHERE key='lifecycle.armed_ttl_h'),24)-EXTRACT(EPOCH FROM now()-created_at)/3600.0
		 FROM fin_proposal WHERE state='fin_armed' ORDER BY created_at`)
	if err == nil {
		defer ar.Close()
		for ar.Next() {
			var a dashboardv1.CockpitArmed
			var t time.Time
			if ar.Scan(&a.Id, &a.InstId, &a.Direction, &t, &a.TtlHoursLeft) == nil {
				a.Created = tsProto(t)
				r.Armed = append(r.Armed, &a)
			}
		}
	}
	return r, nil
}
