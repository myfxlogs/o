// silent_test.go —— 研究员席静默看门桩测（T-0121 验收锚）：双源锚（卡 ts/
// registry Beat）窗内不报、超窗 Alert 恰一次、持续静默不刷屏、锚回新清标
// 再报资格、零行零卡 boot 宽限锚。
package researcher

import (
	"context"
	"strings"
	"testing"
	"time"

	"o/internal/platform/notify"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

type capSender struct{ sent []string }

func (f *capSender) SendText(ctx context.Context, chatID, text string) (string, error) {
	f.sent = append(f.sent, text)
	return "m1", nil
}

func setup(t *testing.T) (context.Context, *SilentWatch) {
	t.Helper()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	return context.Background(), NewSilentWatch(pool)
}

// beat 直刷研究员行（tools/hypo metrics 同径——在岗锚）。
func beat(t *testing.T, ctx context.Context, w *SilentWatch, at time.Time) {
	t.Helper()
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_employee_registry(name,dept,title,last_beat)
		VALUES('研究员','research','test',$1)
		ON CONFLICT(name) DO UPDATE SET last_beat=$1`, at); err != nil {
		t.Fatal(err)
	}
}

func card(t *testing.T, ctx context.Context, w *SilentWatch, at time.Time) {
	t.Helper()
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_opp_hypothesis(venue,surface,family,ts)
		VALUES('okx','test-surface','carry',$1)
		ON CONFLICT (venue,surface) DO NOTHING`, at); err != nil {
		t.Fatal(err)
	}
}

// 锚：零行零卡→boot 宽限；超窗→Alert 恰一次（humanize 标题在载荷）；
// 持续静默不刷屏；Beat 回新清标；再超窗复归置标。
func TestResearcherSilentOnceAndRecover(t *testing.T) {
	ctx, w := setup(t)
	fs := &capSender{}
	w.AttachAlerter(notify.NewAlerter(fs, func() string { return "c1" }, func() bool { return true }))
	base := time.Unix(1700000000, 0)
	w.SetNow(func() time.Time { return base })
	w.boot = base

	w.Check(ctx) // 零行零卡——boot 宽限窗内零警
	if len(fs.sent) != 0 {
		t.Fatalf("宽限窗内不得告警: %v", fs.sent)
	}
	base = base.Add(silentWindow + time.Hour)
	w.Check(ctx)
	if len(fs.sent) != 1 {
		t.Fatalf("超窗应告警恰一次: %v", fs.sent)
	}
	if !strings.Contains(fs.sent[0], "研究臂静默") || !strings.Contains(fs.sent[0], "研究员席") {
		t.Fatalf("载荷应 humanize 标题+席名: %q", fs.sent[0])
	}
	w.Check(ctx)
	if len(fs.sent) != 1 {
		t.Fatalf("持续静默不得刷屏: %v", fs.sent)
	}
	// 锚回新（Beat 刷讫）→清标；再超窗→置标记（再报资格）。
	base = base.Add(24 * time.Hour)
	beat(t, ctx, w, base)
	w.Check(ctx)
	if len(fs.sent) != 1 {
		t.Fatalf("锚回新应清标不告警: %v", fs.sent)
	}
	base = base.Add(silentWindow + time.Hour)
	w.Check(ctx)
	w.mu.Lock()
	remarked := w.alerted
	w.mu.Unlock()
	if !remarked {
		t.Fatal("恢复后再静默应复归置标（再报资格）")
	}
	// 第二次发送面由 Alerter 真钟退避管——断言看门判定语义非通道节流
	// （同 armwatch_test 口径）。
}

// 锚双源取大：旧卡+新 Beat=在岗不报；新卡+无行=产出不报；卡旧 Beat 陈→报。
func TestResearcherSilentAnchorDualSource(t *testing.T) {
	ctx, w := setup(t)
	fs := &capSender{}
	w.AttachAlerter(notify.NewAlerter(fs, func() string { return "c1" }, func() bool { return true }))
	base := time.Unix(1700000000, 0)
	w.SetNow(func() time.Time { return base })
	w.boot = base.Add(-2 * silentWindow) // boot 宽限已过——锚全凭 DB 双源

	card(t, ctx, w, base.Add(-silentWindow-time.Hour)) // 卡陈
	beat(t, ctx, w, base)                              // Beat 新——锚取大=在岗
	w.Check(ctx)
	if len(fs.sent) != 0 {
		t.Fatalf("Beat 新应在岗不报: %v", fs.sent)
	}

	w.mu.Lock()
	w.alerted = false
	w.mu.Unlock()
	card(t, ctx, w, base.Add(-silentWindow-time.Hour)) // 旧卡（UNIQUE 面已占——不增）
	beat(t, ctx, w, base.Add(-silentWindow-time.Hour)) // Beat 也陈
	w.Check(ctx)
	if len(fs.sent) != 1 {
		t.Fatalf("双源皆陈应告警: %v", fs.sent)
	}
}
