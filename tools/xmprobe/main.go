// xmprobe —— XM 户会话面硬验探针（O-37449⑤）：OpenSession 共享会话
// （零新 login——owner=汇市巡独占 ConnectEx，本件纯消费侧）→
// SymbolList 实读→候选族 SymbolSpec+GetQuote 全字段+SpecHash 落证。
// 用途：fin_symbol_dialect XM 行 venue_symbol 实读定版（禁猜名判据原点）。
// env：FIN_PG_DSN（app_config/app_secrets 读面——roster_rw 窄白名单在案）。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/mt5"
	"o/internal/platform/config"
)

type ev struct {
	VenueSymbol  string  `json:"venue_symbol"`
	ContractSize float64 `json:"contract_size"`
	Digits       int32   `json:"digits"`
	TickSize     float64 `json:"tick_size"`
	Points       float64 `json:"points"`
	ProfitCcy    string  `json:"profit_currency"`
	MarginCcy    string  `json:"margin_currency"`
	MinLots      float64 `json:"min_lots"`
	LotsStep     float64 `json:"lots_step"`
	MaxLots      float64 `json:"max_lots"`
	TradeMode    int     `json:"trade_mode"`
	Bid          float64 `json:"bid"`
	Ask          float64 `json:"ask"`
	SpreadBps    float64 `json:"spread_bps"`
	SpecHash     string  `json:"spec_hash"`
}

func main() {
	venue := flag.String("venue", "xmglobal", "名册 venue token（共享会话键 <venue>.session_token）")
	match := flag.String("match", "", "候选过滤子串（空=全量 SymbolList 只列名）")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dsn := os.Getenv("FIN_PG_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "FIN_PG_DSN 缺")
		os.Exit(2)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pool:", err)
		os.Exit(2)
	}
	defer pool.Close()
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "cfg load:", err)
		os.Exit(2)
	}
	sess, err := mt5.OpenSession(ctx, pool, cfg, *venue)
	if err != nil {
		fmt.Fprintln(os.Stderr, "OpenSession:", err)
		os.Exit(1)
	}
	defer sess.Close()

	syms, err := sess.SymbolList(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "SymbolList:", err)
		os.Exit(1)
	}
	sort.Strings(syms)
	fmt.Fprintf(os.Stderr, "[xmprobe] %s SymbolList=%d 只\n", *venue, len(syms))

	var cand []string
	for _, s := range syms {
		if *match == "" || strings.Contains(s, *match) {
			cand = append(cand, s)
		}
	}
	out := make([]ev, 0, len(cand))
	for _, s := range cand {
		si, sg, err := sess.SymbolSpec(ctx, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[xmprobe] %s SymbolSpec 败: %v\n", s, err)
			continue
		}
		e := ev{VenueSymbol: s, SpecHash: mt5.SpecHash(si, sg)}
		if si != nil {
			e.ContractSize, e.Digits, e.TickSize, e.Points = si.GetContractSize(), si.GetDigits(), si.GetTickSize(), si.GetPoints()
			e.ProfitCcy, e.MarginCcy = si.GetProfitCurrency(), si.GetMarginCurrency()
		}
		if sg != nil {
			e.MinLots, e.LotsStep, e.MaxLots, e.TradeMode = sg.GetMinLots(), sg.GetLotsStep(), sg.GetMaxLots(), int(sg.GetTradeMode())
		}
		if q, err := sess.GetQuoteOne(ctx, s); err == nil {
			e.Bid, e.Ask, e.SpreadBps = q.Bid, q.Ask, q.SpreadBPS()
		} else {
			fmt.Fprintf(os.Stderr, "[xmprobe] %s GetQuote 败: %v\n", s, err)
		}
		out = append(out, e)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
