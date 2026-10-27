// trx_faces.go —— TRX 自营盘 proto 读面（T-0156：REST 手拼 JSON 面全撤，
// CoreService 单源——nox service_trx_*/service_hedge_* 对拍语义原样；
// found=false=读件缺显形不造零；D-629 业绩外自营盘零 NAV 计入）。
package main

import (
	"context"

	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"o/internal/data/cmc"
	"o/internal/gen/dashboardv1"
	"o/internal/store/pgstore"
)

// trxAccountRow 快照行→proto（GetTrxAccount/TrxAccountRefresh 共用——折值后端算定）。
func trxAccountRow(a pgstore.TRXAccount) *dashboardv1.GetTrxAccountResponse {
	sr := ""
	if a.SRAddress != nil {
		sr = *a.SRAddress
	}
	return &dashboardv1.GetTrxAccountResponse{
		Found: true, Address: a.Address, SrAddress: sr,
		TrxBalance:      float64(a.BalanceSun) / 1e6,
		TrxAllowance:    float64(a.RewardNumSun) / 1e6,
		TronPower:       float64(a.TronPower) / 1e6,
		EnergyDelegated: float64(a.EnergyDelegated) / 1e6,
		BandwidthStaked: float64(a.BandwidthStaked) / 1e6,
		NetLimit:        float64(a.NetLimit), NetUsed: float64(a.NetUsed),
		FreeNetLimit: float64(a.FreeNetLimit), FreeNetUsed: float64(a.FreeNetUsed),
		EnergyLimit: float64(a.EnergyLimit), EnergyUsed: float64(a.EnergyUsed),
		Votes:           float64(a.VotesTotal),
		BandwidthStatus: bandwidthStatus(a.NetLimit, a.NetUsed, a.FreeNetLimit, a.FreeNetUsed),
		FetchedAt:       timestamppb.New(a.FetchedAt),
	}
}

// GetTrxAccount 账户资源快照读面（nox 对拍——空表 found=false 读件缺显形）。
func (c coreService) GetTrxAccount(ctx context.Context,
	_ *dashboardv1.GetTrxAccountRequest) (*dashboardv1.GetTrxAccountResponse, error) {
	a, err := c.trx.st.LastTRXAccount(ctx)
	if err != nil {
		return &dashboardv1.GetTrxAccountResponse{}, nil
	}
	return trxAccountRow(a), nil
}

// TrxAccountRefresh 刷新踢（开页/手按——读侧拉新 venue→库写，静默败不阻断
// 渲染；非钱径。回带最新快照段）。
func (c coreService) TrxAccountRefresh(ctx context.Context,
	_ *dashboardv1.TrxAccountRefreshRequest) (*dashboardv1.TrxAccountRefreshResponse, error) {
	w := c.trx
	if w.accts != nil {
		_, _ = w.accts.Poll(ctx)
	}
	if w.apr != nil {
		w.apr.TriggerNow(ctx)
	}
	resp := &dashboardv1.TrxAccountRefreshResponse{}
	if a, err := w.st.LastTRXAccount(ctx); err == nil {
		resp = &dashboardv1.TrxAccountRefreshResponse{Found: true, Address: a.Address,
			TrxBalance: float64(a.BalanceSun) / 1e6, FetchedAt: timestamppb.New(a.FetchedAt)}
	}
	return resp, nil
}

// ListTrxTxs 链上流水读面（limit/kind 过滤下推——前端零聚合令）。
func (c coreService) ListTrxTxs(ctx context.Context,
	req *dashboardv1.ListTrxTxsRequest) (*dashboardv1.ListTrxTxsResponse, error) {
	rows, err := c.trx.st.ListTRXTxs(ctx, int(req.GetLimit()), req.GetKind())
	if err != nil {
		return nil, err
	}
	out := make([]*dashboardv1.TrxTx, 0, len(rows))
	for _, t := range rows {
		cp := ""
		if t.Counterparty != nil {
			cp = *t.Counterparty
		}
		out = append(out, &dashboardv1.TrxTx{Id: t.ID, TxId: t.TxID,
			Ts: timestamppb.New(t.Ts), Kind: t.Kind,
			AmountTrx: float64(t.AmountSun) / 1e6, Counterparty: cp, CheatStatus: t.CheatStatus})
	}
	return &dashboardv1.ListTrxTxsResponse{Txs: out}, nil
}

