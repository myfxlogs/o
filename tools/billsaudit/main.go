// billsaudit —— T-0055 对账差根查件：全谱 /account/bills（无 instType 过滤）逐行
// dump，与 fin_cashflow 已记行按 billId 对拍——缺口逐类归因（Earn 利息/现货费/
// 划转漏类）。只读零下单。用法：-dsn <dsn> -key <keyfile> [-days 90]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/trading/okxexec"
)

func main() {
	dsn := flag.String("dsn", "", "")
	keyf := flag.String("key", config.MasterKeyPath(), "")
	days := flag.Int("days", 90, "")
	flag.Parse()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	die(err)
	mk, err := config.ReadMasterKeyFrom(*keyf)
	die(err)
	var cred okxexec.Credentials
	for k, dst := range map[string]*string{"okx.api_key": &cred.Key, "okx.api_secret": &cred.Secret, "okx.passphrase": &cred.Passphrase} {
		var ct []byte
		die(pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct))
		p, err := cryptoutil.Open(mk, ct)
		die(err)
		*dst = p
	}
	cli := okxexec.NewClient(cred, false)

	// 全谱 bills 无 instType——逐页向旧推进，直到超过 -days 窗。
	type bill struct {
		BillID, Type, SubType, InstID, InstType, Ccy, Ts, BalChg, Fee, Pnl string
	}
	var all []bill
	after := ""
	for {
		url := "/api/v5/account/bills?limit=100" // begin 与 after 组合实测翻页空——3 月窗内全拉后本地按 ts 过滤
		if after != "" {
			url += "&after=" + after
		}
		raw, err := cli.Get(ctx, url)
		die(err)
		var rows []bill
		die(json.Unmarshal(raw, &rows))
		fmt.Fprintf(os.Stderr, "page after=%s n=%d\n", after, len(rows))
		if len(rows) == 0 {
			break
		}
		all = append(all, rows...)
		after = rows[len(rows)-1].BillID
		if len(rows) < 100 {
			break
		}
	}
	// 已入账 billId 集（fin_venue_bill）。
	ingested := map[string]bool{}
	rows2, err := pool.Query(ctx, "SELECT bill_id FROM fin_venue_bill")
	die(err)
	for rows2.Next() {
		var b string
		die(rows2.Scan(&b))
		ingested[b] = true
	}
	rows2.Close()
	// 逐类汇总：type/subType 分桶（balChg/fee 聚合+是否已入账）。
	type agg struct {
		n, ing int
		bal    float64
	}
	m := map[string]*agg{}
	for _, b := range all {
		k := "type=" + b.Type + " sub=" + b.SubType + " " + b.Ccy
		a := m[k]
		if a == nil {
			a = &agg{}
			m[k] = a
		}
		a.n++
		if ingested[b.BillID] {
			a.ing++
		}
		a.bal += pf(b.BalChg)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// ts 窗过滤（bills 端点 3 月窗；-days 本地截）。
	cut := time.Now().AddDate(0, 0, -*days).UnixMilli()
	filt := all[:0]
	for _, b := range all {
		if ts, _ := strconv.ParseInt(b.Ts, 10, 64); ts >= cut {
			filt = append(filt, b)
		}
	}
	all = filt
	fmt.Printf("总账单 %d 行（%d 天窗）；已入账 %d\n", len(all), *days, len(ingested))
	for _, k := range keys {
		a := m[k]
		fmt.Printf("%-38s n=%-4d 已记=%-4d ΣbalChg=%.4f\n", k, a.n, a.ing, a.bal)
	}
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func pf(s string) float64 { f, _ := strconv.ParseFloat(s, 64); return f }
