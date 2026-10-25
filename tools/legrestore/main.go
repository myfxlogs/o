// legrestore —— 事故运维件：被旁挂实例残币归集错搬的**活仓腿**归位
// funding 户→trading 户（T-0104 实爆修复：仓 23 ETH 腿 0.00998 被本机实例
// sweepResidual 当残币归口——本机库无仓位行故 FundCcy 盲，parked 至 savings/
// funding，致 hedge 裸腿+venue USDT 键虚失）。实读 avail 定额，禁快照。
// 用法：DB_DSN=… O_MASTER_KEY_FILE=… legrestore -ccy ETH -amt 0 [-dry=false]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

func main() {
	ccy := flag.String("ccy", "ETH", "归位币")
	amt := flag.Float64("amt", 0, "归位额（0=搬 funding 实余全额）")
	dry := flag.Bool("dry", true, "默认干跑")
	flag.Parse()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "db:", err)
		os.Exit(1)
	}
	defer pool.Close()
	mk, _ := config.ReadMasterKey()
	cred := okxCreds(ctx, pool, mk)
	cli := okxexec.NewClient(cred, false)
	if cli == nil {
		fmt.Fprintln(os.Stderr, "okx 凭据缺席——fail-closed")
		os.Exit(1)
	}
	// 三户实读：funding avail + trading avail + savings 持仓
	fAvail := fundingAvail(ctx, cli, *ccy)
	sAvail := savingsAvail(ctx, cli, *ccy)
	tAvail, _ := cli.TradingAvailBal(ctx, *ccy)
	mv := *amt
	if mv <= 0 {
		mv = fAvail
	}
	fmt.Printf("%s funding=%.8f savings=%.8f trading=%.8f → 拟归位 %.8f\n", *ccy, fAvail, sAvail, tAvail, mv)
	if *dry {
		fmt.Println("[dry] -dry=false 真归位")
		return
	}
	if mv <= 0 {
		fmt.Fprintln(os.Stderr, "额零——fail-closed 不搬")
		os.Exit(1)
	}
	if short := mv - fAvail; short > 1e-9 {
		if short > sAvail+1e-9 {
			fmt.Fprintf(os.Stderr, "funding 缺 %.8f 且 savings 不足——fail-closed 不赎\n", short)
			os.Exit(1)
		}
		fmt.Printf("funding 不足 → SavingsRedeem %.8f\n", short)
		if err := cli.SavingsRedeem(ctx, *ccy, short); err != nil {
			fmt.Fprintln(os.Stderr, "redeem 拒:", err)
			os.Exit(1)
		}
		fAvail = fundingAvail(ctx, cli, *ccy)
		fmt.Printf("赎回后 funding=%.8f\n", fAvail)
	}
	if mv > fAvail+1e-9 {
		fmt.Fprintln(os.Stderr, "赎回未补足额——fail-closed 不搬")
		os.Exit(1)
	}
	cltid, transid, err := cli.AssetTransfer(ctx, *ccy, mv, okxexec.AcctFunding, okxexec.AcctTrading)
	if err != nil {
		st, _ := cli.AssetTransferState(ctx, *ccy, cltid)
		fmt.Fprintf(os.Stderr, "transfer 歧态势: %v state=%s clTId=%s——禁盲重发\n", err, st, cltid)
		os.Exit(1)
	}
	fmt.Printf("transfer ok transId=%s clTId=%s\n", transid, cltid)
	tAvail2, _ := cli.TradingAvailBal(ctx, *ccy)
	fmt.Printf("终态对拍 trading=%.8f\n", tAvail2)
}

func fundingAvail(ctx context.Context, cli *okxexec.Client, ccy string) float64 {
	rows, err := cli.AssetBalances(ctx)
	if err != nil {
		return 0
	}
	for _, r := range rows {
		if r.Ccy == ccy {
			return okxexec.ParseF(r.AvailBal)
		}
	}
	return 0
}

// savingsAvail Earn 灵活持仓实读（amt+pendingAmt 同 earn 消费侧口径）。
func savingsAvail(ctx context.Context, cli *okxexec.Client, ccy string) float64 {
	raw, err := cli.SavingsBalance(ctx)
	if err != nil {
		return 0
	}
	var rows []struct {
		Ccy        string `json:"ccy"`
		Amt        string `json:"amt"`
		AvailBal   string `json:"availBal"`
		PendingAmt string `json:"pendingAmt"`
	}
	if json.Unmarshal(raw, &rows) != nil {
		return 0
	}
	for _, r := range rows {
		if r.Ccy == ccy {
			a := okxexec.ParseF(r.Amt) + okxexec.ParseF(r.PendingAmt)
			if a == 0 {
				a = okxexec.ParseF(r.AvailBal)
			}
			return a
		}
	}
	return 0
}

func okxCreds(ctx context.Context, pool *pgxpool.Pool, mk []byte) okxexec.Credentials {
	var cred okxexec.Credentials
	for k, dst := range map[string]*string{"okx.api_key": &cred.Key, "okx.api_secret": &cred.Secret, "okx.passphrase": &cred.Passphrase} {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			fmt.Fprintln(os.Stderr, k, err)
			os.Exit(1)
		}
		p, _ := cryptoutil.Open(mk, ct)
		*dst = p
	}
	return cred
}
