// Package dirmom —— 方向族评估臂（棒1 trend_follow 假想卡采纳件③：
// adopted→模板注册+direction 枚举+评估臂 显式批准径的臂件）。
// md_klines bar='1D' 确认棒→动量 z 分→fin_opp_eval（dir_perp_mom;
// long_perp/short_perp 单腿方向位）→fin_proposal（dedupe 同 funding
// persist）。paper-only 族——真单径未建（execfam.PaperOnly 册载，
// excell 免演登记+Drillable=false 拦 probe→live）。
//
// 信号：r_i=ln(c_i/c_{i-1}) 日对数收益；sigma_d=近 lookback 窗 r 样本
// 标准差；mom=ln(c_last/c_{last-lookback})；z=mom/(sigma_d·√lookback)。
// |z|≥z_entry=信号成立（z>0=long_perp；z<0=short_perp）；信号侧之外
// 反方向行落 rejected+reason=counter_signal（翻面主动证伪——反侧陈
// proposed 行留尸=毕业闸 dirEntryGate 误导源）。
//
// 期望净 APR（可证伪预期——校准账 pred vs real 消费面）：
//
//	exp_ret_h = |z|·sigma_d·√hold_days·persist（持有期动量延续期望）
//	gross_apr = exp_ret_h·365/hold_days
//	net_apr   = gross_apr − friction_rt·365/hold_days（往返摩擦年化摊）
//
// verdict=proposed ⟺ |z|≥z_entry ∧ net_apr≥min_net_apr。
// funding 正负租不进评估——纸面账 paperAccounting 对 perp 方向行照记
// funding 腿（动量 edge 与租流分解诚实不混账）。
package dirmom

import (
	"context"
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
	"o/internal/research/armwatch"
)

const (
	Template    = "dir_perp_mom" // execfam 册键同源
	bar         = "1D"
	staleBars   = 36 * time.Hour // 末确认棒陈界（1D 棒日收线+扫拍余量——陈=不评估 fail-closed）
	minDataBars = 12             // 棒数下界（sigma 稳定性——不足=unevaluable 不虚构）
)

var paramKeys = []string{
	"research.dirmom.scan_interval_min", "research.dirmom.lookback_days",
	"research.dirmom.z_entry", "research.dirmom.hold_days",
	"research.dirmom.persist", "research.dirmom.friction_rt",
	"research.dirmom.min_net_apr",
}

type Params struct {
	ScanIv     time.Duration
	Lookback   int
	ZEntry     float64
	HoldDays   float64
	Persist    float64
	FrictionRT float64
	MinNetAPR  float64
}

// card 一 inst 一拍评估中间件（信号+判词+证据载荷）。
type card struct {
	inst, dir, opp, verdict, reason string
	z, sigma, mom, netAPR           float64
	bars                            int
	lastTs                          time.Time
}

// Runner 方向评估岗（md 池——fin_opp_eval/fin_proposal/fin_opp_template
// 正本+fin_strategy 只读副本+md_klines 桥面，研究域同池惯例）。
type Runner struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	watch *armwatch.Watch
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New("方向评估"),
		rej: rejects.New("research", "dir_missing_params", "dir_stale_bars",
			"dir_thin_bars", "dir_write_err", "gate_kill")}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.watch.AttachAlerter(a); return r }

// ScanOnce 单拍壳（工具/实证件消费——scan 本体私有语义不变）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

