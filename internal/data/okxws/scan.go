// scan.go —— 粗扫层 REST 轮询臂（T-0401 两级采集制①——「禁 REST 轮询补
// 行情」的钦定例外：本件非执行价源，是发现面筛选精度层；执行/记账价仍
// 归 md_ticks WS 流——两级分工票面钦定）。T-0427 粗扫扩全宇宙：按族各一
// 发 REST（SWAP/SPOT/FUTURES 三族——60s 节律不变，扫面 ×3 实机约 +800
// 行最新行；overwrite 语义表型不变——体积纪律钦定只存最新行禁流水），
// 行携 inst_type 族标（显式 > inst_id 键内隐式——族判禁散落正则复制件）。
// 谱域闸按族各判：该族目录读败=该族本拍让位不虚构，他族照常——fail-
// closed 粒度到族；全族败=面病灶本拍 loud 岗终（原形同守）。零交易语义
// ——只读公共行情零凭据。
package okxws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/scanwm"
)

// scanPollS 钦定节拍（秒）——票面「60s 一拍」钉死值，config 覆写仅限调优。
const scanPollS = 60.0

// ScanGet REST 读件形（装配层挂 okxexec.Client.PublicGet——复用现有 HTTP
// 面禁新框架；返回体=data 数组原文）。
type ScanGet func(ctx context.Context, pathWithQuery string) (json.RawMessage, error)

// ScanGate 族谱域闸读件形（族→目录全集——装配层挂 UniverseCatalog.FamSet，
// 同供给件同 TTL 同 fail-closed）。
type ScanGate func(ctx context.Context) (map[string]bool, error)

// venueOKX 本面 venue 域（md_tick_scan.venue 列默认值——水位闸判例域锚）。
const venueOKX = "okx"

// famReg 族注册行（scanFams 册行形——instType=OKX 原词；col=md_tick_scan
// 族标小写归一）。
type famReg struct{ instType, col string }

// scanFams 扫族册（T-0427——instType=OKX 原词按序各一发；col=md_tick_scan.
// inst_type 族标小写归一；OPTION/MARGIN 票面不收——无消费面）。
// T-0440：本册=scan 注册面正本——水位闸期望族集唯一派生源（非闸内写死；
// 新族注册即自动入闸域）。
var scanFams = []famReg{
	{"SWAP", "swap"}, {"SPOT", "spot"}, {"FUTURES", "futures"},
}

// ScanPoller 粗扫轮询臂（独立协程非 WS 面——节拍件）。
type ScanPoller struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	rej  *rejects.Door
	get  ScanGet
	// Gates 逐族谱域闸（T-0427——族=目录全集读件，同供给件同读；族键
	// 缺席=该族 fail-closed 让位不虚构；全域 nil=谱域闸缺席——桩测/兼容
	// 面，行直落不拦，生产装配必挂）。
	Gates map[string]ScanGate
	// fams 注册面=本 venue 应扫族集（T-0440——唯一期望源，装配时定）。
	fams []famReg
	// WM 族谱水位闸（T-0440③——注册面派生自动挂线；nil=守位桩测面）。
	WM *scanwm.Gate
}

// NewScanPoller 装配（get=nil 拒装——零凭据公开面读件仍须显式挂线）。
// T-0440：水位闸随装配自动入闸域（期望集=本册注册面派生——闸体零 venue
// 硬编码；pool 缺=守位 nil 闸）。
func NewScanPoller(pool *pgxpool.Pool, cfg *config.Client, get ScanGet) *ScanPoller {
	p := &ScanPoller{pool: pool, cfg: cfg, get: get, fams: scanFams,
		rej: rejects.New("okxscan", "scan_fetch_err", "scan_write_err", "scan_empty")}
	p.WM = newWatermark(pool, p.fams)
	return p
}

// newWatermark 族谱水位闸装配（T-0440③a）：期望集=注册面派生；实落读缝=
// md_tick_scan 逐族 recv_ts 水位（鲜落族每拍前进——龄超阈=零鲜落持续实证）；
// 升件面=ops_event 同池。
func newWatermark(pool *pgxpool.Pool, fams []famReg) *scanwm.Gate {
	if pool == nil {
		return nil // 守位——桩测/缺池面
	}
	cols := make([]string, len(fams))
	for i, f := range fams {
		cols[i] = f.col
	}
	return scanwm.New(pool, "okxscan", "T-0440", venueOKX, cols,
		func(ctx context.Context, fam string) (*time.Time, error) {
			var wm *time.Time
			err := pool.QueryRow(ctx,
				`SELECT max(recv_ts) FROM md_tick_scan WHERE venue=$1 AND inst_type=$2`,
				venueOKX, fam).Scan(&wm)
			return wm, err
		})
}

