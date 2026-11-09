package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		panic(err)
	}
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

	// trading 户 details 全行（eqUsd/availBal/disEq 逐币）
	if raw, err := cli.Get(ctx, "/api/v5/account/balance"); err == nil {
		var r []struct {
			Details []struct {
				Ccy      string `json:"ccy"`
				Eq       string `json:"eq"`
				EqUsd    string `json:"eqUsd"`
				AvailBal string `json:"availBal"`
			} `json:"details"`
		}
		json.Unmarshal(raw, &r)
		for _, a := range r {
			for _, d := range a.Details {
				fmt.Printf("trading %-8s eq=%s eqUsd=%s avail=%s\n", d.Ccy, d.Eq, d.EqUsd, d.AvailBal)
			}
		}
	} else {
		fmt.Println("trading err:", err)
	}
	if raw, err := cli.Get(ctx, "/api/v5/asset/balances"); err == nil {
		var r []struct {
			Ccy   string `json:"ccy"`
			Bal   string `json:"bal"`
			Avail string `json:"availBal"`
		}
		json.Unmarshal(raw, &r)
		for _, d := range r {
			if d.Bal != "0" && d.Bal != "" {
				fmt.Printf("funding %-8s bal=%s avail=%s\n", d.Ccy, d.Bal, d.Avail)
			}
		}
	} else {
		fmt.Println("funding err:", err)
	}
	if sav, err := cli.SavingsBalance(ctx); err == nil {
		var rows []struct {
			Ccy string `json:"ccy"`
			Amt string `json:"amt"`
		}
		json.Unmarshal(sav, &rows)
		for _, r := range rows {
			fmt.Printf("savings %-8s amt=%s\n", r.Ccy, r.Amt)
		}
	} else {
		fmt.Println("savings err:", err)
	}
	if raw, err := cli.Get(ctx, "/api/v5/account/bills-archive?type=2&ccy=USDT&limit=20"); err == nil {
		fmt.Println("bills:", string(raw)[:1500])
	}
	if raw, err := cli.Get(ctx, "/api/v5/asset/deposit-withdraw-history?limit=10"); err == nil {
		fmt.Println("depwd:", string(raw)[:800])
	}
	if raw, err := cli.Get(ctx, "/api/v5/asset/transfer-state?type=0&limit=10"); err == nil {
		fmt.Println("transfer:", string(raw)[:1200])
	}
	if raw, err := cli.Get(ctx, "/api/v5/asset/bills?ccy=USDT&limit=15"); err == nil {
		fmt.Println("assetbills:", string(raw)[:1800])
	}
	if ords, err := cli.DcdLiveOrders(ctx); err == nil {
		for _, od := range ords {
			b, _ := json.Marshal(od)
			fmt.Printf("dcd %s\n", string(b)[:200])
		}
	} else {
		fmt.Println("dcd err:", err)
	}
}
