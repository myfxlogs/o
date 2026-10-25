// trx_hedge.go —— 对冲结构复合读面（T-0110 nox service_hedge_structure 移植——
// freshLive 三级兜底/carry 统计/带宽状态；trx_faces.go 拆件行数棘轮）。
package main

import (
	"context"
	"time"

	"o/internal/data/cmc"
	"o/internal/portfolio/trxvote"
)

// bandwidthStatus 剩余带宽状态（nox service_trx_resource 移植——阈值引用
// trxvote.BandwidthNeedVoteBP/ClaimBP 同源禁双写）。
func bandwidthStatus(netLimit, netUsed, freeNetLimit, freeNetUsed int64) string {
	remaining := (freeNetLimit - freeNetUsed) + (netLimit - netUsed)
	if remaining < trxvote.BandwidthNeedClaimBP {
		return "red"
	}
	if remaining < trxvote.BandwidthNeedVoteBP {
		return "yellow"
	}
	return "normal"
}

// freshLive 内存态优先、>30s 回退 DB（nox freshLive D-236 变更9 对拍——
// mid 兜底序②③④照搬；nil live=纯 DB 径）。
func (w *trxWiring) freshLive(ctx context.Context) cmc.LiveState {
	var ls cmc.LiveState
	if w.cmcLive != nil {
		ls = w.cmcLive()
	}
	posFresh := !ls.PosTs.IsZero() && time.Since(ls.PosTs) <= 30*time.Second
	if !posFresh || ls.Mid <= 0 {
		if mid, _, err := w.st.LastCMCPriceMidWithTime(ctx, "X-AFQUL"); err == nil && mid > 0 {
			ls.Mid = mid
		}
	}
	if !posFresh || ls.Ask <= 0 {
		if cmcPos, err := w.st.LastCMCPositionByInstrument(ctx, "X-AFQUL"); err == nil {
			if !posFresh || ls.Ask <= 0 {
				ls.Ask = cmcPos.EvaluationAskPrice
			}
			if !posFresh || ls.Mid <= 0 {
				ls.Mid = cmcPos.EvaluationBidPrice
				if ls.Mid <= 0 {
					ls.Mid = cmcPos.EvaluationAskPrice
				}
			}
			if !posFresh || ls.Qty <= 0 {
				ls.Qty = cmcPos.Quantity
			}
			if !posFresh || ls.Lev <= 0 {
				if cmcPos.MarginPercentage > 0 {
					ls.Lev = 1.0 / cmcPos.MarginPercentage
				}
			}
			if !posFresh || ls.PnL == 0 {
				ls.PnL = cmcPos.UnrealisedPnL
			}
			if !posFresh {
				ls.PosTs = cmcPos.FetchedAt
			}
		}
	}
	sumFresh := !ls.SumTs.IsZero() && time.Since(ls.SumTs) <= 30*time.Second
	if !sumFresh {
		if cmcSum, err := w.st.LastCMCAccountSummary(ctx); err == nil {
			ls.Cash = cmcSum.TotalCashBalance
			ls.Equity = cmcSum.TotalAccountValue
			ls.Margin = cmcSum.TotalEquityUsedInMargin
			ls.SumTs = cmcSum.FetchedAt
		}
	}
	return ls
}

// computeHedgeCarry carry 统计（nox 同名件对拍——窗口/年化/分段照搬）。
func (w *trxWiring) computeHedgeCarry(ctx context.Context, mid, qtyStaked float64) map[string]any {
	now := time.Now().UTC()
	from7d := now.Add(-7 * 24 * time.Hour)
	energy7d, voting7d, _, _ := w.st.SumDailyIncomeByRange(ctx, &from7d, &now)
	overnight7d, _ := w.st.SumCMCOvernightByRange(ctx, &from7d, &now)
	energyAll, votingAll, interestAll, _ := w.st.SumDailyIncomeByRange(ctx, nil, nil)
	cumUsd := (energyAll+votingAll)*mid + interestAll
	var annualized float64
	if mid > 0 && qtyStaked > 0 {
		if firstDay, err := w.st.FirstDailyIncomeDay(ctx); err == nil && firstDay != nil {
			days := now.Sub(*firstDay).Hours() / 24
			if days < 1 {
				days = 1
			}
			annualized = cumUsd / (qtyStaked * mid) / days * 365 * 100
		}
	}
	hedgedCumUsd, hedgedAnnual := cumUsd, annualized
	if hedgeStart, err := w.st.FirstCMCPositionOpenedAt(ctx); err == nil && hedgeStart != nil {
		if energyH, votingH, interestH, err := w.st.SumDailyIncomeByRange(ctx, hedgeStart, &now); err == nil {
			hedgedCumUsd = (energyH+votingH)*mid + interestH
			if daysH := now.Sub(*hedgeStart).Hours() / 24; mid > 0 && qtyStaked > 0 {
				if daysH < 1 {
					daysH = 1
				}
				hedgedAnnual = hedgedCumUsd / (qtyStaked * mid) / daysH * 365 * 100
			}
		}
	}
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	mtdE, mtdV, mtdI, _ := w.st.SumDailyIncomeByRange(ctx, &monthStart, &now)
	mtdDays, _ := w.st.CountDailyIncomeByRange(ctx, &monthStart, &now)
	return map[string]any{
		"trx_rental_7d_trx": energy7d, "trx_vote_7d_trx": voting7d,
		"cmc_overnight_7d_usd": overnight7d, "cum_usd": cumUsd,
		"annualized_pct": annualized, "hedged_cum_usd": hedgedCumUsd,
		"hedged_annualized_pct": hedgedAnnual, "mtd_net_usd": (mtdE+mtdV)*mid + mtdI,
		"mtd_sample_days": mtdDays,
	}
}

// energyRentalAPR 能量租金年化（nox TrxEnergyRentalAPR 对拍——冷启动不猜）。
func (w *trxWiring) energyRentalAPR(ctx context.Context, days int) (aprPct float64, sampleDays int) {
	if days <= 0 {
		days = 7
	}
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -days)
	rentalSun, err := w.st.SumTRXTxRentalByRange(ctx, from, now)
	if err != nil {
		return 0, 0
	}
	firstTs, ok, err := w.st.FirstTRXTxRentalTsInRange(ctx, from, now)
	if err != nil || !ok {
		return 0, 0
	}
	sampleDays = int(now.Sub(firstTs)/(24*time.Hour)) + 1
	if sampleDays < days {
		return 0, sampleDays
	}
	acct, err := w.st.LastTRXAccount(ctx)
	if err != nil || acct.EnergyDelegated == 0 {
		return 0, sampleDays
	}
	aprPct = (float64(rentalSun) / 1e6 / float64(sampleDays) * 365) / (float64(acct.EnergyDelegated) / 1e6) * 100
	return aprPct, sampleDays
}

// legQtyStaked 质押量提取（carry 分母）。
func legQtyStaked(legSpot map[string]any) float64 {
	if v, ok := legSpot["qty_staked"].(float64); ok {
		return v
	}
	return 0
}
