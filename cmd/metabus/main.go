// metabus —— T-0312 第五刀 dev 订阅件（localhost:15436 Watch 循环+退避
// 1s→30s 帽+resume 内存水位+帧→systemctl --user start 唤醒映射；零 DB
// 凭据零 PG 依赖红线）。
package main

import (
	"context"
	"fmt"
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
	"dm":   "o-decision-seat.service", // 第六刀：待裁事件唤醒决策席
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wm []*eventv1.SourceWatermark
	for s, m := range resume {
		wm = append(wm, &eventv1.SourceWatermark{Source: s, MaxId: m})
	}
	st, err := cli.Watch(ctx, &eventv1.WatchRequest{Resume: wm})
	if err != nil {
		return err
	}
	defer st.Close()
	type res struct {
		fr  *eventv1.DataAdvanced
		err error
	}
	rch := make(chan res, 1)
	go func() { // Receive 串行喂通道——看门狗超时时 cancel 解阻塞
		for {
			fr, err := st.Receive()
			rch <- res{fr, err}
			if err != nil {
				return
			}
		}
	}()
	idle := time.NewTimer(45 * time.Second) // 45s 无帧无 __hb__=流死（半死 TCP/隧道异态自愈闸）
	for {
		select {
		case r := <-rch:
			if r.err != nil {
				return r.err
			}
			if !idle.Stop() {
				<-idle.C
			}
			idle.Reset(45 * time.Second)
			fr := r.fr
			if fr.Source == "__hb__" { // 心跳哨兵——刷活不录不唤
				continue
			}
			resume[fr.Source] = fr.MaxId // 水位内存册——重启归零补当前帧幂等无害
			u, ok := wake[fr.Kind]
			log.Printf("[metabus] 帧 %s/%s max=%d →%s", fr.Source, fr.Kind, fr.MaxId,
				map[bool]string{true: "systemctl start " + u, false: "无映射不唤"}[ok])
			if ok {
				// T-0319：activating 态同步 start 排队绞死帧循环根修——
				// --no-block job 入队即返（唤醒=有活即醒非等干完）。
				if err := exec.Command("systemctl", "--user", "start", "--no-block", u).Run(); err != nil {
					log.Printf("[metabus] 唤醒 %s 败: %v", u, err)
				}
			}
		case <-idle.C:
			cancel()
			return fmt.Errorf("45s 无帧无心跳——判流死重连")
		}
	}
}
