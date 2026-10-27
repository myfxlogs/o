// Package researcher —— 研究员席静默看门（T-0121/D-0016 Ⅲ）。研究员=短会话
// 岗（周节拍+手动，同告警分诊架构）——无常驻臂拍可挂 armwatch.Watch，改锚
// DB 双源：fin_opp_hypothesis max(ts)（产出锚）与 fin_employee_registry
// 「研究员」行 last_beat（在岗锚，tools/hypo metrics 刷）。双源皆超
// silentWindow（14 天）零更新=research_arm_silent 一次（恢复清标再报，
// 抑刷屏同 armwatch 语义）；双源皆无（行/卡零）=boot 锚起算（上岗宽限窗）。
package researcher

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/notify"
	"o/internal/platform/roster"
)

const silentWindow = 14 * 24 * time.Hour

// SilentWatch 研究员席静默判定体。
type SilentWatch struct {
	pool    *pgxpool.Pool
	al      *notify.Alerter
	boot    time.Time // 双源皆无时的宽限锚（进程 boot 注入）
	mu      sync.Mutex
	alerted bool
	now     func() time.Time // 时钟注入面（测试）——nil=time.Now
}

// NewSilentWatch 构造（boot=进程启动时点——宽限窗基准）。
func NewSilentWatch(pool *pgxpool.Pool) *SilentWatch {
	return &SilentWatch{pool: pool, boot: time.Now()}
}

// AttachAlerter 挂业主通道（nil=log 退路——测试/无通知装配）。
func (w *SilentWatch) AttachAlerter(a *notify.Alerter) { w.al = a }

// SetNow 时钟注入（桩测期界——生产零调用）。
func (w *SilentWatch) SetNow(fn func() time.Time) { w.now = fn }

func (w *SilentWatch) clock() time.Time {
	if w.now != nil {
		return w.now()
	}
	return time.Now()
}

// newerOf 双源锚取新（GREATEST 语义——NULL 旁路，双 NULL=无效）。
func newerOf(a, b sql.NullTime) sql.NullTime {
	switch {
	case !a.Valid:
		return b
	case !b.Valid:
		return a
	case a.Time.After(b.Time):
		return a
	default:
		return b
	}
}

// Run 在岗（小时节拍判定——短会话岗的看门由常驻 core 代守）。
func (w *SilentWatch) Run(ctx context.Context) {
	t := time.NewTicker(time.Hour) // sched-exempt-ok 看门判定节拍（纯本地比对锚查询——研究员短会话无臂拍，core 代守兜底非轮询拉取）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.Check(ctx)
		}
	}
}

// Check 静默判定：锚=max(最新卡 ts, 研究员行 last_beat)；皆无=boot 锚。
// 双源分域拆读（T-0172——产出锚在 md 池，在岗锚 fin_employee_registry 归
// fin 域册走 roster 钉池路由；原跨域 GREATEST 单句在 md 池恒 42P01 fail-closed
// 不判=看门全盲）。任一源读败=fail-closed 不判。
// 超窗零更新→Alert 一次；锚回新（新卡/新 Beat）→清标再报资格。
func (w *SilentWatch) Check(ctx context.Context) {
	var card, beat sql.NullTime
	if err := w.pool.QueryRow(ctx,
		"SELECT max(ts) FROM fin_opp_hypothesis WHERE evidence->>'provenance'='researcher'").Scan(&card); err != nil {
		log.Printf("[research-silent] 研究员锚读败（fail-closed 不判）: %v", err)
		return
	}
	beat, err := roster.LastBeat(ctx, w.pool, "研究员")
	if err != nil {
		log.Printf("[research-silent] 研究员锚读败（fail-closed 不判）: %v", err)
		return
	}
	anchor := newerOf(card, beat)
	a := w.boot // 双源皆无（零行零卡）——boot 宽限锚
	if anchor.Valid {
		a = anchor.Time
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.clock().Sub(a) <= silentWindow {
		w.alerted = false // 锚回新清标（再静默可再报）
		return
	}
	if w.alerted {
		return
	}
	w.alerted = true
	msg := "研究员席超 14 天零产出（无新假想卡+registry 心跳陈旧）——外部进水口停摆（D-0016 Ⅲ），周节拍桥面或起草链路断"
	if w.al != nil {
		w.al.Alert(ctx, "research_arm_silent", msg)
	} else {
		log.Printf("[research_arm_silent] %s", msg)
	}
}
