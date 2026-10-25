// armwatch_test.go —— 臂活性看门桩测（T-0117 返修验收锚）：0 产出 2×节拍→
// 业主通道 Alert 一次（kind=research_arm_silent 入 kindSpec=飞书可达）；
// 恢复后不刷屏（抑刷屏+清标再报）。
package armwatch

import (
	"context"
	"strings"
	"testing"
	"time"

	"o/internal/platform/notify"
)

type ctxAlias = context.Context

func bg() context.Context { return context.Background() }

type capSender struct{ sent []string }

func (f *capSender) SendText(ctx ctxAlias, chatID, text string) (string, error) {
	f.sent = append(f.sent, text)
	return "m1", nil
}

// 锚：静默→告警一次（humanize 标题在载荷=业主通道形）；持续静默不刷屏；
// 恢复清标→再静默可再报。
func TestSilentAlertOnceAndRecover(t *testing.T) {
	fs := &capSender{}
	al := notify.NewAlerter(fs, func() string { return "c1" }, func() bool { return true })
	w := New("测试臂")
	w.AttachAlerter(al)
	beat := time.Hour
	w.SetBeat(beat)
	now := time.Unix(1700000000, 0)
	w.SetNow(func() time.Time { return now })
	w.OK() // t0 产出
	now = now.Add(30 * time.Minute)
	w.Check(bg()) // 窗内——零警
	if len(fs.sent) != 0 {
		t.Fatalf("窗内不得告警: %v", fs.sent)
	}
	now = now.Add(2 * time.Hour) // 超 2×beat
	w.Check(bg())
	if len(fs.sent) != 1 {
		t.Fatalf("静默应告警恰一次: %v", fs.sent)
	}
	if !strings.Contains(fs.sent[0], "研究臂静默") || !strings.Contains(fs.sent[0], "测试臂") {
		t.Fatalf("载荷应 humanize 标题+臂名: %q", fs.sent[0])
	}
	now = now.Add(time.Hour)
	w.Check(bg())
	if len(fs.sent) != 1 {
		t.Fatalf("持续静默不得刷屏: %v", fs.sent)
	}
	w.OK() // 恢复产出清标
	w.mu.Lock()
	cleared := !w.silent
	w.mu.Unlock()
	if !cleared {
		t.Fatal("恢复产出应清标")
	}
	// 再静默→标记复归（再次置位=再报资格；发送面退避由 Alerter 真钟管，
	// 已被首推断言+notify 包测试覆盖——此处断言看门判定语义非通道节流）。
	now = now.Add(3 * time.Hour)
	w.Check(bg())
	w.mu.Lock()
	remarked := w.silent
	w.mu.Unlock()
	if !remarked {
		t.Fatal("恢复后再静默应复归置标（再报资格）")
	}
}
