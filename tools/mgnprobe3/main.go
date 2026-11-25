// mgnprobe3 —— max-loan 三参补全 + borrow 面 GET 族普查。
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
		if len(s) > 500 {
			s = s[:500] + "…"
		}
		fmt.Printf("== %s\n   %s\n", path, s)
	}
	// max-loan: ccy+instId+mgnMode——instId 试 MARGIN 对与 SPOT 对两形
	for _, pair := range [][3]string{
		{"BTC", "BTC-USDT", "cross"}, {"BTC", "BTC-USDT", "isolated"},
		{"BTC", "BTC-USDT-SWAP", "cross"},
		{"BIGTIME", "BIGTIME-USDT", "cross"}, {"BIGTIME", "BIGTIME-USDT", "isolated"},
		{"SLX", "SLX-USDT", "cross"}, {"SLX", "SLX-USDT", "isolated"},
		{"RAY", "RAY-USDT", "cross"}, {"RAY", "RAY-USDT", "isolated"},
		{"XOM", "XOM-USDT", "cross"}, {"XOM", "XOM-USDT", "isolated"},
	} {
		get(fmt.Sprintf("/api/v5/account/max-loan?ccy=%s&instId=%s&mgnMode=%s", pair[0], pair[1], pair[2]))
	}
	// borrow/repay 端点存在性——用空 body POST 只验路由不验参数（预期 50014 参数错=路由在）
	for _, p := range []string{"/api/v5/account/borrow", "/api/v5/account/repay"} {
		raw, err := cli.Post(ctx, p, []byte(`{"ccy":"BTC","amt":"0.001"}`))
		if err != nil {
			fmt.Printf("== POST %s\n   ERR %v\n", p, err)
			continue
		}
		fmt.Printf("== POST %s\n   %s\n", p, string(raw))
	}
}
