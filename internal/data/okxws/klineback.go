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
	"sort"
	"strconv"
	"strings"
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
	// klineTopN 流动性序扩集缺省额度（okxws.kline_topn 可覆,0=关动扩）。
	klineTopN = 50
	// klineCensusHours 全 USDT-SWAP 普查缺省降频（okxws.kline_census_hours
	// 可覆,0=关普查——新上零量标的补盲窗）。
	klineCensusHours = 168
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

// backfillSweep 单扫：（活跃集∪观察名单∪动扩集）逐 inst 拉单页→滤 confirm=1
// 收线行→剔库内已有 ts→缺齿行入队（阻塞式——队列背压等 writer 排空,
// 回填行不弃；ctx 死让位）。业主令 2026-11-15：okxws.kline_watchlist
// 观察名单并入集——活跃集=「在交易什么」闭集,无名单补充则新标的结构性
// 失明（无棒→dirmom 无法评估→不提案→永不进集=闭环锁死实证）。
// 工单#18314（自我裁#19987）：名单之外再挂流动性序动扩——机制件见 univExpand。
func (c *Collector) backfillSweep(ctx context.Context) {
	set, err := c.activeSet(ctx)
	if err != nil {
		log.Printf("[okxws] kline 回填活跃集读败（本扫让位）: %v", err)
		return
	}
	if c.cfg != nil {
		watchUnion(set, c.cfg.GetStringList("okxws.kline_watchlist"))
	}
	c.univExpand(ctx, set)
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

// watchUnion 观察名单并入扫集（纯函数可测面——空名单零副作用,
// 已入集键幂等）。名单=基建感知面（venue 无该 inst=单发败不连坐）。
func watchUnion(set map[string]bool, watch []string) {
	for _, inst := range watch {
		if inst != "" {
			set[inst] = true
		}
	}
}

// univExpand 宇宙动态扩集（工单#18314——自我裁#19987：①流动性序动态扩集
// 为主径,②全 SWAP 普查降频为辅）。venue SWAP tickers 单发全谱拉取按
// volCcy24h 降序：①主径=top okxws.kline_topn（缺省 klineTopN,0=关）每扫
// 并入集,长尾标的按量序自入视野零人工；②辅径=全 USDT-SWAP 普查降频并集
// （okxws.kline_census_hours 缺省 klineCensusHours,0=关——捕零量新上/极
// 长尾,零量标的永远进不了 topN 序）。棒落 md_klines 即入 dirmom 宇宙
// （dirmom 宇宙=md_klines inst 集）——扩集→蓄史→评估链零配置零写面。
// REST 败=本扫让位不连坐（同 watchUnion 单发律）；读拍驱动不动 app_config。
func (c *Collector) univExpand(ctx context.Context, set map[string]bool) {
	if c.restGet == nil {
		return // 读件缺席=臂停（测试构面免崩——backfillLoop 已先行拦,双闸）
	}
	topn, censusH := klineTopN, klineCensusHours
	if c.cfg != nil {
		if v, err := c.cfg.GetInt("okxws.kline_topn"); err == nil {
			topn = int(v)
		}
		if v, err := c.cfg.GetInt("okxws.kline_census_hours"); err == nil {
			censusH = int(v)
		}
	}
	census := censusH > 0 && time.Since(c.lastCensus) >= time.Duration(censusH)*time.Hour
	if topn <= 0 && !census {
		return // 双径全关=纯活跃集∪名单行为（闸回退零副作用）
	}
	raw, err := c.restGet(ctx, "/api/v5/market/tickers?instType=SWAP")
	if err != nil {
		c.rej.Inc("univ_expand_err")
		return
	}
	ranked := rankSwapTickers(raw)
	if census {
		for _, t := range ranked {
			set[t.inst] = true
		}
		c.lastCensus = time.Now()
		return
	}
	for i, t := range ranked {
		if i >= topn {
			break
		}
		set[t.inst] = true
	}
}

type swapTicker struct {
	inst string
	vol  float64
}

// rankSwapTickers SWAP tickers 载荷排序（纯函数可测面）——滤 USDT-SWAP
// （业主令 USDT 计价闸：币本位/现货不采不评）,volCcy24h 降序。
func rankSwapTickers(raw json.RawMessage) []swapTicker {
	var rows []struct {
		InstID    string `json:"instId"`
		VolCcy24h string `json:"volCcy24h"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil
	}
	out := make([]swapTicker, 0, len(rows))
	for _, r := range rows {
		if !strings.HasSuffix(r.InstID, "-USDT-SWAP") {
			continue
		}
		v, _ := strconv.ParseFloat(r.VolCcy24h, 64)
		out = append(out, swapTicker{r.InstID, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].vol > out[j].vol })
	return out
}
