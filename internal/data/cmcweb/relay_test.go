// relay_test.go —— 本地 WS 中继协议断言：auth 前数据帧禁漏、四种控制帧代答、
// auth 后透传、上游死掐客户端、环形缓冲回灌。
package cmcweb

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func dialRelay(t *testing.T, addr string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func startRelay(t *testing.T) (*Relay, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	r := NewRelay(addr)
	r.SetUpstreamAlive(true) // 上游死时 serveWS 503 拒升——测试默认上游活
	go func() { _ = r.ListenAndServe(context.Background()) }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.Dial("tcp", addr); err == nil {
			_ = c.Close()
			return r, addr
		}
		time.Sleep(20 * time.Millisecond) // sched-exempt-ok：中继监听就绪轮询
	}
	t.Fatal("relay not listening")
	return nil, ""
}

func readFrame(t *testing.T, c *websocket.Conn, timeout time.Duration) ([]byte, error) {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(timeout))
	_, data, err := c.ReadMessage()
	return data, err
}

func TestRelayGateBeforeAuth(t *testing.T) {
	r, addr := startRelay(t)
	r.Feed(`{"msg":{"type":6}}`) // 先喂——客户端未 auth 前禁漏
	c := dialRelay(t, addr)
	defer c.Close()
	time.Sleep(300 * time.Millisecond) // sched-exempt-ok：泄漏窗口——若漏帧已在 TCP 缓冲
	// auth → 首帧必须是 ack（漏则首帧=msg 帧，直接判败）
	if err := c.WriteMessage(1, []byte(`{"auth":{"token":"x"}}`)); err != nil {
		t.Fatal(err)
	}
	data, err := readFrame(t, c, 2*time.Second)
	if err != nil || string(data) != `{"auth":{"status":"ok"}}` {
		t.Fatalf("auth ack: %s %v", data, err)
	}
	// 闸开后回灌帧到
	data, err = readFrame(t, c, 2*time.Second)
	if err != nil || string(data) != `{"msg":{"type":6}}` {
		t.Fatalf("backlog 回灌: %s %v", data, err)
	}
}

func TestRelayAcks(t *testing.T) {
	_, addr := startRelay(t)
	c := dialRelay(t, addr)
	defer c.Close()
	_ = c.WriteMessage(1, []byte(`{"auth":{"token":"x"}}`))
	_, _ = readFrame(t, c, 2*time.Second)
	for _, tc := range []struct{ in, want string }{
		{`{"subscribe":{"subscriptionId":"1","type":7}}`, `{"subscribe":{"status":"ok"}}`},
		{`{"heartbeat":{}}`, `{"heartbeat":{"status":"ok"}}`},
		{`{"auth":{"token":"renewed"}}`, `{"auth":{"status":"ok"}}`},
	} {
		if err := c.WriteMessage(1, []byte(tc.in)); err != nil {
			t.Fatal(err)
		}
		data, err := readFrame(t, c, 2*time.Second)
		if err != nil || string(data) != tc.want {
			t.Fatalf("%s → %s (%v)", tc.in, data, err)
		}
	}
}

func TestRelayUpstreamDeath(t *testing.T) {
	r, addr := startRelay(t)
	c := dialRelay(t, addr)
	defer c.Close()
	_ = c.WriteMessage(1, []byte(`{"auth":{"token":"x"}}`))
	_, _ = readFrame(t, c, 2*time.Second)
	r.SetUpstreamAlive(false)
	if _, err := readFrame(t, c, 2*time.Second); err == nil {
		t.Fatal("上游死客户端连未掐")
	}
	// 死态下新连接拒升（陈旧回放假报活闸——dial 应 503 败）
	if _, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/", addr), nil); err == nil {
		t.Fatal("上游死态下新 WS 升级未拒")
	}
}
