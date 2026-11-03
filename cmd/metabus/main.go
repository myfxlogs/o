// metabus —— T-0312 第五刀 dev 订阅件（localhost:15436 Watch 循环+退避
// 1s→30s 帽+resume 内存水位+帧→systemctl --user start 唤醒映射；零 DB
// 凭据零 PG 依赖红线）。
package main

import (
	"context"
	"log"
	"net/http"
	"os/exec"
	"time"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"
	"o/internal/gen/eventv1"
	"o/internal/gen/eventv1/eventv1connect"
)

// wake 映射（kind→unit——钦定两席 ctx/econ 各唤一）。
var wake = map[string]string{
	"ctx":  "o-regime-analyst.service",
	"econ": "o-econ-verdict.service",
}

func main() {
	cli := eventv1connect.NewEventBusClient(connect.NewClient(
		connecthttp.NewTransport(http.DefaultClient, "http://localhost:15436")))
	resume := map[string]int64{}
	backoff := time.Second
	for {
		err := watchOnce(cli, resume)
		log.Printf("[metabus] watch 断 %v——%v 后重连", err, backoff)
		time.Sleep(backoff) // sched-exempt-ok 重连退避（钦定 1s→30s 帽）
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func watchOnce(cli eventv1connect.EventBusClient, resume map[string]int64) error {
	ctx := context.Background()
	var wm []*eventv1.SourceWatermark
	for s, m := range resume {
		wm = append(wm, &eventv1.SourceWatermark{Source: s, MaxId: m})
	}
	st, err := cli.Watch(ctx, &eventv1.WatchRequest{Resume: wm})
	if err != nil {
		return err
	}
	defer st.Close()
	for {
		fr, err := st.Receive()
		if err != nil {
			return err
		}
		resume[fr.Source] = fr.MaxId // 水位内存册——重启归零补当前帧幂等无害
		if u, ok := wake[fr.Kind]; ok {
			log.Printf("[metabus] 帧 %s/%s max=%d →systemctl start %s", fr.Source, fr.Kind, fr.MaxId, u)
			if err := exec.Command("systemctl", "--user", "start", u).Run(); err != nil {
				log.Printf("[metabus] 唤醒 %s 败: %v", u, err)
			}
		}
	}
}
