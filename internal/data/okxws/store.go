package okxws

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

// writer 异步批量落库协程：行队列→500ms/128 行触发 SendBatch（读循环零同步 DB 写——
// 背压不连坐 WS 读径，满丢弃已计 write_err 闸）。
func (c *Collector) writer(ctx context.Context) {
	buf := make([]rowItem, 0, 128)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	var lastFlushFunding []rowItem // T-0197②：funding 行败后入缓重试一拍（幂等键同名无害）
	flush := func() {
		if len(buf) == 0 {
			return
		}
		// T-0197②：败弃全批静默根治——WARN 批大小/桶分布落行；
		// funding 行败后重试一拍（窗口消费幂等键同拍同值无害）。
		if fset := append(lastFlushFunding, fundingOnly(buf)...); len(fset) > 0 {
			buf2 := append(fset, buf...)
			lastFlushFunding = nil
			buf = buf2 // funding 缓行重试一拍再入批
		}
		sort.SliceStable(buf, func(i, j int) bool { // T-0198：锁序归一（键序与 projector 镜像批同律——同键集恒序死锁根修）
			ki, kj := rowKey(buf[i]), rowKey(buf[j])
			return ki < kj
		})
		if err := c.insertBatch(ctx, buf); err != nil {
			c.rej.Inc("write_err")
			lastFlushFunding = fundingOnly(buf) // funding 行入缓待下拍重试
			log.Printf("[okxws] insertBatch 败：批=%d 行（ticks=%d funding=%d mark=%d depth5=%d）: %v",
				len(buf), countKind(buf, "tick"), countKind(buf, "funding"), countKind(buf, "mark"), countKind(buf, "depth5"), err)
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
			// T-0169：md_funding UNIQUE(venue,inst_id,ts)（fin_0066 双链同加）——
			// funding-rate 帧 ts=结算拍（结算前恒值），逐拍裸 INSERT 必撞键；
			// 同键同拍即同值 DO NOTHING（D-0030 §六 per-table 键表）。
			b.Queue(`INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_ts,next_funding_rate) VALUES($1,$2,$3,$4,$5)
				ON CONFLICT (venue,inst_id,ts) DO NOTHING`,
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

// rowKey 批行冲突键序件（锁序归一判键——venue 恒 'okx' 不入键；
// inst_id 主键+channel/桶类次键）。
func rowKey(it rowItem) string {
	switch {
	case it.tick != nil:
		return "1:" + it.tick.instID
	case it.funding != nil:
		return "2:" + it.funding.instID
	case it.mark != nil:
		return "3:" + it.mark.instID + ":" + it.mark.channel
	case it.depth5 != nil:
		return "4:" + it.depth5.instID
	}
	return "9:"
}

// fundingOnly 批内 funding 行过滤件（T-0197②——重试一拍缓行径）。
func fundingOnly(buf []rowItem) []rowItem {
	out := make([]rowItem, 0, len(buf))
	for _, it := range buf {
		if it.funding != nil {
			out = append(out, it)
		}
	}
	return out
}

// countKind 桶分布计数（WARN 日志断点诊断件）。
func countKind(buf []rowItem, kind string) int {
	n := 0
	for _, it := range buf {
		switch kind {
		case "tick":
			if it.tick != nil {
				n++
			}
		case "funding":
			if it.funding != nil {
				n++
			}
		case "mark":
			if it.mark != nil {
				n++
			}
		case "depth5":
			if it.depth5 != nil {
				n++
			}
		}
	}
	return n
}
