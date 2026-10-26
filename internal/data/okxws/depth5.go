// depth5.go —— books5 复活信号域（T-0139/D-0022 桥二期）：订阅标的集可配置
// （bridge.depth5.symbols）+帧蒸馏 top5 USD 额/spread_bps 特征行（非原始盘口）。
package okxws

import (
	"context"
	"encoding/json"
	"strconv"
	"time"
)

// parseBooks5 → depth5Row（books5 帧 bids/asks=[px,sz,liq,num,ordPx]×5 档；
// USD 额=px×sz×ctv——SWAP sz=张需 ctv 换算，缺 ctv=nil 拒入如实）。
func parseBooks5(instID string, d map[string]json.RawMessage, ctv float64) *depth5Row {
	px1 := lvlPx(d, "bids")
	px2 := lvlPx(d, "asks")
	if px1 == 0 || px2 == 0 || ctv <= 0 {
		return nil
	}
	r := &depth5Row{instID: instID, ts: tsms(d, "ts"),
		bestBid: px1, bestAsk: px2}
	if r.ts.IsZero() {
		r.ts = time.Now().UTC()
	}
	r.bidUSD = depthUSD(d, "bids", ctv)
	r.askUSD = depthUSD(d, "asks", ctv)
	r.spreadBP = (px2 - px1) / ((px2 + px1) / 2) * 10000
	return r
}

// lvlPx 档位数组首档价（[px,sz,…] 首元素——OKX 档位=字符串形；零档/坏形=0 拒入）。
func lvlPx(d map[string]json.RawMessage, k string) float64 {
	var lvls [][]json.RawMessage
	if v, ok := d[k]; ok {
		_ = json.Unmarshal(v, &lvls)
	}
	if len(lvls) == 0 || len(lvls[0]) < 1 {
		return 0
	}
	return lvlNum(lvls[0][0])
}

// lvlNum 档位元素（JSON 字符串/数字双形兼容→float64）。
func lvlNum(raw json.RawMessage) float64 {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		_ = json.Unmarshal(raw, &s)
		if err != nil {
			return 0
		}
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// depthUSD top5 档 USD 额列（px×sz×ctv；档位≤5 截断）。
func depthUSD(d map[string]json.RawMessage, k string, ctv float64) []float64 {
	var lvls [][]json.RawMessage
	if v, ok := d[k]; ok {
		_ = json.Unmarshal(v, &lvls)
	}
	out := []float64{}
	for i, l := range lvls {
		if i >= 5 {
			break
		}
		if len(l) < 2 {
			continue
		}
		out = append(out, lvlNum(l[0])*lvlNum(l[1])*ctv)
	}
	return out
}

// AttachCTV 挂 SWAP ctVal 读件（T-0139 depth5 USD 额换算——装配层 instruments
// SWAP 实读闭包；缓存 10min，缺 inst=该 inst 蒸馏跳过如实）。
func (c *Collector) AttachCTV(fn func(ctx context.Context) (map[string]float64, error)) {
	c.ctvFeed = fn
}

// depth5Symbols 深度标的集（config 键 bridge.depth5.symbols JSON 数组；缺/空
// =面缺席如实不订阅——T-0123 撤订语义对无消费域保持）。
func (c *Collector) depth5Symbols(ctx context.Context) []string {
	raw, err := c.cfg.Get("bridge.depth5.symbols")
	if err != nil {
		return nil
	}
	var syms []string
	if err := json.Unmarshal(raw, &syms); err != nil {
		c.rej.Inc("parse_fail")
		return nil
	}
	return syms
}
