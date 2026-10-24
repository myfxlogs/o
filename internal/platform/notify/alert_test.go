package notify

import (
	"context"
	"testing"
)

type fakeSender struct{ sent []string }

func (f *fakeSender) SendText(ctx context.Context, chatID, text string) (string, error) {
	f.sent = append(f.sent, text)
	return "m1", nil
}

func TestAlerterDebounceAndDegrade(t *testing.T) {
	f := &fakeSender{}
	a := NewAlerter(f, "c1", func() bool { return true })
	ctx := context.Background()
	a.Alert(ctx, "dd_hardline_tripped", "dd=0.031")
	a.Alert(ctx, "dd_hardline_tripped", "dd=0.031") // 同指纹 5min 去抖
	a.Alert(ctx, "dd_hardline_tripped", "dd=0.040") // 异指纹照推
	if len(f.sent) != 2 {
		t.Fatalf("去抖失败 sent=%d", len(f.sent))
	}
	// enabled=false 降级不炸
	off := NewAlerter(f, "c1", func() bool { return false })
	off.Alert(ctx, "x", "y")
	if len(f.sent) != 2 {
		t.Fatal("enabled=false 仍推")
	}
	// nil 面不炸
	var nilA *Alerter
	nilA.Alert(ctx, "x", "y")
}
