// Package regime —— regime→结构映射器（工单#25750 业主钦定「举一反三·
// 策略版」有界件：市场态检测→人维护映射表→族层提案；无界版业主裁=不建
// ——过拟合假阳性工厂+钱机自改风险对象）。
//
// 职责=族层提案非符层信号（SSOT②）：量全宇宙市场态占比 vs 族覆盖面——
// dominant regime 缺映射族→「族提案」升件（ops_event event='族提案'
// seat='研究席' esc_class='商业产出' esc_scope='regime_map_gap'）；有族
// 但判词稀薄→稀薄提案（同事件面 note 分型）。影子产出零执行——提案
// 升件喊人，不动钱不动策略册；映射表外族永不提案（fail-closed）。
//
// 分类（SSOT③，eval.go 落地式）：每 (venue,inst_id) 取 lookback+1 根
// 1D 确认棒——|z|≥z_thr→trend；vr=Var(r2)/(2·Var(r1))<vr_thr∧含带率
// ≥cont_thr→chop；sigma_d<quiet_thr→quiet；sigma_d>vol_thr→volatile；
// 余=mixed。占比=分类台数/足窗台数；态账行（event='态账'）每拍落
// 占比证据（ops_event 驻 fin 域——AttachEscPool 挂接同 ctx 先例）。
package regime

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
	"o/internal/platform/writefail"
	"o/internal/research/armwatch"
)

const (
	bar  = "1D"
	seat = "态评估"
)

var paramKeys = []string{
	"research.regime.scan_interval_min", "research.regime.lookback_days",
	"research.regime.z_thr", "research.regime.vr_thr",
	"research.regime.cont_thr", "research.regime.quiet_thr",
	"research.regime.vol_thr", "research.regime.thin_days",
}

// Params 判据全 app_config 机读（映射表 research.regime.map 走
// regimeMap() JSON 径不入本结构——标量/对象分径同 config 契约）。
type Params struct {
	ScanIv   time.Duration
	Lookback int
	ZThr     float64
	VrThr    float64
	ContThr  float64
	QuietThr float64
	VolThr   float64
	ThinDays int
}

// Runner 态评估岗（md 池——md_klines/fin_opp_template/fin_opp_eval 桥面
// 读径；ops_event 写径=escPool fin 池同 ctx 先例——研究节点 md 池无
// ops_event 面）。
type Runner struct {
	pool    *pgxpool.Pool
	cfg     *config.Client
	rej     *rejects.Door
	watch   *armwatch.Watch
	escPool *pgxpool.Pool // ops_event 态账/族提案写径（fin 域）
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New(seat),
		rej: rejects.New("research", "regime_missing_params", "regime_stale_bars",
			"regime_thin_bars", "regime_map_err", "regime_write_err", "gate_kill")}
}

// AttachEscPool ops_event 写径挂接（fin 池——未挂=守位不写账不提案，
// 分类读径照常；ctx AttachEscPool 同形先例）。
func (r *Runner) AttachEscPool(p *pgxpool.Pool) *Runner { r.escPool = p; return r }

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修同径）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.watch.AttachAlerter(a); return r }

// ScanOnce 单拍壳（工具/实证件消费——scan 本体私有语义不变）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

