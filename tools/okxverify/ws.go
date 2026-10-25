// ws.go —— okxverify WS 流面段（公开+私有帧收集/私有流泵——行数棘轮自主拆分）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"o/internal/platform/signutil"
	"o/internal/trading/okxexec"
)

const okxPublicURL = "wss://ws.okx.com:8443/ws/v5/public"
const okxPrivateURL = "wss://ws.okx.com:8443/ws/v5/private"

// wsCollect 公开面通用订阅采样——返 channel→帧数。
func wsCollect(ctx context.Context, url string, args []map[string]string, win time.Duration) map[string]int {
	out := map[string]int{}
	c, _, err := websocket.DefaultDialer.DialContext(ctx, url, http.Header{})
	if err != nil {
		return out
	}
	defer c.Close()
	_ = c.WriteJSON(map[string]any{"op": "subscribe", "args": args})
	deadline := time.Now().Add(win)
	var mu2 sync.Mutex
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, data, err := c.ReadMessage()
			if err != nil {
				return
			}
			var m struct {
				Arg struct {
					Channel string `json:"channel"`
				} `json:"arg"`
				Data json.RawMessage `json:"data"`
			}
			// 只计数据帧——订阅 ACK 亦携 arg.channel，计入则活性证伪
			if json.Unmarshal(data, &m) == nil && m.Arg.Channel != "" && len(m.Data) > 0 {
				mu2.Lock()
				out[m.Arg.Channel]++
				mu2.Unlock()
			}
		}
	}()
	time.Sleep(time.Until(deadline)) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
	c.Close()
	<-done
	return out
}

// privStream 私有流 login+订阅 account/balance_and_position/positions，帧推 privCh。
func privStream(ctx context.Context, creds okxexec.Credentials, sim bool, out chan<- string) {
	c, _, err := websocket.DefaultDialer.DialContext(ctx, okxPrivateURL, http.Header{})
	if err != nil {
		out <- "dialerr: " + err.Error()
		return
	}
	defer c.Close()
	ts := fmt.Sprintf("%d", time.Now().Unix())
	_ = c.WriteJSON(map[string]any{"op": "login", "args": []map[string]string{{
		"apiKey": creds.Key, "passphrase": creds.Passphrase, "timestamp": ts,
		"sign": signutil.OkxSign(creds.Secret, ts, "GET", "/users/self/verify", "")}}})
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, ack, err := c.ReadMessage()
	if err != nil {
		out <- "login ack err: " + err.Error()
		return
	}
	_ = c.SetReadDeadline(time.Time{})
	out <- "LOGIN_ACK " + string(ack)
	if !strings.Contains(string(ack), `"event":"login"`) || !strings.Contains(string(ack), `"code":"0"`) {
		return
	}
	_ = c.WriteJSON(map[string]any{"op": "subscribe", "args": []map[string]string{
		{"channel": "account"},
		{"channel": "balance_and_position"},
		{"channel": "positions", "instType": "ANY"},
		{"channel": "orders", "instType": "ANY"},
	}})
	out <- "SUBSCRIBED"
	for {
		_ = c.SetReadDeadline(time.Now().Add(30 * time.Second))
		_, data, err := c.ReadMessage()
		if err != nil {
			out <- "end: " + err.Error()
			return
		}
		if string(data) == "pong" {
			continue
		}
		select {
		case out <- string(data):
		default:
		}
	}
}

func drainPriv(ch chan string, tag string) {
	n := 0
	for {
		select {
		case f := <-ch:
			n++
			if n <= 40 {
				s := f
				if len(s) > 300 {
					s = s[:300] + "…"
				}
				fmt.Printf("  PRIV[%d]%s %s\n", n, tag, s)
			}
		default:
			if n > 0 {
				fmt.Printf("  …私有帧合计 %d\n", n)
			}
			return
		}
	}
}

// mutateAll mutating 实单谱——摆渡→开→定态→平→复位。每步实读仲裁。
func mutateAll(ctx context.Context, cli *okxexec.Client, mvusd float64, posMode string, priv chan string, restore string, restoremax float64) {
	fmt.Println("\n== mutating 实单谱（真钱——每步实读仲裁）==")
	// ① 自筹资 ② 划转往返 ③ Earn ④ 合约腿 ⑤ 现货腿 ⑥ 挂单谱 ⑦ bills ⑧ 台账复位
	tra, dustInst, soldUSD := mSelfFund(ctx, cli)
	mTransfers(ctx, cli, mvusd, tra)
	mEarn(ctx, cli)
	tra, _ = cli.TradingAvailBal(ctx, "USDT")
	mPerp(ctx, cli, tra, posMode, priv)
	mSpot(ctx, cli, tra, priv)
	mPending(ctx, cli)
	mBills(ctx, cli)
	mRestore(ctx, cli, dustInst, soldUSD, restore, restoremax)
}
