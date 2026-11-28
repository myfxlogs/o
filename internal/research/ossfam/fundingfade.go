// fundingfade.go —— funding-extreme-fade 评估臂（工单O-41394：卡#578381
// |funding|>p95 fade——资金费极值反向 mean_reversion；criterion 真值=
// fin_0243 research.funding_fade.* app_config 机读，缺/坏=fail-closed
// 不评估不虚构判据）。
//
// 信号：宇宙=md_funding venue='okx' 逐 inst 最新 funding_rate（stale_h
// 新鲜度界内）；|rate| 截面 p95 分位=极值阈（funding_pctile_gate 键——
// 卡载 |funding|>p95 语义）。极值侧 fade：rate>0=多头拥挤付租→fade=空
// （evidence.fade_side 记 'short_perp'）；rate<0→'long_perp'。
//
// 期望净 APR（可证伪预期——校准账消费面）：
//
//	gross_apr = |rate| × periods_per_year（资金费年化=拥挤度代理——
//	  极值资金费≈持仓极端≈预期反转溢价代理，evidence.gross_basis
//	  如实标 proxy 非实测 edge）
//	net_apr   = gross_apr − est_trades_per_year × friction_rt
//	  （卡 ev_formula：30trades×7.2bp≈2.2%/yr 摩擦摊）
//
// verdict=proposed ⟺ 极值 ∧ net_apr≥target_net_annual；非极值 inst
// 也落 rejected/'funding_not_extreme' 行——全宇宙评估覆盖（candidate
// 态零产出探针口径=eval 行存在即覆盖，非只信号行）。影子径：verdict
// 实判但零提案零执行（ossfam.commit 单点）。
package ossfam

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
)

const fundingVenue = "okx"

type fundingParams struct {
	ScanIv        time.Duration
	StaleH        float64
	PctileGate    float64
	FrictionRT    float64
	EstTrades     float64
	TargetNet     float64
	PeriodsYear   float64
	FalsifyWinMin float64
	FalsifyNetMin float64
}

var fundingKeys = []string{
	"research.funding_fade.scan_interval_min", "research.funding_fade.stale_h",
	"research.funding_fade.funding_pctile_gate", "research.funding_fade.friction_rt",
	"research.funding_fade.est_trades_per_year", "research.funding_fade.target_net_annual",
	"research.funding_fade.periods_per_year",
	"research.funding_fade.falsify_winrate_min", "research.funding_fade.falsify_net_annual_min",
}

// FundingRunner 资金费极值反向评估岗（md 池——fin_opp_eval 正本+
// md_funding 桥面，研究域同池惯例）。
type FundingRunner struct{ *arm }

