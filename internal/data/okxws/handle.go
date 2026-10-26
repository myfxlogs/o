// handle.go —— WS 帧分流（T-0137 拆件；T-0139 加 books5 深度蒸馏支）。
package okxws

import (
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
			if ctv := c.ctvCache[instID]; ctv > 0 {
				if c.depth5At == nil || time.Since(c.depth5At[instID]) > 5*time.Second {
					it.depth5 = parseBooks5(instID, d, ctv)
					if it.depth5 != nil && c.depth5At != nil {
						c.depth5At[instID] = time.Now()
					}
				}
			} else {
				c.rej.Inc("depth5_no_ctval")
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
		select {
		case c.queue <- it:
		default:
			c.rej.Inc("write_err")
		}
	}
}

// （books5 撤订后序列完整性判据随面退役——T-0123，seq 缺口无消费者）。
