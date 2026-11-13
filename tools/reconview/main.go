// reconview —— 对账自助面（手工SQL退休件）：
// 三段印全——①账本净额按 (venue,ccy) 分解（cashflow/parked/dcdBase/mtm/
// known_residual 各分量）；②venue 现读同径（trading 基金eq+funding/
// savings+dcd LIVE 名义）；③对拍 diff（USD 折算判容差,同 Runner.reconcile
// 口径）+ambiguous 腿待仲裁面+live 策略真敞口（live 而零仓零动=僵尸甄别）。
// 只读零写。凭据=app_secrets 密文轨（config.ReadMasterKey+cryptoutil，
// okxread/mt5wrap 同径）。用法：DB_DSN=... O_SECRET_KEY=<mk> go run ./tools/reconview
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/ledger"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

func pf(s string) float64 { return okxexec.ParseF(s) }

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		panic(err)
	}

	// ── venue 现读（venue_readers.go 同径：trading=md_account_events 最新
	// account 快照帧基金口径 eqUsd 求和——T-0123 状态表下跨频道取最新行会
	// 采到 balance_and_position 帧（data 无 details[]→trading 整量漏算零，
	// 假差≈trading 全额）;帧缺/读败→REST /account/balance 兜底
	// （tradingFundEq 同径,双径俱败显名 venErr 非静默）;
	// funding/savings/dcd=REST 活读,usd=false 原币量；
	// 非本位币走 FundCcy 白名单闸——托管尘不键）。
	ven := map[[2]string]float64{}
	fset, _ := ledger.FundCcy(ctx, pool)
	emitOK := func(dom, ccy string) bool {
		return ccy == "USDT" || ccy == "USD" || fset[[2]string{"okx:" + dom, ccy}]
	}
	var venErr []string
	var data json.RawMessage
	type detailRow struct {
		Ccy   string `json:"ccy"`
		EqUsd string `json:"eqUsd"`
	}
	tradingSum := func(rows []detailRow) float64 {
		var sum float64
		for _, d := range rows {
			if emitOK("trading", d.Ccy) {
				sum += pf(d.EqUsd)
			}
		}
		return sum
	}
	tradingRead := false
	var top struct {
		Details []detailRow `json:"details"`
	}
	if err := pool.QueryRow(ctx, `SELECT data FROM md_account_events
		WHERE channel='account' ORDER BY ts DESC LIMIT 1`).Scan(&data); err == nil &&
		json.Unmarshal(data, &top) == nil && tradingSum(top.Details) > 0 {
		ven[[2]string{"okx", "USDT"}] += tradingSum(top.Details)
		tradingRead = true
	}
	var cred okxexec.Credentials
	var cli *okxexec.Client
	mk, mkErr := config.ReadMasterKey()
	for k, dst := range map[string]*string{
		"okx.api_key": &cred.Key, "okx.api_secret": &cred.Secret, "okx.passphrase": &cred.Passphrase} {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			mkErr = err
			break
		}
		p, err := cryptoutil.Open(mk, ct)
		if err != nil {
			mkErr = err
			break
		}
		*dst = p
	}
	if mkErr != nil {
		venErr = append(venErr, "凭据读败(仅账本面印出): "+mkErr.Error())
	} else {
		cli = okxexec.NewClient(cred, false)
		if bals, err := cli.AssetBalances(ctx); err != nil {
			venErr = append(venErr, "funding 读败: "+err.Error())
		} else {
			for _, b := range bals {
				if v := pf(b.Bal); v != 0 && emitOK("funding", b.Ccy) {
					ven[[2]string{"okx", b.Ccy}] += v
				}
			}
		}
		if sav, err := cli.SavingsBalance(ctx); err != nil {
			venErr = append(venErr, "savings 读败: "+err.Error())
		} else {
			var rows []struct {
				Ccy string `json:"ccy"`
				Amt string `json:"amt"`
			}
			_ = json.Unmarshal(sav, &rows)
			for _, r := range rows {
				if v := pf(r.Amt); v != 0 && emitOK("earn", r.Ccy) {
					ven[[2]string{"okx", r.Ccy}] += v
				}
			}
		}
		if ords, err := cli.DcdLiveOrders(ctx); err != nil {
			venErr = append(venErr, "dcd 读败: "+err.Error())
		} else {
			for _, od := range ords {
				ven[[2]string{"okx", od.NotionalCcy}] += pf(od.NotionalSz)
			}
		}
		if !tradingRead {
			// 帧缺/读败→REST /account/balance 兜底（tradingFundEq 同径;
			// Get 已剥 data 壳）——禁静默归零,双径俱败显名上账。
			raw, err := cli.Get(ctx, "/api/v5/account/balance")
			var rows []struct {
				Details []detailRow `json:"details"`
			}
			var sum float64
			if err == nil && json.Unmarshal(raw, &rows) == nil && len(rows) > 0 {
				sum = tradingSum(rows[0].Details)
			}
			if sum > 0 {
				ven[[2]string{"okx", "USDT"}] += sum
			} else if err != nil {
				venErr = append(venErr, "trading 读败(帧缺+REST兜底俱败,分量缺读数偏低): "+err.Error())
			} else {
				venErr = append(venErr, "trading 分量缺: account 帧+REST 均无 details 行")
			}
		}
	}
	var mt5eq float64
	_ = pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE venue='mt5' AND ccy='USD' AND kind NOT IN ('earn_in','earn_out')`).Scan(&mt5eq)
	fmt.Printf("== mt5 USD 账本净额 %.2f（venue 现读走 mt5wrap/mt5verify——容器外径）\n\n", mt5eq)

	// ── 账本净额分解（run.go reconcile 同口径）：
	wl, parked, dcdBase, err := ledger.ReconSets(ctx, pool)
	if err != nil {
		panic(err)
	}
	keys := map[[2]string]bool{}
	for k := range wl {
		if eq, ok := ven[k]; ok || k[0] == "mt5" {
			_ = eq
			keys[k] = true
		}
	}
	for k := range ven {
		keys[k] = true
	}
	var ks [][2]string
	for k := range keys {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i][0]+ks[i][1] < ks[j][0]+ks[j][1] })

	fmt.Printf("%-6s %-6s %14s %12s %10s %10s %12s | %14s %10s %10s\n",
		"venue", "ccy", "cashflow净", "parked", "dcdBase", "mtm", "账本净", "venue现读", "diff", "豁免后")
	for _, k := range ks {
		var cf, ex float64
		_ = pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
			WHERE venue=$1 AND ccy=$2 AND kind NOT IN ('earn_in','earn_out')`, k[0], k[1]).Scan(&cf)
		_ = pool.QueryRow(ctx, `SELECT COALESCE(sum(amount_usd),0) FROM fin_known_residual
			WHERE venue=$1 AND ccy=$2 AND expires_at>now()`, k[0], k[1]).Scan(&ex)
		mtm := mtmOf(ctx, pool, cli, k)
		net := cf + parked[k] + dcdBase[k] + mtm
		eq := ven[k]
		diff := ledger.Diff(net, eq)
		usdDiff := diff
		if k[1] != "USDT" && k[1] != "USD" && diff != 0 {
			var px float64
			_ = pool.QueryRow(ctx, `SELECT v FROM (SELECT last v,ts FROM md_ticks WHERE inst_id=$1||'-USDT'
				UNION ALL SELECT px,ts FROM md_mark_index WHERE inst_id=$1||'-USDT') q
				ORDER BY ts DESC LIMIT 1`, k[1]).Scan(&px)
			if px > 0 {
				usdDiff = diff * px
			}
		}
		wlv := ""
		if !wl[k] {
			wlv = "（白名单外—不报）"
		}
		fmt.Printf("%-6s %-6s %14.4f %12.4f %10.4f %10.4f %12.4f | %14.4f %10.2f %10.2f %s\n",
			k[0], k[1], cf, parked[k], dcdBase[k], mtm, net, eq, usdDiff, usdDiff-ex, wlv)
	}
	for _, e := range venErr {
		fmt.Println("venue 读缺:", e)
	}

	// ── ambiguous 腿待仲裁面
	fmt.Println("\n== ambiguous 腿（venue 回报歧态势——禁补单,待 venue 实录仲裁）")
	legs, _ := pool.Query(ctx, `SELECT id,execution_id,leg_no,inst_id,side,sz,ts,venue_err_code
		FROM fin_order_leg WHERE state='ambiguous' ORDER BY ts`)
	defer legs.Close()
	for legs.Next() {
		var id, execID, no int64
		var inst, side string
		var sz float64
		var ts time.Time
		var vc *string
		_ = legs.Scan(&id, &execID, &no, &inst, &side, &sz, &ts, &vc)
		vcv := ""
		if vc != nil {
			vcv = *vc
		}
		fmt.Printf("  leg#%d exec=%d %s %s %.4f @%s err=%s\n", id, execID, inst, side, sz, ts.Format("2006-01-02"), vcv)
	}

	// ── live 策略真敞口（live 而零开仓腿零近动=僵尸候选）
	fmt.Println("\n== live 策略敞口甄别（live≠敞口——开仓腿=真敞口铁证,零腿=armed 待信号）")
	srows, _ := pool.Query(ctx, `SELECT id,inst_id,template_ref,entered_at FROM fin_strategy
		WHERE state='live' ORDER BY id`)
	defer srows.Close()
	for srows.Next() {
		var id int64
		var inst, tpl string
		var entered time.Time
		_ = srows.Scan(&id, &inst, &tpl, &entered)
		var openLegs, act7d int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_order_leg l
			JOIN fin_position p ON p.execution_id=l.execution_id
			WHERE p.closed_at IS NULL AND l.state='filled' AND l.inst_id=$1`, inst).Scan(&openLegs)
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_order_leg WHERE inst_id=$1
			AND ts>now()-interval '7 days'`, inst).Scan(&act7d)
		tag := ""
		if openLegs == 0 && act7d == 0 {
			tag = "  ←零敞口（armed 待信号,非僵尸——funding-hedge 设计态）"
		} else if openLegs == 0 {
			tag = "  ←零敞口但近7d有腿动（刚平或新开）"
		}
		fmt.Printf("  #%d %-18s %-22s 开仓腿=%d 近7d腿动=%d 入场=%s%s\n",
			id, inst, tpl, openLegs, act7d, entered.Format("2006-01-02"), tag)
	}
}