// Run 60s 拍环（起即首拍——不等首 tick 消盲区）。拍败=让位记拍下拍续试
// 不拖死采集主进程（T-0446：errc 岗终=全臂死——同进程 WS tick/sigdist
// 姊妹臂陪葬；瞬断自愈，真死由 scanwm 族谱水位闸+depblind 观测盲探针
// 升件收口，拒计数拍内已记）。
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
			log.Printf("[okxscan] 粗扫拍让位: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// ScanOnce 单拍：逐族 REST 全谱拉取→族闸滤→解析→单事务批量 upsert
// （同键最新赢——EXCLUDED.ts>现 ts 守卫同 uTicks 形，回戳帧不覆新）。
// 族败让位他族照常（fail-closed 粒度到族）；全族败=本拍 loud 上抛岗终。
func (p *ScanPoller) ScanOnce(ctx context.Context) error {
	b := &pgx.Batch{}
	counts := make(map[string]int, len(p.fams))
	var famErrs []error
	for _, f := range p.fams {
		n, err := p.scanFam(ctx, f.instType, f.col, b)
		if err != nil {
			famErrs = append(famErrs, fmt.Errorf("%s 族: %w", f.instType, err))
			continue // 族让位——他族照常（票面钦定 fail-closed 粒度到族）
		}
		counts[f.instType] = n
	}
	if len(famErrs) == len(p.fams) {
		return errors.Join(famErrs...) // 全族败=面病灶——同原 loud 形
	}
	for _, e := range famErrs {
		log.Printf("[okxscan] 粗扫族让位: %v", e) // 部分让位显式不静默
	}
	br := p.pool.SendBatch(ctx, b)
	if err := br.Close(); err != nil {
		p.rej.Inc("scan_write_err")
		return err
	}
	log.Printf("[okxscan] 粗扫拍 swap=%d spot=%d futures=%d upsert md_tick_scan",
		counts["SWAP"], counts["SPOT"], counts["FUTURES"])
	p.WM.Beat(ctx) // 族谱水位闸拍（T-0440——观测面零侵入：nil 守位/败 WARN 不拖扫径）
	return nil
}

// scanFam 单族拍：族谱域闸→REST tickers→解析→键域滤→族标落行队列。
// 族内任一环败=族让位（err 上抛——rej 计数在环内）；队列行携 inst_type
// 族标（显式 > 键内隐式——票面钦定形）。
func (p *ScanPoller) scanFam(ctx context.Context, instType, col string, b *pgx.Batch) (int, error) {
	var gate map[string]bool
	if p.Gates != nil {
		g, ok := p.Gates[instType]
		if !ok {
			p.rej.Inc("scan_fetch_err")
			return 0, fmt.Errorf("谱域闸缺席——fail-closed 该族让位")
		}
		var err error
		if gate, err = g(ctx); err != nil {
			p.rej.Inc("scan_fetch_err")
			return 0, fmt.Errorf("宇宙目录读败——fail-closed 不虚构谱域: %w", err)
		}
	}
	raw, err := p.get(ctx, "/api/v5/market/tickers?instType="+instType)
	if err != nil {
		p.rej.Inc("scan_fetch_err")
		return 0, err
	}
	rows, err := parseScanTickers(raw)
	if err != nil {
		p.rej.Inc("scan_fetch_err")
		return 0, err
	}
	if len(rows) == 0 {
		p.rej.Inc("scan_empty")
		return 0, fmt.Errorf("tickers 空集——fail-closed（%s 全谱拉取零行=面病灶非真空）", instType)
	}
	n := 0
	for _, r := range rows {
		if gate != nil && !gate[r.instID] {
			continue // 目录全集外——谱域闸滤除（tickers 回包非目录员不落表）
		}
		b.Queue(`INSERT INTO md_tick_scan(ts,inst_id,inst_type,last,bid,ask) VALUES($1,$2,$3,$4,$5,$6)
			ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last,
			bid=EXCLUDED.bid,ask=EXCLUDED.ask,inst_type=EXCLUDED.inst_type,recv_ts=now()
			WHERE EXCLUDED.ts > md_tick_scan.ts`,
			r.ts, r.instID, col, r.last, r.bid, r.ask)
		n++
	}
	return n, nil
}

// parseScanTickers REST tickers data 数组解析（instId/last/bidPx/askPx/ts
// 毫秒戳；USDT 键域闸同订阅面——杂币不采令全域同律；族标由扫描臂按族
// 发面显式携带——本件不做 inst_id 形状族判，禁猜钦定）。
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
