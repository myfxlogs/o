// Package trendmech —— 机械趋势假想卡②评估臂（工单O-32604：BTC-USDT-SWAP
// 日级 Donchian 单范式手术刀——收盘破20日高开多/破20日低开空，10日反向
// 突破平仓；信号 t 收盘→t+1 开盘成交，taker 费+funding 日摊，净值曲线落
// fin_opp_eval.evidence=paper 评估径产出）。paper-only 影子产出——不建
// proposal/strategy 径：假想卡留 proposed 归方向族死刑审查裁（spec 钉），
// 本臂只产可后验打分的评估卡（verdict=net_apr>0→proposed 否则 rejected
// ——净期望≤费调后零=判死输入件，>0=升审队输入件）。
// 全窗重放确定性径：每拍全量确认棒重算（无增量状态漂移）；evalgate 指纹
// 闸去重（判词/净额变迁才产新版史）。
package trendmech

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
	"o/internal/platform/writefail"
	"o/internal/research/armwatch"
	"o/internal/research/evalgate"
)

const (
	beatIv    = 4 * time.Hour // 日棒粒度评估拍（新棒确认才有新信号——慢特征节拍）
	lookback  = 200           // 确认棒回看上限（≈6.5 月——通道+样本窗余量）
	minBars   = 35            // 评估下界：20 通道预热+≥30d paper 窗语义下界
	staleHint = 56 * time.Hour
	notional  = 10000.0 // 单腿 1x 名义 USDT（spec：perp 单腿 1x notional 上限）
)

// Runner 评估岗（md 池——md_klines/md_funding 桥面+fin_opp_eval 正本，
// 研究域同池惯例）。
type Runner struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	watch *armwatch.Watch
	rej   *rejects.Door
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New("趋势机械评估"),
		rej: rejects.New("research", "trendmech_thin_bars", "trendmech_stale_bars",
			"trendmech_write_err", "trendmech_feesrc", "gate_kill")}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修同径）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.watch.AttachAlerter(a); return r }

// ScanOnce 单拍壳（工具/实证件消费——scan 本体私有语义不变）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

// Run 在岗（注册+心跳+ticker 拍——klines 无事件脊，日棒粒度 ticker 正解；
// sched-exempt-ok：慢特征评估节拍非轮询拉取）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "趋势机械评估", "research",
		"md_klines 1D BTC-USDT-SWAP→Donchian20/10 机械趋势 paper 模拟→fin_opp_eval（影子产出零执行）", nil); err != nil {
		errc <- fmt.Errorf("趋势机械评估 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "趋势机械评估", 30*time.Second, errc)
	for {
		if err := r.scan(ctx); err != nil {
			log.Printf("[research/trendmech] 拍败: %v", err)
			writefail.Report(ctx, "research/trendmech", "工单#32604", "RESEARCH-TICK/trendmech", "拍败: "+err.Error())
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(beatIv): // sched-exempt-ok：评估节拍非轮询拉取
		}
	}
}

// feeBps taker 费率——复用 funding 臂已播种的 venue taker 键
// （research.funding.taker_fee_bps=5.0 OKX perp 实测档；读败/缺=缺省档
// 5.0 不 fail-closed——费率是成本输入非判据，判据钉死在 sim 常量）。
func (r *Runner) feeBps() float64 {
	if v, err := r.cfg.GetFloat("research.funding.taker_fee_bps"); err == nil && v > 0 {
		return v
	}
	r.rej.Inc("trendmech_feesrc")
	return 5.0
}

// scan 一拍：确认棒+funding 结算行→全窗重放→eval 卡落表（指纹闸去重）。
func (r *Runner) scan(ctx context.Context) error {
	to := 2 * time.Minute
	if v, err := r.cfg.GetFloat("funding.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx
	if k, err := r.cfg.GetBool("gate.kill.trendmech"); err == nil && k {
		r.rej.Inc("gate_kill")
		return nil
	}
	r.watch.SetBeat(24 * time.Hour)
	r.watch.Check(ctx)

	bars, err := r.bars(ctx)
	if err != nil {
		return err
	}
	if len(bars) < minBars {
		r.rej.Inc("trendmech_thin_bars")
		return nil
	}
	if time.Since(bars[len(bars)-1].Ts) > staleHint {
		r.rej.Inc("trendmech_stale_bars")
		return nil // 陈棒面不评估（快照禁令同 funding 新鲜度语义）
	}
	funds, err := r.funds(ctx, bars[0].Ts)
	if err != nil {
		return err
	}
	res := Simulate(bars, funds, r.feeBps(), notional)
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		return r.writeEval(ctx, tx, res, len(bars), bars[0].Ts, bars[len(bars)-1].Ts)
	})
}

