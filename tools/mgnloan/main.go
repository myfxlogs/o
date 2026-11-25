// mgnloan —— O-40707 S0:max-loan 正确参形(ccy XOR instId)+借币余额读面复核。
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	mk, err := config.ReadMasterKey()
	if err != nil {
		panic(err)
	}
	var cred okxexec.Credentials
	for k, dst := range map[string]*string{"okx.api_key": &cred.Key, "okx.api_secret": &cred.Secret, "okx.passphrase": &cred.Passphrase} {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			panic(err)
		}
		p, err := cryptoutil.Open(mk, ct)
		if err != nil {
			panic(err)
		}
		*dst = p
	}
	cli := okxexec.NewClient(cred, false)
	get := func(path string) {
		raw, err := cli.Get(ctx, path)
		fmt.Printf("== %s\n   ", path)
		if err != nil {
			fmt.Printf("ERR %v\n", err)
			return
		}
		s := string(raw)
		if len(s) > 400 {
			s = s[:400] + "…"
		}
		fmt.Printf("%s\n", s)
	}
	for _, q := range []string{
		"/api/v5/account/max-loan?ccy=BTC&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=SOL&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=BIGTIME&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?instId=BTC-USDT&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=BTC&mgnMode=isolated&mgnCcy=BTC",
		"/api/v5/account/max-loan?ccy=USDT&mgnMode=cross&mgnCcy=USDT",
	} {
		get(q)
	}
	get("/api/v5/account/balance")
}
