// hlprobe —— T-0263 验收实证件（overviewprobe 同形）：unary GetHeadless 快照
// +Watch ev_headless 流帧时延对拍+断流重连恢复语义验（杀连接→自动重连→
// unary 回补→seq 水印去重补洞——与 web store.applyHeadless/stream.ts 同一
// 恢复合约的可跑参照）。只读 RPC，零副作用。
// 用法：hlprobe [-addr http://127.0.0.1:50051] [-watch] [-reconnect]
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"google.golang.org/protobuf/encoding/protojson"

	dashboardv1 "o/internal/gen/dashboardv1"
	"o/internal/gen/dashboardv1/dashboardv1connect"
)

// seqWater 客户端水位（applyHeadless 同律——陈帧/重帧 seq≤水位弃，seq>水位补洞）。
type seqWater struct{ mark int64 }

func (w *seqWater) admit(tag string, r *dashboardv1.GetHeadlessResponse) bool {
	if r.LastSeq <= w.mark {
		fmt.Printf("[DEDUP] %s lastSeq=%d ≤ 水位 %d——弃\n", tag, r.LastSeq, w.mark)
		return false
	}
	w.mark = r.LastSeq
	fmt.Printf("[APPLY] %s lastSeq=%d > 水位——纳\n", tag, r.LastSeq)
	return true
}

func snap(client dashboardv1connect.CoreServiceClient, ctx context.Context,
	w *seqWater) {
	r, err := client.GetHeadless(ctx, &dashboardv1.GetHeadlessRequest{})
	if err != nil {
		fmt.Println("[UNARY] err:", err)
		return
	}
	if w.admit("unary", r) {
		for _, row := range r.Rows {
			fmt.Printf("  %s seat=%s model=%s phase=%s state=%s seq=%d\n",
				row.Ticket, row.Seat, row.Model, row.Phase, row.State, row.Seq)
		}
	}
}

func stream(client dashboardv1connect.CoreServiceClient, ctx context.Context,
	w *seqWater) error {
	st, err := client.Watch(ctx, &dashboardv1.WatchRequest{})
	if err != nil {
		return err
	}
	fmt.Println("[STREAM] attached")
	for {
		ev, err := st.Receive()
		if err != nil {
			fmt.Println("[STREAM] closed:", err)
			return err
		}
		line := fmt.Sprintf("[FRAME %s] %s", time.Now().Format("15:04:05.000"), ev.Kind)
		if ev.Kind == "ev_headless" && ev.Data != nil {
			r := &dashboardv1.GetHeadlessResponse{}
			if b, e := protojson.Marshal(ev.Data); e == nil {
				_ = protojson.Unmarshal(b, r)
			}
			line += fmt.Sprintf(" lastSeq=%d rows=%d", r.LastSeq, len(r.Rows))
			w.admit("frame", r)
		}
		fmt.Println(line)
	}
}

func main() {
	addr := flag.String("addr", "http://127.0.0.1:50051", "CoreService 基址")
	reconnect := flag.Bool("reconnect", false, "断流自动重连演示环（ctrl-c 退）")
	flag.Parse()
	client := dashboardv1connect.NewCoreServiceClient(
		connect.NewClient(connecthttp.NewTransport(http.DefaultClient, *addr)))
	ctx := context.Background()
	w := &seqWater{}
	snap(client, ctx, w) // 挂载/重连=unary 全量快照回补
	if !*reconnect {
		_ = stream(client, ctx, w)
		return
	}
	for ctx.Err() == nil { // 断流自动重连环——恢复合约：快照→接流→seq 补洞
		if err := stream(client, ctx, w); err == nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second): // sched-exempt-ok 断线重连退避（stream.ts 同梯）
		}
		snap(client, ctx, w)
	}
}
