// Package gridcfd —— CFD 网格评估影子族臂（工单O-41369：OCI md_klines
// venue='exness' bar='1D' 368 符×202d 史在库零采集欠账——grid_spot
// 谓词只认 OKX 现货形，CFD 宇宙在库无人评）。md_klines 确认棒→网格带
// 回放仿真（eval.go 与 grid_spot 逐字同构）→fin_opp_eval
// （template/direction='grid_cfd'）→fin_proposal（dedupe 同 funding
// persist）。影子族——execfam 不册 grid_cfd：proposed 判词经 T-0330
// 同闸落 rejected/exec_family_missing（评估可见执行零径）；执行面
// =后续工单另件。
//
// 成本模型（生死件——CFD 无显式手续费，成本=买卖价差）：每 inst
// cost_rt=max(research.gridcfd.fee_floor, md_ticks 该 inst 7d 中位
// spread_pct=(ask-bid)/bid)；ticks 全缺=fail-closed 跳过不评（不虚构
// 成本——exotic 价差未知即判死向）；cost_rt 注入仿真=单边 fee_side=
// cost_rt/2（买卖各半价差——ask 买 bid 卖一次穿价≈spread，RT 合计=
// spread 诚实口径，禁 fee_rt 平移虚构）。禁线：grid_spot 谓词零放宽
// （模板分 venue 族律）；对称双腿网格不入本票；xmglobal 不并扩。
package gridcfd

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
	Template = "grid_cfd" // execfam 册键同源（未册=影子族闸）
	bar      = "1D"
)

var paramKeys = []string{
	"research.gridcfd.scan_interval_min", "research.gridcfd.lookback_days",
	"research.gridcfd.band_pct", "research.gridcfd.step_pct",
	"research.gridcfd.line_notional_usd", "research.gridcfd.fee_floor",
	"research.gridcfd.min_net_apr", "research.gridcfd.blown_tol",
}

type Params struct {
	ScanIv       time.Duration
	Lookback     int
	BandPct      float64
	StepPct      float64
	LineNotional float64
	FeeFloor     float64 // cost_rt 下界（CFD 成本地板——exotic 实测 spread 恒超界）
	MinNetAPR    float64
	BlownTol     float64
	FeeRT        float64 // 仿真单边费槽——scan 逐 inst 注 cost_rt/2（基值不读键）
}

// Runner CFD 网格评估岗（md 池——fin_opp_eval/fin_proposal/fin_opp_template
// 正本+fin_strategy 只读副本+md_klines/md_ticks 桥面，研究域同池惯例）。
type Runner struct {
	pool      *pgxpool.Pool
	cfg       *config.Client
	rej       *rejects.Door
	watch     *armwatch.Watch
	venue     string // md_klines/md_ticks venue 域（exness=首 venue；xmglobal 另票）
	template  string // fin_opp_eval.template + execfam 册键
	direction string // fin_opp_eval.direction 词表值（grid 族=template 同名）
	seat      string // roster 座名+watch 标（双臂并存须分席）
}

func NewRunnerVenue(pool *pgxpool.Pool, cfg *config.Client, venue, template, seat string) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New(seat),
		venue: venue, template: template, direction: template, seat: seat,
		rej: rejects.New("research", "gridcfd_missing_params", "gridcfd_stale_bars",
			"gridcfd_thin_bars", "gridcfd_no_ticks", "gridcfd_write_err", "gate_kill")}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修同径）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.watch.AttachAlerter(a); return r }

// ScanOnce 单拍壳（工具/实证件消费——scan 本体私有语义不变）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

