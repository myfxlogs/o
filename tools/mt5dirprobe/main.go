// mt5dirprobe —— O-35162 一次性取证探针：共享会话消费腿 SymbolList 全目录
// →GetQuoteMany 点差→SymbolParamsMany tradeable 过滤→品类分组印出。
// 消费腿纪律：SetSharedToken+CheckAlive，永不 ConnectEx（双 login 互撞坑）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
)

type sess struct {
	Addr  string `json:"addr"`
	Token string `json:"token"`
}

func chunk(s []string, n int) [][]string {
	var out [][]string
	for i := 0; i < len(s); i += n {
		j := i + n
		if j > len(s) {
			j = len(s)
		}
		out = append(out, s[i:j])
	}
	return out
}

func main() {
	dsn := os.Getenv("FIN_PG_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "FIN_PG_DSN 缺")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pg:", err)
		os.Exit(1)
	}
	defer pool.Close()
	var raw string
	if err := pool.QueryRow(ctx, `SELECT value FROM app_config WHERE key='mt5.session_token'`).Scan(&raw); err != nil {
		fmt.Fprintln(os.Stderr, "session row:", err)
		os.Exit(1)
	}
	var ss sess
	if err := json.Unmarshal([]byte(raw), &ss); err != nil || ss.Addr == "" || ss.Token == "" {
		fmt.Fprintln(os.Stderr, "session 行坏:", err, raw[:min(60, len(raw))])
		os.Exit(1)
	}
	cl, err := mt5.Dial(ss.Addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial:", err)
		os.Exit(1)
	}
	defer cl.Close()
	cl.SetSharedToken(ss.Token)
	if err := cl.CheckAlive(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "CheckAlive:", err)
		os.Exit(1)
	}
	fmt.Println("== session alive, addr=", ss.Addr)

	syms, err := cl.SymbolList(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "SymbolList:", err)
		os.Exit(1)
	}
	sort.Strings(syms)
	fmt.Printf("== SymbolList n=%d\n", len(syms))

	// GetQuoteMany 分批（bid/ask→mid+spread%）
	type qd struct {
		bid, ask, mid, spreadPct float64
		has                      bool
	}
	quotes := map[string]qd{}
	for _, b := range chunk(syms, 100) {
		r, err := cl.MT5().GetQuoteMany(cl.Authed(ctx), &mt5pb.GetQuoteManyRequest{Id: cl.Token(), Symbols: b})
		if err != nil {
			fmt.Fprintln(os.Stderr, "GetQuoteMany:", err)
			continue
		}
		for _, q := range r.GetResult() {
			bid, ask := q.GetBid(), q.GetAsk()
			if bid <= 0 || ask <= 0 {
				continue
			}
			mid := (bid + ask) / 2
			quotes[q.GetSymbol()] = qd{bid, ask, mid, (ask - bid) / mid * 100, true}
		}
		time.Sleep(300 * time.Millisecond) // sched-exempt-ok：限频步进（RPC 批间限速——klinePace 同款）
	}
	fmt.Printf("== quotes n=%d\n", len(quotes))

	// SymbolParamsMany 分批（trade_mode/group/currency）
	type pd struct {
		tradeMode int32
		group     string
		ccy       string
		desc      string
		points    float64
	}
	params := map[string]pd{}
	for _, b := range chunk(syms, 100) {
		r, err := cl.MT5().SymbolParamsMany(cl.Authed(ctx), &mt5pb.SymbolParamsManyRequest{Id: cl.Token(), Symbol: b})
		if err != nil {
			fmt.Fprintln(os.Stderr, "SymbolParamsMany:", err)
			continue
		}
		for _, p := range r.GetResult() {
			if p == nil {
				continue
			}
			var ccy, desc string
			var pts float64
			if si := p.GetSymbolInfo(); si != nil {
				ccy, desc, pts = si.GetCurrency(), si.GetDescription(), si.GetPoints()
			}
			params[p.GetSymbol()] = pd{int32(p.GetSymbolGroup().GetTradeMode()),
				p.GetSymbolGroup().GetGroupName(), ccy, desc, pts}
		}
		time.Sleep(300 * time.Millisecond) // sched-exempt-ok：限频步进（RPC 批间限速——klinePace 同款）
	}
	fmt.Printf("== params n=%d\n", len(params))

	// 过滤：tradeable（非 Disabled/CloseOnly）+ 有报价 + 点差≤阈
	type row struct {
		sym       string
		grp       string
		ccy       string
		desc      string
		tm        int32
		mid       float64
		spreadPct float64
	}
	var pass, fail []row
	for _, s := range syms {
		p, ok := params[s]
		if !ok {
			continue
		}
		q, qok := quotes[s]
		r := row{s, p.group, p.ccy, p.desc, p.tradeMode, q.mid, q.spreadPct}
		if qok && p.tradeMode != 0 && p.tradeMode != 3 && q.spreadPct <= 1.0 {
			pass = append(pass, r)
		} else {
			fail = append(fail, r)
		}
	}
	fmt.Printf("== pass(tradeable+quote+spread<=1%%) n=%d ; fail n=%d\n", len(pass), len(fail))

	// SOLUSDm 等在册旧符定点判词
	for _, s := range []string{"SOLUSDm", "BTCUSDm", "ETHUSDm"} {
		p, pok := params[s]
		q, qok := quotes[s]
		fmt.Printf("PIN %-9s params=%v quote=%v(pok=%v qok=%v)\n", s, pok, qok, pok, qok)
		if pok {
			fmt.Printf("    tm=%d grp=%s sprd=%.3f%%\n", p.tradeMode, p.group, q.spreadPct)
		}
	}
	// 全过滤集 JSON（bridge.mt5.symbols 候选——官名实读序）
	passNames := make([]string, 0, len(pass))
	for _, r := range pass {
		passNames = append(passNames, r.sym)
	}
	sort.Strings(passNames)
	jb, _ := json.Marshal(passNames)
	fmt.Printf("PASSJSON %s\n", jb)

	// 品类分组（官名实读——名字面分类仅分组展示，集成员=过滤集全集）
	cats := map[string][]row{}
	for _, r := range pass {
		c := "other"
		switch {
		case strings.Contains(r.sym, "XAU") || strings.Contains(r.sym, "XAG") ||
			strings.Contains(r.sym, "XPT") || strings.Contains(r.sym, "XPD") ||
			strings.Contains(strings.ToUpper(r.desc), "GOLD") || strings.Contains(strings.ToUpper(r.desc), "SILVER"):
			c = "metals"
		case strings.Contains(r.sym, "XTI") || strings.Contains(r.sym, "XBR") ||
			strings.Contains(r.sym, "XNG") || strings.Contains(r.sym, "OIL") ||
			strings.Contains(strings.ToUpper(r.desc), "OIL") || strings.Contains(strings.ToUpper(r.desc), "CRUDE"):
			c = "energy"
		case strings.Contains(r.sym, "BTC") || strings.Contains(r.sym, "ETH") ||
			strings.Contains(r.sym, "SOL") || strings.Contains(r.sym, "XRP") ||
			strings.Contains(r.sym, "LTC") || strings.Contains(r.sym, "DOGE") ||
			strings.Contains(r.sym, "ADA") || strings.Contains(r.sym, "DOT") ||
			strings.Contains(r.sym, "BNB") || strings.Contains(r.sym, "LINK") ||
			strings.Contains(r.sym, "AVAX") || strings.Contains(r.sym, "MATIC") ||
			strings.Contains(strings.ToUpper(r.desc), "BITCOIN") || strings.Contains(strings.ToUpper(r.desc), "ETHEREUM"):
			c = "crypto"
		case strings.Contains(r.sym, "US30") || strings.Contains(r.sym, "USTEC") ||
			strings.Contains(r.sym, "US500") || strings.Contains(r.sym, "NAS") ||
			strings.Contains(r.sym, "SPX") || strings.Contains(r.sym, "DE30") ||
			strings.Contains(r.sym, "UK100") || strings.Contains(r.sym, "JP225") ||
			strings.Contains(r.sym, "AUS200") || strings.Contains(r.sym, "HK50") ||
			strings.Contains(strings.ToUpper(r.desc), "DOW") || strings.Contains(strings.ToUpper(r.desc), "NASDAQ") ||
			strings.Contains(strings.ToUpper(r.desc), "S&P"):
			c = "index"
		case strings.Contains(r.ccy, "JPY") && !strings.Contains(r.sym, "XAU") && !strings.Contains(r.sym, "XAG"):
			c = "fx-jpy"
		case len(r.sym) >= 6 && math.IsNaN(0):
			c = "fx-other"
		}
		cats[c] = append(cats[c], r)
	}
	for _, c := range []string{"metals", "energy", "index", "crypto", "fx-jpy", "fx-other", "other"} {
		rows := cats[c]
		if len(rows) == 0 {
			continue
		}
		fmt.Printf("\n## %s n=%d\n", c, len(rows))
		for _, r := range rows {
			fmt.Printf("  %-14s grp=%-22s ccy=%-4s tm=%d mid=%.5g sprd=%.3f%%  %s\n",
				r.sym, r.grp, r.ccy, r.tm, r.mid, r.spreadPct, r.desc)
		}
	}
	// 失败面摘要（为什么被滤掉——取证用）
	fmt.Println("\n## filtered-out sample (tm=0/3 or no quote or spread>1%)")
	shown := 0
	for _, r := range fail {
		if shown >= 25 {
			break
		}
		fmt.Printf("  %-14s grp=%-22s tm=%d sprd=%.3f%% %s\n", r.sym, r.grp, r.tm, r.spreadPct, r.desc)
		shown++
	}
}
