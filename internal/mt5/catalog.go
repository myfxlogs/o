// catalog.go —— MT5 全目录粗扫臂（T-0408①影子评估层事实源）：broker
// SymbolList 自报目录（366 符）周期扫——逐符 SymbolParams 步进限速（低频
// RPC 非流，节拍守 swapLoop 同款 ≤1/4h+mtapi 限频面）+GetQuoteMany 批价
// 快照（mid 源——目录符无 tick 流，venue 侧未订阅径缺价=0 缺席如实）。
// 落 md_signal kind='mt5_swap'（覆盖面扩目录非新 kind——同键 upsert 存储=
// 目录行数非流水；消费者全谱按键读面零误伤）；事实域=swap_long/short+
// mid+contract_size——影子提名/影子蒸馏的上游。会话断=拍随会话终（同
// swapLoop sessCtx 语义）。
package mt5

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// catalogEvery 目录粗扫节拍（≤1/4h 与 swapEvery 同款——日级变量低频拍；
// sched-exempt-ok 慢特征节拍）。
const catalogEvery = 4 * time.Hour

// catalogPace 逐符步进限速（366 符连发=限频风险——拍内匀速步进，var=桩测
// 覆写锚）。120ms≈目录 44s/拍——4h 拍内零压。
var catalogPace = 120 * time.Millisecond

// catalogBatch quoteMany 单批上限（整目录一次拉=网关载荷风险——分批步进）。
const catalogBatch = 100

// catalogLoop 目录粗扫拍（swapLoop 同款 ticker 形——sessCtx 止=零泄漏）。
func (c *BridgeCollector) catalogLoop(ctx context.Context, cl *Client) {
	t := time.NewTicker(catalogEvery) // sched-exempt-ok：日级变量低频拍
	defer t.Stop()
	c.catalogSweep(ctx, cl)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.catalogSweep(ctx, cl)
		}
	}
}

// catalogSweep 单拍：SymbolList 全目录→GetQuoteMany mid 快照（败=mid 空集
// 降级不阻扫——InPoints 方言符无 mid 不可年化 val=NULL 缺席如实）→逐符
// SymbolParams 步进限速落 mt5_swap 行。全灭（零落行）=WARN 可观测事件
// （swapLoop 消哑同款——日志链拾取，collector 无 ops_event 写径）。
func (c *BridgeCollector) catalogSweep(ctx context.Context, cl *Client) {
	list := c.symList
	if list == nil {
		list = func(ctx context.Context, cl *Client) ([]string, error) { return cl.SymbolList(ctx) }
	}
	syms, err := list(ctx, cl)
	if err != nil {
		c.rej.Inc("symbols_read_failed")
		log.Printf("WARN [mt5bridge] 目录粗扫 SymbolList 败——本拍让位: %v", err)
		return
	}
	if len(syms) == 0 {
		c.rej.Inc("symbols_read_failed")
		log.Printf("WARN [mt5bridge] 目录粗扫 SymbolList 零行——本拍让位（venue 缺陷 fail-closed）")
		return
	}
	mids := c.catalogMids(ctx, cl, syms)
	okN, sample := 0, ""
	for i, sym := range syms {
		if ctx.Err() != nil {
			return
		}
		if i > 0 && catalogPace > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(catalogPace): // sched-exempt-ok：逐符限速步进（限频面守）
			}
		}
		mid := mids[sym]
		if mid == 0 {
			mid = c.midOf(sym) // 订阅符 tick 面回落（目录∩订阅集——鲜 mid 正道）
		}
		if why := c.fetchSwapMid(ctx, cl, sym, mid); why == "" {
			okN++
		} else if sample == "" {
			sample = why
		}
	}
	if okN == 0 {
		log.Printf("WARN [mt5bridge] 目录粗扫全灭——%d sym 零落行（面上盲拍，样本: %s）",
			len(syms), sample)
		return
	}
	log.Printf("[mt5bridge] 目录粗扫拍：%d/%d 落行（mid 快照 %d 符）", okN, len(syms), len(mids))
}

