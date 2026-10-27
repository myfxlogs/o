// Package hostmetrics —— 主机资源自报（T-0173 业主令：面板 CPU/内存/磁盘/
// 网络）。/proc 直读（docker 容器内默认透传宿主值）；每机一行 upsert 进
// fin_host_metrics（D-0018 有界态——30s 拍覆写零增长）；写方=各机进程自报
// （SG core 写 sg / dev collector 经 ROSTER 池写 dev——与席位 beat 同频同径）。
package hostmetrics

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sample 单拍资源快照。
type Sample struct {
	Cores       int
	CPUPct      float64
	MemTotalMB  int64
	MemUsedMB   int64
	DiskTotalGB float64
	DiskUsedGB  float64
	NetRxKbps   float64
	NetTxKbps   float64
	UptimeS     int64
}

var (
	mu       sync.Mutex
	lastCPU  [2]uint64 // idle, total
	lastNet  map[string][2]uint64
	haveLast bool
)

// procUptime 宿主运行秒。
func procUptime() int64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	f := strings.Fields(string(b))
	v, _ := strconv.ParseFloat(f[0], 64)
	return int64(v)
}

// sampleCPU /proc/stat 累计值→占用率（需两次拍差分；首拍 0）。
func sampleCPU() float64 {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		f := strings.Fields(line)[1:]
		var vals []uint64
		for _, x := range f {
			v, _ := strconv.ParseUint(x, 10, 64)
			vals = append(vals, v)
		}
		var idle, total uint64
		for i, v := range vals {
			total += v
			if i == 3 || i == 4 { // idle+iowait
				idle += v
			}
		}
		mu.Lock()
		defer mu.Unlock()
		if !haveLast {
			lastCPU = [2]uint64{idle, total}
			haveLast = true
			return 0
		}
		dTotal := total - lastCPU[1]
		dIdle := idle - lastCPU[0]
		lastCPU = [2]uint64{idle, total}
		if dTotal == 0 {
			return 0
		}
		return float64(dTotal-dIdle) / float64(dTotal) * 100
	}
	return 0
}

// sampleMem /proc/meminfo（容器内=宿主值）。
func sampleMem() (totalMB, usedMB int64) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	var avail int64
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		v, _ := strconv.ParseInt(f[1], 10, 64)
		switch f[0] {
		case "MemTotal:":
			totalMB = v / 1024
		case "MemAvailable:":
			avail = v / 1024
		}
	}
	return totalMB, totalMB - avail
}

// sampleDisk 根文件系统用量（statfs——面板面非账面）。
func sampleDisk(path string) (totalGB, usedGB float64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	total := float64(st.Blocks) * float64(st.Bsize)
	free := float64(st.Bavail) * float64(st.Bsize)
	return total / (1 << 30), (total - free) / (1 << 30)
}

// sampleNet /proc/net/dev 全接口字节差分→kbps（lo 除外）。
func sampleNet(interval time.Duration) (rxKbps, txKbps float64) {
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	cur := map[string][2]uint64{}
	for _, line := range strings.Split(string(b), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}
		f := strings.Fields(parts[1])
		if len(f) < 9 {
			continue
		}
		rx, _ := strconv.ParseUint(f[0], 10, 64)
		tx, _ := strconv.ParseUint(f[8], 10, 64)
		cur[iface] = [2]uint64{rx, tx}
	}
	mu.Lock()
	prev := lastNet
	lastNet = cur
	mu.Unlock()
	if !haveLast {
		return 0, 0
	}
	sec := interval.Seconds()
	if sec <= 0 {
		return 0, 0
	}
	for iface, c := range cur {
		p, ok := prev[iface]
		if !ok {
			continue
		}
		rxKbps += float64(c[0]-p[0]) * 8 / 1000 / sec
		txKbps += float64(c[1]-p[1]) * 8 / 1000 / sec
	}
	return rxKbps, txKbps
}

// Take 抓一拍（连续调用间自动差分——进程内常驻件按固定间隔调用）。
func Take(diskPath string, interval time.Duration) Sample {
	totalMB, usedMB := sampleMem()
	dTotal, dUsed := sampleDisk(diskPath)
	rx, tx := sampleNet(interval)
	cores := int64(0)
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		cores = int64(strings.Count(string(b), "processor"))
	}
	return Sample{
		Cores:       int(cores),
		CPUPct:      sampleCPU(),
		MemTotalMB:  totalMB,
		MemUsedMB:   usedMB,
		DiskTotalGB: dTotal,
		DiskUsedGB:  dUsed,
		NetRxKbps:   rx,
		NetTxKbps:   tx,
		UptimeS:     procUptime(),
	}
}

// Upsert 单拍入册（host 一行覆写——D-0018 有界态）。
func Upsert(ctx context.Context, pool *pgxpool.Pool, host string, s Sample) error {
	_, err := pool.Exec(ctx, `INSERT INTO fin_host_metrics
		(host,ts,cores,cpu_pct,mem_total_mb,mem_used_mb,disk_total_gb,disk_used_gb,net_rx_kbps,net_tx_kbps,uptime_s)
		VALUES ($1,now(),$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (host) DO UPDATE SET ts=now(),cores=$2,cpu_pct=$3,mem_total_mb=$4,
		mem_used_mb=$5,disk_total_gb=$6,disk_used_gb=$7,net_rx_kbps=$8,net_tx_kbps=$9,uptime_s=$10`,
		host, s.Cores, s.CPUPct, s.MemTotalMB, s.MemUsedMB,
		s.DiskTotalGB, s.DiskUsedGB, s.NetRxKbps, s.NetTxKbps, s.UptimeS)
	return err
}

// Run 常驻自报循环（interval 拍；失败 log 不炸——观测面缺不拖业务面）。
func Run(ctx context.Context, pool *pgxpool.Pool, host, diskPath string, interval time.Duration) {
	if pool == nil {
		return
	}
	prev := Take(diskPath, interval) // 首拍建立差分基线
	if err := Upsert(ctx, pool, host, prev); err != nil {
		log.Printf("[hostmetrics] %s 首拍写败: %v", host, err)
	}
	t := time.NewTicker(interval) // sched-exempt-ok：主机自报心跳拍（30s 有界态 upsert——心跳/退避语义非轮询拉取，D-0008-B 豁免径）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s := Take(diskPath, interval)
			if err := Upsert(ctx, pool, host, s); err != nil {
				log.Printf("[hostmetrics] %s 写败: %v", host, err)
			}
		}
	}
}
