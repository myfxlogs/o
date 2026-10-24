// poschk —— 只读仓位/余额快查（T-0056 运维件）。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/trading/okxexec"
)

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	d(err)
	mk, err := config.ReadMasterKey()
	d(err)
	var cred okxexec.Credentials
	for k, dst := range map[string]*string{"okx.api_key": &cred.Key, "okx.api_secret": &cred.Secret, "okx.passphrase": &cred.Passphrase} {
		var ct []byte
		d(pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct))
		p, err := cryptoutil.Open(mk, ct)
		d(err)
		*dst = p
	}
	cli := okxexec.NewClient(cred, false)
	raw, err := cli.Get(ctx, "/api/v5/account/positions")
	fmt.Println("positions:", string(raw), err)
	bal, err := cli.Get(ctx, "/api/v5/account/balance?ccy=ETH")
	fmt.Println("ETH:", extractAvail(string(bal)), err)
	av, _ := cli.TradingAvailBal(ctx, "USDT")
	fmt.Println("USDT trading:", av)
}

func extractAvail(s string) string {
	i := indexOf(s, `"availBal":"`)
	if i < 0 {
		return "?"
	}
	r := s[i+12:]
	j := 0
	for j < len(r) && r[j] != '"' {
		j++
	}
	return r[:j]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func d(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
