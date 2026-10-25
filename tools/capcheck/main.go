// capcheck —— 能力账致密化机检（T-0119）：skills/okx-*/mt5-*/cmc prose 面
// 清单 vs fin_capability_surface 种子 SQL 行对账——面缺席种子=FAIL 退回。
// 面清单=keyset 硬钉（skills 头注面名——prose 面名钉表即锚）。
package main

import (
	"fmt"
	"os"
	"strings"
)

// want skills prose 钦定面集（逐面源=skills/<venue>-*/SKILL.md 节名钉）。
var want = []string{
	"okx/market-tickers", "okx/market-books5", "okx/market-funding-rate",
	"okx/market-mark-price", "okx/market-index-tickers", "okx/market-instruments",
	"okx/priv-account", "okx/priv-balance-and-position",
	"okx/account-balance", "okx/account-positions", "okx/account-bills",
	"okx/asset-balances", "okx/asset-transfer", "okx/asset-transfer-state",
	"okx/savings-balance", "okx/savings-rate", "okx/savings-purchase", "okx/savings-redeem",
	"okx/trade-order", "okx/trade-order-precheck", "okx/trade-cancel",
	"mt5/account", "mt5/account-summary", "mt5/symbols", "mt5/quote",
	"mt5/price-history", "mt5/opened-orders", "mt5/order-history",
	"mt5/stream-tick", "mt5/required-margin",
	"cmc/prices", "cmc/account-summary", "cmc/positions", "cmc/history", "cmc/ws-stream",
}

func main() {
	b, err := os.ReadFile("migrations/0049_capability_dense.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "capcheck: 种子件缺席: %v\n", err)
		os.Exit(1)
	}
	s := string(b)
	var miss []string
	for _, w := range want {
		parts := strings.SplitN(w, "/", 2)
		if !strings.Contains(s, "('"+parts[0]+"','"+parts[1]+"'") {
			miss = append(miss, w)
		}
	}
	if len(miss) > 0 {
		fmt.Fprintf(os.Stderr, "capcheck: %d 面种子缺席——%v\n", len(miss), miss)
		os.Exit(1)
	}
	fmt.Printf("capcheck: %d 面种子齐（0049）\n", len(want))
}
