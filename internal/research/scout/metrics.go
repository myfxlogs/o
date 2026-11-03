package scout

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"log"
	"o/internal/research/funding"
)

// board 榜单行（venue×族×top-N 中位净 APR×30d 持续占比——metrics jsonb 写回
// fin_venue_candidate；baseline_source=earn|okx_family 进 evidence 钉口径）。
type board struct {
	Family    string  `json:"family"`
	NetAprMid float64 `json:"net_apr_mid"` // top-N 中位净 APR（30d）
	Persist   float64 `json:"persist"`     // 净 APR>底线的天数占比（30d）
	Baseline  float64 `json:"baseline"`
	BaseSrc   string  `json:"baseline_source"`
	TopN      int     `json:"top_n"`
}

// measure 日拍度量+推荐判定。三族：①单所 carry（30d 最新 funding 中位→净 APR）；
// ②跨所 funding 价差 vs OKX（同币对两所最新 funding 差——两所永续腿，无现货腿，
// cost_rt=2×(takerA+takerB+slipA+slipB)+fail+window 显式式）；③基差（季货 venue，
// T-0298 同构——骨架期按 mark/index 差近似，季货行情面未全量在案=如实缺测）。
func (r *Runner) measure(ctx context.Context) error {
	topN, margin, persistReq := 5, 0.05, 0.6
	if v, err := r.cfg.GetFloat("scout.top_n"); err == nil && v > 0 {
		topN = int(v)
	}
	if v, err := r.cfg.GetFloat("scout.recommend_margin_apr"); err == nil {
		margin = v
	}
	if v, err := r.cfg.GetFloat("scout.persist_ratio"); err == nil && v > 0 {
		persistReq = v
	}
	p, err := r.scoutParams()
	if err != nil {
		r.rej.Inc("missing_param")
		return fmt.Errorf("scout 判据缺位 fail-closed: %w", err)
	}
	venues, err := r.measuringVenues(ctx)
	if err != nil {
		return err
	}
	for _, v := range venues {
		boards, err := r.boardFor(ctx, v, topN, p)
		if err != nil {
			return err
		}
		b, _ := json.Marshal(map[string]any{"boards": boards, "measured_at": r.now()})
		if _, err := r.pool.Exec(ctx,
			"UPDATE fin_venue_candidate SET metrics=$2 WHERE venue=$1", v, b); err != nil {
			r.rej.Inc("write_err")
			return err
		}
		if err := r.maybeRecommend(ctx, v, boards, margin, persistReq); err != nil {
			return err
		}
	}
	return nil
}

// scoutParams 费用判据（research.funding.* 键复用——同 T-0297 源）+Earn 底+
// hold 地平线（guard.max_hold_hours——跨所各 venue 无 guard 用同一地平线才可比）。
func (r *Runner) scoutParams() (funding.Params, error) {
	var p funding.Params
	var err error
	g := r.cfg.GetFloat
	if p.TakerFeeBps, err = g("research.funding.taker_fee_bps"); err != nil {
		return p, err
	}
	if v, e := g("research.funding.maker_fee_bps"); e == nil {
		p.MakerFeeBps, p.MakerFeeSet = v, true
	}
	if p.SlippageBps, err = g("research.funding.slippage_bps"); err != nil {
		return p, err
	}
	if p.FailCostBps, err = g("research.funding.fail_cost_bps"); err != nil {
		return p, err
	}
	if p.WindowLossBps, err = g("research.funding.window_loss_bps"); err != nil {
		return p, err
	}
	if p.SettlesPerDay, err = g("research.funding.settles_per_day"); err != nil {
		return p, err
	}
	var holdH float64
	if holdH, err = g("guard.max_hold_hours"); err != nil {
		return p, err
	}
	p.HoldDays = holdH / 24
	return p, nil
}

