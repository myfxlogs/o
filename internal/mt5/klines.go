// klines.go —— MT5 D1 日K喂径（工单#25563 异因子数据面扩）：全目录
// PriceHistoryExMany 分批拉 D1 棒→md_klines venue='mt5' 同键 upsert
// （venue 正本=broker 服务端棒——共写语义同键最新帧赢）。
// 判词面：末棒=形成中帧 confirm=false（次日扫覆写至收线——okxws klines
// 同款语义）；TimeFrame=16408 MT5 PERIOD_D1 枚举实证锚（mt5-quotes-history
// skill 注 16408 存歧——以落行 ts 间距自验，非 D1 即日志现形）。
package mt5

import (
	"context"
	"log"
	"time"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// klineEvery 日K扫拍（D1=日级变量低频拍——catalogEvery 同款；拍内分批步进）。
const klineEvery = 4 * time.Hour

// klineBatch ExMany 单批符号上限（369 符整目录一次拉=网关载荷风险——catalogBatch 同款分批）。
const klineBatch = 60

// klineNumBars 单符回填深度（dirtrend eval 需 lookback20+replay60+1=81 棒——
// 200 棒冗余：FX 周线休市+假日缺口,cal 窗 200d 保底 81 根确认棒；
// 后续拍同量幂等覆写零增量成本）。
const klineNumBars = 200

// klineFrom 回填锚窗——From 为网关必填 string 形 DateTime 且语义=**末锚**
// （首扫实证：From=now-210d 拉回棒齐截 2026-04-19=锚前一日——网关从
// From 回数 numBars 非起点顺拉；now+2d 兜住今日进行棒，远端忽略未来段）。
func klineFrom(now time.Time) string {
	return now.AddDate(0, 0, 2).UTC().Format("2006-01-02T15:04:05Z")
}

// mt5FrameD1 PERIOD_D1 枚举值（MT5 标准 ENUM_TIMEFRAMES——实证钉:扫后
// md_klines ts 间距≠24h 即枚举错配,WARN 行现形,不静默）。
const mt5FrameD1 = 16408

// klinePace 批间限速（369 符/60=7 批连发=限频面守——catalogPace 同款步进）。
var klinePace = 300 * time.Millisecond

// klineLoop D1 扫拍环（catalogLoop 同款 ticker 形——sessCtx 止=零泄漏）。
func (c *BridgeCollector) klineLoop(ctx context.Context, cl *Client) {
	t := time.NewTicker(klineEvery) // sched-exempt-ok：日级变量低频拍
	defer t.Stop()
	c.klineSweep(ctx, cl)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.klineSweep(ctx, cl)
		}
	}
}

// klineSweep 单拍：SymbolList 全目录→ExMany 分批 D1 棒→逐棒 upsert。
// 目录读败/全灭（零落行）=WARN 可观测事件（catalogSweep 消哑同款——
// collector 无 ops_event 写径，日志链拾取）。
func (c *BridgeCollector) klineSweep(ctx context.Context, cl *Client) {
	list := c.symList
	if list == nil {
		list = func(ctx context.Context, cl *Client) ([]string, error) { return cl.SymbolList(ctx) }
	}
	syms, err := list(ctx, cl)
	if err != nil {
		c.rej.Inc("kline_symbols_failed")
		log.Printf("WARN [mt5bridge] 日K扫 SymbolList 败——本拍让位: %v", err)
		return
	}
	okN, sample := 0, ""
	for i := 0; i < len(syms); i += klineBatch {
		if ctx.Err() != nil {
			return
		}
		if i > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(klinePace): // sched-exempt-ok：分批步进限速
			}
		}
		end := i + klineBatch
		if end > len(syms) {
			end = len(syms)
		}
		if n, why := c.klineBatchWrite(ctx, cl, syms[i:end]); why != "" {
			sample = why
		} else {
			okN += n
		}
	}
	if okN == 0 {
		log.Printf("WARN [mt5bridge] 日K扫全灭——%d sym 零落行（样本: %s）", len(syms), sample)
		return
	}
	log.Printf("[mt5bridge] 日K扫拍：%d 棒落行（目录 %d 符）", okN, len(syms))
}

// klineBatchWrite 单批:ExMany→逐符逐棒 upsert;返落棒数+败样（批败=why非空）。
func (c *BridgeCollector) klineBatchWrite(ctx context.Context, cl *Client, batch []string) (int, string) {
	r, err := mt5pb.NewQuoteHistoryClient(cl.Conn()).PriceHistoryExMany(cl.Authed(ctx),
		&mt5pb.PriceHistoryExManyRequest{
			Id: cl.Token(), Symbol: batch, From: klineFrom(time.Now()),
			NumBars: klineNumBars, TimeFrame: mt5FrameD1})
	if err != nil {
		c.rej.Inc("kline_read_failed")
		return 0, err.Error()
	}
	if r.GetError() != nil {
		c.rej.Inc("kline_read_failed")
		return 0, r.GetError().String()
	}
	n := 0
	for _, res := range r.GetResult() {
		if res.GetException() != "" {
			continue // 单符 venue 侧缺史=缺席如实（Exception 域载真因,不算批败）
		}
		bars := res.GetBars()
		for j, b := range bars {
			confirm := j < len(bars)-1 // 末棒=形成中帧(收线后次拍终态落)
			if _, err := c.pool.Exec(ctx, `INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm)
				VALUES('mt5',$1,'1D',$2,$3,$4,$5,$6,$7,$8,$9)
				ON CONFLICT (venue,inst_id,bar,ts) DO UPDATE SET o=EXCLUDED.o,h=EXCLUDED.h,l=EXCLUDED.l,
				c=EXCLUDED.c,vol=EXCLUDED.vol,vol_ccy=EXCLUDED.vol_ccy,confirm=EXCLUDED.confirm,recv_ts=now()`,
				res.GetSymbol(), b.GetTime().AsTime(), b.GetOpenPrice(), b.GetHighPrice(),
				b.GetLowPrice(), b.GetClosePrice(), float64(b.GetTickVolume()), 0.0, confirm); err != nil {
				c.rej.Inc("kline_write_err")
				return n, err.Error()
			}
			n++
		}
	}
	return n, ""
}
