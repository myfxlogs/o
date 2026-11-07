// scanwm_test.go —— T-0440③a 桩测：零行史族首见锚起→跨阈升件行落（族=0
// 升件行落——票面验收④）/陈水水位即时达阈/鲜落复常清零（滞回）/注册面
// 缺席=不可判升件/nil 守位。Exec 捕获桩断言载荷。
package scanwm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// escStub Exec 捕获桩（零去重模拟——载荷断言位；note=args[2]）。
type escStub struct {
	notes []string
}

func (z *escStub) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	if len(args) > 2 {
		if note, ok := args[2].(string); ok {
			z.notes = append(z.notes, note)
		}
	}
	return pgconn.CommandTag{}, nil
}

func newTestGate(esc *escStub, fams []string, landed Landed, now func() time.Time) *Gate {
	g := New(esc, "test-seat", "T-0440", "testvenue", fams, landed)
	g.now = now
	return g
}

// TestZeroRowFamilyEscalates 验收④锚：族=0（零行史）→首见锚起→阈满升件
// 行落（谱域缺行 scope=venue:族）。鲜落族水位随钟前进（实拍每拍刷recv_ts）。
func TestZeroRowFamilyEscalates(t *testing.T) {
	esc := &escStub{}
	cur := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	landed := func(_ context.Context, fam string) (*time.Time, error) {
		if fam == "swap" {
			wm := cur.Add(-30 * time.Second) // 鲜落——随钟前进
			return &wm, nil
		}
		return nil, nil // spot 零行史（病灶形态——T-0440 SPOT=0）
	}
	g := newTestGate(esc, []string{"swap", "spot"}, landed, func() time.Time { return cur })
	g.Beat(context.Background()) // spot 首见锚起
	if len(esc.notes) != 0 {
		t.Fatalf("首见拍不应升件: %v", esc.notes)
	}
	cur = cur.Add(9 * time.Minute)
	g.Beat(context.Background()) // 裕度未满
	if len(esc.notes) != 0 {
		t.Fatalf("阈内不应升件: %v", esc.notes)
	}
	cur = cur.Add(10 * time.Minute)
	g.Beat(context.Background()) // 跨阈——升件
	if len(esc.notes) != 1 {
		t.Fatalf("跨阈应升件 1 行: %v", esc.notes)
	}
	if !strings.Contains(esc.notes[0], "spot 族零鲜落") || !strings.Contains(esc.notes[0], "零行史") {
		t.Fatalf("升件 note 应携族/零行史锚: %s", esc.notes[0])
	}
}

// TestStaleWatermarkEscalates 即时达阈：有史陈水（水位龄≥阈）——水位龄即
// 持续实证，单拍即升（重启复拍不重等）。
func TestStaleWatermarkEscalates(t *testing.T) {
	esc := &escStub{}
	t0 := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	landed := func(_ context.Context, _ string) (*time.Time, error) {
		wm := t0.Add(-11 * time.Minute) // 陈水 11min
		return &wm, nil
	}
	g := newTestGate(esc, []string{"spot"}, landed, func() time.Time { return t0 })
	g.Beat(context.Background())
	if len(esc.notes) != 1 || !strings.Contains(esc.notes[0], "末落") {
		t.Fatalf("陈水应即时升件携末落锚: %v", esc.notes)
	}
}

// TestFreshRecoveryClears 滞回复常清零：缺行锚起后复鲜——清零不升；再缺
// 重锚（不陈账）。
func TestFreshRecoveryClears(t *testing.T) {
	esc := &escStub{}
	t0 := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	cur := t0
	wmFresh := t0
	stale := false
	landed := func(_ context.Context, _ string) (*time.Time, error) {
		if stale {
			return nil, nil
		}
		return &wmFresh, nil
	}
	g := newTestGate(esc, []string{"spot"}, landed, func() time.Time { return cur })
	g.Beat(context.Background()) // 鲜
	stale = true
	cur = t0.Add(5 * time.Minute)
	g.Beat(context.Background()) // 首见缺
	cur = t0.Add(8 * time.Minute)
	stale = false
	g.Beat(context.Background()) // 复鲜——清零
	stale = true
	cur = t0.Add(12 * time.Minute)
	g.Beat(context.Background()) // 再缺——重锚（距首见 4min<阈）
	if len(esc.notes) != 0 {
		t.Fatalf("复常清零后重锚不应陈账升件: %v", esc.notes)
	}
	cur = t0.Add(22 * time.Minute) // 重锚+10min
	g.Beat(context.Background())
	if len(esc.notes) != 1 {
		t.Fatalf("重锚跨阈应升件: %v", esc.notes)
	}
}

// TestUndecidableOnEmptyRegistration 注册面缺席=闸自身 fail-closed 升
// 「不可判」——非静默放行。
func TestUndecidableOnEmptyRegistration(t *testing.T) {
	esc := &escStub{}
	t0 := time.Now()
	g := newTestGate(esc, nil, func(_ context.Context, _ string) (*time.Time, error) { return nil, nil },
		func() time.Time { return t0 })
	g.Beat(context.Background())
	if len(esc.notes) != 1 || !strings.Contains(esc.notes[0], "不可判") {
		t.Fatalf("注册面缺席应升不可判: %v", esc.notes)
	}
}

// TestNilGateGuard nil 闸/缺件装配 Beat 零炸（观测臂缺席不拖主径）。
func TestNilGateGuard(t *testing.T) {
	var g *Gate
	g.Beat(context.Background()) // nil 收者守位
	esc := &escStub{}
	h := New(esc, "s", "t", "v", []string{"f"}, nil) // landed 缺件
	h.Beat(context.Background())
	if len(esc.notes) != 0 {
		t.Fatalf("缺件应守位: %v", esc.notes)
	}
}
