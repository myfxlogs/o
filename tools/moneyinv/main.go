// moneyinv —— 钱径实盘底账只读探针（临时诊断件，不提交）：
// 交易户 balance+positions、资金户 asset/balances、Earn savings/balance、
// DCD live orders 一把拉。凭据零落盘零回显。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
}

func main() {
	dsn := flag.String("dsn", "", "")
	keyf := flag.String("key", config.MasterKeyPath(), "")
	flag.Parse()
	ctx := context.Background()
	mk, err := config.ReadMasterKeyFrom(*keyf)
	die(err)
	pool, err := pgxpool.New(ctx, *dsn)
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
	for _, ep := range []struct{ name, path string }{
		{"TRADING-BALANCE", "/api/v5/account/balance"},
		{"TRADING-POSITIONS", "/api/v5/account/positions"},
		{"FUNDING-BALANCES", "/api/v5/asset/balances"},
		{"EARN-SAVINGS", "/api/v5/finance/savings/balance"},
		{"DCD-LIVE", "/api/v5/finance/sfp/dcd/order-history?state=live"},
		{"ORDERS-PENDING", "/api/v5/trade/orders-pending"},
	} {
		raw, err := cli.Get(ctx, ep.path)
		if err != nil {
			fmt.Printf("== %s: ERR %v\n", ep.name, err)
			continue
		}
		var rows []json.RawMessage
		die(json.Unmarshal(raw, &rows))
		fmt.Printf("== %s (%d)\n", ep.name, len(rows))
		for _, r := range rows {
			fmt.Printf("   %s\n", string(r))
		}
	}
}
