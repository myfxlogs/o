package notify

import (
	"context"
	"strings"
	"testing"
)

// TestEventPushOffNoSend 开关缺/false→零发（fail-closed 静默锚）。
func TestEventPushOffNoSend(t *testing.T) {
	f := &fakeSender{}
	e := NewEventPush(nil, f, func() string { return "oc_x" }, func() bool { return false })
	_ = e.Scan(context.Background())
	if len(f.sent) != 0 {
		t.Fatalf("off 应零发 got=%v", f.sent)
	}
}

// TestEventPushDedup 同 key 一键一推——重注入不重复。
func TestEventPushDedup(t *testing.T) {
	f := &fakeSender{}
	e := NewEventPush(nil, f, func() string { return "oc_x" }, func() bool { return true })
	ctx := context.Background()
	e.push(ctx, "opened:1", "开仓 BTC-USDT-SWAP")
	e.push(ctx, "opened:1", "开仓 BTC-USDT-SWAP")
	if len(f.sent) != 1 || !strings.Contains(f.sent[0], "[经营事件]") {
		t.Fatalf("去重推=%v", f.sent)
	}
}
