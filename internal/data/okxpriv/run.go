package okxpriv

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"o/internal/platform/signutil"
)

// Run 单户私有流主循环：roster connecting 自报→心跳刷源协程→拨号循环。
// login 失败/权限错 fail-loud（err 上抛外层退避重连，不静默退化）；
// degraded/active 自报归 roster.go reportVenueState（名册=生命周期真源）。
func (c *Collector) Run(ctx context.Context, errc chan<- error) {
	if _, err := reportVenueState(ctx, c.pool, c.venueOr(), "connecting", nil); err != nil {
		log.Printf("[okxpriv] %s roster connecting 自报败: %v", c.venueOr(), err)
	}

	backoff := backoffMin
	for ctx.Err() == nil {
		err := c.dialLoop(ctx)
		if ctx.Err() != nil {
			return
		}
		log.Printf("[okxpriv] %s 连接终：%v——%s 后重连", c.venueOr(), err, backoff)
		reportVenueFail(ctx, c.pool, c.venueOr(), err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, backoffMax)
	}
}

// venueOr venue 写键归一（空='okx' 存量兼容——测试直构 Collector 零值场景）。
func (c *Collector) venueOr() string {
	if c.venue == "" {
		return "okx"
	}
	return c.venue
}

// login op=login args={apiKey,passphrase,timestamp,sign}——
// sign=base64(HMAC-SHA256(secret, ts+"GET"+"/users/self/verify"))，ts=秒级串。
func (c *Collector) login(conn *websocket.Conn) error {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	args := map[string]any{"op": "login", "args": []map[string]string{{
		"apiKey":     c.creds.Key,
		"passphrase": c.creds.Passphrase,
		"timestamp":  ts,
		"sign":       signutil.OkxSign(c.creds.Secret, ts, "GET", "/users/self/verify", ""),
	}}}
	return conn.WriteJSON(args)
}

func (c *Collector) dialLoop(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, http.Header{})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	if err := c.login(conn); err != nil {
		c.rej.Inc("login_fail")
		return fmt.Errorf("login write: %w", err)
	}
	// login ack 先行：60011「Please log in」=未等 ack 抢订阅（服务端拒未认证频道的
	// subscribe）——同步读一帧，login code≠0 即 fail-loud 上抛。
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, ack, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("login ack: %w", err)
	}
	_ = conn.SetReadDeadline(time.Time{})
	var lm privMsg
	if err := json.Unmarshal(ack, &lm); err != nil || lm.Event != "login" || lm.Code != "0" {
		c.rej.Inc("login_fail")
		return fmt.Errorf("login 败 fail-loud: %s", ack)
	}
	sub := map[string]any{"op": "subscribe", "args": []map[string]string{
		{"channel": "account"}, {"channel": "balance_and_position"}}}
	if err := conn.WriteJSON(sub); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	log.Printf("[okxpriv] %s login+subscribed url=%s", c.venueOr(), c.url)
	if _, err := reportVenueState(ctx, c.pool, c.venueOr(), "active", nil); err != nil {
		log.Printf("[okxpriv] %s roster active 自报败: %v", c.venueOr(), err)
	}
	// boot 首读（事件锚定形 REST 播种——快照写径禁周期轮询，仅此一次性）：
	// 静户零推送期快照即鲜；之后流事件+心跳续源。
	c.bootstrapEquity(ctx)
	c.tryBills(ctx) // boot 补漏：断连窗口内 bills 一次性拉齐（事件驱动主的兜底）

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			if mt == websocket.TextMessage && string(data) == "pong" {
				continue
			}
			c.handle(ctx, data)
			if c.meter != nil {
				var msg struct {
					Arg struct {
						Channel string `json:"channel"`
					} `json:"arg"`
				}
				if json.Unmarshal(data, &msg) == nil {
					c.meter.Add("okx_priv/"+c.venueOr()+"/"+msg.Arg.Channel, len(data))
				}
			}
		}
	}()
	ping := time.NewTicker(pingEvery)
	defer ping.Stop()
	watch := time.NewTicker(30 * time.Second)
	defer watch.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return fmt.Errorf("read: %w", err)
		case <-ping.C:
			if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				return fmt.Errorf("ping: %w", err)
			}
		case <-watch.C:
			// 私有面事件低频（静户零事件正常）——宽阈 5min 判死连
			if last := c.lastFrm.Load(); last != 0 && time.Since(time.Unix(0, last)) > dataIdle {
				return fmt.Errorf("私有帧静默 >%s——判死重连", dataIdle)
			}
		}
	}
}
