package notify

import (
	"context"
	"strings"
	"testing"
)

// fakeCardSender 双径假件——SendText/CreateInteractive 分录（chat 列断言路由）。
type fakeCardSender struct {
	texts []string // "chat|text"
	cards []string // "chat|cardJSON"
}

func (f *fakeCardSender) SendText(ctx context.Context, chatID, text string) (string, error) {
	f.texts = append(f.texts, chatID+"|"+text)
	return "t", nil
}
func (f *fakeCardSender) CreateInteractive(ctx context.Context, chatID, content string) (string, error) {
	f.cards = append(f.cards, chatID+"|"+content)
	return "c", nil
}

// TestSysChannelRouting 双频分流锚（业主令 2026-11-15）：钱 kinds→主频文本；
// 系统 kinds→sys 频富卡；sys 空=回落主频；无 CardSender=系统件文本降级。
func TestSysChannelRouting(t *testing.T) {
	ctx := context.Background()

	// ① 双频齐：钱走主频文本，系统走 sys 卡
	f := &fakeCardSender{}
	a := NewAlerter(f, func() string { return "money" }, func() bool { return true }).
		WithSysChat(func() string { return "sys" })
	a.Alert(ctx, "kill_switch", "on")     // 钱径集
	a.Alert(ctx, "cognition_dead", "2h")  // 系统集
	a.Alert(ctx, "newkind_unlisted", "x") // 未定性=默认系统
	if len(f.texts) != 1 || !strings.HasPrefix(f.texts[0], "money|") {
		t.Fatalf("钱 kind 未走主频文本: %+v", f.texts)
	}
	if len(f.cards) != 2 || !strings.HasPrefix(f.cards[0], "sys|") ||
		!strings.HasPrefix(f.cards[1], "sys|") {
		t.Fatalf("系统 kind 未走 sys 卡: %+v", f.cards)
	}
	if !strings.Contains(f.cards[0], "interactive") && !strings.Contains(f.cards[0], "header") {
		t.Fatal("卡体非 interactive 装配")
	}

	// ② sys 空=回落主频零漏件
	f2 := &fakeCardSender{}
	a2 := NewAlerter(f2, func() string { return "money" }, func() bool { return true }).
		WithSysChat(func() string { return "" })
	a2.Alert(ctx, "cognition_dead", "2h")
	if len(f2.cards) != 1 || !strings.HasPrefix(f2.cards[0], "money|") {
		t.Fatalf("sys 空未回落主频: %+v", f2.cards)
	}

	// ③ 无 CardSender 实现=系统件文本降级仍落 sys 频
	f3 := &fakeSender{}
	a3 := NewAlerter(f3, func() string { return "money" }, func() bool { return true }).
		WithSysChat(func() string { return "sys" })
	a3.Alert(ctx, "cognition_dead", "2h")
	if len(f3.sent) != 1 {
		t.Fatalf("文本降级未发: %+v", f3.sent)
	}
}
