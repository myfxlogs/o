// calendar_escalate_test.go —— T-0410 锚③零静默：feed 拉取败/空返/解包败
// =升件不静默（事实律——原仅 log+计数，断更两日零升件病灶根修）；
// 升件同窗 24h 去重不暴；pool 缺席（桩测面）不炸不升件。
package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestCalendarFetchFailEscalates 锚③a：死源传输错→升件（esc_class='读败'
// esc_scope='CALENDAR-FEED'）+同窗去重；原静默计数面根修。
func TestCalendarFetchFailEscalates(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // 死源——Do 传输错
	c := NewCalendar(pool, &fakeSender{}, func() string { return "c1" }, func() bool { return true },
		func() string { return "high" }, func() int64 { return 60 })
	c.src = srv.URL
	if err := c.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='读败' AND esc_scope='CALENDAR-FEED'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("拉取败升件行应=1: %d err=%v", n, err)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_scope='CALENDAR-FEED'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "拉取") {
		t.Fatalf("升件 note 缺拉取相位: %s", note)
	}
	// 同窗去重：快进退避再败仍=1
	c.mu.Lock()
	c.nextTry = time.Now().Add(-time.Second)
	c.mu.Unlock()
	_ = c.Scan(ctx)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='CALENDAR-FEED'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件同窗应去重仍=1: %d", n)
	}
}

// TestCalendarHTTPFailEscalates 非 200 同桶升件（429/5xx 面——T-0296 退避
// 不抵升件义务，零静默事实律两径同闸）。
func TestCalendarHTTPFailEscalates(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := NewCalendar(pool, &fakeSender{}, func() string { return "c1" }, func() bool { return true },
		func() string { return "high" }, func() int64 { return 60 })
	c.src = srv.URL
	if err := c.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='CALENDAR-FEED'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("非200升件行应=1: %d", n)
	}
}

// TestCalendarEmptyFeedEscalates 锚③b：200 但零事件行=空返升件
// （上游周历空返=断更前兆——对拍锚落账非静默滑过）。
func TestCalendarEmptyFeedEscalates(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<?xml version="1.0"?><weekly></weekly>`))
	}))
	defer srv.Close()
	c := NewCalendar(pool, &fakeSender{}, func() string { return "c1" }, func() bool { return true },
		func() string { return "high" }, func() int64 { return 60 })
	c.src = srv.URL
	if err := c.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	var note string
	if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(max(note),'') FROM ops_event
		WHERE event='升件' AND esc_scope='CALENDAR-FEED'`).Scan(&n, &note); err != nil || n != 1 {
		t.Fatalf("空返升件行应=1: %d err=%v", n, err)
	}
	if !strings.Contains(note, "空返") {
		t.Fatalf("升件 note 缺空返相位: %s", note)
	}
}

// TestCalendarDecodeFailEscalates 锚③c：200 但非 XML=解包败升件（原仅 log）。
func TestCalendarDecodeFailEscalates(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<?xml version="1.0"?><weekly><event><title>trunc`)) // 截断畸形 XML——解包必败
	}))
	defer srv.Close()
	c := NewCalendar(pool, &fakeSender{}, func() string { return "c1" }, func() bool { return true },
		func() string { return "high" }, func() int64 { return 60 })
	c.src = srv.URL
	if err := c.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	var note string
	if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(max(note),'') FROM ops_event
		WHERE event='升件' AND esc_scope='CALENDAR-FEED'`).Scan(&n, &note); err != nil || n != 1 {
		t.Fatalf("解包败升件行应=1: %d err=%v", n, err)
	}
	if !strings.Contains(note, "解包") {
		t.Fatalf("升件 note 缺解包相位: %s", note)
	}
}

// TestCalendarEscalateNilPoolSafe pool 缺席（桩测面 NewCalendar(nil,...)）=
// 不炸不升件——结构性限度静默退（升件不能再升）。
func TestCalendarEscalateNilPoolSafe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	c := NewCalendar(nil, &fakeSender{}, func() string { return "c1" }, func() bool { return true },
		func() string { return "high" }, func() int64 { return 60 })
	c.src = srv.URL
	if err := c.Scan(context.Background()); err != nil {
		t.Fatalf("nil 池 Scan 不应上抛: %v", err)
	}
}
