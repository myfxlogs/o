// Package pgguard —— pgxpool 连接卫生守卫（T-0154/canon
// pool-tx-residue-vacuum-pin）：事务残留借出拦+backend_xmin 年龄哨兵。
// 独立包（pgstore/testutil/cmd 自建池共用件——环引隔离）。
package pgguard

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// txResidueCount 事务态还池累计（观测面——抓真凶借用者后续专票根修）。
var txResidueCount atomic.Int64

// TxResidueCount 残留计数读（status/巡检观测面）。
func TxResidueCount() int64 { return txResidueCount.Load() }

// GuardPool 池配挂守卫（Connect 默认挂；cmd/collector/testutil 自建池同守——
// 一切 pgxpool 借用者共用件非三份复制）。借出前 TxStatus 检查：非 idle→
// ROLLBACK 清态放行+log 计数（抓泄漏借用者定位面）；ROLLBACK 败=销毁连接。
func GuardPool(cfg *pgxpool.Config) {
	// PrepareConn（BeforeAcquire 已弃用——v5.10 起本位被 PrepareConn 取代，
	// 双设时后者优先）：借出前 TxStatus 检查，语义同守卫。
	cfg.PrepareConn = func(ctx context.Context, conn *pgx.Conn) (bool, error) {
		if st := conn.PgConn().TxStatus(); st != 'I' {
			if _, err := conn.Exec(ctx, "ROLLBACK"); err != nil {
				log.Printf("[pgguard] WARN 池连接事务态还池且 ROLLBACK 败——销毁连接（TxStatus=%q）", st)
				return false, err
			}
			txResidueCount.Add(1)
			log.Printf("[pgguard] WARN 池连接事务态还池已守卫回滚（TxStatus=%q 残留计数=%d——泄漏借用者定位面）",
				st, txResidueCount.Load())
		}
		return true, nil
	}
}

// Alerter 告警窄接口（notify.Alerter 同形——包依赖反向禁）。
type Alerter interface {
	Alert(ctx context.Context, kind, payload string)
}

// Sentinel backend_xmin 年龄哨兵（先报不杀——杀权留人工/后续裁定）：
// pg_stat_activity 中 xact_start 超 10min 的会话=快照钉死 vacuum 水平线实锤
// （命中 LISTEN/长驻会话=钉手）；另带 notify 队列水位探针（T-0162——
// pg_notification_queue_usage() 供方洪泛先于队列爆顶可见，爆顶=发送方
// NOTIFY 直接报错）。
type Sentinel struct {
	pool     *pgxpool.Pool
	al       Alerter
	alerted  bool    // xmin 钉手 arm/disarm 位
	qAlerted bool    // notify 水位 arm/disarm 位（独立条件独立清标）
	qMax     float64 // 水位告警线（NewSentinel 置 notifyQueueMax——桩测可调）
	mu       sync.Mutex
}

// notifyQueueMax 水位告警线（0..1 分数——过半即供方洪泛实锤先报，
// 离 1.0 爆顶留处置余量；治洪后常态≈0，非零即值得看）。
const notifyQueueMax = 0.5

// NewSentinel 构造。
func NewSentinel(pool *pgxpool.Pool) *Sentinel {
	return &Sentinel{pool: pool, qMax: notifyQueueMax}
}

// AttachAlerter 挂业主告警通道（nil=log 退路）。
func (x *Sentinel) AttachAlerter(a Alerter) { x.al = a }

// Run 在岗（10min 节拍——sched-exempt-ok 观测拍非轮询拉取）。
func (x *Sentinel) Run(ctx context.Context) {
	t := time.NewTicker(10 * time.Minute) // sched-exempt-ok 观测拍（哨兵判定节拍非轮询拉取——先报不杀）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			x.Check(ctx)
			x.CheckNotifyQueue(ctx)
		}
	}
}

// Check 一拍：命中即告警一次（恢复清标再报——抑刷屏同 armwatch 语义）。
// 取证粒度（T-0154 返修判据④）：wait_event（ClientWrite=读饥饿型实锤指纹——
// notify 交付卡在对端饿死的套接字上）+backend_start（会话年龄——跨容器
// 复发判定锚）逐会话入报。
func (x *Sentinel) Check(ctx context.Context) {
	rows, err := x.pool.Query(ctx, `SELECT pid, coalesce(application_name,''), left(query,60),
		coalesce(wait_event,''), backend_start
		FROM pg_stat_activity
		WHERE backend_xmin IS NOT NULL AND xact_start < now()-interval '10 minutes'
		  AND backend_type='client backend'`) // T-0209：background worker 长事务排除（TimescaleDB 调度器系统会话非真空钉手——共享实例误报病灶根修）
	if err != nil {
		log.Printf("[pgguard] xmin 哨兵读败（不判）: %v", err)
		return
	}
	type hit struct {
		pid       int32
		app       string
		query     string
		waitEvent string
		started   time.Time
	}
	var hits []hit
	for rows.Next() {
		var h hit
		if rows.Scan(&h.pid, &h.app, &h.query, &h.waitEvent, &h.started) == nil {
			hits = append(hits, h)
		}
	}
	rows.Close()
	x.mu.Lock()
	defer x.mu.Unlock()
	if len(hits) == 0 {
		x.alerted = false // 恢复清标（再钉可再报）
		return
	}
	if x.alerted {
		return
	}
	x.alerted = true
	detail := make([]string, len(hits))
	for i, h := range hits {
		detail[i] = fmt.Sprintf("pid %d backend_start %s wait_event %s app %q query %q",
			h.pid, h.started.Format("01-02T15:04:05"), h.waitEvent, h.app, h.query)
	}
	msg := fmt.Sprintf("backend_xmin 哨兵：%d 会话事务超 10min 钉死 vacuum 水平线（先报不杀——杀权留人工，LISTEN 臂重连兜底可后续升自动）｜%s",
		len(hits), strings.Join(detail, "；"))
	if x.al != nil {
		x.al.Alert(ctx, "pool_tx_residue", msg)
	} else {
		log.Printf("[pgguard] WARN %s", msg)
	}
}

// CheckNotifyQueue 一拍：pg_notification_queue_usage() 水位过线即告警一次
// （恢复清标再报——同 xmin 臂抑刷屏语义）。读败不判（观测面非工作面）。
func (x *Sentinel) CheckNotifyQueue(ctx context.Context) {
	var usage float64
	if err := x.pool.QueryRow(ctx, "SELECT pg_notification_queue_usage()").Scan(&usage); err != nil {
		log.Printf("[pgguard] notify 水位读败（不判）: %v", err)
		return
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if usage < x.qMax {
		x.qAlerted = false // 恢复清标（再洪可再报）
		return
	}
	if x.qAlerted {
		return
	}
	x.qAlerted = true
	msg := fmt.Sprintf("notify 队列水位哨兵：pg_notification_queue_usage()=%.0f%%≥%.0f%%——供方洪泛实锤（洪泛先于队列爆顶可见；爆顶=发送方 NOTIFY 直接报错），先报不杀", usage*100, x.qMax*100)
	if x.al != nil {
		x.al.Alert(ctx, "notify_queue_usage", msg)
	} else {
		log.Printf("[pgguard] WARN %s", msg)
	}
}
