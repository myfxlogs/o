// xsecmom.go —— xsec-perp-mom-basket 评估臂（工单O-41394：卡#578380
// 截面动量篮——md_klines 宇宙相对强弱 mean_reversion 族 xsec_mom；
// criterion 真值=fin_0243 research.xsec_mom.* app_config 机读）。
//
// 信号：宇宙=md_klines venue='okx' bar='1D' confirm perp（inst_id
// '-SWAP' 尾缀）逐 inst lookback_days+1 确认棒收盘——mom_i=
// ln(c_last/c_first)（L 日对数动量）；截面降序排名 top/bottom
// basket_k 各成腿=多篮/空篮（卡 mut 载 k3）。宇宙<2k=篮不可成
// →rejected/'thin_universe'（仍落行——评估覆盖非只信号行）。
//
// 期望净 APR（卡 ev_formula 可算口径）：
//
//	spread_l_days = mean(mom_top_k) − mean(mom_bottom_k)；
//	gross_apr = spread × 365/lookback_days（gross_basis 如实标 proxy）；
//	friction_annual = rebal_per_year × friction_rt × 2×basket_k
//	  （结构腿数全换手=保守上界，卡面 4legs 约数注记 evidence）；
//	net_apr = gross − friction
//
// verdict=proposed ⟺ 宇宙≥2k ∧ net_apr≥target_net_annual；falsify
// 键（sharpe_lb/gross_min/paper_window_d）卡载=paper 窗判据非本臂
// 可算面——evidence 回声供复审不混判词。影子径零提案零执行；
// 单行篮级卡 inst_id='XSEC-MOM-BASKET'。
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

const xsecBasketInst = "XSEC-MOM-BASKET"

// xsecRanked 截面排名行（mom 降序——top=多篮侧、tail=空篮侧）。
type xsecRanked struct {
	inst string
	mom  float64
}

type xsecParams struct {
	ScanIv        time.Duration
	LookbackDays  int
	StaleH        float64
	BasketK       int
	RebalPerYear  float64
	FrictionRT    float64
	TargetNet     float64
	FalsifySharpe float64
	FalsifyGross  float64
	PaperWindowD  float64
}

var xsecKeys = []string{
	"research.xsec_mom.scan_interval_min", "research.xsec_mom.lookback_days",
	"research.xsec_mom.stale_h", "research.xsec_mom.basket_k",
	"research.xsec_mom.rebal_per_year", "research.xsec_mom.friction_rt",
	"research.xsec_mom.target_net_annual",
	"research.xsec_mom.falsify_sharpe_lb", "research.xsec_mom.falsify_gross_annual_min",
	"research.xsec_mom.paper_window_d",
}

// XsecRunner 截面动量篮评估岗（md 池——md_klines 1D perp 宇宙+
// fin_opp_eval 正本，研究域同池惯例）。
type XsecRunner struct{ *arm }