// bars 末 lookback 根 1D 确认棒 OHLC（新→旧取回倒序——不足实得）。
func (r *Runner) bars(ctx context.Context) ([]Bar, error) {
	rows, err := r.pool.Query(ctx, `SELECT ts,o,h,l,c FROM md_klines
		WHERE venue='okx' AND inst_id=$1 AND bar=$2 AND confirm
		ORDER BY ts DESC LIMIT $3`, Inst, bar, lookback)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Bar
	for rows.Next() {
		var b Bar
		if err := rows.Scan(&b.Ts, &b.O, &b.H, &b.L, &b.C); err != nil {
			return nil, err
		}
		out = append([]Bar{b}, out...) // 倒序→正序
	}
	return out, rows.Err()
}

// funds 窗内 funding 结算行（ts≥窗首——持仓期付息原料）。
func (r *Runner) funds(ctx context.Context, since time.Time) ([]Fund, error) {
	rows, err := r.pool.Query(ctx, `SELECT ts,funding_rate FROM md_funding
		WHERE venue='okx' AND inst_id=$1 AND ts>=$2 ORDER BY ts`, Inst, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Fund
	for rows.Next() {
		var f Fund
		if err := rows.Scan(&f.Ts, &f.Rate); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// writeEval 评估卡落表（evalgate 指纹闸——verdict/net_apr 未变不产新版）。
// verdict=net_apr>0→proposed（升审队输入件）否则 rejected+net_nonpositive
// （判死输入件）——证伪判据 spec 钉：净期望≤费调后零→判死。
func (r *Runner) writeEval(ctx context.Context, tx pgx.Tx, res Result,
	nBars int, firstTs, lastTs time.Time) error {
	verdict, reason := "rejected", "net_nonpositive"
	if res.NetAPR > 0 {
		verdict, reason = "proposed", ""
	}
	nav := make([]float64, len(res.Equity))
	for i, e := range res.Equity {
		nav[i] = mathRound(e/notional, 6)
	}
	gnav := make([]float64, len(res.GrossEq))
	for i, e := range res.GrossEq {
		gnav[i] = mathRound(e/notional, 6)
	}
	days := make([]string, len(res.Days))
	for i, d := range res.Days {
		days[i] = d.Format("2006-01-02")
	}
	ev, _ := json.Marshal(map[string]any{
		"rule":    "donchian20_entry/10_reverse_exit_tclose_signal_t+1_open_fill",
		"entry_n": EntryN, "exit_n": ExitN, "bar": bar,
		"notional": notional, "fee_bps": r.feeBps(),
		"window": map[string]any{"first": firstTs, "last": lastTs, "bars": nBars},
		"fills":  len(res.Fills), "trades": len(res.Trades),
		"wins":    countWins(res.Trades),
		"net_ret": res.NetRet, "gross_ret": res.GrossRet,
		"net_apr": res.NetAPR, "gross_apr": res.GrossAPR,
		"final_eq":  res.FinalEq,
		"nav_curve": nav, "gross_nav": gnav, "nav_days": days,
		"last_fills": tailFills(res.Fills, 5), "reason": reason,
		"falsify": "paper窗≥30d∨confGate判据先到——net_apr≤0判死引方向死刑审查并档;>0升confGate排队(仍无live权)",
	})
	dir := "perp_trend"
	_, same, err := evalgate.SameFingerprint(ctx, tx, Inst, Template, dir, verdict, res.NetAPR)
	if err != nil {
		return err
	}
	if same {
		return nil
	}
	legs, _ := json.Marshal([]string{"perp_long", "perp_short"})
	_, err = tx.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		Inst, Template, dir, res.GrossAPR, res.NetAPR, verdict, legs, ev)
	if err == nil {
		log.Printf("[research/trendmech] 评估卡 %s verdict=%s net_apr=%.4f fills=%d trades=%d",
			Inst, verdict, res.NetAPR, len(res.Fills), len(res.Trades))
	}
	return err
}

func mathRound(v float64, n int) float64 {
	p := math.Pow10(n)
	return math.Round(v*p) / p
}

func countWins(ts []Trade) int {
	n := 0
	for _, t := range ts {
		if t.Pnl > 0 {
			n++
		}
	}
	return n
}

func tailFills(fs []Fill, n int) []Fill {
	if len(fs) > n {
		return fs[len(fs)-n:]
	}
	return fs
}
