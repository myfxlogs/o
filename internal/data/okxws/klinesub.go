// klinesub.go —— 工单#13065 日K采集臂：business 端点独立连接。
// 2026-11-13 venue 实测：candle* 频道全族（1D/1Dutc/1W/1H）于 /public 端点
// 全拒 60018「channel doesn't exist」，于 /business 返真帧——K 线面不在
// 公开端点，须独立连接。订阅集=活跃集同面（ActiveSetUnion 唯一真源），
// 帧径=公共 handle()→parseCandle→queue→writer 同幂等写面。
package okxws

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const BusinessURL = "wss://ws.okx.com:443/ws/v5/business"

// klineLoop business 端点连接臂：拨号→klineDial→终则退避重连。
// 独立于 public 连接的生命周期——candle 面死连不拖行情面（同 backoff 谱）。
func (c *Collector) klineLoop(ctx context.Context) {
	backoff := backoffMin
	sub := map[string]bool{}
	for ctx.Err() == nil {
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.bizURL, http.Header{})
		if err != nil {
			log.Printf("[okxws] business 拨号: %v——%s 后重试", err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff): // sched-exempt-ok 退避重连期界
			}
			backoff = min(backoff*2, backoffMax)
			continue
		}
		err = c.klineDial(ctx, conn, sub)
		_ = conn.Close()
		if ctx.Err() != nil {
			return
		}
		log.Printf("[okxws] business 连接终: %v——%s 后重连", err, backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff): // sched-exempt-ok 退避重连期界
		}
		backoff = min(backoff*2, backoffMax)
	}
}

// klineDial 单连接生命周期：活跃集 candle1D 订阅→读循环+ping+30s 集重订。
func (c *Collector) klineDial(ctx context.Context, conn *websocket.Conn, sub map[string]bool) error {
	c.klineSync(ctx, conn, sub)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			c.handle(data)
		}
	}()
	ping := time.NewTicker(pingEvery) // sched-exempt-ok WS 心跳节
	defer ping.Stop()
	syncTick := time.NewTicker(30 * time.Second) // sched-exempt-ok 活跃集重订轮询
	defer syncTick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case <-ping.C:
			if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				return err
			}
		case <-syncTick.C:
			c.klineSync(ctx, conn, sub)
		}
	}
}

// klineSync 活跃集 candle1D 动态重订：新入者加订、出集者退订、
// 60018 拒收不再重发（共表拒集读件——USDT 键域+拒集滤同 tickGate 判据族）。
func (c *Collector) klineSync(ctx context.Context, conn *websocket.Conn, sub map[string]bool) {
	set, err := c.activeSet(ctx)
	if err != nil {
		c.rej.Inc("activeset_read_failed")
		return
	}
	var adds []map[string]string
	for inst := range set {
		if !sub[inst] && !c.refused(inst) && isUSDTInst(inst) {
			adds = append(adds, map[string]string{"channel": "candle1D", "instId": inst})
		}
	}
	if err := subscribeAll(conn, adds); err != nil {
		return
	}
	for _, a := range adds {
		sub[a["instId"]] = true
	}
	for inst := range sub {
		if !set[inst] {
			if err := conn.WriteJSON(subArgs("unsubscribe", "candle1D", inst)); err != nil {
				return
			}
			delete(sub, inst)
		}
	}
	if len(adds) > 0 {
		log.Printf("[okxws] business candle1D 订阅 insts=%d", len(sub))
	}
}

// refuseMark/refused 60018 拒收集互斥面（public/business 双臂读环并发写同
// 表——拒因=inst 不存在者 venue 侧双臂同真，共表合理；重连清表=自然放行）。
func (c *Collector) refuseMark(inst string) {
	c.refuseMu.Lock()
	defer c.refuseMu.Unlock()
	if c.refuseSub != nil {
		c.refuseSub[inst] = true
	}
}
func (c *Collector) refused(inst string) bool {
	c.refuseMu.Lock()
	defer c.refuseMu.Unlock()
	return c.refuseSub != nil && c.refuseSub[inst]
}

// refuseSnap 拒集读快照（迭代面前先快照再遍历——锁内读指针防换表竞）。
func (c *Collector) refuseSnap() map[string]bool {
	c.refuseMu.Lock()
	defer c.refuseMu.Unlock()
	out := make(map[string]bool, len(c.refuseSub))
	for k := range c.refuseSub {
		out[k] = true
	}
	return out
}

// refuseReset 重连清表（dialLoop 指件换表——指针读写同锁）。
func (c *Collector) refuseReset() {
	c.refuseMu.Lock()
	defer c.refuseMu.Unlock()
	c.refuseSub = map[string]bool{}
}
