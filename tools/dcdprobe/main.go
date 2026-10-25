// dcdprobe —— dcd order-history 现读探针（T-0101 字段名取证+信封双型验证）。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

func main() {
	dsn := flag.String("dsn", "", "")
	key := flag.String("key", config.MasterKeyPath(), "")
	raw := flag.Bool("raw", false, "打原始 data")
	flag.Parse()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	die(err)
	mk, err := config.ReadMasterKeyFrom(*key)
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
	if *raw {
		b, err := cli.Get(ctx, "/api/v5/finance/sfp/dcd/order-history?state=live")
		die(err)
		fmt.Println(string(b))
		return
	}
	rows, err := cli.DcdLiveOrders(ctx)
	die(err)
	for _, r := range rows {
		fmt.Printf("%+v\n", r)
	}
}
func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
}
