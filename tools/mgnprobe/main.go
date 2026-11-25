// mgnprobe —— O-39460 S0 取证探针：OKX 保证金做空能力面只读盘点。
// 只发 GET——borrow/repay/place-order 一律不碰（变更面另票实测）。
// 用法：DB_DSN=<pg> O_MASTER_KEY_FILE=<key> mgnprobe [ccy1,ccy2,...]
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
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

	ccys := []string{"BTC", "ETH", "SOL", "DOGE", "XRP", "USDT"}
	if len(os.Args) > 1 {
		ccys = strings.Split(os.Args[1], ",")
	}

	get := func(path string) {
		raw, err := cli.Get(ctx, path)
		if err != nil {
			fmt.Printf("== %s\n   ERR %v\n", path, err)
			return
		}
		// 只打前 600 字节防爆屏
		s := string(raw)
		if len(s) > 600 {
			s = s[:600] + "…"
		}
		fmt.Printf("== %s\n   %s\n", path, s)
	}

	fmt.Println("### 1) /account/config 全量（acctLv/posMode/保证金模式字段）")
	get("/api/v5/account/config")

	fmt.Println("\n### 2) /account/balance details（保证金可用字段面）")
	get("/api/v5/account/balance")

	fmt.Println("\n### 3) 每币借贷面：interest-rate + max-loan + interest-rate-history")
	for _, c := range ccys {
		fmt.Printf("--- %s ---\n", c)
		get("/api/v5/account/interest-rate?ccy=" + c)
		get("/api/v5/account/max-loan?ccy=" + c)
		get("/api/v5/account/interest-rate-history?ccy=" + c + "&limit=3")
	}

	fmt.Println("\n### 4) MARGIN instType 目录（borrowable/杠杆档字段）")
	get("/api/v5/public/instruments?instType=MARGIN")

	fmt.Println("\n### 5) SPOT 标的 borrowable 字段抽查（负 funding 热币）")
	get("/api/v5/public/instruments?instType=SPOT&instId=BIGTIME-USDT")
	get("/api/v5/public/instruments?instType=SPOT&instId=SLX-USDT")

	fmt.Println("\n### 6) 负 funding 基币借贷面（小币可借性命门）")
	for _, c := range []string{"RAY", "BIGTIME", "CC", "BABY", "NEIRO", "SLX", "ANIME", "FWDI", "USO", "XOM", "EGLD", "SAHARA"} {
		fmt.Printf("--- %s ---\n", c)
		get("/api/v5/account/interest-rate?ccy=" + c)
		get("/api/v5/account/max-loan?ccy=" + c)
	}
}
