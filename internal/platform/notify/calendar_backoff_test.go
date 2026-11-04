// calendar_backoff_test.go —— T-0296 桩锚：非 200 连败指数退避（×2 封顶 4h）
// +退避窗内零出向零发送、200 复常归零回 15min 拍、UA/Accept 头落、
// tzdata 真命中 DST 边界（冬令 ET=-5 非兜底 -4h）。
package notify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 锚：429 连败→退避 15m/30m/1h/2h/4h/4h 递增封顶序列+窗内复扫零出向+零推送；
// UA/Accept 请求头落桩断言。
func TestCalendarBackoffSequence(t *testing.T) {
	var hits int
	var ua, accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		ua = r.Header.Get("User-Agent")
		accept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	fs := &fakeSender{}
	c := NewCalendar(nil, fs, func() string { return "c1" }, func() bool { return true },
		func() string { return "low" }, func() int64 { return 60 })
	c.src = srv.URL
	ctx := context.Background()

	for i, d := range []time.Duration{calStep, 30 * time.Minute, time.Hour, 2 * time.Hour, calBackoffMax, calBackoffMax} {
		t0 := time.Now()
		if err := c.Scan(ctx); err != nil {
			t.Fatalf("Scan %d: %v", i, err)
		}
		if got := c.nextTry.Sub(t0); got < d-time.Second || got > d+time.Second {
			t.Fatalf("连败%d 退避=%v 期望~%v", i+1, got, d)
		}
		if c.fails != i+1 {
			t.Fatalf("连败=%d 期望 %d", c.fails, i+1)
		}
		// 退避窗内复扫=零出向（限流期不续砸——退避闸实证）
		_ = c.Scan(ctx)
		if hits != i+1 {
			t.Fatalf("退避窗内仍打源 hits=%d 期望 %d", hits, i+1)
		}
		c.mu.Lock()
		c.nextTry = time.Now().Add(-time.Second) // 快进退避窗
		c.mu.Unlock()
	}
	if len(fs.sent) != 0 {
		t.Fatalf("限流期零推送钦定，实发 %d", len(fs.sent))
	}
	if !strings.Contains(ua, "Mozilla/5.0") || !strings.Contains(accept, "xml") {
		t.Fatalf("UA/Accept 未挂: ua=%q accept=%q", ua, accept)
	}
	if c.fetchErrs != 6 {
		t.Fatalf("拉取败计数=%d 期望 6", c.fetchErrs)
	}
}

// 锚：传输错同桶连败（断源亦退避不轰炸）；403/5xx 同桶已含上测序列义。
func TestCalendarBackoffTransportErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // 死源——Do 传输错
	fs := &fakeSender{}
	c := NewCalendar(nil, fs, func() string { return "c1" }, func() bool { return true },
		func() string { return "low" }, func() int64 { return 60 })
	c.src = srv.URL
	_ = c.Scan(context.Background())
	if c.fails != 1 || c.fetchErrs != 1 || len(fs.sent) != 0 {
		t.Fatalf("传输错未进连败桶: fails=%d fetchErrs=%d sent=%d", c.fails, c.fetchErrs, len(fs.sent))
	}
}

// 锚：403 后 200=限流复常——退避档归零+事件照推+幂等锚重扫零推（DB 面）。
func TestCalendarRecoverPush(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// ET 当下+30min 高影响事件——落 [T-lead,T] 预推窗（lead=60）
	et := time.Now().In(etLoc).Add(30 * time.Minute)
	body := fmt.Sprintf(`<?xml version="1.0"?><weekly><event>`+
		`<title>CPI m/m</title><country>USD</country><date>%s</date><time>%s</time>`+
		`<impact>High</impact><forecast>0.2%%</forecast><previous>0.3%%</previous>`+
		`</event></weekly>`, et.Format("01-02-2006"), et.Format("3:04pm"))
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		io.WriteString(w, body)
	}))
	defer srv.Close()
	fs := &fakeSender{}
	c := NewCalendar(pool, fs, func() string { return "c1" }, func() bool { return true },
		func() string { return "high" }, func() int64 { return 60 })
	c.src = srv.URL

	_ = c.Scan(ctx) // 首拍 403→退避档 1
	if c.fails != 1 || len(fs.sent) != 0 {
		t.Fatalf("403 未进退避: fails=%d sent=%d", c.fails, len(fs.sent))
	}
	c.mu.Lock()
	c.nextTry = time.Now().Add(-time.Second)
	c.mu.Unlock()
	if err := c.Scan(ctx); err != nil { // 二拍 200=复常
		t.Fatal(err)
	}
	if c.fails != 0 || !c.nextTry.IsZero() {
		t.Fatalf("200 未复位退避: fails=%d nextTry=%v", c.fails, c.nextTry)
	}
	if len(fs.sent) != 1 || !strings.Contains(fs.sent[0], "CPI") || !strings.Contains(fs.sent[0], "美国") {
		t.Fatalf("复常推缺/文不符: %+v", fs.sent)
	}
	// 幂等锚落行——重扫零推不重推
	if err := c.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 {
		t.Fatalf("重推违幂等: %+v", fs.sent)
	}
}

// 锚：tzdata 真命中+DST 边界——2026-11-02 入冬令 ET=-5（8:30am ET=13:30 UTC），
// 夏令对照 -4（=12:30 UTC）；兜底 FixedZone(-4h) 病灶在案不得复发。
func TestFFTimeDSTBoundary(t *testing.T) {
	if _, err := time.LoadLocation("America/New_York"); err != nil {
		t.Skip("host tzdata 缺席——容器面 tzdata 件由 Dockerfile.core apk 验收")
	}
	if got := ffTime("11-05-2026", "8:30am"); !got.Equal(time.Date(2026, 11, 5, 13, 30, 0, 0, time.UTC)) {
		t.Fatalf("冬令 ET 错（应 -5h=13:30 UTC）: %v", got)
	}
	if got := ffTime("07-15-2026", "8:30am"); !got.Equal(time.Date(2026, 7, 15, 12, 30, 0, 0, time.UTC)) {
		t.Fatalf("夏令 ET 错（应 -4h=12:30 UTC）: %v", got)
	}
	_, off := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC).In(etLoc).Zone()
	if off != -5*3600 {
		t.Fatalf("etLoc 冬令 offset=%d 期望 -18000（兜底 -4h 病灶）", off)
	}
}
