// Package earn —— Earn 停泊态+动态比较底执行面（D-0006 业主令「金钱不眠」）。
// 员工「配管」（编制册投资域既有卡：闲置不瞒报/承接位自动枚）在岗三件套：
//  1. Earn 现率采集：公开面 lending-rate-summary→md_earn_rate 事实行（采不到=告警
//     计数不入假值）；研究域动态比较底与停泊决策同源此表。
//  2. 扫额回拢：交易户闲余 USDT（WS 快照行 avail，鲜<5min）>earn.sweep_min_usd（热启）
//     →EarnTransfer(deposit) 回存；失败=告警计数+下拍重试（不静默）。
//  3. 开仓前置赎回 RedeemToTrading：执行链 leg1 前调用（EarnTransfer(redeem,
//     notional+buffer)）；赎回/划转败=fail-closed 不开仓。
//
// 留痕：每次钱动落 fin_risk_decision（reason=transfer——内部移动不走敞口判）+
// fin_cashflow earn_in/earn_out（source_ref=decision 行，禁推断记账）。
// 快照纪律：earn 余额快照只事件锚定写（boot 首读+移动后刷）——周期态摄入禁新增。
package earn

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/rejects"
	"phase/internal/platform/roster"
	"phase/internal/trading/okxexec"
)

const (
	defaultVenue     = "okx"
	defaultCcy       = "USDT" // Earn 停泊本位（杂币不入账闭集；枚举面限 USDT 同规）
	beatEvery        = 30 * time.Second
	scanEvery        = 60 * time.Second
	snapshotFreshMax = 5 * time.Minute // unified 快照新鲜度门（okx-rest-boundary 同律）
	parkingSrcBoot   = "earn_boot"     // 快照行 src：boot 首读
	parkingSrcMove   = "earn_move"     // 快照行 src：移动后事件锚定刷
)

// VenueAPI venue 停泊面（注入式——earn 不识 venue 方言，产线=okxexec.Client）。
type VenueAPI interface {
	SavingsPurchase(ctx context.Context, ccy string, amt float64) error
	SavingsRedeem(ctx context.Context, ccy string, amt float64) error
	// SavingsBalance raw 返回（client.go T-holding 契约——earn 侧按 ccy 过滤解析）。
	SavingsBalance(ctx context.Context) (json.RawMessage, error)
	AssetTransfer(ctx context.Context, ccy string, amt float64, from, to string) (string, error)
	AssetTransferState(ctx context.Context, ccy, clTID string) (string, error)
}

// Worker 配管在岗体（rateFn 可注入=测试面；api 缺席=仅现率采集在岗；
// venue/ccy 字段化——生产默认 okx/USDT，测试键域隔离用）。
type Worker struct {
	pool   *pgxpool.Pool
	cfg    *config.Client
	rej    *rejects.Door
	api    VenueAPI
	venue  string
	ccy    string
	rateFn func(ctx context.Context) (okxexec.RateSummary, error)
	mu     sync.Mutex // 单飞锁：redeem 持锁必进（开仓不排队），sweep TryLock 让位
}

// NewWorker 装配（api 未挂接=现率采集岗；AttachVenue 后停泊钱动上岗）。
func NewWorker(pool *pgxpool.Pool, cfg *config.Client) *Worker {
	return &Worker{pool: pool, cfg: cfg, venue: defaultVenue, ccy: defaultCcy,
		rej: rejects.New("earn", "rate_fetch_failed", "rate_parse_failed", "missing_param",
			"snapshot_stale", "sweep_skip_inflight", "sweep_busy", "venue_absent",
			"deposit_failed", "transfer_ambiguous", "compensation_failed",
			"snapshot_refresh_failed", "write_err"),
		rateFn: func(ctx context.Context) (okxexec.RateSummary, error) {
			return okxexec.LendingRateSummary(ctx, defaultCcy)
		}}
}

// AttachVenue 挂接 venue 停泊面（cmd 装配层凭据就位后调用——earn 不解凭据）。
func (w *Worker) AttachVenue(api VenueAPI) { w.api = api }