func NewXsecRunner(pool *pgxpool.Pool, cfg *config.Client) *XsecRunner {
	return &XsecRunner{newArm(pool, cfg, rejects.New("research", "xsec_mom_missing_params", "xsec_mom_write_err", "gate_kill"), "评估臂-xsec-mom",
		"截面动量篮评估臂（xsec-perp-mom-basket candidate 态——md_klines 1D perp 宇宙→fin_opp_eval 影子产出零执行零提案，O-41394）",
		"xsec-perp-mom-basket", "xsec_mom_basket")}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修同径；具型返回保链）。
func (r *XsecRunner) AttachAlerter(a *notify.Alerter) *XsecRunner {
	r.arm.AttachAlerter(a)
	return r
}

// ScanOnce 单拍壳。
func (r *XsecRunner) ScanOnce(ctx context.Context) error {
	_, err := r.tick(ctx)
	return err
}

// Run 在岗。
func (r *XsecRunner) Run(ctx context.Context, errc chan<- error) {
	r.arm.Run(ctx, errc, r.tick)
}

// params 判据全 app_config 机读（缺/坏=fail-closed）。结构下界：
// lookback≥5（日频动量窗下界）、stale_h>0、basket_k≥1、rebal>0、
// friction≥0、target>0。
func (r *XsecRunner) params() (xsecParams, error) {
	var p xsecParams
	get := func(k string) (float64, error) { return r.cfg.GetFloat(k) }
	var err error
	vals := make([]float64, len(xsecKeys))
	for i, k := range xsecKeys {
		if vals[i], err = get(k); err != nil {
			return p, fmt.Errorf("%s: %w", k, err)
		}
	}
	p.ScanIv = time.Duration(vals[0]) * time.Minute
	p.LookbackDays = int(vals[1])
	p.StaleH, p.BasketK = vals[2], int(vals[3])
	p.RebalPerYear, p.FrictionRT, p.TargetNet = vals[4], vals[5], vals[6]
	p.FalsifySharpe, p.FalsifyGross, p.PaperWindowD = vals[7], vals[8], vals[9]
	if p.LookbackDays < 5 || p.StaleH <= 0 || p.BasketK < 1 ||
		p.RebalPerYear <= 0 || p.FrictionRT < 0 || p.TargetNet <= 0 {
		return p, fmt.Errorf("判据下界违例 lookback=%d stale_h=%.1f k=%d rebal=%.1f friction=%.5f target=%.3f（结构下界）",
			p.LookbackDays, p.StaleH, p.BasketK, p.RebalPerYear, p.FrictionRT, p.TargetNet)
	}
	return p, nil
}

// tick 单拍：判据→gate→宇宙评估→批落表。
func (r *XsecRunner) tick(ctx context.Context) (time.Duration, error) {
	to := 2 * time.Minute
	if v, err := r.cfg.GetFloat("funding.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx
	if k, err := r.cfg.GetBool("gate.kill.xsec_mom"); err == nil && k {
		r.rej.Inc("gate_kill")
		return 0, nil
	}
	r.watch.SetBeat(24 * time.Hour)
	r.watch.Check(ctx)
	p, err := r.params()
	if err != nil {
		r.rej.Inc("xsec_mom_missing_params")
		return 0, fmt.Errorf("判据缺位=不评估（fail-closed）: %w", err)
	}
	row, err := r.evalBasket(ctx, p)
	if err != nil {
		r.rej.Inc("xsec_mom_write_err")
		return 0, err
	}
	var rows []evalRow
	if row.inst != "" {
		rows = []evalRow{row}
	}
	if err := r.commit(ctx, rows); err != nil {
		r.rej.Inc("xsec_mom_write_err")
		return 0, err
	}
	if len(rows) > 0 {
		r.watch.OK()
	} else {
		r.watch.Quiet()
	}
	return p.ScanIv, nil
}

// evalBasket 截面动量评估——返回单行篮级卡；宇宙真空（无 perp klines
// 或全陈）返零值行=不写卡（断供面归 breed_stall 探针裁非臂裁）。
func (r *XsecRunner) evalBasket(ctx context.Context, p xsecParams) (evalRow, error) {
	var zero evalRow
	instRows, err := r.pool.Query(ctx, `SELECT DISTINCT inst_id FROM md_klines
		WHERE venue=$1 AND bar='1D' AND confirm AND inst_id LIKE '%-SWAP'
		ORDER BY inst_id`, fundingVenue)
	if err != nil {
		return zero, err
	}
	var insts []string
	for instRows.Next() {
		var s string
		if err := instRows.Scan(&s); err != nil {
			instRows.Close()
			return zero, err
		}
		insts = append(insts, s)
	}
	instRows.Close()
	if err := instRows.Err(); err != nil {
		return zero, err
	}
	if len(insts) == 0 {
		return zero, nil
	}
	staleBefore := time.Now().Add(-time.Duration(p.StaleH * float64(time.Hour)))
	type momSnap struct {
		mom  float64
		last time.Time
	}
	moms := map[string]momSnap{}
	var lastTs time.Time
	for _, inst := range insts {
		m, ts, ok, err := r.instMom(ctx, inst, p.LookbackDays+1, staleBefore)
		if err != nil {
			return zero, err
		}
		if !ok {
			continue // 棒不足/陈棒——不入截面（不虚构动量）
		}
		moms[inst] = momSnap{m, ts}
		if ts.After(lastTs) {
			lastTs = ts
		}
	}
	n := len(moms)
	frictionAnnual := p.RebalPerYear * p.FrictionRT * float64(2*p.BasketK)
	base := map[string]any{
		"n_universe": n, "basket_k": p.BasketK, "lookback_days": p.LookbackDays,
		"rebal_per_year": p.RebalPerYear, "friction_rt": p.FrictionRT,
		"friction_annual": frictionAnnual, "friction_legs": 2 * p.BasketK,
		"friction_note":     "结构腿数 2k 全换手——保守上界（卡面 4legs×12 约 3.5%/yr 为 k2 口径）",
		"target_net_annual": p.TargetNet,
		"falsify_sharpe_lb": p.FalsifySharpe, "falsify_gross_annual_min": p.FalsifyGross,
		"paper_window_d": p.PaperWindowD, "last_bar_ts": lastTs, "stale_h": p.StaleH,
	}
	if n < 2*p.BasketK {
		base["reason"] = "thin_universe"
		return evalRow{inst: xsecBasketInst, verdict: "rejected", reason: "thin_universe",
			legs: []string{"perp_long_basket", "perp_short_basket"}, evidence: base}, nil
	}
	all := make([]xsecRanked, 0, n)
	for inst, s := range moms {
		all = append(all, xsecRanked{inst, s.mom})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].mom > all[j].mom })
	top := all[:p.BasketK]
	bot := all[n-p.BasketK:]
	mean := func(rs []xsecRanked) float64 {
		var s float64
		for _, r := range rs {
			s += r.mom
		}
		return s / float64(len(rs))
	}
	spread := mean(top) - mean(bot) // L 日截面动量差
	ann := 365.0 / float64(p.LookbackDays)
	gross := spread * ann
	net := gross - frictionAnnual
	verdict, reason := "rejected", "net_thin"
	if net >= p.TargetNet {
		verdict, reason = "proposed", ""
	}
	base["spread_l_days"] = spread
	base["gross_basis"] = "xsec_mom_spread_annualized_proxy"
	base["gross_apr"] = gross
	base["top_basket"] = topList(top)
	base["bottom_basket"] = topList(bot)
	base["reason"] = reason
	return evalRow{inst: xsecBasketInst, verdict: verdict, reason: reason,
		grossAPR: gross, netAPR: net,
		legs: []string{"perp_long_basket", "perp_short_basket"}, evidence: base}, nil
}