// ListHedgeFlow 对冲流水读面（trx/cmc 双源 UNION 归一——cheat/vote/买卖成交
// 剔除照 nox；total+offset 分页下推）。
func (c coreService) ListHedgeFlow(ctx context.Context,
	req *dashboardv1.ListHedgeFlowRequest) (*dashboardv1.ListHedgeFlowResponse, error) {
	w := c.trx
	items, err := w.st.ListHedgeFlowPage(ctx, req.GetSourceFilter(), int(req.GetLimit()), int(req.GetOffset()))
	if err != nil {
		return nil, err
	}
	total, err := w.st.ListHedgeFlowCount(ctx, req.GetSourceFilter())
	if err != nil {
		return nil, err
	}
	out := make([]*dashboardv1.HedgeFlowItem, 0, len(items))
	for _, it := range items {
		out = append(out, &dashboardv1.HedgeFlowItem{Source: it.Source,
			Ts: timestamppb.New(it.Ts), Kind: it.Kind, AmountValue: it.AmountValue,
			AmountUnit: it.AmountUnit, Counterparty: it.Counterparty, TxId: it.TxID})
	}
	return &dashboardv1.ListHedgeFlowResponse{Total: int32(total), Txs: out}, nil
}

// GetHedgeStructure 对冲结构复合读面（nox GetHedgeStructure 对拍——freshLive
// 三级兜底/平台口径 50%/80% CMC 标定/双腿镜像）。
func (c coreService) GetHedgeStructure(ctx context.Context,
	_ *dashboardv1.GetHedgeStructureRequest) (*dashboardv1.GetHedgeStructureResponse, error) {
	w := c.trx
	trxAcc, hasTRX := w.st.LastTRXAccount(ctx)
	trxApr, hasApr := w.st.LastTRXVoteApr(ctx)
	ls := w.freshLive(ctx)
	cmcSum, hasSum := w.st.LastCMCAccountSummary(ctx)
	var baseline float64
	if p, err := w.st.LastCMCPositionByInstrument(ctx, "X-AFQUL"); err == nil {
		baseline = p.EntryPrice
	}
	spot := &dashboardv1.HedgeLegSpot{Label: "质押现货"}
	if hasTRX == nil {
		spot.QtyStaked = float64(trxAcc.TronPower) / 1e6
		spot.QtyLiquid = float64(trxAcc.BalanceSun) / 1e6
		spot.Stake = &dashboardv1.HedgeStake{
			Energy:    float64(trxAcc.EnergyDelegated) / 1e6,
			Bandwidth: float64(trxAcc.BandwidthStaked) / 1e6}
	}
	if hasApr == nil {
		spot.VoteAprCurrent = trxApr.CurrentRate
		spot.VoteAprMax = trxApr.MaxRate
		spot.VoteAprSpread = trxApr.Spread
		if trxApr.MaxSr != nil {
			spot.VoteAprMaxSr = *trxApr.MaxSr
		}
	}
	spot.PnlVsBase = -ls.PnL // 镜像近似（现货购价未知）
	short := &dashboardv1.HedgeLegShort{Label: "CMC 空单"}
	if ls.Qty > 0 {
		short.Qty = ls.Qty
	}
	short.PnlVsBase = ls.PnL // 平台真值（业主令不反推）
	if ls.Lev > 0 {
		short.Lev = ls.Lev
	}
	if hasSum == nil {
		short.Liq = cmcSum.CloseoutPrice
		short.DistPct = cmcSum.DistanceToCloseoutPct
	}
	var netTrx, netUsd float64
	if hasTRX == nil && ls.Qty > 0 {
		netTrx = float64(trxAcc.BalanceSun)/1e6 + float64(trxAcc.TronPower)/1e6 - ls.Qty
		netUsd = netTrx * ls.Mid
	}
	accts := &dashboardv1.HedgeAccounts{}
	if hasTRX == nil {
		accts.TrxBalance = float64(trxAcc.BalanceSun) / 1e6
		accts.TrxAllowance = float64(trxAcc.RewardNumSun) / 1e6
		accts.EnergyDelegated = float64(trxAcc.EnergyDelegated) / 1e6
		accts.Votes = float64(trxAcc.VotesTotal)
		accts.NetLimit = float64(trxAcc.NetLimit)
		accts.NetUsed = float64(trxAcc.NetUsed)
		accts.FreeNetLimit = float64(trxAcc.FreeNetLimit)
		accts.FreeNetUsed = float64(trxAcc.FreeNetUsed)
		accts.EnergyLimit = float64(trxAcc.EnergyLimit)
		accts.EnergyUsed = float64(trxAcc.EnergyUsed)
		accts.BandwidthStatus = bandwidthStatus(trxAcc.NetLimit, trxAcc.NetUsed, trxAcc.FreeNetLimit, trxAcc.FreeNetUsed)
	}
	if ls.Cash > 0 || ls.Equity > 0 {
		accts.CmcCash, accts.CmcEquity = ls.Cash, ls.Equity
	} else if hasSum == nil {
		accts.CmcCash, accts.CmcEquity = cmcSum.TotalCashBalance, cmcSum.TotalAccountValue
	}
	margin := ls.Margin
	if margin <= 0 && hasSum == nil {
		margin = cmcSum.TotalEquityUsedInMargin
	}
	alertState := "normal"
	if hasSum == nil {
		alertState = cmc.AlertsToState(cmcSum.Alerts)
	}
	return &dashboardv1.GetHedgeStructureResponse{Structure: &dashboardv1.HedgeStructure{
		BaselinePrice: baseline, Mid: ls.Mid, Ask: ls.Ask,
		LegSpot: spot, LegShort: short,
		NetExposureTrx: netTrx, NetExposureUsd: netUsd,
		Carry:              w.hedgeCarry(ctx, ls.Mid, spot.QtyStaked),
		Accts:              accts,
		WsConnected:        w.cmcWsHealth != nil && w.cmcWsHealth(),
		CloseoutSource:     "platform_rule_50",
		PlatformAlertState: alertState,
		CloseoutLevelUsd:   margin * 0.5,
		WarningLevelUsd:    margin * 0.8,
	}}, nil
}

