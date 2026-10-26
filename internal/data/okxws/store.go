package okxws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// writer 异步批量落库协程：行队列→500ms/128 行触发 SendBatch（读循环零同步 DB 写——
// 背压不连坐 WS 读径，满丢弃已计 write_err 闸）。
func (c *Collector) writer(ctx context.Context) {
	buf := make([]rowItem, 0, 128)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	flush := func() {
		if len(buf) == 0 {
			return
		}
		if err := c.insertBatch(ctx, buf); err != nil {
			c.rej.Inc("write_err")
		}
		buf = buf[:0]
	}
	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case it := <-c.queue:
			buf = append(buf, it)
			if len(buf) >= 128 {
				flush()
			}
		case <-tick.C:
			flush()
		}
	}
}

// insertBatch 单事务三表分桶批量插（pgx.Batch——一次往返，venue 列默认 'okx'；
// T-0123 状态表化：ticks/mark 改 upsert 末帧覆盖，books5 零读者停写撤订阅，
// funding 有窗口消费者仍追加+30d 保留期）。
func (c *Collector) insertBatch(ctx context.Context, rows []rowItem) error {
	b := &pgx.Batch{}
	n := 0
	for _, it := range rows {
		switch {
		case it.tick != nil:
			r := it.tick
			b.Queue(`INSERT INTO md_ticks(ts,inst_id,last,bid,ask,vol_24h) VALUES($1,$2,$3,$4,$5,$6)
				ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last,
				bid=EXCLUDED.bid,ask=EXCLUDED.ask,vol_24h=EXCLUDED.vol_24h,recv_ts=now()`,
				r.ts, r.instID, r.last, r.bid, r.ask, r.vol24h)
			n++
		case it.funding != nil:
			r := it.funding
			b.Queue(`INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_ts,next_funding_rate) VALUES($1,$2,$3,$4,$5)`,
				r.ts, r.instID, r.fundingRate, r.nextFundingTs, r.nextFundingRate)
			n++
		case it.mark != nil:
			r := it.mark
			b.Queue(`INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES($1,$2,$3,$4)
				ON CONFLICT (venue,inst_id,channel) DO UPDATE SET ts=EXCLUDED.ts,
				px=EXCLUDED.px,recv_ts=now()`,
				r.ts, r.instID, r.channel, r.px)
			n++
		case it.depth5 != nil: // T-0139：md_signal kind='depth5'——特征值 upsert（有界状态表）
			r := it.depth5
			meta, _ := json.Marshal(map[string]any{
				"bid_usd": r.bidUSD, "ask_usd": r.askUSD,
				"best_bid": r.bestBid, "best_ask": r.bestAsk})
			b.Queue(`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('depth5',$1,$2,$3::jsonb,$4)
				ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val,
				meta=EXCLUDED.meta, ts=EXCLUDED.ts`,
				r.instID, r.spreadBP, string(meta), r.ts)
			n++
		}
	}
	if n == 0 {
		return nil
	}
	br := c.pool.SendBatch(ctx, b)
	err := br.Close()
	return err
}
