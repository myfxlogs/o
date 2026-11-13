// klineback.go —— 日K REST 自举回填臂（工单#13065 切片③）。
// 病灶原型：WS candle 频道只产尾流（连接起向前流）——trend 类判据需
// 30d+ 史,纯 WS 面需蓄史一月方可评估。/api/v5/market/history-candles
// =venue 史面唯一径（WS 无史等价=事务型单发域,okx-rest-boundary
// 钦定保留形非退役轮询;venue 实读定案:bar=1D 与 candle1D 开点同位
// 16:00Z,confirm=1 收线史新→旧序单页≤100）。
// 臂律：只补「confirm=1 收线行且库内无该 ts」——形成中帧归 WS 臂
// （REST 快照陈,禁覆写 WS 新鲜态/收线终态）;已收线行跳过（终态
// 幂等省写,稳态扫≈零写）。启动即扫+定节复扫（断连窗缺齿回补）。
// 写径=queue 同径（writer upsert 单写者,回填不另开写面）。
package okxws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const (
	// klineBackLimit 单页行数（OKX history-candles 上限 100≈100d 史）。
	klineBackLimit = 100
	// klineBackEvery 复扫节拍（断连窗缺齿回补——日级足够,6h 收两头）。
	klineBackEvery = 6 * time.Hour
	// klineBackGap 逐 inst 间隔（20req/2s 界内慢放——活跃集 40+inst
	// 全扫≈10s,不占 venue 配额）。
	klineBackGap = 200 * time.Millisecond
)

// AttachRestGet 注 REST 公开读件（装配层挂 okxexec.PublicGet——
// okxws→okxexec 方向闸外,ScanGet 缝注入同 universeGet 先例；
// nil=面缺席,回填臂停拍不炸）。
func (c *Collector) AttachRestGet(get ScanGet) { c.restGet = get }

// backfillLoop 回填臂拍：启动即扫（冷启蓄史）+定节复扫（缺齿回补）。
func (c *Collector) backfillLoop(ctx context.Context) {
	if c.restGet == nil {
		return // 读件缺席=面未装——静默退（装配层负全责注入）
	}
	c.backfillSweep(ctx)
	t := time.NewTicker(klineBackEvery) // sched-exempt-ok 史面补齿定节拍
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.backfillSweep(ctx)
		}
	}
}

// backfillSweep 单扫：活跃集逐 inst 拉单页→滤 confirm=1 收线行→
// 剔库内已有 ts→缺齿行入队（阻塞式——队列背压等 writer 排空,
// 回填行不弃；ctx 死让位）。
func (c *Collector) backfillSweep(ctx context.Context) {
	set, err := c.activeSet(ctx)
	if err != nil {
		log.Printf("[okxws] kline 回填活跃集读败（本扫让位）: %v", err)
		return
	}
	filled := 0
	for inst := range set {
		if ctx.Err() != nil {
			return
		}
		raw, err := c.restGet(ctx, fmt.Sprintf(
			"/api/v5/market/history-candles?instId=%s&bar=1D&limit=%d", inst, klineBackLimit))
		if err != nil {
			c.rej.Inc("klineback_err")
			continue // 单 inst 败不连坐（期权/SIM 面 venue 无 K 线=常态拒）
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			c.rej.Inc("klineback_err")
			continue
		}
		// 库内已收线 ts 集（终态幂等——只补缺齿不覆写）。
		have := map[int64]bool{}
		qrows, err := c.pool.Query(ctx,
			`SELECT ts FROM md_klines WHERE venue='okx' AND inst_id=$1 AND bar='1D' AND confirm`, inst)
		if err != nil {
			c.rej.Inc("klineback_err")
			continue
		}
		for qrows.Next() {
			var ts time.Time
			if qrows.Scan(&ts) == nil {
				have[ts.UnixMilli()] = true
			}
		}
		qrows.Close()
		for _, e := range rows {
			k := parseCandle(inst, "1D", e)
			if k == nil || !k.confirm || have[k.ts.UnixMilli()] {
				continue // 形成中帧归 WS 臂/已收线终态免写
			}
			select {
			case c.queue <- rowItem{kline: k}:
				filled++
			case <-ctx.Done():
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(klineBackGap): // sched-exempt-ok inst 慢放间隔
		}
	}
	log.Printf("[okxws] kline 回填拍: insts=%d 补行=%d", len(set), filled)
}
