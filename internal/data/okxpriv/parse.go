package okxpriv

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"
)

// privMsg 私有帧（arg.channel 分流；data[]=载荷行族）。
type privMsg struct {
	Event string `json:"event"`
	Arg   struct {
		Channel string `json:"channel"`
	} `json:"arg"`
	Code string                       `json:"code"`
	Msg  string                       `json:"msg"`
	Data []map[string]json.RawMessage `json:"data"`
}

func str(d map[string]json.RawMessage, k string) string {
	var s string
	if v, ok := d[k]; ok {
		_ = json.Unmarshal(v, &s)
	}
	return s
}
func num(d map[string]json.RawMessage, k string) float64 {
	s := str(d, k)
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// handle 帧分流：login ack 验败=fail-loud 关连（外层重连）；事件→事实行+末帧缓存+快照刷写。
func (c *Collector) handle(ctx context.Context, data []byte) {
	var msg privMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		c.rej.Inc("parse_fail")
		return
	}
	if msg.Event != "" {
		if msg.Event == "error" || (msg.Event == "login" && msg.Code != "0") {
			c.rej.Inc("login_fail")
			log.Printf("[okxpriv] 服务端 error/login 帧: %s", data)
		}
		return
	}
	c.frames.Add(1)
	c.lastFrm.Store(time.Now().UnixNano())
	for _, d := range msg.Data {
		ccy := str(d, "ccy")
		if ccy == "" {
			ccy = "USDT" // balance_and_position 账户行币种归一（快照行 Currency=USDT 口径）
		}
		eq := num(d, "eqUsd")
		if eq == 0 {
			eq = num(d, "eq")
		}
		instID := str(d, "instId")
		blob, _ := json.Marshal(d)
		if _, err := c.pool.Exec(ctx, `INSERT INTO md_account_events(ts,channel,ccy,inst_id,data)
			VALUES(now(),$1,$2,$3,$4)`, msg.Arg.Channel, ccy, instID, blob); err != nil {
			c.rej.Inc("write_err")
		}
		if eq != 0 {
			c.lastEq[ccy] = eq
		}
	}
	c.flushSnapshot(ctx, "okx_event")
}

// flushSnapshot 末帧 equity 写快照行（src 标事件/心跳源——unified 键域归一）。
func (c *Collector) flushSnapshot(ctx context.Context, src string) {
	for ccy, eq := range c.lastEq {
		if _, err := c.pool.Exec(ctx, `INSERT INTO fin_venue_account_snapshot(venue,venue_type,ccy,equity,src,updated_at)
			VALUES('okx','unified',$1,$2,$3,now())
			ON CONFLICT(venue,venue_type,ccy) DO UPDATE SET equity=EXCLUDED.equity,src=EXCLUDED.src,updated_at=now()`,
			ccy, eq, src); err != nil {
			c.rej.Inc("write_err")
		}
	}
}

// bootstrapEquity 一次性账户首读播种末帧 equity（boot 锚定——非轮询面；
// 拉取件=装配层注入 bootFn）。读败=warn 不炸（流事件续源仍可成立）。
func (c *Collector) bootstrapEquity(ctx context.Context) {
	if c.bootFn == nil {
		return
	}
	eqs, err := c.bootFn(ctx)
	if err != nil {
		log.Printf("[okxpriv] boot 首读败（流事件续源仍成立）: %v", err)
		return
	}
	for ccy, v := range eqs {
		c.lastEq[ccy] = v
	}
	c.flushSnapshot(ctx, "okx_event_boot")
}
