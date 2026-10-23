package okxws

import (
	"context"
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

// insertBatch 单事务四表分桶批量插（pgx.Batch——一次往返，venue 列默认 'okx'）。
func (c *Collector) insertBatch(ctx context.Context, rows []rowItem) error {
	b := &pgx.Batch{}
	n := 0
	for _, it := range rows {
		switch {
		case it.tick != nil:
			r := it.tick
			b.Queue(`INSERT INTO md_ticks(ts,inst_id,last,bid,ask,vol_24h) VALUES($1,$2,$3,$4,$5,$6)`,
				r.ts, r.instID, r.last, r.bid, r.ask, r.vol24h)
			n++
		case it.books != nil:
			r := it.books
			b.Queue(`INSERT INTO md_books5(ts,inst_id,bids,asks,seq_id,checksum) VALUES($1,$2,$3,$4,$5,$6)`,
				r.ts, r.instID, r.bids, r.asks, r.seqID, r.checksum)
			n++
		case it.funding != nil:
			r := it.funding
			b.Queue(`INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_ts) VALUES($1,$2,$3,$4)`,
				r.ts, r.instID, r.fundingRate, r.nextFundingTs)
			n++
		case it.mark != nil:
			r := it.mark
			b.Queue(`INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES($1,$2,$3,$4)`,
				r.ts, r.instID, r.channel, r.px)
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
