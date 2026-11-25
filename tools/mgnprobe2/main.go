// mgnprobe2 —— O-39460 S0 第二轮：max-loan 带 mgnMode + borrow 面形状。
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
			fmt.Printf("== %s\n   ERR %v\n", path, err)
			return
		}
		s := string(raw)
		if len(s) > 700 {
			s = s[:700] + "…"
		}
		fmt.Printf("== %s\n   %s\n", path, s)
	}
	for _, c := range []string{"BTC", "ETH", "SOL", "BIGTIME", "SLX", "RAY", "USDT", "XOM"} {
		for _, m := range []string{"cross", "isolated"} {
			fmt.Printf("--- %s mgnMode=%s ---\n", c, m)
			get("/api/v5/account/max-loan?ccy=" + c + "&mgnMode=" + m)
		}
	}
	// config 里 enableSpotBorrow=false 的语义——试 borrow 权限面读（GET 族）
	get("/api/v5/account/borrow-interest-rate?ccy=BTC")
	get("/api/v5/account/bills?type=2&limit=5")
	get("/api/v5/account/bills?instType=MARGIN&limit=5")
	get("/api/v5/public/instruments?instType=MARGIN&instId=BTC-USDT")
}