// catalogMids GetQuoteMany 批价快照→sym mid 表（分批步进；败=空集降级——
// venue 未订阅径返空亦同径降级，不阻 params 扫）。mid=(bid+ask)/2，双边零
// =不入表（缺席如实）。
func (c *BridgeCollector) catalogMids(ctx context.Context, cl *Client, syms []string) map[string]float64 {
	qm := c.quoteMany
	if qm == nil {
		qm = func(ctx context.Context, cl *Client, batch []string) map[string]float64 {
			r, err := cl.MT5().GetQuoteMany(cl.Authed(ctx), &mt5pb.GetQuoteManyRequest{
				Id: cl.Token(), Symbols: batch})
			out := map[string]float64{}
			if err != nil || r.GetError() != nil {
				return out // 批量败=空集降级（不阻 params 扫——缺席如实）
			}
			for _, q := range r.GetResult() {
				if q.GetBid() > 0 && q.GetAsk() > 0 {
					out[q.GetSymbol()] = (q.GetBid() + q.GetAsk()) / 2
				}
			}
			return out
		}
	}
	mids := map[string]float64{}
	for i := 0; i < len(syms); i += catalogBatch {
		end := i + catalogBatch
		if end > len(syms) {
			end = len(syms)
		}
		for sym, mid := range qm(ctx, cl, syms[i:end]) {
			mids[sym] = mid
		}
	}
	return mids
}

// fetchSwap 单 sym SymbolParams 读→裁定→换算→md_signal 落行（mid 自 md_ticks
// 现取——订阅符正道）。空返=落行成；非空=败因样本。FIX-02 消哑分键：
// transport err/in-band err/Result nil→symbols_read_failed；SymbolGroup nil→
// symbol_group_nil（原 177 行静默 continue 第二哑径——独立键留痕）；
// 写败→tick_write_err（既有键）。
func (c *BridgeCollector) fetchSwap(ctx context.Context, cl *Client, sym string) string {
	return c.fetchSwapMid(ctx, cl, sym, c.midOf(sym))
}

// fetchSwapMid 单 sym SymbolParams 读→裁定→换算→md_signal 落行（mid 外源——
// 目录粗扫携 GetQuoteMany 快照 mid，订阅符回落 midOf tick 面）。
func (c *BridgeCollector) fetchSwapMid(ctx context.Context, cl *Client, sym string, mid float64) string {
	fetch := c.symFetch
	if fetch == nil {
		fetch = func(ctx context.Context, cl *Client, sym string) (*mt5pb.SymbolParamsReply, error) {
			return cl.MT5().SymbolParams(cl.Authed(ctx), &mt5pb.SymbolParamsRequest{Id: cl.Token(), Symbol: sym})
		}
	}
	r, err := fetch(ctx, cl, sym)
	switch {
	case err != nil:
		c.rej.Inc("symbols_read_failed")
		return sym + ": " + err.Error()
	case r.GetError() != nil:
		c.rej.Inc("symbols_read_failed")
		return fmt.Sprintf("%s: mtapi err %d %q", sym, r.GetError().GetCode(), r.GetError().GetMessage())
	case r.Result == nil:
		c.rej.Inc("symbols_read_failed")
		return sym + ": Result nil"
	}
	sg := r.Result.SymbolGroup
	if sg == nil {
		c.rej.Inc("symbol_group_nil")
		return sym + ": SymbolGroup nil"
	}
	// 换算链全携 meta（票面钦定——点数→bps 链逐环可查）
	si := r.Result.SymbolInfo
	ann, ok := SwapAnnualBPS(si, sg, mid)
	longAnn, shortAnn, ok2 := SwapAnnualBoth(si, sg, mid) // 返修①双腿年化
	meta, _ := json.Marshal(map[string]any{
		"swap_long": sg.GetSwapLong(), "swap_short": sg.GetSwapShort(),
		"swap_type": int32(sg.GetSwapType())})
	if si != nil {
		meta, _ = json.Marshal(map[string]any{
			"swap_long": sg.GetSwapLong(), "swap_short": sg.GetSwapShort(),
			"swap_type":  int32(sg.GetSwapType()),
			"tick_value": si.GetTickValue(), "tick_size": si.GetTickSize(),
			"points": si.GetPoints(), "contract_size": si.GetContractSize(),
			"max_volume": float64(sg.GetMaxVolume()), "initial_margin": sg.GetInitialMargin(),
			// T-0144 返修①：双腿年化随 meta（正 funding 径 swap_long 成本端入账源）。
			"swap_long_ann_bps": longAnn, "swap_short_ann_bps": shortAnn, "ann_ok": ok2,
			"mid": mid})
	}
	var val any // NULL=单位方言不可年化（缺换算件缺席如实）
	if ok {
		val = ann // 空腿年化 bps（负 funding 对冲径正主）
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
		VALUES('mt5_swap',$1,$2,$3::jsonb,now())
		ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val,
		meta=EXCLUDED.meta, ts=EXCLUDED.ts`, sym, val, string(meta)); err != nil {
		c.rej.Inc("tick_write_err")
		return sym + ": write " + err.Error()
	}
	return ""
}
