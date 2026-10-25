// money_card.go —— 钱在哪卡面件（T-0086 ②三分卡：行组装/在册集/遗留资产
// 命名/amt 估值——自主文件拆分，行数棘轮）。
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/gen/dashboardv1"
	"o/internal/ledger"
)

// moneyRow 组装（T-0086 ②：分区+币种+遗留资产标注——非在册币种涨跌不报警）。
func moneyRow(section, ccy, label string, amt float64, legacy bool) *dashboardv1.MoneyRow {
	return &dashboardv1.MoneyRow{Label: label, Amount: amt, Section: section, Ccy: ccy, Legacy: legacy}
}

// ledgeredCcy 在册币种集（T-0090：fin_cashflow 有账 ∪ 已成交腿基币——
// 现货腿底层币非遗留；ledger.LedgeredCcy 唯一真源按 venue 过滤——身份豁免判据）。
func (c coreService) ledgeredCcy(ctx context.Context, venue string) map[string]bool {
	out := map[string]bool{}
	set, err := ledger.LedgeredCcy(ctx, c.pool)
	if err != nil {
		return out
	}
	for k := range set {
		if k[0] == venue {
			out[k[1]] = true
		}
	}
	return out
}

// legacyName 遗留资产人类名（常见杂币——未知币种回退代码本身）。
var legacyName = map[string]string{
	"AVAX": "Avalanche", "COMP": "Compound", "NFT": "NFT 凭证",
	"DOGE": "Dogecoin", "FIL": "Filecoin", "BTC": "Bitcoin", "ETH": "Ethereum",
	"PENGU": "Pudgy Penguins", "MEW": "cat in a dogs world",
}

// Overview 总览四联活读（所见即所得——零快照零缓存；数据出生证=查询时戳）。
// valCcy 代币数→USD 估值（T-0086 ① amt估值）：USDT/USD 直入；其余=md_ticks
// 最新价×量（无 tick=0——卡面仍展示，NAV 不虚计 51M NFT 级原生数）。
func valCcy(ctx context.Context, pool *pgxpool.Pool, ccy string, units float64) float64 {
	if ccy == "USDT" || ccy == "USD" {
		return units
	}
	var last *float64
	_ = pool.QueryRow(ctx, `SELECT last FROM md_ticks WHERE inst_id=$1 ORDER BY ts DESC LIMIT 1`,
		ccy+"-USDT").Scan(&last)
	if last == nil {
		return 0
	}
	return units * *last
}

// ccyName 币种人类名（遗留资产卡可读性）。
func ccyName(ccy string) string {
	if n, ok := legacyName[ccy]; ok {
		return n
	}
	return ccy
}
