// swapsurvey —— T-0099 MT5 品种全集+swap/carry 调查探针（只读面——SymbolList/
// SymbolParams/RequiredMargin/GetQuote 族；mutating 面零触）。凭据自
// app_secrets 密文列解（OpenSession 同径）；输出 JSON 落 -out 档。
// 用法：O_MASTER_KEY_FILE=secrets/o_master_key go run ./tools/swapsurvey -dsn <dsn> -out docs/能力账/x.json
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/platform/config"
)

func main() {
	dsn := flag.String("dsn", "", "")
	out := flag.String("out", "", "")
	perSym := flag.Duration("per-sym", 150*time.Millisecond, "SymbolParams 节拍（限频礼貌）")
	flag.Parse()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	die(err)
	cfg := config.New(pool)
	die(cfg.Load(ctx))
	cli, err := mt5.OpenSession(ctx, pool, cfg)
	die(err)
	defer func() { cli.Disconnect(ctx); cli.Close() }()

	// 户头面（login/leverage/equity——保证金匹配分母）。
	login, lev, ccy, err := cli.AccountFacts(ctx)
	die(err)
	eq, fm, bal, err := cli.AccountSummary(ctx)
	die(err)

	sl, err := cli.MT5().SymbolList(cli.Authed(ctx), &mt5pb.SymbolListRequest{Id: cli.Token()})
	die(err)
	if sl.Error != nil {
		die(fmt.Errorf("SymbolList apierr: %s", sl.Error.Message))
	}
	syms := sl.Result
	log.Printf("符号全集 %d 件（login=%d lev=%d ccy=%s eq=%.2f fm=%.2f bal=%.2f）",
		len(syms), login, lev, ccy, eq, fm, bal)

	var rows []map[string]any
	for i, s := range syms {
		row := map[string]any{"symbol": s}
		r, err := cli.MT5().SymbolParams(cli.Authed(ctx),
			&mt5pb.SymbolParamsRequest{Id: cli.Token(), Symbol: s})
		if err != nil || r.Error != nil || r.Result == nil {
			row["err"] = fmt.Sprint(err, r.GetError())
			rows = append(rows, row)
			continue
		}
		si, sg := r.Result.SymbolInfo, r.Result.SymbolGroup
		row["desc"] = si.GetDescription()
		row["digits"] = si.GetDigits()
		row["spread_pts"] = si.GetSpread()
		row["tick_size"] = si.GetTickSize()
		row["tick_value"] = si.GetTickValue()
		row["contract_size"] = si.GetContractSize()
		row["margin_ccy"] = si.GetMarginCurrency()
		row["profit_ccy"] = si.GetProfitCurrency()
		row["trade_mode"] = sg.GetTradeMode().String()
		row["min_lots"] = sg.GetMinLots()
		row["lots_step"] = sg.GetLotsStep()
		row["swap_type"] = sg.GetSwapType().String()
		row["swap_long"] = sg.GetSwapLong()
		row["swap_short"] = sg.GetSwapShort()
		row["triple_swap_day"] = sg.GetThreeDaysSwap().String()
		row["init_margin"] = sg.GetInitialMargin()
		row["mntn_margin"] = sg.GetMaintenanceMargin()
		if q, err := cli.MT5().GetQuote(cli.Authed(ctx),
			&mt5pb.GetQuoteRequest{Id: cli.Token(), Symbol: s}); err == nil && q.Error == nil && q.Result != nil {
			row["bid"], row["ask"] = q.Result.Bid, q.Result.Ask
		}
		// 1 lot 保证金试算（buy 向——err 照录不虚不猜）。
		buy := mt5pb.DealType_DealType_DealBuy
		if mr, err := cli.MT5().RequiredMargin(cli.Authed(ctx),
			&mt5pb.RequiredMarginRequest{Id: cli.Token(), Symbol: s, Lots: 1, Type: &buy}); err == nil && mr.Error == nil {
			row["margin_1lot"] = mr.Result
		}
		rows = append(rows, row)
		if i%20 == 0 {
			log.Printf("… %d/%d", i, len(syms))
		}
		time.Sleep(*perSym) // sched-exempt-ok：一次性探针 venue 限频礼貌退避（非周期环）
	}
	doc := map[string]any{"ts": time.Now().UTC().Format(time.RFC3339),
		"account": map[string]any{"login": login, "leverage": lev, "ccy": ccy,
			"equity": eq, "free_margin": fm, "balance": bal},
		"symbols": rows}
	b, _ := json.MarshalIndent(doc, "", "  ")
	if *out != "" {
		die(os.WriteFile(*out, b, 0o644))
		log.Printf("落档 %s（%d 符号）", *out, len(rows))
	} else {
		fmt.Println(string(b))
	}
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
}
