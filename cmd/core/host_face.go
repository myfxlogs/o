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
