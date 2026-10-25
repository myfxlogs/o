// rescuespot —— 事故运维件：LEGGED 残仓人工救助。决策时点现读 /account/balance
// 卖 BTC-USDT 现货残仓（availBal 实读，禁快照）。执行后打印 venue 终态对拍。
// 用法：DB_DSN=… O_MASTER_KEY_FILE=… rescuespot [-dry]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

func main() {
	dry := flag.Bool("dry", false, "干跑——只读残仓不下单")
	dump := flag.Bool("dump", false, "只打 venue 余额/成交明细（取证）")
	flag.Parse()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	mk, err := config.ReadMasterKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
	cli := okxexec.NewClient(cred, false)

	if *dump {
		raw, err := cli.Get(ctx, "/api/v5/account/balance")
		if err != nil {
			fmt.Fprintln(os.Stderr, "balance:", err)
			os.Exit(1)
		}
		fmt.Println("== /account/balance ==")
		fmt.Println(string(raw))
		if sv, err := cli.SavingsBalance(ctx); err == nil {
			fmt.Println("== /finance/savings/balance ==")
			fmt.Println(string(sv))
		}
		for _, iid := range []string{"BTC-USDT-SWAP", "BTC-USDT"} {
			if f, err := cli.Get(ctx, "/api/v5/trade/fills-history?instType=SPOT&instId="+iid); err == nil {
				fmt.Printf("== fills %s ==\n%s\n", iid, string(f))
			}
		}
		if f, err := cli.Get(ctx, "/api/v5/trade/fills-history?instType=SWAP&instId=BTC-USDT-SWAP"); err == nil {
			fmt.Printf("== fills SWAP ==\n%s\n", string(f))
		}
		if b, err := cli.Get(ctx, "/api/v5/account/bills?type=8"); err == nil {
			fmt.Printf("== bills funding ==\n%s\n", string(b))
		}
		return
	}

	// 决策时点现读：交易户 BTC availBal + SWAP 持仓残量（D-0008——禁快照）。
	raw, err := cli.Get(ctx, "/api/v5/account/balance")
	if err != nil {
		fmt.Fprintln(os.Stderr, "balance:", err)
		os.Exit(1)
	}
	var rows []struct {
		Details []struct {
			Ccy      string `json:"ccy"`
			AvailBal string `json:"availBal"`
		} `json:"details"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		fmt.Fprintln(os.Stderr, "balance parse:", err)
		os.Exit(1)
	}
	var btcAvail float64
	for _, d := range rows[0].Details {
		if d.Ccy == "BTC" {
			btcAvail = okxexec.ParseF(d.AvailBal)
		}
	}
	fmt.Printf("venue 实读: BTC availBal=%.11f\n", btcAvail)

	praw, err := cli.Get(ctx, "/api/v5/account/positions?instId=BTC-USDT-SWAP")
	if err == nil {
		var pos []struct {
			Data []struct {
				Pos string `json:"pos"`
			} `json:"data"`
		}
		if json.Unmarshal(praw, &pos) == nil && len(pos) > 0 {
			for _, p := range pos[0].Data {
				fmt.Printf("venue 实读: SWAP pos=%s\n", p.Pos)
			}
		}
	}

	if btcAvail <= 0 {
		fmt.Println("无现货残仓——退出")
		return
	}
	if *dry {
		fmt.Println("dry——不下单")
		return
	}
	m, err := cli.InstrumentMeta(ctx, "SPOT", "BTC-USDT")
	if err != nil {
		fmt.Fprintln(os.Stderr, "instrument:", err)
		os.Exit(1)
	}
	sz := math.Floor(btcAvail/m.LotSz) * m.LotSz
	if sz < m.MinSz {
		fmt.Printf("残仓 %.11f < minSz %.8f（尘埃，无需处置）\n", sz, m.MinSz)
		return
	}
	clID := fmt.Sprintf("phresc%d", time.Now().Unix()%1000000)
	resp, err := cli.PlaceOrder(ctx, okxexec.OrderParams{
		InstID: "BTC-USDT", TdMode: "cash", Side: "sell", OrdType: "market",
		Sz: strconv.FormatFloat(sz, 'f', 11, 64), ClOrdID: clID, TgtCcy: "base_ccy"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "place:", err)
		os.Exit(1)
	}
	fmt.Printf("sell submitted clOrdId=%s sz=%.11f resp=%s\n", clID, sz, string(resp))
}
