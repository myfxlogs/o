// earnpoll —— T-0023 dev 实证件：直拉 lending-rate-summary→md_earn_rate 落行。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/data/earnrate"
	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/trading/okxexec"
)

func main() {
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
	cfg := config.New(pool)
	_ = cfg.Load(ctx)
	c := earnrate.NewCollector(pool, cfg, func(ctx context.Context) (json.RawMessage, error) {
		return cli.Get(ctx, "/api/v5/finance/savings/lending-rate-summary?ccy=USDT")
	})
	raw, _ := cli.Get(ctx, "/api/v5/finance/savings/lending-rate-summary?ccy=USDT")
	fmt.Printf("raw=%s\n", raw)
	if err := c.CollectOnce(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "collect:", err)
		os.Exit(1)
	}
	var n int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM md_earn_rate").Scan(&n)
	fmt.Printf("COLLECT-OK rows=%d\n", n)
}
