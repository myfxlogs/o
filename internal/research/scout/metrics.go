package scout

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"o/internal/gen/scoutv1"
	"o/internal/research/funding"
)

// board 榜单行（venue×族×top-N 中位净 APR×30d 持续占比——metrics jsonb 写回
// fin_venue_candidate；baseline_source=earn|okx_family 进 evidence 钉口径）。
// Board 型别名=scoutv1.MetricBoard 生成共享型（返修①丙案修订：proto 申
// protojson 出入 metrics jsonb——生产/消费同用生成型字段漂移编译期炸；
// scout/board 叶子件退役）。
type Board = scoutv1.MetricBoard

// measure 日拍度量+推荐判定（返修③：baseline=max(Earn, OKX 同族中位)实算；
// 跨所族基线=Earn 底——两所同在才存在无 OKX 同族）。
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
	// Earn 底共用读件（funding.EarnRate——返修③ 零复制）。
	var earn float64
	staleH, herr := r.cfg.GetFloat("research.earn.stale_hours")
	e, fresh, eerr := funding.EarnRate(ctx, r.pool, staleH)
	switch {
	case herr != nil || eerr != nil || !fresh || e == nil:
		r.rej.Inc("earn_rate_degraded") // 降级=基线退回零底+计数留痕
	default:
		earn = *e
	}
	venues, err := r.measuringVenues(ctx)
	if err != nil {
		return err
	}
	for _, v := range venues {
		boards, err := r.boardFor(ctx, v, topN, p, earn)
		if err != nil {
			return err
		}
		// T-0304：rejects Door 快照随 metrics 出观察面（venue_down 运行期不盲）。
		var pb []*scoutv1.MetricBoard
		for i := range boards {
			pb = append(pb, &boards[i])
		}
		vm := &scoutv1.VenueMetrics{Boards: pb}
		bj, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(vm)
		var merged map[string]any
		json.Unmarshal(bj, &merged)
		merged["measured_at"] = r.now()
		merged["rejects"] = r.rej.Snapshot()
		b, _ := json.Marshal(merged)
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

// scoutParams 费用判据（research.funding.* 键复用——同 T-0297 源）+
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

// boardFor 单 venue 族榜行（返修② OKX 读 md_funding 正本；返修⑤ 双腿费读
// fin_venue_candidate.taker_bps——venue 侧无档=config taker 回退）。
func (r *Runner) boardFor(ctx context.Context, venue string, topN int, p funding.Params, earn float64) ([]Board, error) {
	vrows, err := loadVenue(ctx, r, venue)
	if err != nil {
		return nil, err
	}
	costAPR := funding.CostAPR(p)
	var vTaker float64
	if err := r.pool.QueryRow(ctx,
		"SELECT COALESCE(taker_bps,0)::float8 FROM fin_venue_candidate WHERE venue=$1",
		venue).Scan(&vTaker); err != nil || vTaker <= 0 {
		vTaker = p.TakerFeeBps // venue 档缺=同费表回退（不虚构费率）
	}
	out := []Board{}

	// settles 分所归一（T-0304：interval_h>0→24/iv；缺/0→config 回退——
	// binance 8h→3 恰同回退值；OKX 侧 NULL 回退如实）。
	settles := func(iv float64) float64 {
		if iv > 0 {
			return 24 / iv
		}
		return p.SettlesPerDay
	}
	// OKX 同族基线（carry 族：md_funding 同式日序列中位——返修③）。
	var okxMid float64
	if orows, err := loadVenue(ctx, r, "okx"); err == nil {
		okxMid = med(dailySeries(orows, topN, func(rate, iv float64) float64 {
			return abs(rate)*settles(iv)*365 - costAPR
		}))
	}
	carryBaseline := earn
	baseSrc := "earn"
	if okxMid > carryBaseline {
		carryBaseline, baseSrc = okxMid, "okx_family"
	}

	// ① 单所 carry：逐日净 APR 序列（|rate|×settles×365−cost_apr）。
	if len(vrows) > 0 {
		ser := dailySeries(vrows, topN, func(rate, iv float64) float64 {
			return abs(rate)*settles(iv)*365 - costAPR
		})
		out = append(out, Board{Family: "carry", NetAprMid: med(ser),
			Persist: fracOver(ser, carryBaseline), Baseline: carryBaseline,
			BaselineSource: baseSrc, TopN: int32(topN)})
	}

	// ② 跨所价差 vs OKX：同 canon 同日 funding 差×settles×365−双腿费年化
	// （双腿皆永续无现货腿=2×(takerA+takerB+slipA+slipB)+fail+window——返修⑤
	// takerA=venue 档、takerB=OKX 档读台账）。
	if venue != "okx" {
		orows, err := loadVenue(ctx, r, "okx")
		if err != nil {
			return nil, err
		}
		var oTaker float64
		if err := r.pool.QueryRow(ctx,
			"SELECT COALESCE(taker_bps,0)::float8 FROM fin_venue_candidate WHERE venue='okx'").Scan(&oTaker); err != nil || oTaker <= 0 {
			oTaker = p.TakerFeeBps
		}
		xCostRt := 2*(vTaker+oTaker+2*p.SlippageBps) + p.FailCostBps + p.WindowLossBps
		xCostAPR := xCostRt / 1e4 * 365 / p.HoldDays
		// canon×day join——跨所差式分列（T-0304：rate_v×settles_v−
		// rate_okx×settles_okx 各带本所结算频次，同×3 错比病灶修）。
		joined := instRows{}
		for c, dm := range vrows {
			om, ok := orows[c]
			if !ok {
				continue
			}
			for d, vr := range dm {
				or, ok := om[d]
				if !ok {
					continue
				}
				if joined[c] == nil {
					joined[c] = map[string]dayRow{}
				}
				joined[c][d] = dayRow{rate: abs(vr.rate*settles(vr.iv) - or.rate*settles(or.iv))}
			}
		}
		if len(joined) > 0 {
			ser := dailySeries(joined, topN, func(diff, _ float64) float64 {
				return diff*365 - xCostAPR
			})
			out = append(out, Board{Family: "xvenue_spread", NetAprMid: med(ser),
				Persist: fracOver(ser, earn), Baseline: earn, BaselineSource: "earn", TopN: int32(topN)})
		}
	}
	return out, nil
}

// maybeRecommend 达阈→条件请求（30d 内不重发——last_request_at 闸；KYC 不可
// 及 venue 永远不进 recommended）。
func (r *Runner) maybeRecommend(ctx context.Context, venue string, boards []Board, margin, persistReq float64) error {
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
	for i := range boards { // T-0316：索引迭代不拷锁（Board=proto 型含
		// MessageState——值拷贝 vet copylocks 诊；型本身零改）
		b := &boards[i]
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
