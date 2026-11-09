// legarb —— ambiguous 腿仲裁件（只读取证,不写账）:
// 逐 ambiguous 腿按 clOrdId 拉 OKX ord-history/archive 实录,
// 印 venue 真态(filled/canceled/absent)+fillPx/fillSz 凭据。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
	"os"
)

func main() {
	ctx := context.Background()
	pool, _ := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	mk, _ := config.ReadMasterKey()
	var cred okxexec.Credentials
	for k, dst := range map[string]*string{"okx.api_key": &cred.Key, "okx.api_secret": &cred.Secret, "okx.passphrase": &cred.Passphrase} {
		var ct []byte
		pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct)
		p, _ := cryptoutil.Open(mk, ct)
		*dst = p
	}
	cli := okxexec.NewClient(cred, false)
	rows, _ := pool.Query(ctx, `SELECT id,cl_ord_id,inst_id FROM fin_order_leg WHERE state='ambiguous' ORDER BY id`)
	type leg struct {
		id       int64
		cl, inst string
	}
	var legs []leg
	for rows.Next() {
		var l leg
		rows.Scan(&l.id, &l.cl, &l.inst)
		legs = append(legs, l)
	}
	rows.Close()
	for _, l := range legs {
		fmt.Printf("== leg#%d %s clOrdId=%s\n", l.id, l.inst, l.cl)
		for _, ep := range []string{
			"/api/v5/trade/orders-history-archive?instType=SWAP&instId=" + l.inst + "&clOrdId=" + l.cl,
			"/api/v5/trade/orders-history?instType=SWAP&instId=" + l.inst + "&clOrdId=" + l.cl,
		} {
			raw, err := cli.Get(ctx, ep)
			if err != nil {
				fmt.Println("  err:", err)
				continue
			}
			var arr []struct {
				OrdId, ClOrdId, State, AvgPx, AccFillSz, Sz, Side string
			}
			if json.Unmarshal(raw, &arr) != nil || len(arr) == 0 {
				continue
			}
			var hit bool
			for _, o := range arr {
				if o.ClOrdId != l.cl {
					continue
				}
				hit = true
				fmt.Printf("  ordId=%s state=%s side=%s sz=%s fill=%s avgPx=%s\n", o.OrdId, o.State, o.Side, o.Sz, o.AccFillSz, o.AvgPx)
			}
			if hit {
				break
			}
		}
	}
}