// Run 配管上岗（Register+BeatLoop 30s+扫描循环；boot 首读刷停泊快照——事件锚定形）。
func (w *Worker) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, w.pool, "配管", "portfolio", "闲置承接岗——Earn 停泊+现率采集", nil); err != nil {
		errc <- fmt.Errorf("配管 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, w.pool, "配管", beatEvery, errc)
	if err := w.refreshParkingSnapshot(ctx, parkingSrcBoot); err != nil {
		log.Printf("[earn] boot 停泊快照缺席（凭据落位后移动时自愈）: %v", err)
	}
	for {
		if err := w.scan(ctx); err != nil {
			log.Printf("[earn] scan: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(w.pollEvery()):
		}
	}
}

// pollEvery 采样拍长（data.earn.poll_sec 热启；节拍非判据——键缺/坏降默认拍
// 不驻停，停泊面不因节拍键缺而停摆）。
func (w *Worker) pollEvery() time.Duration {
	if sec, err := w.cfg.GetInt("data.earn.poll_sec"); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}
	return scanEvery
}

// scan 一拍两件：现率采样（独立——败不连坐扫额）+扫额回拢。
func (w *Worker) scan(ctx context.Context) error {
	w.pollRate(ctx)
	return w.sweepIdle(ctx)
}

// pollRate 现率采样→md_earn_rate 事实行（append-only；采不到=计数不入假值）。
func (w *Worker) pollRate(ctx context.Context) {
	rs, err := w.rateFn(ctx)
	if err != nil {
		w.rej.Inc("rate_fetch_failed")
		log.Printf("ALERT [earn] Earn 现率采样败（不入假值）: %v", err)
		return
	}
	if rs.AvgRate <= 0 {
		w.rej.Inc("rate_parse_failed")
		log.Printf("ALERT [earn] Earn 现率采样空值（avgRate=%v）——不入假值", rs.AvgRate)
		return
	}
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_earn_rate(ts,venue,ccy,rate) VALUES(now(),$1,$2,$3)`, w.venue, w.ccy, rs.AvgRate); err != nil {
		w.rej.Inc("write_err")
		log.Printf("[earn] md_earn_rate 写败: %v", err)
	}
}

// sweepIdle 闲余>阈值（热启）→回存停泊。判据源：unified 快照行 avail（WS 事件源，
// 鲜<5min fail-closed 不动钱）；执行在途（state=open）让位防饿腿。
func (w *Worker) sweepIdle(ctx context.Context) error {
	thr, err := w.getF("earn.sweep_min_usd")
	if err != nil {
		w.rej.Inc("missing_param")
		return fmt.Errorf("earn.sweep_min_usd: %w", err)
	}
	idle, ok, err := w.tradingIdleUSDT(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return nil // 计数门已挂（snapshot_stale）——陈快照 fail-closed 不动钱
	}
	if idle <= thr {
		return nil
	}
	var inFlight int
	if err := w.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_execution WHERE state='open'").Scan(&inFlight); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	if inFlight > 0 {
		w.rej.Inc("sweep_skip_inflight")
		return nil
	}
	if !w.mu.TryLock() {
		w.rej.Inc("sweep_busy")
		return nil
	}
	defer w.mu.Unlock()
	w.depositIdle(ctx, idle)
	return nil
}

// tradingIdleUSDT 交易户闲余（unified 快照行 avail；缺行/陈=fail-closed ok=false）。
func (w *Worker) tradingIdleUSDT(ctx context.Context) (float64, bool, error) {
	var avail float64
	var up time.Time
	err := w.pool.QueryRow(ctx, `SELECT avail,updated_at FROM fin_venue_account_snapshot
		WHERE venue=$1 AND venue_type='unified' AND ccy=$2`, w.venue, w.ccy).Scan(&avail, &up)
	if err != nil {
		w.rej.Inc("snapshot_stale")
		return 0, false, nil // 快照行缺席=闲余不可知——禁动钱（禁虚构判据）
	}
	if time.Since(up) > snapshotFreshMax {
		w.rej.Inc("snapshot_stale")
		log.Printf("ALERT [earn] unified 快照陈（updated_at=%s>5min）——扫额让位 fail-closed", up)
		return 0, false, nil
	}
	return avail, true, nil
}

// getF 标量判据读（typed client——T-0018 标量契约根修口径；缺/坏=fail-closed 上抛）。
func (w *Worker) getF(k string) (float64, error) { return w.cfg.GetFloat(k) }