// Run 在岗（注册+心跳+ticker 拍——klines 无事件脊，日棒粒度 ticker 正解；
// sched-exempt-ok：慢特征评估节拍非轮询拉取）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, r.seat, "research",
		"md_klines 1D OHLC→网格带回放+价差成本→CFD 网格评估卡+提案（影子族 execfam 未册）", nil); err != nil {
		errc <- fmt.Errorf("%s register: %w", r.seat, err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, r.seat, 30*time.Second, errc)
	iv := 4 * time.Hour // 判据未读前兜底节距（拍内 params 重读热启）
	for {
		if err := r.scan(ctx); err != nil {
			log.Printf("[research/gridcfd] 拍败: %v", err)
			writefail.Report(ctx, "research/gridcfd", "工单#O-41369", "RESEARCH-TICK/gridcfd", "拍败: "+err.Error())
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

// staleBars 末确认棒陈界（research.gridcfd.stale_h 懒读缺省 56h——SSOT⑦
// 同 gridfit 钉值：1D 棒=确认滞~16h+日 cadence 24h→常龄≤40h，56h 顶
// =真断供仍捕陈=不评估 fail-closed）。
func (r *Runner) staleBars() time.Duration {
	if v, err := r.cfg.GetFloat("research.gridcfd.stale_h"); err == nil && v > 0 {
		return time.Duration(v * float64(time.Hour))
	}
	return 56 * time.Hour
}

// params 判据全 app_config 机读（缺/坏=fail-closed 不评估不虚构判据）。
// 结构下界：lookback≥2（锚棒+仿真窗）、band_pct>step_pct>0（买线档
// ≥1——否则 capital=0 除零/空网格）、step_pct≥0.001（档数爆炸防挂）、
// line_notional>0、fee_floor≥0、0<blown_tol<1。
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
	if p.FeeFloor, err = get(paramKeys[5]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[5], err)
	}
	if p.MinNetAPR, err = get(paramKeys[6]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[6], err)
	}
	if p.BlownTol, err = get(paramKeys[7]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[7], err)
	}
	if p.Lookback < 2 || p.BandPct <= p.StepPct || p.StepPct < 0.001 ||
		p.LineNotional <= 0 || p.FeeFloor < 0 || p.BlownTol <= 0 || p.BlownTol >= 1 {
		return p, fmt.Errorf("判据下界违例 lookback=%d band=%.4f step=%.4f notional=%.2f floor=%.4f tol=%.4f（结构下界）",
			p.Lookback, p.BandPct, p.StepPct, p.LineNotional, p.FeeFloor, p.BlownTol)
	}
	return p, nil
}

// scan 一拍：klines venue 宇宙逐 inst 取 lookback+1 确认棒→价差成本
// 实测→网格回放→写批事务落卡（宇宙=venue 全目录——CFD 无 spot/perp
// 尾缀分层，venue 谓词即宇宙界；exness=首 venue，xmglobal 另票）。
func (r *Runner) scan(ctx context.Context) error {
	to := 2 * time.Minute
	if v, err := r.cfg.GetFloat("funding.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second // 期界包络同键复用（不另造判据）
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx
	if k, err := r.cfg.GetBool("gate.kill.gridcfd"); err == nil && k {
		r.rej.Inc("gate_kill")
		return nil
	}
	r.watch.SetBeat(24 * time.Hour)
	r.watch.Check(ctx)
	p, err := r.params()
	if err != nil {
		r.rej.Inc("gridcfd_missing_params")
		return fmt.Errorf("判据缺位=不评估（fail-closed）: %w", err)
	}
	instRows, err := r.pool.Query(ctx, `SELECT DISTINCT inst_id FROM md_klines
		WHERE venue=$1 AND bar=$2 ORDER BY inst_id`, r.venue, bar)
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
	spreads, err := r.spreadMedian(ctx)
	if err != nil {
		return err
	}
	var cards []card
	for _, inst := range insts {
		bs, ts, n, err := r.bars(ctx, inst, p.Lookback+1)
		if err != nil {
			return err
		}
		if n < p.Lookback+1 {
			r.rej.Inc("gridcfd_thin_bars")
			continue
		}
		if time.Since(ts) > r.staleBars() {
			r.rej.Inc("gridcfd_stale_bars")
			continue // 陈棒面不评估（快照禁令同 funding 新鲜度语义）
		}
		sp, ok := spreads[inst]
		if !ok {
			r.rej.Inc("gridcfd_no_ticks")
			continue // ticks 全缺=价差不可证——fail-closed 不评不虚构成本
		}
		costRT := maxF(p.FeeFloor, sp)
		pc := p
		pc.FeeRT = costRT / 2 // 单边费率=cost_rt/2（买卖各半价差——RT 合计=cost_rt）
		c := evalGrid(inst, bs, ts, pc)
		c.spreadPct, c.costRT, c.feeSide = sp, costRT, pc.FeeRT
		cards = append(cards, c)
	}
	if err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		for _, c := range cards {
			if err := r.writeEval(ctx, tx, c, p); err != nil {
				r.rej.Inc("gridcfd_write_err")
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	// O-38640 产出语义：判词卡>0=实质产出 OK；宇宙空=合理零产出
	// Quiet；宇宙在零卡=病灶窗不调——静默计时让位 2×beat 告警。
	if len(cards) > 0 {
		r.watch.OK()
	} else if len(insts) == 0 {
		r.watch.Quiet()
	}
	return nil
}

// maxF float64 大者（stdlib 无内建——params 域小函数同包私有）。
func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
