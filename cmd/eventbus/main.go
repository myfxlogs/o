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
	"time"

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
	hb := time.NewTicker(15 * time.Second) // sched-exempt-ok 心跳哨兵帧——订阅端活性探针兼隧道保活（流死检测钦定周期件）
	defer hb.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-hb.C:
			if err := st.Send(&eventv1.DataAdvanced{Source: "__hb__", Ts: timestamppb.Now()}); err != nil {
				return err
			}
		case fr := <-ch:
			fr.Ts = timestamppb.Now()
			if err := st.Send(fr); err != nil {
				return err
			}
		}
	}
}

// envDSN 取 DSN（进程 env 优先——EnvironmentFile 注入径；回退 EVENTBUS_ENV_FILE
// 或默认 %h/o-research/env 件——键缺=该源静默不轮）。
func envDSN(key, file string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if file == "" {
		if h, err := os.UserHomeDir(); err == nil {
			file = h + "/o-research/env"
		}
	}
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
		"a": {Table: "md_ctx_fact", Kind: "ctx", DSN: "md",
			Q: "SELECT COALESCE(max(id),0) FROM md_ctx_fact"},
		"b": {Table: "md_macro_event", Kind: "ctx", DSN: "md",
			Q: "SELECT COALESCE(EXTRACT(EPOCH FROM max(event_date))::bigint,0) FROM md_macro_event"},
		"c": {Table: "fin_econ_lesson", Kind: "econ", DSN: "fin",
			Q: "SELECT COALESCE(max(id),0) FROM fin_econ_lesson"},
		"d": {Table: "ops_event_wip", Kind: "dm", DSN: "fin",
			// T-0354 件①：升件入 kick 白名单——升件行亦唤醒决策席
			//（拍壳待裁闸自挡空拍零 LLM；队列为空=心跳即退）。
			// T-0431：诊断草案行入白名单——草案落位即唤醒决策席裁决
			//（判词归流头段；status='drafted' 不入裁定席水印不回烧）。
			Q: "SELECT COALESCE(max(id),0) FROM ops_event WHERE event IN ('收口','移交','收口待审','升件','诊断草案')"},
	}
	pools := map[string]eventbus.Querier{}
	for _, kv := range [][2]string{{"md", "MD_PG_DSN"}, {"fin", "FIN_PG_DSN"}} {
		if d := envDSN(kv[1], ""); d != "" {
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
