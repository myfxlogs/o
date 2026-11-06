package notify

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0410 锚②提醒实证：造 event_date=今/明日 行→下一拍提醒行落（渠道=现有
// 告警扇出径——Sender+chatID+enabled 装配同 price_alert）；内容=事件
// kind+title+date，时刻缺如实标「日级精度」。已提醒锚=ops_event `宏观提醒`
// 行（送达才记账 T-0093 序——推败/凭据缺不占位下拍重试，escalate 自报）。

func TestMacroRemindTomorrowPushesOnce(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 种键断言（price_alert 同钉——fixture on()=true 桩与生产种值分歧案防再发）。
	var seed string
	if err := pool.QueryRow(ctx, `SELECT value::text FROM app_config
		WHERE key='notify.feishu.macro_remind.enabled'`).Scan(&seed); err != nil || seed != "true" {
		t.Fatalf("开关键种应='true': %q err=%v", seed, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_macro_event(event_date,kind,title)
		VALUES(current_date+1,'nfp','桩事件明日')`); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSender{}
	m := NewMacroRemind(pool, fs, func() string { return "c1" }, func() bool { return true })
	if err := m.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 || !strings.Contains(fs.sent[0], "明日") ||
		!strings.Contains(fs.sent[0], "nfp") || !strings.Contains(fs.sent[0], "桩事件明日") ||
		!strings.Contains(fs.sent[0], "日级精度") {
		t.Fatalf("推送文不符: %+v", fs.sent)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='宏观提醒' AND ticket='T-0410'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("宏观提醒行应=1: %d err=%v", n, err)
	}
	// 幂等锚：重扫零推零行（一事件一提醒）
	if err := m.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 {
		t.Fatalf("重扫重推违幂等: %d", len(fs.sent))
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='宏观提醒'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("重扫提醒行应仍=1: %d", n)
	}
}

// 今日行同推（今日/明日双窗——标「今日」非明日）。
func TestMacroRemindTodayPushes(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_macro_event(event_date,kind,title)
		VALUES(current_date,'fomc','桩事件今日')`); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSender{}
	m := NewMacroRemind(pool, fs, func() string { return "c1" }, func() bool { return true })
	if err := m.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 || !strings.Contains(fs.sent[0], "今日") ||
		!strings.Contains(fs.sent[0], "fomc") {
		t.Fatalf("今日推送文不符: %+v", fs.sent)
	}
}

// 窗外行不推（后日=非今/明双窗）。
func TestMacroRemindOutOfWindowQuiet(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_macro_event(event_date,kind,title)
		VALUES(current_date+2,'cpi','桩事件后日')`); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSender{}
	m := NewMacroRemind(pool, fs, func() string { return "c1" }, func() bool { return true })
	if err := m.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 0 {
		t.Fatalf("窗外行不应推: %+v", fs.sent)
	}
}

// 关态=不扫不记零行（复归后窗内行照推——不占位语义）。
func TestMacroRemindOffSkips(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_macro_event(event_date,kind,title)
		VALUES(current_date,'nfp','桩事件关态')`); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSender{}
	m := NewMacroRemind(pool, fs, func() string { return "c1" }, func() bool { return false })
	if err := m.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 0 {
		t.Fatalf("关态不应推: %+v", fs.sent)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='宏观提醒'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("关态不应落提醒行: %d", n)
	}
	// 复归后同窗行照推（关态未占位）
	m2 := NewMacroRemind(pool, fs, func() string { return "c1" }, func() bool { return true })
	if err := m2.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 {
		t.Fatalf("复归后应补推: %d", len(fs.sent))
	}
}

// 推送败=不落提醒锚（票不丢下拍重推）+升件自报行；败愈重拍=补推。
func TestMacroRemindPushFailRetries(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_macro_event(event_date,kind,title)
		VALUES(current_date,'nfp','桩事件推败')`); err != nil {
		t.Fatal(err)
	}
	m := NewMacroRemind(pool, &failSender{}, func() string { return "c1" }, func() bool { return true })
	if err := m.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='宏观提醒'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("推败不应落提醒锚（送达才记账）: %d", n)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='NOTIFY-MACRO-REMIND'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("推败自报升件行应=1: %d", n)
	}
	// 败愈重拍=补推+落锚
	fs := &fakeSender{}
	m2 := NewMacroRemind(pool, fs, func() string { return "c1" }, func() bool { return true })
	if err := m2.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 {
		t.Fatalf("败愈应补推: %d", len(fs.sent))
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='宏观提醒'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("败愈提醒行应=1: %d", n)
	}
}
