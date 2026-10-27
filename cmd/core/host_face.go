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