func NewFundingRunner(pool *pgxpool.Pool, cfg *config.Client) *FundingRunner {
	return &FundingRunner{newArm(pool, cfg, rejects.New("research", "funding_fade_missing_params", "funding_fade_write_err", "gate_kill"), "评估臂-funding-fade",
		"资金费极值反向评估臂（funding-extreme-fade candidate 态——md_funding→fin_opp_eval 影子产出零执行零提案，O-41394）",
		"funding-extreme-fade", "perp_fade")}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修同径；具型返回保链）。
func (r *FundingRunner) AttachAlerter(a *notify.Alerter) *FundingRunner {
	r.arm.AttachAlerter(a)
	return r
}

// ScanOnce 单拍壳（工具/实证件消费——tick 本体私有语义不变）。
func (r *FundingRunner) ScanOnce(ctx context.Context) error {
	_, err := r.tick(ctx)
	return err
}

// Run 在岗（臂壳同构——注册+心跳+ticker 拍）。
func (r *FundingRunner) Run(ctx context.Context, errc chan<- error) {
	r.arm.Run(ctx, errc, r.tick)
}

// params 判据全 app_config 机读（缺/坏=fail-closed 不评估不虚构判据）。
// 结构下界：pctile_gate∈(0,1)、friction_rt≥0、est_trades>0、
// periods_per_year>0（年化分母）、target_net>0。
func (r *FundingRunner) params() (fundingParams, error) {
	var p fundingParams
	get := func(k string) (float64, error) { return r.cfg.GetFloat(k) }
	var err error
	var iv float64
	if iv, err = get(fundingKeys[0]); err != nil {
		return p, fmt.Errorf("%s: %w", fundingKeys[0], err)
	}
	p.ScanIv = time.Duration(iv) * time.Minute
	vals := make([]float64, len(fundingKeys)-1)
	for i, k := range fundingKeys[1:] {
		if vals[i], err = get(k); err != nil {
			return p, fmt.Errorf("%s: %w", k, err)
		}
	}
	p.StaleH, p.PctileGate, p.FrictionRT = vals[0], vals[1], vals[2]
	p.EstTrades, p.TargetNet, p.PeriodsYear = vals[3], vals[4], vals[5]
	p.FalsifyWinMin, p.FalsifyNetMin = vals[6], vals[7]
	if p.StaleH <= 0 || p.PctileGate <= 0 || p.PctileGate >= 1 ||
		p.FrictionRT < 0 || p.EstTrades <= 0 || p.PeriodsYear <= 0 || p.TargetNet <= 0 {
		return p, fmt.Errorf("判据下界违例 stale_h=%.1f pctile=%.3f friction=%.5f trades=%.1f ppy=%.1f target=%.3f（结构下界）",
			p.StaleH, p.PctileGate, p.FrictionRT, p.EstTrades, p.PeriodsYear, p.TargetNet)
	}
	return p, nil
}

// tick 单拍：判据→gate→宇宙评估→批落表。返 scan_interval 作下拍距。
func (r *FundingRunner) tick(ctx context.Context) (time.Duration, error) {
	to := 2 * time.Minute
	if v, err := r.cfg.GetFloat("funding.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second // 期界包络同键复用（不另造判据）
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx
	if k, err := r.cfg.GetBool("gate.kill.funding_fade"); err == nil && k {
		r.rej.Inc("gate_kill")
		return 0, nil
	}
	r.watch.SetBeat(24 * time.Hour)
	r.watch.Check(ctx)
	p, err := r.params()
	if err != nil {
		r.rej.Inc("funding_fade_missing_params")
		return 0, fmt.Errorf("判据缺位=不评估（fail-closed）: %w", err)
	}
	rows, err := r.evalUniverse(ctx, p)
	if err != nil {
		r.rej.Inc("funding_fade_write_err")
		return 0, err
	}
	if err := r.commit(ctx, rows); err != nil {
		r.rej.Inc("funding_fade_write_err")
		return 0, err
	}
	// 产出语义：卡>0=实质产出 OK；宇宙空=合理零产出 Quiet（断供真空
	// 由 breed_stall 探针面管——评估臂不替探针裁）。
	if len(rows) > 0 {
		r.watch.OK()
	} else {
		r.watch.Quiet()
	}
	return p.ScanIv, nil
}

// evalUniverse 逐 inst 最新资金费→p95 极值阈→全宇宙评估卡。
func (r *FundingRunner) evalUniverse(ctx context.Context, p fundingParams) ([]evalRow, error) {
	cutoff := time.Now().Add(-time.Duration(p.StaleH * float64(time.Hour)))
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ON (inst_id) inst_id, funding_rate, ts
		FROM md_funding
		WHERE venue=$1 AND ts > $2 AND funding_rate IS NOT NULL
		ORDER BY inst_id, ts DESC`, fundingVenue, cutoff)
	if err != nil {
		return nil, err
	}
	type snap struct {
		rate float64
		ts   time.Time
	}
	var insts []string
	rates := map[string]snap{}
	for rows.Next() {
		var inst string
		var rate float64
		var ts time.Time
		if err := rows.Scan(&inst, &rate, &ts); err != nil {
			rows.Close()
			return nil, err
		}
		insts = append(insts, inst)
		rates[inst] = snap{rate, ts}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(insts) == 0 {
		return nil, nil
	}
	abs := make([]float64, 0, len(insts))
	for _, s := range rates {
		abs = append(abs, math.Abs(s.rate))
	}
	sort.Float64s(abs)
	// p95 分位（线性插值最近秩——funding_pctile_gate=0.95 卡载语义；
	// 宇宙<20 时分位退化近似=最大值邻域，结构下界不另设键）。
	q := quantile(abs, p.PctileGate)
	frictionAnnual := p.EstTrades * p.FrictionRT
	out := make([]evalRow, 0, len(insts))
	for _, inst := range insts {
		s := rates[inst]
		af := math.Abs(s.rate)
		extreme := af > q // 严格大于 p95 阈——等于阈值=分位内非极值（全同值宇宙 q=常数→零极值正确）
		fadeSide := "long_perp"
		if s.rate > 0 {
			fadeSide = "short_perp" // rate>0 多头付租拥挤→fade=空 perp
		}
		gross := af * p.PeriodsYear // 拥挤度代理（gross_basis 如实标）
		net := gross - frictionAnnual
		verdict, reason := "rejected", "funding_not_extreme"
		switch {
		case extreme && net >= p.TargetNet:
			verdict, reason = "proposed", ""
		case extreme:
			reason = "net_thin"
		}
		out = append(out, evalRow{
			inst: inst, verdict: verdict, reason: reason,
			grossAPR: gross, netAPR: net,
			legs: []string{"perp_fade_dyn"},
			evidence: map[string]any{
				"funding_rate": s.rate, "abs_funding": af,
				"pctile_gate": p.PctileGate, "p95_threshold": q,
				"funding_ts": s.ts, "stale_h": p.StaleH,
				"fade_side":           fadeSide,
				"gross_basis":         "funding_annualized_proxy", // 极值资金费=拥挤度代理非实测 edge
				"periods_per_year":    p.PeriodsYear,
				"friction_annual":     frictionAnnual,
				"est_trades_per_year": p.EstTrades, "friction_rt": p.FrictionRT,
				"target_net_annual":   p.TargetNet,
				"falsify_winrate_min": p.FalsifyWinMin, "falsify_net_annual_min": p.FalsifyNetMin,
				"paper_window_d": 90, "reason": reason,
			},
		})
	}
	return out, nil
}

// quantile 最近秩分位（abs 已升序；q∈(0,1)——idx 越界钳位）。
func quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(q * float64(len(sorted)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
