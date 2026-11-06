// scan.go —— 粗扫层 REST 轮询臂（T-0401 两级采集制①——「禁 REST 轮询补
// 行情」的钦定例外：本件非执行价源，是发现面筛选精度层；执行/记账价仍
// 归 md_ticks WS 流——两级分工票面钦定）。OKX /market/tickers instType=
// SWAP 单请求全谱 last/bid/ask → md_tick_scan 同键 upsert 最新行（体积
// 纪律钦定：只存最新行，禁 tick 流水全谱）。节拍钦定 60s（okxws.
// tick_scan_s 热启可覆）；零交易语义——只读公共行情零凭据。
package okxws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
)

// scanPollS 钦定节拍（秒）——票面「60s 一拍」钉死值，config 覆写仅限调优。
const scanPollS = 60.0

// ScanGet REST 读件形（装配层挂 okxexec.Client.PublicGet——复用现有 HTTP
// 面禁新框架；返回体=data 数组原文）。
type ScanGet func(ctx context.Context, pathWithQuery string) (json.RawMessage, error)

// ScanPoller 粗扫轮询臂（独立协程非 WS 面——节拍件）。
type ScanPoller struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	rej  *rejects.Door
	get  ScanGet
	// Catalog 粗扫谱域=宇宙扩件目录（T-0416——探针与扫臂同一供给件同读：
	// tickers 行须∈SWAP 目录全集才落表，目录读败=fail-closed 本拍让位不
	// 虚构谱域；nil=谱域闸缺席——桩测/兼容面，生产装配必挂）。
	Catalog func(ctx context.Context) (map[string]bool, error)
}

// NewScanPoller 装配（get=nil 拒装——零凭据公开面读件仍须显式挂线）。
func NewScanPoller(pool *pgxpool.Pool, cfg *config.Client, get ScanGet) *ScanPoller {
	return &ScanPoller{pool: pool, cfg: cfg, get: get,
		rej: rejects.New("okxscan", "scan_fetch_err", "scan_write_err", "scan_empty")}
}

// Run 60s 拍环（起即首拍——不等首 tick 消盲区）。败=上抛 errc 岗终。
func (p *ScanPoller) Run(ctx context.Context, errc chan<- error) {
	iv := scanPollS
	if p.cfg != nil {
		if v, err := p.cfg.GetFloat("okxws.tick_scan_s"); err == nil && v > 0 {
			iv = v
		}
	}
	t := time.NewTicker(time.Duration(iv * float64(time.Second))) // sched-exempt-ok 钦定节拍件
	defer t.Stop()
	for {
		if err := p.ScanOnce(ctx); err != nil {
			errc <- fmt.Errorf("[okxscan] 粗扫拍: %w", err)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// ScanOnce 单拍：REST 全谱拉取→解析→单事务批量 upsert（同键最新赢——
// EXCLUDED.ts>现 ts 守卫同 uTicks 形，回戳帧不覆新）。
func (p *ScanPoller) ScanOnce(ctx context.Context) error {
	var cat map[string]bool
	if p.Catalog != nil {
		var err error
		if cat, err = p.Catalog(ctx); err != nil {
			p.rej.Inc("scan_fetch_err")
			return fmt.Errorf("宇宙目录读败——fail-closed 不虚构谱域: %w", err)
		}
	}
	raw, err := p.get(ctx, "/api/v5/market/tickers?instType=SWAP")
	if err != nil {
		p.rej.Inc("scan_fetch_err")
		return err
	}
	rows, err := parseScanTickers(raw)
	if err != nil {
		p.rej.Inc("scan_fetch_err")
		return err
	}
	if len(rows) == 0 {
		p.rej.Inc("scan_empty")
		return fmt.Errorf("tickers 空集——fail-closed（全谱拉取零行=面病灶非真空）")
	}
	b := &pgx.Batch{}
	for _, r := range rows {
		if cat != nil && !cat[r.instID] {
			continue // 目录全集外——谱域闸滤除（tickers 回包非目录员不落表）
		}
		b.Queue(`INSERT INTO md_tick_scan(ts,inst_id,last,bid,ask) VALUES($1,$2,$3,$4,$5)
			ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last,
			bid=EXCLUDED.bid,ask=EXCLUDED.ask,recv_ts=now()
			WHERE EXCLUDED.ts > md_tick_scan.ts`,
			r.ts, r.instID, r.last, r.bid, r.ask)
	}
	br := p.pool.SendBatch(ctx, b)
	if err := br.Close(); err != nil {
		p.rej.Inc("scan_write_err")
		return err
	}
	log.Printf("[okxscan] 粗扫拍 %d 符 upsert md_tick_scan", len(rows))
	return nil
}

// parseScanTickers REST tickers data 数组解析（instId/last/bidPx/askPx/ts
// 毫秒戳；USDT 键域闸同订阅面——杂币不采令全域同律）。
func parseScanTickers(raw json.RawMessage) ([]tickRow, error) {
	var rows []struct {
		InstID string `json:"instId"`
		Last   string `json:"last"`
		BidPx  string `json:"bidPx"`
		AskPx  string `json:"askPx"`
		Ts     string `json:"ts"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("tickers 解析: %w", err)
	}
	f := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
	out := make([]tickRow, 0, len(rows))
	for _, r := range rows {
		if !isUSDTInst(r.InstID) {
			continue
		}
		ms, _ := strconv.ParseInt(r.Ts, 10, 64)
		out = append(out, tickRow{
			instID: r.InstID,
			ts:     time.UnixMilli(ms).UTC(),
			last:   f(r.Last), bid: f(r.BidPx), ask: f(r.AskPx),
		})
	}
	return out, nil
}
