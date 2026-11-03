// eventbus —— T-0312 第五刀 OCI 服务端（MD 本地+FIN 隧道双 DSN 三源 3s 轮
// max_id→Watch server-stream 扇出；bind 127.0.0.1:15436 loopback 零公网暴露——
// ssh 隧道鉴权起步 mTLS 二期债）。
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"
	"o/internal/eventbus"
	"o/internal/gen/eventv1"

	"o/internal/gen/eventv1/eventv1connect"
)

type svc struct{ h *eventbus.Hub }

func (s *svc) Watch(ctx context.Context, req *eventv1.WatchRequest,
	st eventv1connect.EventBusWatchServerStream) error {
	resume := map[string]int64{}
	for _, w := range req.Resume {
		resume[w.Source] = w.MaxId
	}
	ch, done := s.h.Watch(resume)
	defer done()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case fr := <-ch:
			fr.Ts = timestamppb.Now()
			if err := st.Send(fr); err != nil {
				return err
			}
		}
	}
}

// envDSN 读 env 件径（MD_PG_DSN/FIN_PG_DSN 钦定键——部署件凭据载体；
// 键缺=该源静默不轮）。
func envDSN(key, file string) string {
	b, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(ln, key+"=") {
			return strings.Trim(strings.TrimPrefix(ln, key+"="), `"`)
		}
	}
	return ""
}

func main() {
	ctx := context.Background()
	h := eventbus.NewHub()
	srcs := map[string]eventbus.Source{
		"a": {Table: "md_ctx_fact", Kind: "ctx", DSN: "md"},
		"b": {Table: "md_macro_event", Kind: "ctx", DSN: "md"},
		"c": {Table: "fin_econ_lesson", Kind: "econ", DSN: "fin"},
	}
	pools := map[string]eventbus.Querier{}
	for _, kv := range [][2]string{{"md", "MD_PG_DSN"}, {"fin", "FIN_PG_DSN"}} {
		if d := envDSN(kv[1], "/opt/o-research/env"); d != "" {
			if p, err := pgxpool.New(ctx, d); err == nil {
				pools[kv[0]] = p
			}
		}
	}
	go eventbus.NewPoller(srcs).Run(ctx, pools, h)
	server := connect.NewServer()
	eventv1connect.RegisterEventBusHandler(server, &svc{h: h})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	log.Printf("[eventbus] 127.0.0.1:15436 watch 起（3s 轮三源——ssh 隧道鉴权）")
	log.Fatal(http.ListenAndServe("127.0.0.1:15436", mux))
}
