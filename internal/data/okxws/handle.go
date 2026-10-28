// handle.go —— WS 帧分流（T-0137 拆件；T-0139 加 books5 深度蒸馏支）。
package okxws

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// handle 帧分流：按 arg.channel 解析→行入异步队列（解析失败/非 USDT 键域计 rejects）。
func (c *Collector) handle(data []byte) {
	var msg wsMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		c.rej.Inc("parse_fail")
		return
	}
	if c.meter != nil {
		c.meter.Add("okxws/"+msg.Arg.Channel, len(data))
	}
	if msg.Event != "" {
		if msg.Event == "error" {
			log.Printf("[okxws] 服务端 error 帧: %s", data)
			// T-0133 ③60018 拒收集：带 instId 的服务端拒收记黑名单——本连接
			// 内 syncActive 不再重发（重连清表自然放行）。
			if c.refuseSub != nil && msg.Arg.InstID != "" {
				c.refuseSub[msg.Arg.InstID] = true
			}
		}
		return // subscribe ack/channel-conn-count/error 留痕非业务帧
	}
	instID := msg.Arg.InstID
	if instID != "" && !isUSDTInst(instID) {
		c.rej.Inc("non_usdt_instid")
		return
	}
	for _, d := range msg.Data {
		var it rowItem
		switch msg.Arg.Channel {
		case "tickers":
			it.tick = parseTickers(instID, d)
		case "funding-rate":
			it.funding = parseFunding(instID, d)
		case "mark-price", "index-tickers":
			it.mark = parseMark(instID, msg.Arg.Channel, d)
		case "books5": // T-0139：深度蒸馏——5s 节流（特征值非原始盘口）
			c.ctvMu.Lock()
			ctv := c.ctvCache[instID]
			stale := !c.ctvReloading && time.Since(c.ctvAt) > 60*time.Second
			c.ctvMu.Unlock()
			if ctv > 0 {
				if c.depth5At == nil || time.Since(c.depth5At[instID]) > 5*time.Second {
					it.depth5 = parseBooks5(instID, d, ctv)
					if it.depth5 != nil && c.depth5At != nil {
						c.depth5At[instID] = time.Now()
					}
				}
			} else {
				c.rej.Inc("depth5_no_ctval")
				// T-0191：载败饿死窗根治——ctvCache 空/此 inst 缺且距上次重载
				// >60s→后台重载一次（飞行中闸节流≤1/60s，venue 调用频不炸；
				// 载成自通不绑重连周期——「载一次败饿死一窗」结构病根治）。
				if stale && c.ctvFeed != nil {
					c.ctvMu.Lock()
					c.ctvReloading = true
					c.ctvMu.Unlock()
					go func() {
						m, err := c.ctvFeed(context.Background())
						c.ctvMu.Lock()
						defer c.ctvMu.Unlock()
						c.ctvReloading = false
						if err == nil && len(m) > 0 {
							c.ctvCache = m
							c.ctvAt = time.Now()
						}
					}()
				}
			}
		default:
			continue
		}
		if it.tick == nil && it.funding == nil && it.mark == nil && it.depth5 == nil {
			if msg.Arg.Channel != "books5" { // books5 节流窗静默非解析败
				c.rej.Inc("parse_fail")
			}
			continue
		}
		c.frames.Add(1)
		c.lastFrame.Store(time.Now().UnixNano())
		// T-0209②：per-inst 3s 最小写间隔（tickers/funding 同拍——写洪防；
		// T-0193 过拍闸量级沿）。
		if !c.passThrottle(it) {
			continue
		}
		// T-0209：SG 活跃集谓词闸（nil=全写 dev 径；set=开仓闸输入集限写
		// ——漏谓词=不落行不写僵尸 fail-closed）。
		if c.LocalWritePred != nil {
			switch {
			case it.tick != nil:
				if !c.LocalWritePred("ticks", it.tick.instID) {
					continue
				}
			case it.funding != nil:
				if !c.LocalWritePred("funding", it.funding.instID) {
					continue
				}
			}
		}
		select {
		case c.queue <- it:
		default:
			c.rej.Inc("write_err")
		}
	}
}

// （books5 撤订后序列完整性判据随面退役——T-0123，seq 缺口无消费者）。
