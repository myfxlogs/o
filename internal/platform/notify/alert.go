// Alert 临界事件即时推送（T-0030 立、T-0032 根修）——复用 FeishuClient token 缓存。
// 三律（业主推送纪律 2026-10-24）：
//  1. 纯事件驱动——无事件零推送；
//  2. typed-nil 双防线：接口装类型化 nil 指针（(*FeishuClient)(nil)）时 Go 语义
//     a.cli==nil 为 false，SendText 调 nil receiver 即 panic（SG 崩环实证）——装配点
//     传 nil 接口为主修，本件 reflect 判为兜底防线（注入不炸锚双落）；
//  3. 持续告警退避：同指纹间隔 5→10→20→40→60min 顶（计数递进），首报即时——
//     kill_switch/dd 触线等瞬时事件首报不受退避；不同指纹各自计 ladder。
package notify

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"reflect"
	"sync"
	"time"
)

// Sender 发送口（FeishuClient 满足；测试注假）。
type Sender interface {
	SendText(ctx context.Context, chatID, text string) (string, error)
}

const (
	backoffBase = 5 * time.Minute
	backoffCap  = 60 * time.Minute
)

// Alerter 退避推送器（同指纹指数退避——持续态告警禁变相刷屏）。
type Alerter struct {
	cli    Sender
	chatID func() string
	on     func() bool // enabled 热启判（nil=off）
	mu     sync.Mutex
	last   map[uint64]time.Time
	sent   map[uint64]int // 同指纹已推次数（ladder 档位）
}

// NewAlerter enabled/chatID 缺席=fail-closed 降级（on=nil 或 false=只 log）。
func NewAlerter(cli Sender, chatID func() string, on func() bool) *Alerter {
	return &Alerter{cli: cli, chatID: chatID, on: on,
		last: map[uint64]time.Time{}, sent: map[uint64]int{}}
}

// cliNil typed-nil 判：接口零值或内裹 nil 指针皆判缺（反射兜底——主修在装配点）。
func cliNil(s Sender) bool {
	if s == nil {
		return true
	}
	return reflect.ValueOf(s).Kind() == reflect.Ptr && reflect.ValueOf(s).IsNil()
}

// Alert 推送（kind+fingerprint 指纹退避；禁重试——单次失败计数返回）。
func (a *Alerter) Alert(ctx context.Context, kind, payload string) {
	if a == nil || cliNil(a.cli) || a.chatID == nil || a.chatID() == "" || (a.on != nil && !a.on()) {
		return
	}
	fp := fnv.New64a()
	_, _ = fp.Write([]byte(kind + "|" + payload))
	k := fp.Sum64()
	a.mu.Lock()
	n := a.sent[k]
	if n > 0 { // 首报即时（n=0）不受退避
		wait := backoffBase << min(n-1, 4) // 5→10→20→40→60 顶（4 档封）
		if wait > backoffCap {
			wait = backoffCap
		}
		if t, ok := a.last[k]; ok && time.Since(t) < wait {
			a.mu.Unlock()
			return
		}
	}
	a.last[k] = time.Now()
	a.sent[k] = n + 1
	a.mu.Unlock()
	text := fmt.Sprintf("[PHASE告警] %s | %s | %s", kind, payload, time.Now().Format("15:04:05"))
	if _, err := a.cli.SendText(ctx, a.chatID(), text); err != nil {
		log.Printf("[alert] feishu push err (%s): %v", kind, err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
