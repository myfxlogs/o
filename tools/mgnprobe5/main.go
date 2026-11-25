// mgnprobe5 —— S0 收官：mgnCcy 方言 + 余额保证金字段 + 持仓面。
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
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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
		if err != nil {
			fmt.Printf("== GET %s\n   ERR %v\n", path, err)
			return
		}
		s := string(raw)
		if len(s) > 900 {
			s = s[:900] + "…"
		}
		fmt.Printf("== GET %s\n   %s\n", path, s)
	}
	// mgnCcy 方言：借币币本位 vs 计价币本位
	for _, q := range []string{
		"/api/v5/account/max-loan?ccy=BTC&instId=BTC-USDT&mgnMode=cross&mgnCcy=BTC",
		"/api/v5/account/max-loan?ccy=BTC&instId=BTC-USDT&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=BTC&instId=BTC-USDT-SWAP&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=BTC&instId=BTC-USD-SWAP&mgnMode=cross&mgnCcy=USD",
		// 不带 mgnCcy 但带 mgnMode+instId——隔离 vs 全仓对照已做
		// 持仓面：现有仓 mgnMode 分布
	} {
		get(q)
	}
	get("/api/v5/account/positions?instType=SWAP&instId=BTC-USDT-SWAP")
	get("/api/v5/account/positions?instType=MARGIN")
	// 余额 details 里保证金关键字段（mgnRatio/interest/borrow）
	get("/api/v5/account/balance?ccy=BTC,USDT,ETH")
}