// mtmOf 盯市偏移（mtmReader 同口径简版——开仓腿 sign×qty×现价）：
// SWAP 腿按 qty_base/币量,现货腿禁推算——仅 qty_base>0 或 sz_unit=base 者计入。
func mtmOf(ctx context.Context, pool *pgxpool.Pool, cli *okxexec.Client, k [2]string) float64 {
	if k[1] != "USDT" {
		return 0 // 非本位键 venue=币数同径——mtm 只作用 USDT 偏移
	}
	rows, err := pool.Query(ctx, `SELECT l.venue,l.inst_id,l.side,l.sz,l.px,l.sz_unit,l.qty_base
		FROM fin_order_leg l JOIN fin_position p ON p.execution_id=l.execution_id
		WHERE p.closed_at IS NULL AND l.state='filled' AND l.px IS NOT NULL`)
	if err != nil {
		return 0
	}
	defer rows.Close()
	var out float64
	for rows.Next() {
		var venue, inst, side string
		var sz, entry float64
		var su *string
		var qb *float64
		if rows.Scan(&venue, &inst, &side, &sz, &entry, &su, &qb) != nil || venue != k[0] {
			continue
		}
		var now float64
		if pool.QueryRow(ctx, `SELECT v FROM (SELECT last v,ts FROM md_ticks WHERE inst_id=$1
			UNION ALL SELECT px,ts FROM md_mark_index WHERE inst_id=$1) q
			ORDER BY ts DESC LIMIT 1`, inst).Scan(&now) != nil || now <= 0 {
			continue
		}
		sign := 1.0
		if side == "sell" {
			sign = -1.0
		}
		if !strings.HasSuffix(inst, "-SWAP") {
			switch {
			case qb != nil && *qb > 0:
				out += sign * *qb * now
			case su != nil && *su == "base":
				out += sign * sz * now
			}
			continue
		}
		mult := 1.0
		if cli != nil {
			if m, err := cli.InstrumentMeta(ctx, "SWAP", inst); err == nil && m.CtVal > 0 {
				mult = m.CtVal
			}
		}
		out += sign * sz * mult * (now - entry)
	}
	return out
}
