// fillsprobe —— T-0085 只读取证件：/trade/fills-history（SPOT 逐成交）+
// /account/bills（subType 1/2 现金行）按窗口拉取——历史卖腿补录的 venue 凭据源
// （okx-account skill 读面）。零写径。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

func main() {
	dsn := flag.String("dsn", "", "")
	keyf := flag.String("key", config.MasterKeyPath(), "")
	inst := flag.String("inst", "", "SPOT instId（如 BTC-USDT）")
	begin := flag.String("begin", "", "窗口起 RFC3339（UTC）")
	end := flag.String("end", "", "窗口止 RFC3339（UTC）")
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
	b, _ := time.Parse(time.RFC3339, *begin)
	e, _ := time.Parse(time.RFC3339, *end)
	fmt.Printf("== fills %s [%s → %s]\n", *inst, b.UTC(), e.UTC())
	url := fmt.Sprintf("/api/v5/trade/fills-history?instType=SPOT&instId=%s&begin=%d&end=%d&limit=100",
		*inst, b.UnixMilli(), e.UnixMilli())
	raw, err := cli.Get(ctx, url)
	die(err)
	pretty(raw)
	fmt.Println("== bills 同窗（subType 1=买 2=卖）")
	url2 := fmt.Sprintf("/api/v5/account/bills?instType=SPOT&instId=%s&begin=%d&end=%d&limit=100",
		*inst, b.UnixMilli(), e.UnixMilli())
	raw2, err := cli.Get(ctx, url2)
	die(err)
	pretty(raw2)
}

func pretty(raw json.RawMessage) {
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		fmt.Println("raw:", string(raw))
		return
	}
	for _, r := range rows {
		keys := []string{"tradeId", "ordId", "side", "fillSz", "fillPx", "fee", "ts", "billId", "subType", "balChg", "sz", "px"}
		fmt.Print("  ")
		for _, k := range keys {
			if v, ok := r[k]; ok && fmt.Sprint(v) != "" {
				fmt.Printf("%s=%v ", k, v)
			}
		}
		fmt.Println()
	}
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
