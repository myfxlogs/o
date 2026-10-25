// dustsweep 粉尘归集件（T-0080：venue 非 USDT 小额余额→USDT——POST
// /api/v5/asset/convert-dust 一次性运营动作非周期环；PENGU 业主令豁免不进单；
// mutating 钱径=手动触发钦定面；账行经 venue_bill 来源行锚非虚写）。
// 用法：DB_DSN=… O_MASTER_KEY_FILE=… dustsweep [-ccy ETC,TRX] [-dry=false]
package main

import (
	"context"
	"encoding/json"
	"flag"
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
	ccyList := flag.String("ccy", "ETC,TRX", "粉尘币清单（逗号分——PENGU 豁免不进）")
	dry := flag.Bool("dry", true, "默认干跑——-dry=false 真扫（mutating 二次确认）")
	flag.Parse()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "db:", err)
		os.Exit(1)
	}
	defer pool.Close()
	mk, _ := config.ReadMasterKey()
	cred := okxCreds(ctx, pool, mk)
	cli := okxexec.NewClient(cred, false)
	if cli == nil {
		fmt.Fprintln(os.Stderr, "okx 凭据缺席——fail-closed 不扫")
		os.Exit(1)
	}
	ccys := strings.Split(*ccyList, ",")
	if *dry {
		fmt.Printf("[dry] convert-dust 干跑——ccy=%v（-dry=false 真扫）\n", ccys)
		return
	}
	body, _ := json.Marshal(map[string]any{"ccy": ccys})
	raw, err := cli.Post(ctx, "/api/v5/asset/convert-dust", body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "convert-dust 败:", err)
		os.Exit(1)
	}
	var resp struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Ccy string `json:"ccy"`
			Amt string `json:"amt"`
		} `json:"data"`
	}
	if uerr := json.Unmarshal(raw, &resp); uerr != nil {
		fmt.Fprintln(os.Stderr, "回包解败:", uerr)
		os.Exit(1)
	}
	if resp.Code != "0" {
		fmt.Fprintf(os.Stderr, "convert-dust 拒: %s %s\n", resp.Code, resp.Msg)
		os.Exit(1)
	}
	var total float64
	for _, d := range resp.Data {
		total += okxexec.ParseF(d.Amt)
		fmt.Printf("%s → %s USDT\n", d.Ccy, d.Amt)
	}
	// 账行=venue_bill 来源行（raw 携回包）+cashflow trade_pnl 关联——dust 原在
	// venue 权益（非 USDT 侧），转 USDT 后账需补行与对账豁免（T-0081）对齐。
	if total > 0 {
		var bid int64
		billID := fmt.Sprintf("dust-%d", time.Now().UnixNano())
		if err := pool.QueryRow(ctx, `INSERT INTO fin_venue_bill
			(bill_id,ts,inst_id,inst_type,ccy,type,sub_type,bal_chg,fee,pnl,sz,px,raw)
			VALUES($1,now(),'DUST','SWAP','USDT','convert','dust_sweep',$2,0,$2,0,0,$3)
			RETURNING id`, billID, total, raw).Scan(&bid); err != nil {
			fmt.Fprintln(os.Stderr, "bill 行落账败:", err)
			os.Exit(1)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow
			(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
			VALUES('trade_pnl','okx','USDT',$1,NULL,$2,'fin_venue_bill',now())`,
			total, bid); err != nil {
			fmt.Fprintln(os.Stderr, "cashflow 入账败:", err)
			os.Exit(1)
		}
	}
	fmt.Printf("归集总 %.4f USDT（bill+cashflow 行已落）@ %s\n", total, time.Now().UTC().Format(time.RFC3339))
}

func okxCreds(ctx context.Context, pool *pgxpool.Pool, mk []byte) okxexec.Credentials {
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
	return cred
}
