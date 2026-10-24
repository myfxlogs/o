// Package netmeter —— 帧级字节计量+带宽预算闸（T-0057：SG ~5M 带宽防占满——
// NOX 占满致整机失联的同类事故防线）。per-channel 字节计数+60s 滑窗速率，
// 超阈两档告警（80% 预警/95% 临界——只告警不自动砍，裁量归决策席）。
package netmeter

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Meter 流面字节计量件（轻采样：按帧字节累加非全量行记——零 DB 开销）。
type Meter struct {
	mu         sync.Mutex
	bytes      map[string]int64 // channel→累计字节
	window     []sample         // 60s 滑窗（事件驱动增量非周期）
	budgetKbps int
	warned     int // 0=无 1=预警 2=临界（同档不重复推）
	alert      func(ctx context.Context, kind, msg string)
}

type sample struct {
	ts time.Time
	n  int64
}

// New budgetKbps=带宽预算（KB/s——net.budget_kbps 键喂入）。
func New(budgetKbps int, alerter func(ctx context.Context, kind, msg string)) *Meter {
	return &Meter{bytes: map[string]int64{}, budgetKbps: budgetKbps, alert: alerter}
}

// Add channel 帧字节入账（调用点=WS 收帧处——轻量 map 累加）。
func (m *Meter) Add(channel string, n int) {
	m.mu.Lock()
	m.bytes[channel] += int64(n)
	m.window = append(m.window, sample{time.Now(), int64(n)})
	cut := time.Now().Add(-60 * time.Second)
	for len(m.window) > 0 && m.window[0].ts.Before(cut) {
		m.window = m.window[1:]
	}
	m.checkLocked() // 帧到达即评估拍——事件驱动非周期扫
	m.mu.Unlock()
}

// RateKbps 60s 滑窗总速率（KB/s）。
func (m *Meter) RateKbps() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for _, s := range m.window {
		n += s.n
	}
	return float64(n) / 60 / 1024
}

// Channels 累计字节账（channel→bytes——瘦身建议数据源）。
func (m *Meter) Channels() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]int64{}
	for k, v := range m.bytes {
		out[k] = v
	}
	return out
}

// Check 外部显式评估（对账/探针面——帧内嵌 checkLocked 已是事件驱动主径）。
func (m *Meter) Check(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkLocked()
}

// checkLocked 预算闸（帧到达即评估拍——80% 预警/95% 临界；复位线 70%）。
// 调用方持锁。
func (m *Meter) checkLocked() {
	var n int64
	for _, s := range m.window {
		n += s.n
	}
	rate := float64(n) / 60 / 1024
	budget := float64(m.budgetKbps)
	if budget <= 0 {
		return
	}
	pct := rate / budget * 100
	if pct >= 95 && m.warned != 2 {
		m.warned = 2
		log.Printf("ALERT [netmeter] 带宽占用 %.0fKB/s=%.0f%%/预算 %.0fKB/s——临界档", rate, pct, budget)
		if m.alert != nil {
			m.alert(context.Background(), "net_budget_critical", fmt.Sprintf("占用 %.0fKB/s=%.0f%%——决策席裁瘦身", rate, pct))
		}
	} else if pct >= 80 && m.warned == 0 {
		m.warned = 1
		log.Printf("ALERT [netmeter] 带宽占用 %.0fKB/s=%.0f%%/预算 %.0fKB/s——预警档", rate, pct, budget)
		if m.alert != nil {
			m.alert(context.Background(), "net_budget_warn", fmt.Sprintf("占用 %.0fKB/s=%.0f%%——观察位", rate, pct))
		}
	} else if pct < 70 {
		m.warned = 0 // 复位线 70%——回落复报径开
	}
}
