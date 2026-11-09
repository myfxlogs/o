// host_face.go —— 主机资源读面（T-0173 业主令：CPU/内存/磁盘/网络——
// 各机自报行直出，stale 显形不假值；判据在服务端）。
package main

import (
	"context"
	"time"

	"o/internal/gen/dashboardv1"
)

// GetHostMetrics 主机资源全集（各机自报行直出——stale=拍断 2min 如实显形）。
func (c coreService) GetHostMetrics(ctx context.Context,
	_ *dashboardv1.GetHostMetricsRequest) (*dashboardv1.GetHostMetricsResponse, error) {
	rows, err := c.pool.Query(ctx, `SELECT host,ts,cores,cpu_pct,mem_total_mb,mem_used_mb,
		disk_total_gb,disk_used_gb,net_rx_kbps,net_tx_kbps,uptime_s FROM fin_host_metrics
		ORDER BY host`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := &dashboardv1.GetHostMetricsResponse{}
	for rows.Next() {
		var host string
		var ts time.Time
		var cores int32
		var cpu, memT, memU, dT, dU, rx, tx float64
		var up int64
		if err := rows.Scan(&host, &ts, &cores, &cpu, &memT, &memU, &dT, &dU, &rx, &tx, &up); err != nil {
			return nil, err
		}
		r.Hosts = append(r.Hosts, &dashboardv1.HostMetric{
			Host: host, Ts: tsProto(ts), Cores: cores,
			CpuPct: cpu, MemTotalMb: memT, MemUsedMb: memU,
			DiskTotalGb: dT, DiskUsedGb: dU,
			NetRxKbps: rx, NetTxKbps: tx,
			UptimeHours: float64(up) / 3600,
			Stale:       time.Since(ts) > 2*time.Minute,
		})
	}
	return r, rows.Err()
}

// ListClosedPositions 已平仓位列表（复盘历史订单——闭仓倒序+分页下推）。
func (c coreService) ListClosedPositions(ctx context.Context,
	req *dashboardv1.ListClosedPositionsRequest) (*dashboardv1.ListClosedPositionsResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	r := &dashboardv1.ListClosedPositionsResponse{}
	var total int32
	if err := c.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_position WHERE closed_at IS NOT NULL").Scan(&total); err != nil {
		return nil, err
	}
	r.Total = total
	rows, err := c.pool.Query(ctx, `SELECT id,inst_id,opened_at,closed_at,
		coalesce(settled_amount,0),coalesce(settled_ccy,'') FROM fin_position
		WHERE closed_at IS NOT NULL ORDER BY closed_at DESC LIMIT $1 OFFSET $2`,
		limit, int(req.GetOffset()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var inst string
		var opened, closed time.Time
		var amt float64
		var ccy string
		if err := rows.Scan(&id, &inst, &opened, &closed, &amt, &ccy); err != nil {
			return nil, err
		}
		r.Positions = append(r.Positions, &dashboardv1.ClosedPosition{
			Id: id, InstId: inst,
			OpenedAt: tsProto(opened), ClosedAt: tsProto(closed),
			SettledAmount: amt, SettledCcy: ccy,
		})
	}
	return r, rows.Err()
}

// ListOpenPositions 在持订单列表（业主令「持仓订单+详细内容」——open 持仓
// 带订单腿明细；DCD 类结构仓零腿=空集如实不虚构；live 无仓=armed 待命
// 区隔非僵尸）。ev_position/ev_strategy 信号驱动重取，零轮询。
func (c coreService) ListOpenPositions(ctx context.Context,
	req *dashboardv1.ListOpenPositionsRequest) (*dashboardv1.ListOpenPositionsResponse, error) {
	r := &dashboardv1.ListOpenPositionsResponse{}
	prows, err := c.pool.Query(ctx, `SELECT p.id,p.execution_id,
		coalesce(s.template_ref,''),coalesce(s.venue,''),p.inst_id,
		coalesce(s.direction,''),p.notional_usd,p.opened_at
		FROM fin_position p LEFT JOIN fin_strategy s ON s.id=p.strategy_ref
		WHERE p.closed_at IS NULL ORDER BY p.opened_at DESC LIMIT 50`)
	if err != nil {
		return nil, err
	}
	var execIDs []int64
	posMap := map[int64]*dashboardv1.OpenPosition{}
	for prows.Next() {
		p := &dashboardv1.OpenPosition{}
		var opened time.Time
		if err := prows.Scan(&p.Id, &p.ExecutionId, &p.Template, &p.Venue,
			&p.InstId, &p.Direction, &p.NotionalUsd, &opened); err != nil {
			prows.Close()
			return nil, err
		}
		p.OpenedAt = tsProto(opened)
		r.Positions = append(r.Positions, p)
		posMap[p.ExecutionId] = p
		execIDs = append(execIDs, p.ExecutionId)
	}
	prows.Close()
	if err := prows.Err(); err != nil {
		return nil, err
	}
	if len(execIDs) > 0 {
		lrows, err := c.pool.Query(ctx, `SELECT execution_id,venue,side,inst_id,sz,
			coalesce(px,0),state,ord_type,coalesce(venue_ticket,0),ts
			FROM fin_order_leg WHERE execution_id=ANY($1) ORDER BY execution_id,leg_no`, execIDs)
		if err != nil {
			return nil, err
		}
		for lrows.Next() {
			var execID int64
			var lts time.Time
			l := &dashboardv1.OpenLeg{}
			if err := lrows.Scan(&execID, &l.Venue, &l.Side, &l.InstId, &l.Sz,
				&l.Px, &l.State, &l.OrdType, &l.VenueTicket, &lts); err != nil {
				lrows.Close()
				return nil, err
			}
			l.Ts = tsProto(lts)
			if p, ok := posMap[execID]; ok {
				p.Legs = append(p.Legs, l)
			}
		}
		lrows.Close()
		if err := lrows.Err(); err != nil {
			return nil, err
		}
	}
	arows, err := c.pool.Query(ctx, `SELECT s.id,s.template_ref,s.venue,s.inst_id,
		s.notional_usd,s.state,s.entered_at FROM fin_strategy s
		WHERE s.state='live' AND NOT EXISTS
		(SELECT 1 FROM fin_position p WHERE p.strategy_ref=s.id AND p.closed_at IS NULL)
		ORDER BY s.entered_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	for arows.Next() {
		a := &dashboardv1.ArmedStrategy{}
		var at time.Time
		if err := arows.Scan(&a.Id, &a.Template, &a.Venue, &a.InstId,
			&a.NotionalUsd, &a.State, &at); err != nil {
			arows.Close()
			return nil, err
		}
		a.EnteredAt = tsProto(at)
		r.Armed = append(r.Armed, a)
	}
	arows.Close()
	return r, arows.Err()
}
