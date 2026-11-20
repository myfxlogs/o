// Package gridfit —— 网格评估族影子评估臂（工单#25737 业主钦点族：
// 震荡市收割=方向族互补面，方向族同市弱构实证）。md_klines bar='1D'
// OHLC 确认棒（okx 现货宇宙——inst_id LIKE '%-USDT' AND NOT LIKE
// '%-SWAP'，现货无强平）→网格带回放仿真→fin_opp_eval（grid_spot
// 方向位）→fin_proposal（dedupe 同 funding persist）。影子族——
// execfam 不册 grid_spot：proposed 判词经 T-0330 同闸落
// rejected/exec_family_missing（T-0408/mt5_dir_mom 先例），评估卡
// 全可见执行零径；执行面=后续工单另件。
//
// 仿真判据 SSOT③④（eval.go 落地式）：band=[c0(1-band_pct),c0(1+band_pct)]
// c0=窗首收盘；买线 c0(1-step)^k≥band_lo；逐棒先卖后买；末持仓按末棒
// 收盘 MTM；capital=Σline_notional；net_apr=(realized+unrealized-Σfee)/
// capital·365/lookback；proposed⟺net_apr≥min_net_apr∧last≥
// band_lo·(1-blown_tol)，否则 rejected net_thin/band_blown（thin_bars
// 在 scan 层 skip 不入卡）。
package gridfit

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
	Template = "grid_spot" // execfam 册键同源（未册=影子族闸）
	bar      = "1D"
	seat     = "网格评估"
)

var paramKeys = []string{
	"research.gridfit.scan_interval_min", "research.gridfit.lookback_days",
	"research.gridfit.band_pct", "research.gridfit.step_pct",
	"research.gridfit.line_notional_usd", "research.gridfit.fee_rt",
	"research.gridfit.min_net_apr", "research.gridfit.blown_tol",
}

type Params struct {
	ScanIv       time.Duration
	Lookback     int
	BandPct      float64
	StepPct      float64
	LineNotional float64
	FeeRT        float64
	MinNetAPR    float64
	BlownTol     float64
}

// Runner 网格评估岗（md 池——fin_opp_eval/fin_proposal/fin_opp_template
// 正本+fin_strategy 只读副本+md_klines 桥面，研究域同池惯例）。
type Runner struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	watch *armwatch.Watch
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New(seat),
		rej: rejects.New("research", "grid_missing_params", "grid_stale_bars",
			"grid_thin_bars", "grid_write_err", "gate_kill")}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修同径）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.watch.AttachAlerter(a); return r }

// ScanOnce 单拍壳（工具/实证件消费——scan 本体私有语义不变）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

// Run 在岗（注册+心跳+ticker 拍——klines 无事件脊，日棒粒度 ticker 正解；
// sched-exempt-ok：慢特征评估节拍非轮询拉取）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, seat, "research",
		"md_klines 1D OHLC→网格带回放→网格评估卡+提案（影子族 execfam 未册）", nil); err != nil {
		errc <- fmt.Errorf("%s register: %w", seat, err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, seat, 30*time.Second, errc)
	iv := 4 * time.Hour // 判据未读前兜底节距（拍内 params 重读热启）
	for {
		if err := r.scan(ctx); err != nil {
			log.Printf("[research/gridfit] 拍败: %v", err)
			writefail.Report(ctx, "research/gridfit", "工单#25737", "RESEARCH-TICK/gridfit", "拍败: "+err.Error())
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

// staleBars 末确认棒陈界（research.gridfit.stale_h 懒读缺省 56h——SSOT⑦
// 钦值入册 fin_0215；1D 棒=确认滞~16h+日 cadence 24h→常龄≤40h，56h 顶
// =真断供仍捕陈=不评估 fail-closed，同 dirmom 先例）。
func (r *Runner) staleBars() time.Duration {
	if v, err := r.cfg.GetFloat("research.gridfit.stale_h"); err == nil && v > 0 {
		return time.Duration(v * float64(time.Hour))
	}
	return 56 * time.Hour
}

// params 判据全 app_config 机读（缺/坏=fail-closed 不评估不虚构判据）。
// 结构下界：lookback≥2（锚棒+仿真窗）、band_pct>step_pct>0（买线档
// ≥1——否则 capital=0 除零/空网格）、step_pct≥0.001（档数爆炸防挂）、
// line_notional>0、0<blown_tol<1。
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
	if p.BandPct, err = get(paramKeys[2]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[2], err)
	}
	if p.StepPct, err = get(paramKeys[3]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[3], err)
	}
	if p.LineNotional, err = get(paramKeys[4]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[4], err)
	}
	if p.FeeRT, err = get(paramKeys[5]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[5], err)
	}
	if p.MinNetAPR, err = get(paramKeys[6]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[6], err)
	}
	if p.BlownTol, err = get(paramKeys[7]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[7], err)
	}
	if p.Lookback < 2 || p.BandPct <= p.StepPct || p.StepPct < 0.001 ||
		p.LineNotional <= 0 || p.FeeRT < 0 || p.BlownTol <= 0 || p.BlownTol >= 1 {
		return p, fmt.Errorf("判据下界违例 lookback=%d band=%.4f step=%.4f notional=%.2f fee=%.4f tol=%.4f（结构下界）",
			p.Lookback, p.BandPct, p.StepPct, p.LineNotional, p.FeeRT, p.BlownTol)
	}
	return p, nil
}

// scan 一拍：klines 现货宇宙逐 inst 取 lookback+1 确认棒→网格回放→
// 写批事务落卡（SSOT②宇宙=venue okx spot——LIKE '%-USDT' AND NOT
// LIKE '%-SWAP' 双谓词钉现货域，SWAP 尾缀排除防 perp 混入）。
func (r *Runner) scan(ctx context.Context) error {
	to := 2 * time.Minute
	if v, err := r.cfg.GetFloat("funding.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second // 期界包络同键复用（不另造判据）
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx
	if k, err := r.cfg.GetBool("gate.kill.gridfit"); err == nil && k {
		r.rej.Inc("gate_kill")
		return nil
	}
	r.watch.SetBeat(24 * time.Hour)
	r.watch.Check(ctx)
	p, err := r.params()
	if err != nil {
		r.rej.Inc("grid_missing_params")
		return fmt.Errorf("判据缺位=不评估（fail-closed）: %w", err)
	}
	instRows, err := r.pool.Query(ctx, `SELECT DISTINCT inst_id FROM md_klines
		WHERE venue='okx' AND bar=$1 AND inst_id LIKE '%-USDT' AND inst_id NOT LIKE '%-SWAP'
		ORDER BY inst_id`, bar)
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
		bs, ts, n, err := r.bars(ctx, inst, p.Lookback+1)
		if err != nil {
			return err
		}
		if n < p.Lookback+1 {
			r.rej.Inc("grid_thin_bars")
			continue
		}
		if time.Since(ts) > r.staleBars() {
			r.rej.Inc("grid_stale_bars")
			continue // 陈棒面不评估（快照禁令同 funding 新鲜度语义）
		}
		cards = append(cards, evalGrid(inst, bs, ts, p))
	}
	if err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		for _, c := range cards {
			if err := r.writeEval(ctx, tx, c, p); err != nil {
				r.rej.Inc("grid_write_err")
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	r.watch.OK() // 评估拍完成=产出（恢复清标）
	return nil
}
