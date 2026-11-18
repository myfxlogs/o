// Package dirtrend —— 方向趋势族评估臂（工单#11481：假想卡#172286
// okx/perp-trend-daily parked 解锁径骨架件——Donchian 突破+winrate-CI+
// vol-est 评估数学=卡薄口径落地式，全参数 app_config 机读禁虚构常数）。
// md_klines bar='1D' 确认棒→信号+判词→fin_opp_eval（perp-trend-daily;
// long_perp/short_perp 单腿方向位）→fin_proposal（dedupe 同 funding
// persist）。paper-only 族——真单径未建（execfam.PaperOnly 册载，
// excell 免演登记+Drillable=false 拦 probe→live）。
//
// 信号：cl=末 W=lookback+replay+1 根确认棒收盘（正序）；今棒通道=
// max/min(cl[W-1-lookback..W-2])（末棒不入通道）；cl[W-1]>hi→long_perp，
// cl[W-1]<lo→short_perp，否则 no_breakout。
//
// winrate-CI：回放 i∈[lookback,W-2]，i 棒收盘破前 lookback 通道记样本——
// 多破 obs=r_{i+1}，空破 obs=−r_{i+1}（多空同构合并）；N=len(obs)，
// p̂=win/N，LB=Wilson(p̂,N,0.95) 下界（保守单调）。
//
// vol-est：sigma_d=stdev(r 全窗)；sigma_ann=sigma_d·√365 入 evidence。
// 净 APR：e_d=mean(obs)；net_apr=(e_d−friction_rt/hold_days)·365；
// gross_apr=e_d·365。
//
// verdict=proposed ⟺ 今棒信号 ∧ N≥min_signal_samples ∧ LB≥winrate_lb ∧
// net_apr≥min_net_apr ∧ sigma_d>0 ∧ 末棒≤stale_h；reject 真因=
// no_breakout|thin_samples|lb_weak|net_thin|sigma_zero；proposed 时反
// 方向行落 rejected+counter_signal（同 dirmom 翻面证伪）；no_breakout
// 无信号向可挂——双向各落一行 rejected 如实记（双向信念同殁）。
package dirtrend

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
	"o/internal/platform/writefail"
	"o/internal/research/armwatch"
)

const (
	Template = "perp-trend-daily" // execfam 册键同源
	bar      = "1D"
)

var paramKeys = []string{
	"research.dirtrend.scan_interval_min", "research.dirtrend.lookback_days",
	"research.dirtrend.replay_days", "research.dirtrend.winrate_lb",
	"research.dirtrend.min_signal_samples", "research.dirtrend.hold_days",
	"research.dirtrend.friction_rt", "research.dirtrend.min_net_apr",
}

type Params struct {
	ScanIv     time.Duration
	Lookback   int
	Replay     int
	WinrateLB  float64
	MinSamples int
	HoldDays   float64
	FrictionRT float64
	MinNetAPR  float64
}

// Runner 趋势评估岗（md 池——fin_opp_eval/fin_proposal/fin_opp_template
// 正本+fin_strategy 只读副本+md_klines 桥面，研究域同池惯例）。
type Runner struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	watch *armwatch.Watch
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New("趋势评估"),
		rej: rejects.New("research", "trend_missing_params", "trend_stale_bars",
			"trend_thin_bars", "trend_write_err", "gate_kill")}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修同径）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.watch.AttachAlerter(a); return r }

// ScanOnce 单拍壳（工具/实证件消费——scan 本体私有语义不变）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

