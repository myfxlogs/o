// trx_rollup.go —— TRX 日收益终局 rollup（T-0110 nox service_trx.go 移植，
// D-211 SSOT §5）：fin_trx_* 汇总当日收入写 position_daily_income(source=auto)。
// energy=rental_income 日合计；voting=相邻快照 reward_num 正增量（D-295：claim
// 物化计量——max(0,·) 天然滤 claim 清零）；overnight=0（CMC 域剔除 D-253）。
// 幂等=ON CONFLICT(day)（0048 唯一约束）——随采集拍补当日，零新环。
package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

// trxRollup 当日 rollup（无账户快照/无 tx 数据=静默缺——冷启动如实零行）。
func trxRollup(ctx context.Context, w *trxWiring) {
	day := time.Now().UTC().Truncate(24 * time.Hour)

	// energy_trx=当日 rental_income 合计（sun→TRX）。
	rentalSun, err := w.st.SumTRXTxRentalByRange(ctx, day, day.Add(24*time.Hour))
	if err != nil {
		log.Printf("[trx-rollup] energy 合计败: %v", err)
		return
	}
	energyTrx := float64(rentalSun) / 1e6

	// voting_trx=Σmax(0,Δreward_num) 相邻快照正增量。
	accounts, err := w.st.ListTRXAccountByDay(ctx, day)
	if err != nil {
		log.Printf("[trx-rollup] voting 增量读败: %v", err)
		return
	}
	var votingTrx float64
	for i := 1; i < len(accounts); i++ {
		if d := accounts[i].RewardNumSun - accounts[i-1].RewardNumSun; d > 0 {
			votingTrx += float64(d) / 1e6
		}
	}

	// price 快照（fin_cmc_prices TRXUSDT mid 最新——D-359 价锚 O 迁 cmc 面）。
	var price *float64
	if mid, _, err := w.st.LastCMCPriceMidWithTime(ctx, "TRXUSDT"); err == nil && mid > 0 {
		price = &mid
	}

	// pnl 快照（空单浮亏=(entry−price)×qty——复用手工语义）。
	var pnl *float64
	if price != nil {
		if p, err := w.st.LastCMCPositionByInstrument(ctx, "X-AFQUL"); err == nil &&
			p.EntryPrice > 0 && p.Quantity > 0 {
			v := (p.EntryPrice - *price) * p.Quantity
			pnl = &v
		}
	}

	// D-253：overnight 成分随 CMC 域剔除——interest_usd 如实置 0。
	note := fmt.Sprintf("auto: energy=租金 %v TRX / voting=Σ正增量 %v TRX / overnight=0（CMC 域剔除）", energyTrx, votingTrx)
	if err := w.st.AddDailyIncomeAuto(ctx, day, energyTrx, votingTrx, 0, price, pnl, note); err != nil {
		log.Printf("[trx-rollup] 写行败: %v", err)
	}
}
