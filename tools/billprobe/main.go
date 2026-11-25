// billprobe —— O-39310 取证探针：/account/bills SPOT 行 ordId/clOrdId 键实证
// + bills-archive 区间全量导出（回填判别素材——billId→clOrdId/ordId 映射）。
// 用法：DB_DSN=<pg> O_MASTER_KEY_FILE=<key> billprobe [export|probe]
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

	mode := "probe"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	if mode == "probe" {
		raw, err := cli.Get(ctx, "/api/v5/account/bills?instType=SPOT&limit=8")
		if err != nil {
			panic(err)
		}
		var rows []map[string]any
		if err := json.Unmarshal(raw, &rows); err != nil {
			panic(err)
		}
		for _, r := range rows {
			fmt.Printf("PROBE billId=%v ordId=%v clOrdId=%v inst=%v ccy=%v sub=%v balChg=%v fee=%v\n",
				r["billId"], r["ordId"], r["clOrdId"], r["instId"], r["ccy"], r["subType"], r["balChg"], r["fee"])
		}
		return
	}
	if mode == "fill" {
		fillQuoteLegs(ctx, pool, cli)
		return
	}

	// export：bills-archive type=2&ccy=USDT 全量（after=billId 游标——新→老序翻页；
	// 窗口下限 2026-10-20——fin_venue_bill 最早行 10-24 前留 4 天余量即停）
	bm := pullArchive(ctx, cli)
	for _, r := range bm {
		b, _ := json.Marshal(r)
		fmt.Printf("BILL %s\n", b)
	}
	fmt.Fprintf(os.Stderr, "TOTAL=%d\n", len(bm))
}

// pullArchive bills-archive type=2&ccy=USDT 全量拉取（after=billId 游标——
// 新→老序翻页；窗口下限 2026-10-20——fin_venue_bill 最早行 10-24 前留
// 4 天余量即停）。返 billId→原始行映射（clOrdId/ordId/fee/balChg 等判别素材）。
func pullArchive(ctx context.Context, cli *okxexec.Client) map[string]map[string]any {
	const cutoffMs = int64(1792454400000) // 2026-10-20T00:00:00Z
	out := map[string]map[string]any{}
	after := ""
	for page := 0; page < 200; page++ {
		q := "/api/v5/account/bills-archive?type=2&ccy=USDT&limit=100"
		if after != "" {
			q += "&after=" + after
		}
		raw, err := cli.Get(ctx, q)
		if err != nil {
			fmt.Fprintln(os.Stderr, "PULL-ERR:", err)
			break
		}
		var rows []map[string]any
		if err := json.Unmarshal(raw, &rows); err != nil {
			panic(err)
		}
		if len(rows) == 0 {
			break
		}
		pageMin := int64(1 << 62)
		for _, r := range rows {
			if id, ok := r["billId"].(string); ok {
				out[id] = r
			}
			if ts, ok := r["ts"].(string); ok && len(ts) >= 13 {
				var ms int64
				fmt.Sscanf(ts, "%d", &ms)
				if ms < pageMin {
					pageMin = ms
				}
			}
		}
		if pageMin < cutoffMs {
			break // 已越出取证窗——更老页不需要
		}
		last, _ := rows[len(rows)-1]["billId"].(string)
		if len(rows) < 100 || last == "" || last == after {
			break
		}
		after = last
	}
	return out
}

// fillQuoteLegs O-39310 存量破口回填（幂等可重跑——部署窗内新行同径补）：
// fin_venue_bill SPOT/MARGIN×USDT/USD×subType1/2 候选行逐腿判别——
// ①billId 不在 archive=无判别素材跳过；②clOrdId 空=无判别键跳过；
// ③gross 域=clOrdId 命中 fin_order_leg 已记腿(legCash)∨fin_risk_decision
// cl_ord_id(residual_clear)=已覆盖跳过；④fee 域=同 clOrdId 有 legCash fee
// 实记行=跳过（clear 径不记 fee→bill fee 腿照补）；⑤余者分量级补记
// （fillOne 镜像 spotNative 对称口径）。全程跳过方向=漏记>双计。
func fillQuoteLegs(ctx context.Context, pool *pgxpool.Pool, cli *okxexec.Client) {
	bm := pullArchive(ctx, cli)
	fmt.Fprintf(os.Stderr, "archive bills=%d\n", len(bm))

	rows, err := pool.Query(ctx, `SELECT id,bill_id,inst_id,ccy,sub_type,bal_chg,fee,ts
		FROM fin_venue_bill
		WHERE inst_type IN ('SPOT','MARGIN') AND ccy IN ('USDT','USD')
		  AND sub_type IN ('1','2') ORDER BY id`)
	if err != nil {
		panic(err)
	}
	var cands []billCand
	for rows.Next() {
		var c billCand
		if err := rows.Scan(&c.id, &c.billID, &c.instID, &c.ccy, &c.subType, &c.balChg, &c.fee, &c.ts); err != nil {
			panic(err)
		}
		cands = append(cands, c)
	}
	rows.Close()

	var nNoArch, nNoKey, nOwn, nFill, nErr, nPart int
	for _, c := range cands {
		ar, ok := bm[c.billID]
		if !ok {
			nNoArch++
			continue
		}
		clOrdID, _ := ar["clOrdId"].(string)
		if clOrdID == "" {
			nNoKey++
			continue
		}
		// gross 域覆盖判别（镜像 spotQuote）：legCash(fin_order_leg)∨
		// residual_clear(fin_risk_decision.cl_ord_id) 已记=跳过。
		var covered bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM fin_order_leg l
			JOIN fin_cashflow cf ON cf.source_table='fin_order_leg' AND cf.source_ref=l.id
			WHERE l.venue='okx' AND l.cl_ord_id=$1)
			OR EXISTS(SELECT 1 FROM fin_risk_decision d
				WHERE d.intent->>'cl_ord_id'=$1)`, clOrdID).Scan(&covered); err != nil {
			panic(err)
		}
		if covered {
			nOwn++
		}
		// fee 域独立判别：legCash fee 实记行在=跳过（feeCcy=USDT 已记）；
		// residual_clear 不记 fee→bill fee 腿照补。
		feeCovered := false
		if c.fee != 0 {
			if err := pool.QueryRow(ctx, `SELECT EXISTS(
				SELECT 1 FROM fin_order_leg l
				JOIN fin_cashflow cf ON cf.source_table='fin_order_leg' AND cf.source_ref=l.id
					AND cf.kind='fee'
				WHERE l.venue='okx' AND l.cl_ord_id=$1)`, clOrdID).Scan(&feeCovered); err != nil {
				panic(err)
			}
		}
		added, err := fillOne(ctx, pool, c, covered, feeCovered)
		if err != nil {
			nErr++
			fmt.Fprintf(os.Stderr, "FILL-ERR bill=%s: %v\n", c.billID, err)
			continue
		}
		if added > 0 {
			nFill++
		} else {
			nPart++ // 分量已齐/零分量——跳过
		}
	}
	fmt.Fprintf(os.Stderr, "cands=%d no_archive=%d no_key=%d own=%d filled=%d noop=%d err=%d\n",
		len(cands), nNoArch, nNoKey, nOwn, nFill, nPart, nErr)
}
