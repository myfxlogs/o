// spotpoll.go —— 现货现价 REST 轮询件（T-0134：index-tickers WS 撤下转
// REST 替——2300 现货标的秒级推送=链路饱和踢连实证；60s 周期热启键
// okxws.spot_poll_s，读败计 rejects fail-closed 不造零）。
package okxws

import (
	"context"
	"encoding/json"
	"time"
)

// spotPoll 现货现价 REST 轮询（GET market/tickers?instType=SPOT 单请求全量
// 返回；restOKAt 活性戳同通道）。
func (c *Collector) spotPoll(ctx context.Context) {
	if c.spotFeed == nil {
		return // 读件未挂=让位（凭据/装配缺席面）
	}
	// T-0135：spotFeed（spot_last 面）+markFeed（mark_price 面——SWAP/
	// FUTURES 两请求批量件）同周期件。
	poll := func() error {
		if c.markFeed != nil {
			raws, err := c.markFeed(ctx)
			if err != nil {
				c.rej.Inc("mark_feed_failed")
				return err
			}
			for _, raw := range raws {
				if err := c.writeMarkRest(ctx, raw, "mark_price"); err != nil {
					c.rej.Inc("write_err")
					return err
				}
			}
		}
		raw, err := c.spotFeed(ctx)
		if err != nil {
			c.rej.Inc("spot_feed_failed")
			return err
		}
		if err := c.writeSpotLast(ctx, raw); err != nil {
			c.rej.Inc("write_err")
			return err
		}
		c.restOKAt.Store(time.Now().UnixNano())
		return nil
	}
	tick := time.NewTicker(60 * time.Second) // sched-exempt-ok：REST 轮询件钦定面（票面 60s 热启）
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if v, err := c.cfg.GetFloat("okxws.spot_poll_s"); err == nil && v > 0 {
				tick.Reset(time.Duration(v) * time.Second)
			}
			if err := poll(); err != nil {
				continue // 计数门已 Inc
			}
		}
	}
}

// writeSpotLast market/tickers 响应→md_mark_index（channel='spot_last'——
// REST 轮询面与 WS mark/index 同表键域分位不混）。
func (c *Collector) writeSpotLast(ctx context.Context, raw []byte) error {
	var rows []struct {
		InstID string `json:"instId"`
		Last   string `json:"last"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return err
	}
	now := time.Now()
	for _, r := range rows {
		if r.InstID == "" || !isUSDTInst(r.InstID) { // T-0135 ③非 USDT 死行滤除
			continue
		}
		if _, err := c.pool.Exec(ctx, `INSERT INTO md_mark_index(ts,inst_id,channel,px)
			VALUES($1,$2,'spot_last',$3)
			ON CONFLICT (venue,inst_id,channel) DO UPDATE SET ts=EXCLUDED.ts,
			px=EXCLUDED.px,recv_ts=now()`,
			now, r.InstID, r.Last); err != nil {
			return err
		}
	}
	return nil
}

// writeMarkRest public/mark-price 响应→md_mark_index channel='mark_price'
// （T-0135：WS 原键域复用——读面零改无缝衔接；非 USDT 形滤除同写径）。
func (c *Collector) writeMarkRest(ctx context.Context, raw []byte, ch string) error {
	var rows []struct {
		InstID string `json:"instId"`
		MarkPx string `json:"markPx"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return err
	}
	now := time.Now()
	for _, r := range rows {
		if r.InstID == "" || !isUSDTInst(r.InstID) {
			continue
		}
		if _, err := c.pool.Exec(ctx, `INSERT INTO md_mark_index(ts,inst_id,channel,px)
			VALUES($1,$2,$3,$4)
			ON CONFLICT (venue,inst_id,channel) DO UPDATE SET ts=EXCLUDED.ts,
			px=EXCLUDED.px,recv_ts=now()`,
			now, r.InstID, ch, r.MarkPx); err != nil {
			return err
		}
	}
	return nil
}
