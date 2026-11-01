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
	"google.golang.org/protobuf/encoding/protojson"

	"o/internal/gen/dashboardv1"
	"o/internal/gen/dashboardv1/dashboardv1connect"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 流推实测（锚 c/d ≤10s——fin_0092 ev_pipeline 触发器→LISTEN→一跳载荷帧，
// in-process connect 服务器全真径：无轮询无定时器——事件驱动律）。
func TestWatchPipelineFrame(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	svc := coreService{pool: pool}
	mux := http.NewServeMux()
	mountCore(mux, svc)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cl := dashboardv1connect.NewCoreServiceClient(connect.NewClient(
		connecthttp.NewTransport(srv.Client(), srv.URL)))
	wctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	stream, err := cl.Watch(wctx, &dashboardv1.WatchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	// watch_open 就绪帧（LISTEN 注册完标志——收到才算连上）。
	first, err := stream.Receive()
	if err != nil || first.Kind != "watch_open" {
		t.Fatalf("首帧应 watch_open: %v %+v", err, first)
	}
	// 锚 d：新票派发→新行入顶。NOTIFY→载荷帧时延 ≤10s（设计稿钉）。
	t0 := time.Now()
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(ts,seat,event,ticket,note)
		VALUES(now(),'Dev','领单','T-9601','流推桩')`); err != nil {
		t.Fatal("事件桩:", err)
	}
	frame := waitPipelineFrame(t, stream)
	latency := time.Since(t0)
	if latency > 10*time.Second {
		t.Fatalf("锚 d 超时: %v", latency)
	}
	if !strings.Contains(frame, "T-9601") {
		t.Fatalf("载荷帧缺新票: %s", truncStr(frame))
	}
	// 锚 c：部署完结→行退场。部署事件插入→下一 ev_pipeline 帧无 T-9601。
	t0 = time.Now()
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(ts,seat,event,ticket,note)
		VALUES(now(),'DEPLOY','部署','','deploy 桩')`); err != nil {
		t.Fatal("部署桩:", err)
	}
	frame = waitPipelineFrame(t, stream)
	deployLatency := time.Since(t0)
	if deployLatency > 10*time.Second {
		t.Fatalf("锚 c 超时: %v", deployLatency)
	}
	if strings.Contains(frame, "T-9601") {
		t.Fatalf("已部署票未退场: %s", truncStr(frame))
	}
	t.Logf("流推时延: 领单帧 %v / 部署帧 %v", latency, deployLatency)
}

// waitPipelineFrame 收帧至下一个 ev_pipeline（跳过心跳/他通道帧——同流多
// 通道共脊；期界=测试 ctx 30s，超期流断 err 抛出=fail）。
func waitPipelineFrame(t *testing.T, stream dashboardv1connect.CoreServiceWatchClientStream) string {
	t.Helper()
	for {
		ev, err := stream.Receive()
		if err != nil {
			t.Fatalf("流断: %v", err)
		}
		if ev.Kind == "ev_pipeline" {
			b, merr := protojson.Marshal(ev.Data)
			if merr != nil {
				t.Fatalf("载荷转败: %v", merr)
			}
			return truncStr(string(b))
		}
	}
}

func truncStr(s string) string {
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}
