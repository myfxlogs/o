// trx_yield_face.go —— 收益雷达+日曲线读面（T-0110 nox service_trx_yield_*
// 移植——后端算好前端零计算；trx_faces.go 拆件行数棘轮）。
package main

import (
	"net/http"
	"strconv"
	"time"
)

// mountTrxYield 收益区两读面（radar/curve——mountTrxAPI 域内调用）。
func mountTrxYield(mux *http.ServeMux, w *trxWiring) {
	get := func(path string, h func(http.ResponseWriter, *http.Request)) {
		mux.HandleFunc("/api/trx/"+path, h)
	}
	get("yield_radar", func(rw http.ResponseWriter, r *http.Request) {
		apr7d, s7 := w.energyRentalAPR(r.Context(), 7)
		apr30d, s30 := w.energyRentalAPR(r.Context(), 30)
		var voteCur, voteMax, voteSpread float64
		var voteMaxSr string
		var updatedAt time.Time
		if v, err := w.st.LastTRXVoteApr(r.Context()); err == nil {
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
		if avg, ok, err := w.st.AvgTRXVoteAprByRange(r.Context(), time.Now().UTC().AddDate(0, 0, -30), time.Now().UTC()); err == nil && ok {
			vote30d = avg
		}
		floorPct, _ := w.cfg.GetFloat("trx.floor_pct")
		// 9-9 过渡期护栏：7d 窗内在押变化>20% → floor 判定无效。
		var transition bool
		if minE, maxE, err := w.st.EnergyDelegatedRangeInWindow(r.Context(),
			time.Now().UTC().AddDate(0, 0, -7), time.Now().UTC()); err == nil && minE > 0 {
			transition = maxE > minE*12/10
		}
		if updatedAt.IsZero() {
			updatedAt = time.Now().UTC()
		}
		writeJSON(rw, map[string]any{
			"energy_apr_7d": apr7d, "energy_apr_30d": apr30d,
			"vote_apr_current": voteCur, "vote_apr_30d_avg": vote30d,
			"floor_pct": floorPct, "sample_days_7d": s7, "sample_days_30d": s30,
			"transition": transition, "vote_apr_max": voteMax,
			"vote_apr_max_sr": voteMaxSr, "vote_apr_spread": voteSpread,
			"updated_at": updatedAt,
		})
	})
	// 净收益率日曲线（nox GetTrxYieldCurve 对拍——分母=$36.5K 池口径 AF4）。
	get("yield_curve", func(rw http.ResponseWriter, r *http.Request) {
		days, _ := strconv.Atoi(r.URL.Query().Get("days"))
		if days <= 0 {
			days = 30
		}
		now := time.Now().UTC()
		ys, err := w.st.ListTRXYieldByDay(r.Context(), now.AddDate(0, 0, -days), now)
		if err != nil {
			writeJSON(rw, map[string]any{"missing": "读件缺"})
			return
		}
		var currentPrice float64
		if mid, _, err := w.st.LastCMCPriceMidWithTime(r.Context(), "TRXUSDT"); err == nil {
			currentPrice = mid
		}
		const poolPrincipal = 36500.0
		type pt struct {
			Day           time.Time `json:"day"`
			EnergyTrx     float64   `json:"energy_trx"`
			VotingTrx     float64   `json:"voting_trx"`
			Price         float64   `json:"price"`
			DailyTotalUsd float64   `json:"daily_total_usd"`
		}
		points := make([]pt, 0, len(ys))
		var sum7, sum30 float64
		var c7, c30 int
		for _, yd := range ys {
			price := currentPrice
			if yd.Price != nil {
				price = *yd.Price
			}
			total := (yd.EnergyTrx + yd.VotingTrx) * price
			points = append(points, pt{Day: yd.Day, EnergyTrx: yd.EnergyTrx,
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
		rentalSun, _ := w.st.SumTRXTxRentalAll(r.Context())
		voteSun, _ := w.st.SumTRXTxVoteClaimAll(r.Context())
		cumTrx := float64(rentalSun+voteSun) / 1e6
		writeJSON(rw, map[string]any{
			"points": points, "annualized_7d": ann7, "annualized_30d": ann30,
			"current_price": currentPrice, "pool_principal_usd": poolPrincipal,
			"sample_days_7d": c7, "sample_days_30d": c30,
			"cumulative_unhedged_income_trx": cumTrx,
			"cumulative_unhedged_income_usd": cumTrx * currentPrice,
		})
	})
}
