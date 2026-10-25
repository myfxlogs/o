// money_card.go —— 钱在哪卡面件（T-0086 ②三分卡起版；T-0095 v2 口径收紧+
// T-0096 venue 分域后=venue→分区→行三级：行组装/在册集/amt 估值）。
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/gen/dashboardv1"
	"o/internal/ledger"
)

// moneyRow 组装（T-0095 v2：零遗留概念——集合外托管行不组装不返不渲染；
// T-0096：venue 一级+note 旁注〔读件缺等如实标注〕）。
func moneyRow(venue, section, ccy, label string, amt, usd float64) *dashboardv1.MoneyRow {
	return &dashboardv1.MoneyRow{Label: label, Amount: amt, Usd: usd,
		Venue: venue, Section: section, Ccy: ccy}
}

// fundCcy 基金的钱币种集·venue 域过滤（T-0095 v2 在册集=USDT∪open 腿基币，
// ledger.FundCcy 唯一真源——历史腿已平的币=托管尘非基金钱）。
func (c coreService) fundCcy(ctx context.Context, venue string) map[string]bool {
	out := map[string]bool{}
	set, err := ledger.FundCcy(ctx, c.pool)
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

// ccyName 币种人类名（行可读性——未知币种回退代码本身）。
var ccyName = map[string]string{
	"AVAX": "Avalanche", "COMP": "Compound", "NFT": "NFT 凭证",
	"DOGE": "Dogecoin", "FIL": "Filecoin", "BTC": "Bitcoin", "ETH": "Ethereum",
	"PENGU": "Pudgy Penguins", "MEW": "cat in a dogs world",
}

func nameOf(ccy string) string {
	if n, ok := ccyName[ccy]; ok {
		return n
	}
	return ccy
}

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
