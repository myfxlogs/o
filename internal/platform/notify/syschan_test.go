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
// 系统 kinds→sys 频富卡（cli+chat 成对翻——跨对=机器人不在席发送必败）；
// sysCli/sysChat 任一缺=整对回落主频；无 CardSender=系统件文本降级。
func TestSysChannelRouting(t *testing.T) {
	ctx := context.Background()

	// ① 双频齐：钱走主频文本，系统走 sys 机器人卡
	m := &fakeCardSender{} // 主机器人
	s := &fakeCardSender{} // 系统机器人
	a := NewAlerter(m, func() string { return "money" }, func() bool { return true }).
		WithSysChat(func() string { return "sys" }).WithSysSender(s)
	a.Alert(ctx, "kill_switch", "on")     // 钱径集
	a.Alert(ctx, "cognition_dead", "2h")  // 系统集
	a.Alert(ctx, "newkind_unlisted", "x") // 未定性=默认系统
	if len(m.texts) != 1 || !strings.HasPrefix(m.texts[0], "money|") {
		t.Fatalf("钱 kind 未走主频文本: %+v", m.texts)
	}
	if len(m.cards)+len(m.texts) != 1 {
		t.Fatalf("主频混入系统件: %+v %+v", m.texts, m.cards)
	}
	if len(s.cards) != 2 || !strings.HasPrefix(s.cards[0], "sys|") ||
		!strings.HasPrefix(s.cards[1], "sys|") {
		t.Fatalf("系统 kind 未走 sys 卡: %+v", s.cards)
	}
	if !strings.Contains(s.cards[0], "header") {
		t.Fatal("卡体非 interactive 装配")
	}

	// ② sysCli 缺=整对回落主频（sysChat 独设不生效——跨对禁）
	m2, s2 := &fakeCardSender{}, &fakeCardSender{}
	a2 := NewAlerter(m2, func() string { return "money" }, func() bool { return true }).
		WithSysChat(func() string { return "sys" })
	_ = s2 // 未挂
	a2.Alert(ctx, "cognition_dead", "2h")
	if len(m2.texts) != 1 || !strings.HasPrefix(m2.texts[0], "money|") {
		t.Fatalf("sysCli 缺未回落主频文本: %+v", m2.texts)
	}

	// ③ sysChat 空=整对回落主频（sysCli 独在不生效）
	m3, s3 := &fakeCardSender{}, &fakeCardSender{}
	a3 := NewAlerter(m3, func() string { return "money" }, func() bool { return true }).
		WithSysChat(func() string { return "" }).WithSysSender(s3)
	a3.Alert(ctx, "cognition_dead", "2h")
	if len(m3.texts) != 1 || !strings.HasPrefix(m3.texts[0], "money|") {
		t.Fatalf("sysChat 空未回落主频: %+v", m3.texts)
	}
	if len(s3.cards)+len(s3.texts) != 0 {
		t.Fatalf("跨对误发 sysCli→主频: %+v %+v", s3.texts, s3.cards)
	}

	// ④ 系统 cli 无 CardSender=文本降级仍落 sys 频
	m4, s4 := &fakeCardSender{}, &fakeSender{}
	a4 := NewAlerter(m4, func() string { return "money" }, func() bool { return true }).
		WithSysChat(func() string { return "sys" }).WithSysSender(s4)
	a4.Alert(ctx, "cognition_dead", "2h")
	if len(s4.sent) != 1 || len(m4.texts)+len(m4.cards) != 0 {
		t.Fatalf("文本降级未落 sys: %+v", s4.sent)
	}
}