// Run 在岗（注册+心跳+ticker 拍——klines 无事件脊，日棒粒度 ticker 正解；
// sched-exempt-ok：慢特征评估节拍非轮询拉取）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "方向评估", "research",
		"md_klines 1D→动量 z 分→方向评估卡+提案（paper-only 族）", nil); err != nil {
		errc <- fmt.Errorf("方向评估 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "方向评估", 30*time.Second, errc)
	iv := 4 * time.Hour // 判据未读前兜底节距（拍内 params 重读热启）
	for {
		if err := r.scan(ctx); err != nil {
			log.Printf("[research/dirmom] 拍败: %v", err)
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
	var lb float64
	if lb, err = get(paramKeys[1]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[1], err)
	}
	p.Lookback = int(lb)
	if p.ZEntry, err = get(paramKeys[2]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[2], err)
	}
	if p.HoldDays, err = get(paramKeys[3]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[3], err)
	}
	if p.Persist, err = get(paramKeys[4]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[4], err)
	}
	if p.FrictionRT, err = get(paramKeys[5]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[5], err)
	}
	if p.MinNetAPR, err = get(paramKeys[6]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[6], err)
	}
	if p.Lookback < minDataBars-1 {
		return p, fmt.Errorf("lookback=%d<%d（sigma 窗下界）", p.Lookback, minDataBars-1)
	}
	return p, nil
}

// scan 一拍：klines 宇宙逐 inst 取 lookback+1 确认棒→信号→写批事务落卡。
func (r *Runner) scan(ctx context.Context) error {
	to := 2 * time.Minute
	if v, err := r.cfg.GetFloat("funding.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second // 期界包络同键复用（不另造判据）
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx
	if k, err := r.cfg.GetBool("gate.kill.dirmom"); err == nil && k {
		r.rej.Inc("gate_kill")
		return nil
	}
	r.watch.SetBeat(24 * time.Hour)
	r.watch.Check(ctx)
	p, err := r.params()
	if err != nil {
		r.rej.Inc("dir_missing_params")
		return fmt.Errorf("判据缺位=不评估（fail-closed）: %w", err)
	}
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
		cl, ts, n, err := r.closes(ctx, inst, p.Lookback+1)
		if err != nil {
			return err
		}
		if n < p.Lookback+1 {
			r.rej.Inc("dir_thin_bars")
			continue
		}
		if time.Since(ts) > staleBars {
			r.rej.Inc("dir_stale_bars")
			continue // 陈棒面不评估（快照禁令同 funding 新鲜度语义）
		}
		rets := make([]float64, 0, n-1)
		for i := 1; i < n; i++ {
			rets = append(rets, math.Log(cl[i]/cl[i-1]))
		}
		var mean float64
		for _, x := range rets {
			mean += x
		}
		mean /= float64(len(rets))
		var sd float64
		for _, x := range rets {
			sd += (x - mean) * (x - mean)
		}
		sigma := math.Sqrt(sd / float64(len(rets)-1))
		if sigma <= 0 {
			continue // 零波动=不可评（除零禁——样本病态如实跳过）
		}
		mom := math.Log(cl[n-1] / cl[0])
		z := mom / (sigma * math.Sqrt(float64(p.Lookback)))
		c := card{inst: inst, z: z, sigma: sigma, mom: mom, bars: n, lastTs: ts}
		if z > 0 {
			c.dir, c.opp = "long_perp", "short_perp"
		} else {
			c.dir, c.opp = "short_perp", "long_perp"
		}
		expRetH := math.Abs(z) * sigma * math.Sqrt(p.HoldDays) * p.Persist
		grossAPR := expRetH * 365 / p.HoldDays
		c.netAPR = grossAPR - p.FrictionRT*365/p.HoldDays
		switch {
		case math.Abs(z) < p.ZEntry:
			c.verdict, c.reason = "rejected", "z_weak"
		case c.netAPR < p.MinNetAPR:
			c.verdict, c.reason = "rejected", "net_thin"
		default:
			c.verdict = "proposed"
		}
		cards = append(cards, c)
	}
	if err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		for _, c := range cards {
			if err := r.writeEval(ctx, tx, c.inst, c.dir, c.verdict, c.reason, c, p); err != nil {
				r.rej.Inc("dir_write_err")
				return err
			}
			if c.verdict == "proposed" {
				if err := r.writeEval(ctx, tx, c.inst, c.opp, "rejected", "counter_signal", c, p); err != nil {
					r.rej.Inc("dir_write_err")
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
