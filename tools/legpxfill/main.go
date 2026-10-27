// legpxfill —— T-0179 老仓腿 px 补录：fin_order_leg.px NULL 行→venue 订单史
// 实拉 avgPx 补录（零写径只 UPDATE 凭据行；禁推算造值——拉不到=留 NULL+
// resolve_note 不可溯记档）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

func die(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func instType(iid string) string {
	if len(iid) > 5 && iid[len(iid)-4:] == "SWAP" {
		return "SWAP"
	}
	return "SPOT"
}

func main() {
	dsn := flag.String("dsn", "", "PG_DSN")
	keyf := flag.String("key", config.MasterKeyPath(), "master key 路径")
	dry := flag.Bool("dry", false, "只出不写（先验径）")
	flag.Parse()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	die(err)
	mk, err := config.ReadMasterKeyFrom(*keyf)
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

	rows, err := pool.Query(ctx, `SELECT id,inst_id,cl_ord_id FROM fin_order_leg
		WHERE px IS NULL AND venue='okx' ORDER BY id`)
	die(err)
	type leg struct {
		id      int64
		instID  string
		clOrdID string
	}
	var legs []leg
	for rows.Next() {
		var x leg
		die(rows.Scan(&x.id, &x.instID, &x.clOrdID))
		legs = append(legs, x)
	}
	rows.Close()
	log.Printf("[legpxfill] NULL 缺口行数=%d（venue=okx 全域）", len(legs))

	var updated, untraceable int
	for _, x := range legs {
		// /trade/order 现役面查；51603=订单缺席→order-archive 终态面补查
		raw, err := cli.Get(ctx, fmt.Sprintf("/api/v5/trade/order?instId=%s&clOrdId=%s", x.instID, x.clOrdID))
		if err != nil {
			var ae *okxexec.APIError
			if errors.As(err, &ae) && ae.Code == "51603" {
				raw, err = cli.Get(ctx, fmt.Sprintf("/api/v5/trade/orders-history-archive?instType=%s&instId=%s&clOrdId=%s",
					instType(x.instID), x.instID, x.clOrdID))
			}
		}
		if err != nil {
			log.Printf("[legpxfill] leg=%d %s %s 读败（禁推算——留 NULL）: %v", x.id, x.instID, x.clOrdID, err)
			untraceable++
			continue
		}
		var ordRows []struct {
			State string `json:"state"`
			AvgPx string `json:"avgPx"`
		}
		if json.Unmarshal(raw, &ordRows) != nil || len(ordRows) == 0 {
			log.Printf("[legpxfill] leg=%d %s %s 订单史缺席（不可溯如实记）", x.id, x.instID, x.clOrdID)
			untraceable++
			continue
		}
		px := okxexec.ParseF(ordRows[0].AvgPx)
		if px <= 0 {
			log.Printf("[legpxfill] leg=%d %s %s avgPx 零/缺（不可溯）state=%s", x.id, x.instID, x.clOrdID, ordRows[0].State)
			untraceable++
			continue
		}
		if *dry {
			log.Printf("[legpxfill] DRY leg=%d %s %s px=%.6g", x.id, x.instID, x.clOrdID, px)
		} else if _, err := pool.Exec(ctx, `UPDATE fin_order_leg SET px=$2 WHERE id=$1`, x.id, px); err != nil {
			die(err)
		}
		updated++
		time.Sleep(200 * time.Millisecond) // sched-exempt-ok：限频件
	}
	log.Printf("[legpxfill] 收尾：updated=%d untraceable=%d 总=%d", updated, untraceable, len(legs))
}
