// Package armwatch —— 研究臂活性看门（T-0117：「fail-closed 让位无告警=静默
// 死亡」四臂共用件——funding/dcd/gapscan/hypothesis 共享，非四份复制）。
// 语义：臂拍成功产出=OK()；节拍窗（2×beat）零 OK=ALERT [research_arm_silent]
// 一次（走 notify.Alerter.Alert 业主通道非 log 前缀——静默告警禁自静默）；
// 恢复产出清标可再报（抑刷屏）。
package armwatch

import (
	"context"
	"o/internal/platform/config"
	"sync"
	"time"

	"o/internal/platform/notify"
)

// Watch 单臂活性锚（零状态外溢——臂拍内同步调用无锁争用热区）。
type Watch struct {
	mu     sync.Mutex
	name   string
	beat   time.Duration // 判定窗基准（=臂节拍；2×beat 零产出=静默）
	al     *notify.Alerter
	lastOK time.Time
	silent bool             // 静默告警已发（恢复清标）
	now    func() time.Time // 时钟注入面（测试）——nil=time.Now
}

// SetNow 时钟注入（桩测期界——生产零调用）。
func (w *Watch) SetNow(fn func() time.Time) { w.now = fn }

func (w *Watch) clock() time.Time {
	if w.now != nil {
		return w.now()
	}
	return time.Now()
}

// New 构造（name=臂名入告警载荷）。
func New(name string) *Watch {
	return &Watch{name: name}
}

// AttachAlerter 挂业主通道（nil=log 退路——测试/无通知装配）。
func (w *Watch) AttachAlerter(a *notify.Alerter) { w.al = a }

// SetBeat 节拍窗基准（臂读热启键后注入）。
func (w *Watch) SetBeat(d time.Duration) { w.beat = d }

// OK 成功产出拍（拍内实质产出落位后调——读败让位拍不调）。
func (w *Watch) OK() {
	w.mu.Lock()
	w.lastOK = w.clock()
	w.silent = false // 恢复产出清标（再静默可再报）
	w.mu.Unlock()
}

// Check 静默判定（臂每拍首调）：首拍前不判（启动窗）；超 2×beat 零 OK=告警
// 一次抑刷屏。
func (w *Watch) Check(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.beat <= 0 || w.lastOK.IsZero() || w.silent {
		return
	}
	if w.clock().Sub(w.lastOK) <= 2*w.beat {
		return
	}
	w.silent = true
	msg := "研究臂 " + w.name + " 超 " + (2 * w.beat).Round(time.Minute).String() +
		" 零成功产出——fail-closed 让位循环（限频/读败），产出面停摆"
	if w.al != nil {
		w.al.Alert(ctx, "research_arm_silent", msg)
	}
}

// DeadlineCtx 期界包络共享件（T-0244——各臂 scan 顶层一行挂：cfg 键
// <域>.scan_timeout_s 热启，缺=def；ctx 透下游全链有界，挂死径封）。
func DeadlineCtx(ctx context.Context, cfg *config.Client, key string, def time.Duration) (context.Context, context.CancelFunc) {
	to := def
	if v, err := cfg.GetFloat(key); err == nil && v > 0 {
		to = time.Duration(v) * time.Second
	}
	return context.WithTimeout(ctx, to)
}
