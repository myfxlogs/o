package notify

import (
	"context"
	"hash/fnv"
	"reflect"
	"testing"
	"time"
)

type fakeSender struct{ sent []string }

func (f *fakeSender) SendText(ctx context.Context, chatID, text string) (string, error) {
	f.sent = append(f.sent, text)
	return "m1", nil
}

func TestAlerterDebounceAndDegrade(t *testing.T) {
	f := &fakeSender{}
	a := NewAlerter(f, func() string { return "c1" }, func() bool { return true })
	ctx := context.Background()
	a.Alert(ctx, "dd_hardline_tripped", "dd=0.031")
	a.Alert(ctx, "dd_hardline_tripped", "dd=0.031") // 同指纹 5min 退避首档
	a.Alert(ctx, "dd_hardline_tripped", "dd=0.040") // 异指纹照推（各自首报即时）
	if len(f.sent) != 2 {
		t.Fatalf("去抖失败 sent=%d", len(f.sent))
	}
	// enabled=false 降级不炸
	off := NewAlerter(f, func() string { return "c1" }, func() bool { return false })
	off.Alert(ctx, "x", "y")
	if len(f.sent) != 2 {
		t.Fatal("enabled=false 仍推")
	}
	// nil 面不炸
	var nilA *Alerter
	nilA.Alert(ctx, "x", "y")
}

// TestTypedNilNoPanic typed-nil 根修锚（T-0032）：(*T)(nil) 装接口注入零 panic
// （SG creds 缺席崩环病灶的反锚——reflect 兜底防线；主修在 cmd 装配点传 nil 接口）。
func TestTypedNilNoPanic(t *testing.T) {
	var p *fakeSender
	var s Sender = p // 接口装类型化 nil——s==nil 恒 false（vet SA4023 在案=语义证明）
	if v := reflect.ValueOf(s); v.Kind() != reflect.Ptr || !v.IsNil() {
		t.Fatal("前置失效：本测试须在接口裹 nil 指针语义下跑")
	}
	a := NewAlerter(s, func() string { return "c1" }, func() bool { return true })
	a.Alert(context.Background(), "x", "y") // 修前此处 panic（nil receiver SendText）
}

// TestAlertBackoffLadder 持续告警退避锚（T-0032）：同指纹第 2/3/4/5/6+ 推须隔
// 5/10/20/40/60min（首报即时——kill_switch/dd 触线等瞬时事件不受退避）。
func TestAlertBackoffLadder(t *testing.T) {
	f := &fakeSender{}
	a := NewAlerter(f, func() string { return "c1" }, func() bool { return true })
	ctx := context.Background()
	now := time.Now()
	a.Alert(ctx, "k", "p")
	if len(f.sent) != 1 {
		t.Fatal("首报应即时")
	}
	fp := fnvOf("k|p")
	for i, want := range []time.Duration{5, 10, 20, 40, 60, 60} {
		n := i + 2 // 第 n 推
		a.mu.Lock()
		a.sent[fp] = n - 1
		a.last[fp] = now.Add(-(want - 1) * time.Minute) // 未满档——应拦
		a.mu.Unlock()
		f.sent = f.sent[:0]
		a.Alert(ctx, "k", "p")
		if len(f.sent) != 0 {
			t.Fatalf("第 %d 推间隔 %v 未满应拦", n, want)
		}
		a.mu.Lock()
		a.last[fp] = now.Add(-(want + 1) * time.Minute) // 满档——应放
		a.mu.Unlock()
		a.Alert(ctx, "k", "p")
		if len(f.sent) != 1 {
			t.Fatalf("第 %d 推间隔满 %v 应放", n, want)
		}
	}
	// 异指纹各自首报即时
	a.Alert(ctx, "k2", "p2")
	if len(f.sent) != 2 {
		t.Fatal("异指纹首报应即时")
	}
}

func fnvOf(s string) uint64 {
	f := fnv.New64a()
	_, _ = f.Write([]byte(s))
	return f.Sum64()
}