// GetTrxVoteStatus 投票引擎状态卡（T-0184——只读显态面：在驶态/下窗/
// 待领/今日改投计数/上限；启停=app_config 业主径不在前端鉴权面内，
// 降级只读+热改径如实）。
func (c coreService) GetTrxVoteStatus(ctx context.Context,
	_ *dashboardv1.GetTrxVoteStatusRequest) (*dashboardv1.GetTrxVoteStatusResponse, error) {
	w := c.trx
	o := &dashboardv1.GetTrxVoteStatusResponse{}
	if w == nil {
		return o, nil
	}
	if v, err := w.cfg.GetBool("trx.vote_enabled"); err == nil {
		o.Enabled = v
	}
	cap := 12
	if n, err := w.cfg.GetInt("trx.vote_max_rebal_per_day"); err == nil && n > 0 {
		cap = int(n)
	}
	o.RebalanceCap = int32(cap)
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(count(*),0) FROM fin_trx_vote_rebalance
		WHERE decided_at::date=CURRENT_DATE`).Scan(&o.RebalanceToday)
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(count(*),0) FROM fin_trx_txs
		WHERE kind='vote_claim' AND ts::date=CURRENT_DATE`).Scan(&o.ClaimToday)
	// 待领取=rewardNum 折值（同 trx_allowance 源——account 快照行）。
	if a, err := w.st.LastTRXAccount(ctx); err == nil {
		o.PendingTrx = float64(a.RewardNumSun) / 1e6
	}
	// 下一领奖窗=00/06/12/18 UTC（boundary.go 实证锚——rewardNum 变化全贴边界）。
	now := time.Now().UTC()
	for h := 0; h <= 24; h += 6 {
		t := time.Date(now.Year(), now.Month(), now.Day(), h%24, 0, 0, 0, time.UTC)
		if h == 24 {
			t = t.Add(24 * time.Hour)
		}
		if t.After(now) {
			o.NextWindow = timestamppb.New(t)
			break
		}
	}
	return o, nil
}
