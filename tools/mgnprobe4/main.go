// mgnprobe4 —— O-39460 S0 第四轮：借贷端点路由存在性（无效参=路由验不执行）+ max-loan 四参。
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
		if len(s) > 400 {
			s = s[:400] + "…"
		}
		fmt.Printf("== GET %s\n   %s\n", path, s)
	}
	post := func(path, body string) {
		raw, err := cli.Post(ctx, path, []byte(body))
		if err != nil {
			fmt.Printf("== POST %s\n   ERR %v\n", path, err)
			return
		}
		fmt.Printf("== POST %s\n   %s\n", path, string(raw))
	}
	// max-loan 四参：ccy+instId+mgnMode+mgnCcy（mgnCcy=计价币 USDT）
	for _, q := range []string{
		"/api/v5/account/max-loan?ccy=BTC&instId=BTC-USDT&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=BTC&instId=BTC-USDT&mgnMode=isolated&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=BIGTIME&instId=BIGTIME-USDT&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=SLX&instId=SLX-USDT&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=RAY&instId=RAY-USDT&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=SOL&instId=SOL-USDT&mgnMode=cross&mgnCcy=USDT",
		"/api/v5/account/max-loan?ccy=ETH&instId=ETH-USDT&mgnMode=cross&mgnCcy=USDT",
	} {
		get(q)
	}
	// 借贷族路由存在性——无效参探测（amt=0/假ccy=参数校验先于执行）
	post("/api/v5/account/borrow-repay", `{"ccy":"FAKECCY","amt":"0","side":"borrow","instId":"BTC-USDT"}`)
	post("/api/v5/account/spot-manual-borrow-repay", `{"ccy":"FAKECCY","amt":"0","side":"borrow"}`)
	post("/api/v5/account/set-auto-repay", `{"autoRepay":true}`)
	get("/api/v5/account/borrow-repay-history?instId=BTC-USDT&limit=3")
	get("/api/v5/account/spot-borrow-repay-history?limit=3")
	// MARGIN 对全量目录计数（负 funding 标的可借对覆盖面）
	get("/api/v5/public/instruments?instType=MARGIN")
}
