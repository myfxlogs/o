// probe.go —— 观测升件面（T-0434）：连败升件（写不进库=观测断供一等事件）
// + 停拍探针（fin_host_metrics 行陈旧→升件，selfprobe 公共模子第三实例，
// core/collector 双挂对机互探——写者进程死=对机活者代吼）。
package hostmetrics

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"o/internal/platform/selfprobe"
)

const (
	// failEscAfter 写败升件阈（30s 拍×3=90s——三拍俱败非抖动）。
	failEscAfter = 3
	// failEscDedupHours 连败升件同窗去重（selfprobe 族 norm 同值——长断供
	// 24h 至多一报）。
	failEscDedupHours = 24
	// escSeat / escTicket 升件身份（T-0434 写器根修件）。
	escSeat   = "hostmetrics"
	escTicket = "T-0434"
)

// escScope 升件判例域单键（按机独立不撞窗——大写与 hostSpec 同形）。
func escScope(host string) string { return "HOSTMETRICS-" + strings.ToUpper(host) }

// escWriteFail 连败升件一拍：观测断供一等事件非 log 噪音（T-0434——写不进
// 库=盯人的探头瞎）。同窗去重族 norm（长断供 24h 至多一报）；升件写败=
// 观测臂同瞎如实 log 非炸环（自报环持续重试，恢复自清）。
func escWriteFail(ctx context.Context, pool DB, host string, fails int) {
	var dup int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3)`,
		selfprobe.EscClass, escScope(host), failEscDedupHours).Scan(&dup); err != nil {
		log.Printf("[hostmetrics] %s 连败升件去重读败: %v", host, err)
		return
	}
	if dup > 0 {
		return
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		escSeat, escTicket,
		fmt.Sprintf("主机自报 %s 连败 %d 拍写不进库（阈=%d 拍）——观测断供一等事件非 log 噪音；自报环退避重试中，恢复自清",
			host, fails, failEscAfter),
		selfprobe.EscClass, escScope(host)); err != nil {
		log.Printf("[hostmetrics] %s 连败升件行写败: %v", host, err)
	}
}

const (
	// staleAfter 自报行陈旧阈（30s 自报拍——10min=20 拍，容忍部署窗重启
	// （SIGTERM 挂死 10s 强杀+健康闸 90s）+启动序；超=停拍实锤）。
	staleAfter = 10 * time.Minute
	// probeEvery 探针拍频（阈 10min 下检测≤12min——单查询零负担）。
	probeEvery = 2 * time.Minute
	// probeSeat / probeTicket 停拍探针身份。
	probeSeat   = "hostmetrics"
	probeTicket = "T-0434"
)

// KnownHosts 自报行全集（写方=cmd/core "sg" + cmd/collector "dev"——生产
// 现值亲证两行；新自报机入列同拍，出生律：新自报机无探针行=交付不齐）。
var KnownHosts = []string{"sg", "dev"}

// hostSpec 主机行规格（scope 单键=HOSTMETRICS-<HOST>——升件按机独立不撞窗）。
func hostSpec(host string) selfprobe.Spec {
	return selfprobe.Spec{
		Site:     probeSeat,
		Ticket:   probeTicket,
		Scope:    "HOSTMETRICS-" + strings.ToUpper(host),
		Label:    "主机自报 " + host,
		Verb:     "停拍",
		TsCol:    "ts",
		MaxAge:   staleAfter,
		Thresh:   "10m",
		Rhythm:   "30s 自报拍×20（容忍部署窗重启+健康闸 90s）",
		ZeroWord: "零行史",
		Ditto:    "停拍形含「部署窗重启」两可如实标，对拍归人肉/运维面",
	}
}

// ProbeStale 自报表停拍探针一拍：每机 fin_host_metrics 末行龄超阈/零行史→
// 病灶行+升件（selfprobe 公共模子第三实例——对机互探：本进程探全表已知行，
// 任一写者进程死=对机活者代吼，进程死不再逃逸观测——2026-10-30 sg 行
// 冻结 8 天病灶对治）。
func ProbeStale(ctx context.Context, escPool selfprobe.EscPool, now func() time.Time) error {
	for _, host := range KnownHosts {
		h := host
		face := func(ctx context.Context) (*time.Time, error) {
			var last *time.Time
			if err := escPool.QueryRow(ctx,
				`SELECT max(ts) FROM fin_host_metrics WHERE host=$1`, h).Scan(&last); err != nil {
				return nil, err
			}
			return last, nil
		}
		if _, err := selfprobe.Probe(ctx, escPool, face, hostSpec(h), now); err != nil {
			return err
		}
	}
	return nil
}

// ProbeLoop 停拍探针常驻环（probeEvery 拍——sched-exempt-ok：观测心跳语义
// 非轮询拉取，D-0008-B 豁免径；escPool nil=守位不探）。
func ProbeLoop(ctx context.Context, escPool selfprobe.EscPool) {
	if escPool == nil {
		return // 守位
	}
	t := time.NewTicker(probeEvery) // sched-exempt-ok：停拍探针观测心跳拍（2min 单查询有界态——心跳语义非轮询拉取，D-0008-B 豁免径）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := ProbeStale(ctx, escPool, time.Now); err != nil {
				log.Printf("[hostmetrics] 停拍探针: %v", err)
			}
		}
	}
}

// Start 自报写器+对机停拍探针一体装配（T-0434——写者与对机监视同生同账，
// 装配点单行不散；两环各自独立 goroutine，ctx 断即齐退）。
func Start(ctx context.Context, pool DB, host, diskPath string, interval time.Duration) {
	go Run(ctx, func() DB { return pool }, host, diskPath, interval)
	go ProbeLoop(ctx, pool)
}
