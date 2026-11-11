// mechwake —— o.event.v1 事件总线订阅唤醒臂（纪元3 队列制正路落地：
// ops_event 落账→eventbus 水印轮→dm/mech 帧→即唤对应 unit；15min
// cognition timer/60s urgent relay timer 降为漏缝兜底非主力——
// 业主令 2026-11-11「队列式，定时轮询最多兜底」）。
// 唤醒映射（帧 Kind 键）：
//
//	dm   （升件/收口/移交/收口待审/诊断草案）→ 唤知派发+急件摆渡+部署守望
//	mech （已批/派单/裁决/处置/部署/起草）   → 唤知派发+急件摆渡+部署守望
//	其余 kind（ctx/econ）无唤件映射——收帧不动作。
//
// dm/mech 同表 ops_event_wip 共水位键互压丢帧属在案设计——union
// 唤三件即撞帧亦必达；唤件幂等（dispatch 单飞闸/deploy_watch 让位
// 三态/relay 水印去重）多唤零害。断流→携末帧水位重连（回填帧=停机
// 期补唤——漏帧追偿语义正确非重复）。
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os/exec"
	"time"

	"o/internal/gen/eventv1"
	"o/internal/gen/eventv1/eventv1connect"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
)

var wakeSet = map[string]bool{"dm": true, "mech": true}

var units = []string{
	"o-cognition-dispatch.service",
	"o-ops-relay-urgent.service",
	"o-deploy-watch.service",
}

func wakeAll() {
	for _, u := range units {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		out, err := exec.CommandContext(ctx, "systemctl", "--user",
			"start", "--no-block", u).CombinedOutput()
		cancel()
		if err != nil {
			log.Printf("[mechwake] 唤 %s 败: %v %.200s", u, err, out)
			continue
		}
		log.Printf("[mechwake] 唤 %s 已递", u)
	}
}

func resumeList(m map[string]int64) []*eventv1.SourceWatermark {
	l := make([]*eventv1.SourceWatermark, 0, len(m))
	for s, id := range m {
		l = append(l, &eventv1.SourceWatermark{Source: s, MaxId: id})
	}
	return l
}

func main() {
	addr := flag.String("addr", "http://127.0.0.1:15436",
		"eventbus 基址（o-oci-bus-tunnel 隧道口）")
	flag.Parse()
	c := eventv1connect.NewEventBusClient(
		connect.NewClient(connecthttp.NewTransport(http.DefaultClient, *addr)))
	ctx := context.Background()
	resume := map[string]int64{}
	log.Printf("[mechwake] 订阅 %s 起（kind∈dm/mech→唤3件）", *addr)
	for ctx.Err() == nil {
		st, err := c.Watch(ctx, &eventv1.WatchRequest{Resume: resumeList(resume)})
		if err != nil {
			log.Printf("[mechwake] watch 起败: %v——3s 重试", err)
			time.Sleep(3 * time.Second) // sched-exempt-ok 断流重连退避（事件流死侦钦定周期——非轮询数据）
			continue
		}
		for {
			fr, err := st.Receive()
			if err != nil {
				log.Printf("[mechwake] 断流: %v——3s 重连", err)
				_ = st.Close()
				break
			}
			if fr == nil || fr.Source == "__hb__" || fr.Source == "" {
				continue
			}
			resume[fr.Source] = max(resume[fr.Source], fr.MaxId)
			if wakeSet[fr.Kind] {
				log.Printf("[mechwake] 帧 %s/%s max=%d", fr.Source, fr.Kind, fr.MaxId)
				wakeAll()
			}
		}
		time.Sleep(3 * time.Second) // sched-exempt-ok 断流重连退避（事件流死侦钦定周期——非轮询数据）
	}
}
