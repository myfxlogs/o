// ws_reauth.go —— 连内续期：auth-JWT 到期前 reauthMargin 秒在同一连接重发 auth 帧。
// CMC auth JWT ttl=300s（auth.go parseExp 实采）；服务端在挂载的 JWT 过期时掐线
// （"Session expired" 生产实采×18）——连内续期=浏览器客户端同款行为，消 ~5min
// 全量重登风暴（6h=14 次全链登录≈288 次/日脚印）。
// 服务端若不支持连内续期：掐线→既有重连/重登径原样兜底，纯加法零回退。
package cmc

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"o/internal/platform/safego"

	"github.com/gorilla/websocket"
)

const (
	reauthMargin  = 45 * time.Second // exp 前续期余量
	reauthMinWait = 30 * time.Second // 等待下限（解析出贴线 exp 防热环）
	reauthDefault = 240 * time.Second
)

// startReauth 起连内续期 goroutine：每轮按 cur JWT 的 exp 重排时序，
// 到期前调 auth.JWT（内部惰性刷新）取新 JWT 发 auth 帧；JWT 刷新败/写败=
// 返回让连接自然死——Run 重连径兜底。
func (w *WsClient) startReauth(ctx context.Context, conn *websocket.Conn, done <-chan struct{}, jwt string) {
	safego.Go("cmc-ws:reauth", func() {
		cur := jwt
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-time.After(reauthWait(cur)): // sched-exempt-ok：期界等待=按 JWT exp 排的续期截止，非轮询节拍
			}
			fresh, err := w.auth.JWT(ctx)
			if err != nil {
				return
			}
			frame, _ := json.Marshal(map[string]any{"auth": map[string]any{"token": fresh}})
			if err := w.writeMsg(conn, frame); err != nil {
				return
			}
			cur = fresh
			log.Printf("[cmc-ws] 连内续期 auth 帧已发")
		}
	})
}

// reauthWait 距下次续期的等待：exp-reauthMargin+±10s 抖动，下限 reauthMinWait；
// 解析败回退 reauthDefault。抖动消节拍器纹（爬虫脚印降噪）——margin45s 兜得住。
func reauthWait(jwt string) time.Duration {
	exp, err := parseExp(jwt)
	if err != nil {
		return reauthDefault
	}
	wait := time.Until(exp.Add(-reauthMargin)) + time.Duration(rand.Intn(21)-10)*time.Second
	if wait < reauthMinWait {
		return reauthMinWait
	}
	return wait
}