// Run 在岗（注册+心跳+ticker 拍——klines 无事件脊，日棒粒度 ticker 正解；
// sched-exempt-ok：慢特征评估节拍非轮询拉取）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "趋势评估", "research",
		"md_klines 1D→Donchian 突破+winrate-CI→趋势评估卡+提案（paper-only 族）", nil); err != nil {
		errc <- fmt.Errorf("趋势评估 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "趋势评估", 30*time.Second, errc)
	iv := 4 * time.Hour // 判据未读前兜底节距（拍内 params 重读热启）
	for {
		if err := r.scan(ctx); err != nil {
			log.Printf("[research/dirtrend] 拍败: %v", err)
			writefail.Report(ctx, "research/dirtrend", "工单#22873", "RESEARCH-TICK/dirtrend", "拍败: "+err.Error())
		}
		if p, err := r.params(); err == nil && p.ScanIv > 0 {
			iv = p.ScanIv
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(iv): // sched-exempt-ok：评估节拍非轮询拉取
		}
	}
}

// staleBars 末确认棒陈界（research.dirtrend.stale_h 懒读缺省 56h——1D 棒=
// 收线确认滞~16h+日 cadence 24h→常龄≤40h；56h 顶=真断供仍捕陈=不评估
// fail-closed，同 dirmom 先例）。
func (r *Runner) staleBars() time.Duration {
	if v, err := r.cfg.GetFloat("research.dirtrend.stale_h"); err == nil && v > 0 {
		return time.Duration(v * float64(time.Hour))
	}
	return 56 * time.Hour
}

// params 判据全 app_config 机读（缺/坏=fail-closed 不评估不虚构判据）。
func (r *Runner) params() (Params, error) {
	var p Params
	get := func(k string) (float64, error) { return r.cfg.GetFloat(k) }
	var err error
	var iv float64
	if iv, err = get(paramKeys[0]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[0], err)
	}
	p.ScanIv = time.Duration(iv) * time.Minute
	var lb, rp, ms float64
	if lb, err = get(paramKeys[1]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[1], err)
	}
	p.Lookback = int(lb)
	if rp, err = get(paramKeys[2]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[2], err)
	}
	p.Replay = int(rp)
	if p.WinrateLB, err = get(paramKeys[3]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[3], err)
	}
	if ms, err = get(paramKeys[4]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[4], err)
	}
	p.MinSamples = int(ms)
	if p.HoldDays, err = get(paramKeys[5]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[5], err)
	}
	if p.FrictionRT, err = get(paramKeys[6]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[6], err)
	}
	if p.MinNetAPR, err = get(paramKeys[7]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[7], err)
	}
	if p.Lookback < 2 || p.Replay < 1 || p.MinSamples < 1 {
		return p, fmt.Errorf("判据下界违例 lookback=%d replay=%d min_samples=%d（通道/回放/样本窗结构下界）",
			p.Lookback, p.Replay, p.MinSamples)
	}
	return p, nil
}

// scan 一拍：klines 宇宙逐 inst 取 lookback+replay+1 确认棒→信号+判词→
// 写批事务落卡。
func (r *Runner) scan(ctx context.Context) error {
	to := 2 * time.Minute
	if v, err := r.cfg.GetFloat("funding.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second // 期界包络同键复用（不另造判据——dirmom 同径）
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx
	if k, err := r.cfg.GetBool("gate.kill.dirtrend"); err == nil && k {
		r.rej.Inc("gate_kill")
		return nil
	}
	r.watch.SetBeat(24 * time.Hour)
	r.watch.Check(ctx)
	p, err := r.params()
	if err != nil {
		r.rej.Inc("trend_missing_params")
		return fmt.Errorf("判据缺位=不评估（fail-closed）: %w", err)
	}
	need := p.Lookback + p.Replay + 1
	instRows, err := r.pool.Query(ctx, `SELECT DISTINCT inst_id FROM md_klines
		WHERE venue='okx' AND bar=$1 AND inst_id LIKE '%-SWAP' ORDER BY inst_id`, bar)
	if err != nil {
		return err
	}
	var insts []string
	for instRows.Next() {
		var s string
		if err := instRows.Scan(&s); err != nil {
			instRows.Close()
			return err
		}
		insts = append(insts, s)
	}
	instRows.Close()
	var cards []card
	for _, inst := range insts {
		cl, ts, n, err := r.closes(ctx, inst, need)
		if err != nil {
			return err
		}
		if n < need {
			r.rej.Inc("trend_thin_bars")
			continue
		}
		if time.Since(ts) > r.staleBars() {
			r.rej.Inc("trend_stale_bars")
			continue // 陈棒面不评估（快照禁令同 funding 新鲜度语义）
		}
		cards = append(cards, evalCard(inst, cl, ts, p))
	}
	if err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		for _, c := range cards {
			if c.sig == "no_breakout" {
				// 无信号向可挂——双向各落 rejected 如实记（双向信念同殁，
				// 翻面出场判据 dirEntryGate/dirpaper 同读本向行）。
				for _, d := range []string{"long_perp", "short_perp"} {
					if err := r.writeEval(ctx, tx, c.inst, d, "rejected", "no_breakout", c, p); err != nil {
						r.rej.Inc("trend_write_err")
						return err
					}
				}
				continue
			}
			if err := r.writeEval(ctx, tx, c.inst, c.sig, c.verdict, c.reason, c, p); err != nil {
				r.rej.Inc("trend_write_err")
				return err
			}
			if c.verdict == "proposed" {
				if err := r.writeEval(ctx, tx, c.inst, c.opp, "rejected", "counter_signal", c, p); err != nil {
					r.rej.Inc("trend_write_err")
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	r.watch.OK() // 评估拍完成=产出（恢复清标）
	return nil
}
