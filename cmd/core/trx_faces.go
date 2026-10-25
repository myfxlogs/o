// trx_faces.go —— TRX 独立页全功能读面（T-0110 nox service_trx_*/service_hedge_*
// 逐面对拍移植——JSON 形字段名对齐 nox proto 消息 snake_case）。每面对拍有数或
// 显形「读件缺」（null/空数组+note），缺失项核销表在票面。D-629：业绩外自营盘，
// 零 NAV/业绩计入。
package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"o/internal/data/cmc"
)

// writeJSON 统一写面（读件缺=null 显形不造零）。
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// mountTrxAPI /api/trx/* 全功能读面（T-0110——独立页四区数据面+刷新踢；
// D-629 自营盘业绩外独挂）。
func mountTrxAPI(mux *http.ServeMux, w *trxWiring) {
	get := func(path string, h func(http.ResponseWriter, *http.Request)) {
		mux.HandleFunc("/api/trx/"+path, h)
	}
	// 账户快照+资源状态（nox TrxAccountRefresh 快照段对拍）。
	get("account", func(rw http.ResponseWriter, r *http.Request) {
		a, err := w.st.LastTRXAccount(r.Context())
		if err != nil {
			writeJSON(rw, map[string]any{"missing": "读件缺"})
			return
		}
		writeJSON(rw, map[string]any{
			"address": a.Address, "sr_address": a.SRAddress,
			"trx_balance":      float64(a.BalanceSun) / 1e6,
			"trx_allowance":    float64(a.RewardNumSun) / 1e6,
			"energy_delegated": float64(a.EnergyDelegated) / 1e6,
			"bandwidth_staked": float64(a.BandwidthStaked) / 1e6,
			"net_limit":        a.NetLimit, "net_used": a.NetUsed,
			"free_net_limit": a.FreeNetLimit, "free_net_used": a.FreeNetUsed,
			"energy_limit": a.EnergyLimit, "energy_used": a.EnergyUsed,
			"votes": a.VotesTotal, "unfrozen": a.Unfrozen,
			"bandwidth_status": bandwidthStatus(a.NetLimit, a.NetUsed, a.FreeNetLimit, a.FreeNetUsed),
			"fetched_at":       a.FetchedAt,
		})
	})
	// 刷新踢（开页/手按——读侧拉新 venue→库写，静默败不阻断渲染；非钱径）。
	get("account_refresh", func(rw http.ResponseWriter, r *http.Request) {
		if w.accts != nil {
			_, _ = w.accts.Poll(r.Context())
		}
		if w.apr != nil {
			w.apr.TriggerNow(r.Context())
		}
		a, err := w.st.LastTRXAccount(r.Context())
		if err != nil {
			writeJSON(rw, map[string]any{"missing": "读件缺"})
			return
		}
		writeJSON(rw, map[string]any{"address": a.Address,
			"trx_balance": float64(a.BalanceSun) / 1e6, "fetched_at": a.FetchedAt})
	})
	// txs 流水（limit/kind 过滤下推——前端零聚合令）。
	get("txs", func(rw http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		rows, err := w.st.ListTRXTxs(r.Context(), limit, r.URL.Query().Get("kind"))
		if err != nil {
			writeJSON(rw, map[string]any{"missing": "读件缺"})
			return
		}
		type tx struct {
			ID           int64     `json:"id"`
			TxID         string    `json:"tx_id"`
			Ts           time.Time `json:"ts"`
			Kind         string    `json:"kind"`
			AmountTrx    float64   `json:"amount_trx"`
			Counterparty string    `json:"counterparty"`
			CheatStatus  bool      `json:"cheat_status"`
		}
		out := make([]tx, 0, len(rows))
		for _, t := range rows {
			cp := ""
			if t.Counterparty != nil {
				cp = *t.Counterparty
			}
			out = append(out, tx{ID: t.ID, TxID: t.TxID, Ts: t.Ts, Kind: t.Kind,
				AmountTrx: float64(t.AmountSun) / 1e6, Counterparty: cp, CheatStatus: t.CheatStatus})
		}
		writeJSON(rw, out)
	})
	mountTrxYield(mux, w) // 收益雷达+日曲线（trx_yield_face.go）
	// 对冲结构（nox GetHedgeStructure 复合面对拍——平台口径 50%/80% CMC 标定）。
	get("hedge_structure", func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		trxAcc, hasTRX := w.st.LastTRXAccount(ctx)
		trxApr, hasApr := w.st.LastTRXVoteApr(ctx)
		ls := w.freshLive(ctx)
		cmcSum, hasSum := w.st.LastCMCAccountSummary(ctx)
		var baseline float64
		if p, err := w.st.LastCMCPositionByInstrument(ctx, "X-AFQUL"); err == nil {
			baseline = p.EntryPrice
		}
		legSpot := map[string]any{"label": "质押现货"}
		if hasTRX == nil {
			legSpot["qty_staked"] = float64(trxAcc.TronPower) / 1e6
			legSpot["qty_liquid"] = float64(trxAcc.BalanceSun) / 1e6
			legSpot["stake"] = map[string]float64{
				"energy":    float64(trxAcc.EnergyDelegated) / 1e6,
				"bandwidth": float64(trxAcc.BandwidthStaked) / 1e6}
		}
		if hasApr == nil {
			legSpot["vote_apr_current"] = trxApr.CurrentRate
			legSpot["vote_apr_max"] = trxApr.MaxRate
			legSpot["vote_apr_spread"] = trxApr.Spread
			if trxApr.MaxSr != nil {
				legSpot["vote_apr_max_sr"] = *trxApr.MaxSr
			}
		}
		legSpot["pnl_vs_base"] = -ls.PnL // 镜像近似（现货购价未知）
		legShort := map[string]any{"label": "CMC 空单"}
		if ls.Qty > 0 {
			legShort["qty"] = ls.Qty
		}
		legShort["pnl_vs_base"] = ls.PnL // 平台真值（业主令不反推）
		if ls.Lev > 0 {
			legShort["lev"] = int(ls.Lev)
		}
		if hasSum == nil {
			legShort["liq"] = cmcSum.CloseoutPrice
			legShort["dist_pct"] = cmcSum.DistanceToCloseoutPct
		}
		var netTrx, netUsd float64
		if hasTRX == nil && ls.Qty > 0 {
			netTrx = float64(trxAcc.BalanceSun)/1e6 + float64(trxAcc.TronPower)/1e6 - ls.Qty
			netUsd = netTrx * ls.Mid
		}
		accts := map[string]any{}
		var updatedAt time.Time
		if hasTRX == nil {
			accts["trx_balance"] = float64(trxAcc.BalanceSun) / 1e6
			accts["trx_allowance"] = float64(trxAcc.RewardNumSun) / 1e6
			accts["energy_delegated"] = float64(trxAcc.EnergyDelegated) / 1e6
			accts["votes"] = trxAcc.VotesTotal
			accts["net_limit"] = trxAcc.NetLimit
			accts["net_used"] = trxAcc.NetUsed
			accts["free_net_limit"] = trxAcc.FreeNetLimit
			accts["free_net_used"] = trxAcc.FreeNetUsed
			accts["energy_limit"] = trxAcc.EnergyLimit
			accts["energy_used"] = trxAcc.EnergyUsed
			accts["bandwidth_status"] = bandwidthStatus(trxAcc.NetLimit, trxAcc.NetUsed, trxAcc.FreeNetLimit, trxAcc.FreeNetUsed)
			updatedAt = trxAcc.FetchedAt
		}
		if ls.Cash > 0 || ls.Equity > 0 {
			accts["cmc_cash"], accts["cmc_equity"] = ls.Cash, ls.Equity
		} else if hasSum == nil {
			accts["cmc_cash"], accts["cmc_equity"] = cmcSum.TotalCashBalance, cmcSum.TotalAccountValue
		}
		if !updatedAt.IsZero() {
			accts["updated_at"] = updatedAt
		}
		margin := ls.Margin
		if margin <= 0 && hasSum == nil {
			margin = cmcSum.TotalEquityUsedInMargin
		}
		writeJSON(rw, map[string]any{
			"baseline_price": baseline, "mid": ls.Mid, "ask": ls.Ask,
			"leg_spot": legSpot, "leg_short": legShort,
			"net_exposure_trx": netTrx, "net_exposure_usd": netUsd,
			"carry": w.computeHedgeCarry(ctx, ls.Mid, legQtyStaked(legSpot)),
			"accts": accts, "ws_connected": w.cmcWsHealth != nil && w.cmcWsHealth(),
			"closeout_source": "platform_rule_50",
			"platform_alert_state": func() string {
				if hasSum == nil {
					return cmc.AlertsToState(cmcSum.Alerts)
				}
				return "normal"
			}(),
			"closeout_level_usd": margin * 0.5,
			"warning_level_usd":  margin * 0.8,
		})
	})
	// 对冲流水（trx/cmc 双源 UNION——cheat/vote/买卖成交剔除照 nox）。
	get("hedge_flow", func(rw http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		items, err := w.st.ListHedgeFlowPage(r.Context(), r.URL.Query().Get("source"), limit, offset)
		if err != nil {
			writeJSON(rw, map[string]any{"missing": "读件缺"})
			return
		}
		total, _ := w.st.ListHedgeFlowCount(r.Context(), r.URL.Query().Get("source"))
		type item struct {
			Source       string    `json:"source"`
			Ts           time.Time `json:"ts"`
			Kind         string    `json:"kind"`
			AmountValue  float64   `json:"amount_value"`
			AmountUnit   string    `json:"amount_unit"`
			Counterparty string    `json:"counterparty"`
			TxID         string    `json:"tx_id"`
		}
		out := make([]item, 0, len(items))
		for _, it := range items {
			out = append(out, item{it.Source, it.Ts, it.Kind, it.AmountValue, it.AmountUnit, it.Counterparty, it.TxID})
		}
		writeJSON(rw, map[string]any{"total": total, "txs": out})
	})
	// CMC 历史（fin_cmc_history 直透）。
	get("cmc_history", func(rw http.ResponseWriter, r *http.Request) {
		var raw json.RawMessage
		if err := w.pool.QueryRow(r.Context(), `SELECT COALESCE(json_agg(t),'[]') FROM
			(SELECT transaction_id,transaction_time,transaction_type,amount,balance,currency
			FROM fin_cmc_history ORDER BY transaction_time DESC LIMIT 30) t`).Scan(&raw); err != nil || raw == nil {
			raw = json.RawMessage("[]")
		}
		writeJSON(rw, raw)
	})
	// trx_cex 比价面（读件缺显形——扫描器未迁另票，核销表在案）。
	get("cex_scan", func(rw http.ResponseWriter, r *http.Request) {
		writeJSON(rw, map[string]any{"missing": "读件缺——cex 扫描器未迁（T-0110 核销：读面就位数据面缺）",
			"earn_aprs": []any{}, "spreads": []any{}, "fundings": []any{}, "balances": []any{}, "judgments": []any{}})
	})
}
