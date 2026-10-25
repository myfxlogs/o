// okxprobe —— E-1 已验探针：解 app_secrets 三件套 → GET /account/balance
// 实测回读。禁线：凭据零落 stdout/log/文件；只回显余额数与成败。
// 用法：okxprobe -dsn <o DSN> [-key <keyfile>] [-sim]
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

func main() {
	dsn := flag.String("dsn", "", "o DSN")
	keyf := flag.String("key", config.MasterKeyPath(), "master key 文件")
	sim := flag.Bool("sim", false, "demo 户头")
	flag.Parse()
	ctx := context.Background()
	mk, err := config.ReadMasterKeyFrom(*keyf)
	if err != nil {
		die(err)
	}
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		die(err)
	}
	defer pool.Close()
	creds := okxexec.Credentials{}
	for k, dst := range map[string]*string{"okx.api_key": &creds.Key, "okx.api_secret": &creds.Secret, "okx.passphrase": &creds.Passphrase} {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			die(fmt.Errorf("%s 读败: %w", k, err))
		}
		p, err := cryptoutil.Open(mk, ct)
		if err != nil {
			die(fmt.Errorf("%s 解密: %w", k, err))
		}
		*dst = p
	}
	cli := okxexec.NewClient(creds, *sim)
	raw, err := cli.Get(ctx, "/api/v5/account/balance")
	if err != nil {
		die(fmt.Errorf("balance probe: %w", err))
	}
	var rows []struct {
		TotalEq string `json:"totalEq"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		die(err)
	}
	if len(rows) == 0 {
		die(fmt.Errorf("空返回"))
	}
	fmt.Printf("PROBE-OK totalEq=%s USD（凭据已验——/account/balance 实返）\n", rows[0].TotalEq)
}

func die(err error) { fmt.Fprintln(os.Stderr, "FAIL:", err); os.Exit(1) }
