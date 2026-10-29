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

// T-0243 层③锚：never-lived 臂 boot 起 2×beat 零 OK→告警落（旧语义=永不报翻案）。
func TestNeverLivedAlerts(t *testing.T) {
	var fakeNow = time.Now()
	w := New("dcd")
	w.SetNow(func() time.Time { return fakeNow })
	w.AttachAlerter(&notify.Alerter{})
	w.SetBeat(time.Minute)
	w.Check(context.Background()) // 首拍定 boot
	fakeNow = fakeNow.Add(90 * time.Second)
	w.Check(context.Background()) // 1.5×beat 窗内不警
	if w.silent {
		t.Fatal("窗内误警——never-lived 提前触发")
	}
	fakeNow = fakeNow.Add(80 * time.Second)
	w.Check(context.Background()) // >2×beat never-lived→警
	if !w.silent {
		t.Fatal("never-lived 窗外未警——boot 基线失效")
	}
	w.OK()
	if w.silent {
		t.Fatal("OK 未清标")
	}
	fakeNow = fakeNow.Add(3 * time.Hour)
	w.Check(context.Background()) // 有 lastOK 后按 lastOK 基线同判
	if !w.silent {
		t.Fatal("lastOK 基线窗外未警")
	}
}
