// trx_yield_face.go —— 收益雷达+日曲线 proto 读面（T-0156：REST 手拼面撤，
// CoreService 单源——nox service_trx_yield_* 对拍语义原样；后端算好前端零
// 计算；冷启动不猜，样本不足透传）。
package main

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"o/internal/gen/dashboardv1"
)

// GetTrxYieldRadar 收益雷达读面（nox GetTrxYieldRadar 对拍——能量实收年化+
// 投票报价年化+floor 参考线+过渡期护栏）。
func (c coreService) GetTrxYieldRadar(ctx context.Context,
	_ *dashboardv1.GetTrxYieldRadarRequest) (*dashboardv1.GetTrxYieldRadarResponse, error) {
	w := c.trx
	apr7d, s7 := w.energyRentalAPR(ctx, 7)
	apr30d, s30 := w.energyRentalAPR(ctx, 30)
	var voteCur, voteMax, voteSpread float64
	var voteMaxSr string
	var updatedAt time.Time
	if v, err := w.st.LastTRXVoteApr(ctx); err == nil {
		voteCur = v.MinRate
		if voteCur == 0 {
			voteCur = v.CurrentRate
		}
		voteMax, voteSpread = v.MaxRate, v.Spread
		if v.MaxSr != nil {
			voteMaxSr = *v.MaxSr
		}
		updatedAt = v.FetchedAt
	}
	var vote30d float64
	if avg, ok, err := w.st.AvgTRXVoteAprByRange(ctx, time.Now().UTC().AddDate(0, 0, -30), time.Now().UTC()); err == nil && ok {
		vote30d = avg
	}
	floorPct, _ := w.cfg.GetFloat("trx.floor_pct")
	// 9-9 过渡期护栏：7d 窗内在押变化>20% → floor 判定无效。
	var transition bool
	if minE, maxE, err := w.st.EnergyDelegatedRangeInWindow(ctx,
		time.Now().UTC().AddDate(0, 0, -7), time.Now().UTC()); err == nil && minE > 0 {
		transition = maxE > minE*12/10
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	return &dashboardv1.GetTrxYieldRadarResponse{
		EnergyApr_7D: apr7d, EnergyApr_30D: apr30d,
		VoteAprCurrent: voteCur, VoteApr_30DAvg: vote30d,
		FloorPct: floorPct, SampleDays_7D: int32(s7), SampleDays_30D: int32(s30),
		Transition: transition, VoteAprMax: voteMax,
		VoteAprMaxSr: voteMaxSr, VoteAprSpread: voteSpread,
		UpdatedAt: timestamppb.New(updatedAt),
	}, nil
}

// GetTrxYieldCurve 净收益率日曲线（nox GetTrxYieldCurve 对拍——分母=$36.5K
// 池口径 AF4；price 缺日=现价兜底）。
func (c coreService) GetTrxYieldCurve(ctx context.Context,
	req *dashboardv1.GetTrxYieldCurveRequest) (*dashboardv1.GetTrxYieldCurveResponse, error) {
	w := c.trx
	days := int(req.GetDays())
	if days <= 0 {
		days = 30
	}
	now := time.Now().UTC()
	ys, err := w.st.ListTRXYieldByDay(ctx, now.AddDate(0, 0, -days), now)
	if err != nil {
		return nil, err
	}
	// T-0192 业主令：价格只认 CMC 平台报价（跨所价替代=漂移风险）——
	// TRXUSDT 行缺席→X-AFQUL 腿原生 mid 回退（对冲基线同源零漂移）。
	var currentPrice float64
	if mid, _, err := w.st.LastCMCPriceMidWithTime(ctx, "TRXUSDT"); err == nil && mid > 0 {
		currentPrice = mid
	}
	if currentPrice == 0 {
		if mid, _, err := w.st.LastCMCPriceMidWithTime(ctx, "X-AFQUL"); err == nil && mid > 0 {
			currentPrice = mid
		}
	}
	const poolPrincipal = 36500.0
	points := make([]*dashboardv1.TrxYieldCurvePoint, 0, len(ys))
	var sum7, sum30 float64
	var c7, c30 int
	for _, yd := range ys {
		price := currentPrice
		if yd.Price != nil {
			price = *yd.Price
		}
		total := (yd.EnergyTrx + yd.VotingTrx) * price
		points = append(points, &dashboardv1.TrxYieldCurvePoint{
			Day: timestamppb.New(yd.Day), EnergyTrx: yd.EnergyTrx,
			VotingTrx: yd.VotingTrx, Price: price, DailyTotalUsd: total})
		age := int(now.Sub(yd.Day) / (24 * time.Hour))
		if age < 7 {
			sum7 += total
			c7++
		}
		if age < 30 {
			sum30 += total
			c30++
		}
	}
	var ann7, ann30 float64
	if c7 > 0 {
		ann7 = sum7 / poolPrincipal * 365 / float64(c7) * 100
	}
	if c30 > 0 {
		ann30 = sum30 / poolPrincipal * 365 / float64(c30) * 100
	}
	rentalSun, _ := w.st.SumTRXTxRentalAll(ctx)
	voteSun, _ := w.st.SumTRXTxVoteClaimAll(ctx)
	cumTrx := float64(rentalSun+voteSun) / 1e6
	return &dashboardv1.GetTrxYieldCurveResponse{
		Points: points, Annualized_7D: ann7, Annualized_30D: ann30,
		CurrentPrice: currentPrice, PoolPrincipalUsd: poolPrincipal,
		SampleDays_7D: int32(c7), SampleDays_30D: int32(c30),
		CumulativeUnhedgedIncomeTrx: cumTrx,
		CumulativeUnhedgedIncomeUsd: cumTrx * currentPrice,
	}, nil
}