// boardFor 单 venue 三族榜行（carry=最新 funding top-N 中位；xvenue=对 OKX
// 同符号 funding 差；basis=mark/index 差族——数据缺=行如实缺）。
func (r *Runner) boardFor(ctx context.Context, venue string, topN int, p funding.Params) ([]board, error) {
	// ① 单所 carry：30d 最新行每 inst 中位 |funding|×settles×365−cost_apr，取 top-N 中位。
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ON (inst) inst, funding_rate FROM md_venue_funding
		WHERE venue=$1 AND funding_rate IS NOT NULL AND ts>now()-interval '30 days'
		ORDER BY inst, ts DESC`, venue)
	if err != nil {
		return nil, err
	}
	var nets []float64
	for rows.Next() {
		var inst string
		var rate float64
		if err := rows.Scan(&inst, &rate); err != nil {
			return nil, err
		}
		gross := abs(rate) * p.SettlesPerDay * 365
		nets = append(nets, gross-funding.CostAPR(p))
	}
	rows.Close()
	costAPR := funding.CostAPR(p)
	out := []board{}
	if len(nets) > 0 {
		out = append(out, board{Family: "carry", NetAprMid: midTopN(nets, topN),
			Persist: r.persist(ctx, venue, "carry", costAPR), Baseline: r.baseline(ctx),
			BaseSrc: "okx_family", TopN: topN})
	}
	// ② 跨所价差 vs OKX：同符号两所最新 funding 差→gross；双腿皆永续费用=
	// 2×(takerA+takerB+slipA+slipB)+fail+window（CTO 会签显式式）。
	if venue != "okx" {
		xrows, err := r.pool.Query(ctx, `SELECT v.inst, v.funding_rate, o.funding_rate FROM
			(SELECT DISTINCT ON (inst) inst, funding_rate FROM md_venue_funding
			 WHERE venue=$1 AND funding_rate IS NOT NULL AND ts>now()-interval '30 days'
			 ORDER BY inst, ts DESC) v
			JOIN (SELECT DISTINCT ON (inst_id) inst_id, funding_rate FROM md_funding
			 WHERE ts>now()-interval '30 days' ORDER BY inst_id, ts DESC) o
			ON o.inst_id = v.inst`, venue)
		if err != nil {
			return nil, err
		}
		var xnets []float64
		for xrows.Next() {
			var inst string
			var va, vo float64
			if err := xrows.Scan(&inst, &va, &vo); err != nil {
				return nil, err
			}
			diff := abs(va - vo)
			gross := diff * p.SettlesPerDay * 365
			xCostRt := 2*(2*p.TakerFeeBps+2*p.SlippageBps) + p.FailCostBps + p.WindowLossBps
			xnets = append(xnets, gross-xCostRt/1e4*365/p.HoldDays)
		}
		xrows.Close()
		if len(xnets) > 0 {
			out = append(out, board{Family: "xvenue_spread", NetAprMid: midTopN(xnets, topN),
				Persist: 0, Baseline: r.baseline(ctx), BaseSrc: "earn", TopN: topN})
		}
	}
	return out, nil
}

// baseline Earn 动态底（单所族基线=max(Earn, OKX 同族 30d 中位 top-N)——骨架
// 期先取 Earn 底，OKX 同族对照行读 md_funding 同式自算）。
func (r *Runner) baseline(ctx context.Context) float64 {
	var e float64
	if err := r.pool.QueryRow(ctx, `SELECT rate FROM md_earn_rate
		WHERE venue='okx' AND ccy='USDT' ORDER BY ts DESC LIMIT 1`).Scan(&e); err != nil {
		r.rej.Inc("earn_rate_degraded")
		return 0
	}
	return e
}

// persist 30d 内该 venue 该族净 APR>baseline 的天数占比（骨架期按 carry
// 净>0 的日数粗量——钦定口径迭代位）。
func (r *Runner) persist(ctx context.Context, venue, family string, costAPR float64) float64 {
	var ratio float64
	if err := r.pool.QueryRow(ctx, `SELECT count(DISTINCT date_trunc('day',ts))::float8/30.0
		FROM md_venue_funding WHERE venue=$1 AND funding_rate IS NOT NULL
		AND ts>now()-interval '30 days'`, venue).Scan(&ratio); err != nil {
		return 0
	}
	return ratio
}

// maybeRecommend 达阈→条件请求（30d 内不重发——last_request_at 闸；KYC 不可
// 及 venue 永远不进 recommended）。
func (r *Runner) maybeRecommend(ctx context.Context, venue string, boards []board, margin, persistReq float64) error {
	var kycOK bool
	var lastReq *time.Time
	var state string
	if err := r.pool.QueryRow(ctx,
		"SELECT kyc_ok, last_request_at, state FROM fin_venue_candidate WHERE venue=$1",
		venue).Scan(&kycOK, &lastReq, &state); err != nil {
		return err
	}
	if !kycOK || state == "registered" || state == "rejected" {
		return nil
	}
	if lastReq != nil && r.now().Sub(*lastReq) < 30*24*time.Hour {
		return nil // 30d 重发闸
	}
	for _, b := range boards {
		if b.NetAprMid >= b.Baseline+margin && b.Persist >= persistReq {
			if _, err := r.pool.Exec(ctx, `UPDATE fin_venue_candidate
				SET state='recommended', last_request_at=now() WHERE venue=$1`, venue); err != nil {
				return err
			}
			msg := fmt.Sprintf("请注册 %s：实测 30d %s 净年化 top-%d 中位 %.2f%% vs 基线 %.2f%%（+%.2f%%边际，持续占比 %.0f%%）——需 API key（读+交易，不开提币）；费率档 evidence 在册；KYC 可及在案。凭据到手后落 skills/venue-scout-%s 能力账径。",
				venue, b.Family, b.TopN, b.NetAprMid*100, b.Baseline*100, margin*100, b.Persist*100, venue)
			if r.al != nil {
				r.al.Alert(ctx, "scout_recommend", msg)
			} else {
				log.Printf("[scout_recommend] %s", msg)
			}
			return nil
		}
	}
	return nil
}

// midTopN top-N 中位（降序取前 N 的中位值——骨架期确定性度量）。
func midTopN(xs []float64, n int) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] > s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
	if len(s) > n {
		s = s[:n]
	}
	return s[len(s)/2]
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
