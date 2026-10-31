// watch_hb_test.go —— T-0269 桩锚：Watch 应用层心跳帧（事件静默期 ≤期界
// 一帧——h2 idle 杀流残余风险关窗）+ NOTIFY 事件透传不被心跳环饿死
// + favicon 显式挂接返真件（HTML-as-favicon 病灶封口）。
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"

	"o/internal/gen/dashboardv1"
	"o/internal/gen/dashboardv1/dashboardv1connect"
	"o/web"
)

// 心跳桩锚：期界缩至 60ms（var 缝）——watch_open 后静默即按拍出 watch_hb
// 纯信号帧（无载荷）；帧间隔实测不超期界 4 倍（调度毛刺余量）。
func TestWatchHeartbeatFrames(t *testing.T) {
	ctx, w, svc := trxTestSetup(t)
	old := watchHeartbeat
	watchHeartbeat = 60 * time.Millisecond
	defer func() { watchHeartbeat = old }()
	mux := http.NewServeMux()
	mountCore(mux, svc)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cl := dashboardv1connect.NewCoreServiceClient(connect.NewClient(
		connecthttp.NewTransport(srv.Client(), srv.URL)))

	wctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	stream, err := cl.Watch(wctx, &dashboardv1.WatchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	open, err := stream.Receive()
	if err != nil || open.Kind != "watch_open" {
		t.Fatalf("首帧应 watch_open: %v %+v", err, open)
	}
	// 静默窗（无 NOTIFY）连收两拍心跳——节拍语义+间隔判据。
	var last time.Time
	for i := 0; i < 2; i++ {
		ev, err := stream.Receive()
		if err != nil {
			t.Fatalf("心跳帧 %d 未达: %v", i, err)
		}
		if ev.Kind != "watch_hb" {
			t.Fatalf("静默期帧应 watch_hb: %s", ev.Kind)
		}
		if ev.GetData() != nil {
			t.Fatalf("心跳帧应无载荷（纯信号）: %v", ev.GetData())
		}
		if !last.IsZero() {
			if gap := time.Since(last); gap > 4*60*time.Millisecond {
				t.Fatalf("心跳间隔超期界×4: %v", gap)
			}
		}
		last = time.Now()
	}
	// NOTIFY 透传不被心跳环饿死：ev_account 纯信号帧照推。
	if _, err := w.pool.Exec(ctx, `SELECT pg_notify('ev_account','')`); err != nil {
		t.Fatal(err)
	}
	ev, err := stream.Receive()
	if err != nil {
		t.Fatalf("ev_account 帧未达: %v", err)
	}
	if ev.Kind != "ev_account" {
		t.Fatalf("应透传 ev_account: %s", ev.Kind)
	}
	cancel()
}

// favicon 桩锚：显式挂接返真件——dist 缺件=条件未备跳过（provision 快照
// 未含 favicon 期如实跳过非谎报）；SPA 兜底不再把 index.html 当 favicon。
func TestFaviconServed(t *testing.T) {
	if _, err := web.Dist.ReadFile("dist/favicon.ico"); err != nil {
		t.Skip("dist/favicon.ico 未产（npm build 未跑于本树——条件未备跳过）")
	}
	mux := http.NewServeMux()
	mountUI(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/favicon.ico", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("favicon 应 200: %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); strings.Contains(ct, "text/html") {
		t.Fatalf("favicon 不应回 HTML: %s", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("favicon 体空")
	}
}
