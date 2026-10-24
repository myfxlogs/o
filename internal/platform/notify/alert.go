// Alert 临界事件即时推送（T-0030：告警到人不等日报）——复用 FeishuClient
// token 缓存；同指纹 5min 去抖（计数照加不逐拍推）；enabled/chat_id 缺=fail-
// closed 只 log 不炸；SendText 10s 超时由 client 兜底不拖调用方节拍。
package notify

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"sync"
	"time"
)

// Sender 发送口（FeishuClient 满足；测试注假）。
type Sender interface {
	SendText(ctx context.Context, chatID, text string) (string, error)
}

// Alerter 去抖推送器（5min 同 kind+fingerprint 只推一次）。
type Alerter struct {
	cli    Sender
	chatID string
	on     func() bool // enabled 热启判（nil=off）
	mu     sync.Mutex
	last   map[uint64]time.Time
}

// NewAlerter enabled/chatID 缺席=fail-closed 降级（on=nil 或 false=只 log）。
func NewAlerter(cli Sender, chatID string, on func() bool) *Alerter {
	return &Alerter{cli: cli, chatID: chatID, on: on, last: map[uint64]time.Time{}}
}

// Alert 推送（kind+fingerprint 指纹去抖；禁重试——单次失败计数返回）。
func (a *Alerter) Alert(ctx context.Context, kind, payload string) {
	if a == nil || a.cli == nil || a.chatID == "" || (a.on != nil && !a.on()) {
		return
	}
	fp := fnv.New64a()
	_, _ = fp.Write([]byte(kind + "|" + payload))
	k := fp.Sum64()
	a.mu.Lock()
	if t, ok := a.last[k]; ok && time.Since(t) < 5*time.Minute {
		a.mu.Unlock()
		return
	}
	a.last[k] = time.Now()
	a.mu.Unlock()
	text := fmt.Sprintf("[PHASE告警] %s | %s | %s", kind, payload, time.Now().Format("15:04:05"))
	if _, err := a.cli.SendText(ctx, a.chatID, text); err != nil {
		log.Printf("[alert] feishu push err (%s): %v", kind, err)
	}
}