// Run 在岗（注册+心跳+ticker 拍——klines 无事件脊，日棒粒度 ticker 正解；
// sched-exempt-ok：慢特征评估节拍非轮询拉取）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, seat, "research",
		"md_klines 1D→regime五态分类→态账+族覆盖缺口→族提案（有界映射表零执行）", nil); err != nil {
		errc <- fmt.Errorf("%s register: %w", seat, err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, seat, 30*time.Second, errc)
	iv := 6 * time.Hour // 判据未读前兜底节距（拍内 params 重读热启）
	for {
		if err := r.scan(ctx); err != nil {
			log.Printf("[research/regime] 拍败: %v", err)
			writefail.Report(ctx, "research/regime", "工单#25750", "RESEARCH-TICK/regime", "拍败: "+err.Error())
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

// staleBars 末确认棒陈界（research.regime.stale_h 懒读缺省 56h——SSOT⑦
// 未播种同 dirtrend 先例；1D 棒=确认滞~16h+日 cadence 24h→常龄≤40h，
// 56h 顶=真断供仍捕陈=不评估 fail-closed）。
func (r *Runner) staleBars() time.Duration {
	if v, err := r.cfg.GetFloat("research.regime.stale_h"); err == nil && v > 0 {
		return time.Duration(v * float64(time.Hour))
	}
	return 56 * time.Hour
}

// params 判据全 app_config 机读（缺/坏=fail-closed 不评估不虚构判据）。
// 结构下界：lookback≥2（锚棒+分类窗）、阈全正、cont_thr∈(0,1]、
// thin_days≥1（覆盖读窗下界）。
func (r *Runner) params() (Params, error) {
	var p Params
	get := func(k string) (float64, error) { return r.cfg.GetFloat(k) }
	var err error
	var iv float64
	if iv, err = get(paramKeys[0]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[0], err)
	}
	p.ScanIv = time.Duration(iv) * time.Minute
	var lb, td float64
	if lb, err = get(paramKeys[1]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[1], err)
	}
	p.Lookback = int(lb)
	if p.ZThr, err = get(paramKeys[2]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[2], err)
	}
	if p.VrThr, err = get(paramKeys[3]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[3], err)
	}
	if p.ContThr, err = get(paramKeys[4]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[4], err)
	}
	if p.QuietThr, err = get(paramKeys[5]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[5], err)
	}
	if p.VolThr, err = get(paramKeys[6]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[6], err)
	}
	if td, err = get(paramKeys[7]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[7], err)
	}
	p.ThinDays = int(td)
	if p.Lookback < 2 || p.ZThr <= 0 || p.VrThr <= 0 ||
		p.ContThr <= 0 || p.ContThr > 1 || p.QuietThr <= 0 ||
		p.VolThr <= 0 || p.ThinDays < 1 {
		return p, fmt.Errorf("判据下界违例 lookback=%d z=%.3f vr=%.3f cont=%.3f quiet=%.4f vol=%.4f thin=%d（结构下界）",
			p.Lookback, p.ZThr, p.VrThr, p.ContThr, p.QuietThr, p.VolThr, p.ThinDays)
	}
	return p, nil
}

// scan 一拍：全宇宙（venue,inst_id）逐对取 lookback+1 确认棒→五态分类
// →态账行落占比证据→dominant regime 映射缺口分析→族提案（销抑+票抑
// 双契）。棒薄/陈棒=unevaluable 记账不评估（SSOT⑩）。
func (r *Runner) scan(ctx context.Context) error {
	to := 2 * time.Minute
	if v, err := r.cfg.GetFloat("funding.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second // 期界包络同键复用（不另造判据）
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx
	if k, err := r.cfg.GetBool("gate.kill.regime"); err == nil && k {
		r.rej.Inc("gate_kill")
		return nil
	}
	r.watch.SetBeat(24 * time.Hour)
	r.watch.Check(ctx)
	p, err := r.params()
	if err != nil {
		r.rej.Inc("regime_missing_params")
		return fmt.Errorf("判据缺位=不评估（fail-closed）: %w", err)
	}
	instRows, err := r.pool.Query(ctx, `SELECT DISTINCT venue,inst_id FROM md_klines
		WHERE bar=$1 ORDER BY venue,inst_id`, bar)
	if err != nil {
		return err
	}
	type pair struct{ venue, inst string }
	var pairs []pair
	for instRows.Next() {
		var pr pair
		if err := instRows.Scan(&pr.venue, &pr.inst); err != nil {
			instRows.Close()
			return err
		}
		pairs = append(pairs, pr)
	}
	instRows.Close()
	counts := map[string]int{}
	var classified, thinN, staleN int
	for _, pr := range pairs {
		bs, ts, n, err := r.bars(ctx, pr.venue, pr.inst, p.Lookback+1)
		if err != nil {
			return err
		}
		if n < p.Lookback+1 {
			thinN++
			continue
		}
		if time.Since(ts) > r.staleBars() {
			staleN++
			continue // 陈棒面不评估（快照禁令同 funding 新鲜度语义）
		}
		label, _ := classify(bs, p)
		counts[label]++
		classified++
	}
	if thinN+staleN > 0 {
		r.rej.Inc("regime_thin_bars")
	}
	dominant := dominantRegime(counts)
	if r.escPool == nil {
		r.watch.OK()
		return nil // 守位：无写径池=不写账不提案（分类读径白做如实）
	}
	if err := r.writeLedger(ctx, counts, classified, thinN, staleN, dominant, p); err != nil {
		r.rej.Inc("regime_write_err")
		return err
	}
	m, err := r.regimeMap()
	if err != nil {
		r.rej.Inc("regime_map_err")
		log.Printf("[research/regime] 映射表读败=不提案（态账照落）: %v", err)
	} else if err := r.maybePropose(ctx, m, dominant, counts, classified, p); err != nil {
		r.rej.Inc("regime_write_err")
		return err
	}
	r.watch.OK() // 评估拍完成=产出（恢复清标）
	return nil
}