// instMom 单 inst 末 n 根确认 1D 棒→L 日对数动量+末棒 ts；棒不足 n
// 或末棒陈于 staleBefore→ok=false 不入截面。
func (r *XsecRunner) instMom(ctx context.Context, inst string, n int, staleBefore time.Time) (float64, time.Time, bool, error) {
	rows, err := r.pool.Query(ctx, `SELECT c, ts FROM md_klines
		WHERE venue=$1 AND inst_id=$2 AND bar='1D' AND confirm
		ORDER BY ts DESC LIMIT $3`, fundingVenue, inst, n)
	if err != nil {
		return 0, time.Time{}, false, err
	}
	defer rows.Close()
	var closes []float64
	var last time.Time
	first := true
	for rows.Next() {
		var c float64
		var ts time.Time
		if err := rows.Scan(&c, &ts); err != nil {
			return 0, time.Time{}, false, err
		}
		if first {
			last, first = ts, false
		}
		closes = append(closes, c)
	}
	if err := rows.Err(); err != nil {
		return 0, time.Time{}, false, err
	}
	if len(closes) < n || last.Before(staleBefore) || closes[0] <= 0 || closes[len(closes)-1] <= 0 {
		return 0, time.Time{}, false, nil
	}
	// closes=倒序（新→旧）：mom=ln(c_newest/c_oldest)=ln(closes[0]/closes[n-1])
	return math.Log(closes[0] / closes[len(closes)-1]), last, true, nil
}

func topList(rs []xsecRanked) []map[string]any {
	out := make([]map[string]any, 0, len(rs))
	for _, r := range rs {
		out = append(out, map[string]any{"inst": r.inst, "mom": r.mom})
	}
	return out
}
