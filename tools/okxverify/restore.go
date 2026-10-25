// restore.go —— okxverify mutating 拆件⑤⑥⑦⑧（现货腿/挂单谱/bills/台账制复位）
// +取件件——行数棘轮自主拆分，行为与单文件版逐行同源。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"o/internal/trading/okxexec"
)

func mSpot(ctx context.Context, cli *okxexec.Client, tra float64, priv chan string) {
	sec := "mut"
	smeta, err := cli.InstrumentMeta(ctx, "SPOT", "BTC-USDT")
	if err == nil {
		buyUsd := 5.0
		if tra < buyUsd+1 {
			buyUsd = tra - 1
		}
		if buyUsd >= smeta.MinSz*1e-4 || buyUsd >= 1 {
			clid := fmt.Sprintf("vfs%d", time.Now().UnixNano()%1e12)
			br, err := cli.PlaceOrder(ctx, okxexec.OrderParams{InstID: "BTC-USDT", TdMode: "cash",
				Side: "buy", OrdType: "market", Sz: fmtAmt(buyUsd), TgtCcy: "quote_ccy", ClOrdID: clid})
			if err != nil {
				recFail(sec, "spot_buy", err.Error())
			} else {
				rec(sec, "spot_buy", jget(br, "ordId"), "ordId", fmt.Sprintf("市价买 %.2fUSD", buyUsd), true)
			}
			time.Sleep(1500 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
			st, _ := cli.ResolveOrder(ctx, "BTC-USDT", clid)
			rec(sec, "spot_buy_state", st, "filled", "", st == okxexec.StateFilled)
			drainPriv(priv, "@现货买帧")
			av := detailAvail(ctx, cli, "BTC")
			if av > 0 {
				clid2 := fmt.Sprintf("vft%d", time.Now().UnixNano()%1e12)
				sr, err := cli.PlaceOrder(ctx, okxexec.OrderParams{InstID: "BTC-USDT", TdMode: "cash",
					Side: "sell", OrdType: "market", Sz: fmtAmt(av), ClOrdID: clid2})
				if err != nil {
					recFail(sec, "spot_sell", err.Error())
				} else {
					rec(sec, "spot_sell", jget(sr, "ordId"), "ordId", fmt.Sprintf("卖回 %s BTC", fmtAmt(av)), true)
				}
				time.Sleep(1500 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
				st2, _ := cli.ResolveOrder(ctx, "BTC-USDT", clid2)
				rec(sec, "spot_sell_state", st2, "filled", "", st2 == okxexec.StateFilled)
			} else {
				rec(sec, "spot_sell", "skip", "availBal>0", "BTC 余量空——买入未成或余量在冻", false)
			}
		}
	}
}

func mPending(ctx context.Context, cli *okxexec.Client) {
	sec := "mut"
	traL, _ := cli.TradingAvailBal(ctx, "USDT")
	tk, _ := cli.Get(ctx, "/api/v5/market/ticker?instId=BTC-USDT")
	deep := f64(jget(tk, "last")) * 0.5
	lsz := traL * 0.75 / deep
	if lsz > 0.0002 {
		lsz = 0.0002
	}
	lsz = float64(int64(lsz*1e8)) / 1e8
	lid := fmt.Sprintf("vfp%d", time.Now().UnixNano()%1e12)
	lr, err := cli.PlaceOrder(ctx, okxexec.OrderParams{InstID: "BTC-USDT", TdMode: "cash",
		Side: "buy", OrdType: "limit", Sz: fmtAmt(lsz), Px: fmtAmt(deep), ClOrdID: lid})
	if err != nil {
		recFail(sec, "limit_place", err.Error())
	} else {
		rec(sec, "limit_place", jget(lr, "ordId"), "ordId", "深价限价挂", true)
	}
	time.Sleep(1200 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
	pend, _ := cli.Get(ctx, "/api/v5/trade/orders-pending?instId=BTC-USDT")
	rec(sec, "pending_visible", jget(pend, "ordId") != "", "挂单可见", "orders-pending 实读", jget(pend, "ordId") != "")
	st3, _ := cli.ResolveOrder(ctx, "BTC-USDT", lid)
	rec(sec, "limit_state", st3, "live", "", st3 == okxexec.StateLive)
	if st3 == okxexec.StateLive {
		raw2, err2 := cli.CancelOrder(ctx, "BTC-USDT", "", lid)
		if err2 != nil {
			recFail(sec, "limit_cancel", err2.Error())
		} else {
			rec(sec, "limit_cancel", jget(raw2, "ordId"), "ok", "撤单", true)
		}
	}
	time.Sleep(1200 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
	st4, _ := cli.ResolveOrder(ctx, "BTC-USDT", lid)
	rec(sec, "limit_canceled", st4, "canceled/absent", "", st4 != okxexec.StateLive && st4 != okxexec.StateFilled)
}

func mBills(ctx context.Context, cli *okxexec.Client) {
	sec := "mut"
	raw, _ := cli.Get(ctx, "/api/v5/account/bills?instType=SPOT&limit=5")
	rec(sec, "bills_rows", jget(raw, "billId"), "有行", "现货账单实读", jget(raw, "billId") != "")
}

// mRestore 台账制复位（⑧——probe-restore-scope-overreach 戒律）：只许动本探针
// 生命周期引入的资金（soldUSD/parked），存量（Earn/funding/既有 trading）禁碰。
func mRestore(ctx context.Context, cli *okxexec.Client, dustInst string, soldUSD float64, restore string, restoremax float64) {
	sec := "mut"
	var parked float64 // 台账：transfer_back 停入 funding 的 USDT
	target := restore
	if target == "" && dustInst != "" {
		target = strings.TrimSuffix(dustInst, "-USDT")
	}
	if target != "" {
		tra3, _ := cli.TradingAvailBal(ctx, "USDT")
		buyback := tra3 - 0.5
		if target == strings.TrimSuffix(dustInst, "-USDT") {
			// 复审 860b8c1：dust 路径封顶无条件——soldUSD 实测 Δ≤0 时=零买回，
			// fail-open 分支物理消失（原 if soldUSD>0 卫=同族残留）。
			buyback = ledgerCap(buyback, soldUSD)
		}
		if restore != "" && buyback > restoremax {
			buyback = restoremax // 显式复位：封顶防扫存量
		}
		if buyback > 2 {
			clid3 := fmt.Sprintf("vfr%d", time.Now().UnixNano()%1e12)
			br, err := cli.PlaceOrder(ctx, okxexec.OrderParams{InstID: target + "-USDT", TdMode: "cash",
				Side: "buy", OrdType: "market", Sz: fmtAmt(buyback), TgtCcy: "quote_ccy", ClOrdID: clid3})
			if err != nil {
				recFail(sec, "dust_buyback", target+" "+err.Error())
			} else {
				time.Sleep(1500 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
				st3, _ := cli.ResolveOrder(ctx, target+"-USDT", clid3)
				rec(sec, "dust_buyback", st3, "filled", fmt.Sprintf("买回 %s 耗 $%.2f ordId=%s", target, buyback, jget(br, "ordId")), st3 == okxexec.StateFilled)
			}
		} else {
			rec(sec, "dust_buyback", buyback, ">2", "余资不足买回——记实", false)
		}
	}
	tra2, _ := cli.TradingAvailBal(ctx, "USDT")
	// 复审 860b8c1：台账封顶无条件——soldUSD==0（无尘币/存量足额起跑路径）=
	// 零摆回零复购，fail-open 分支物理消失（原 if soldUSD>0 卫=事故同族残留）。
	back := ledgerCap(tra2-0.5, soldUSD)
	if back > 0 {
		cltid, _, err := cli.AssetTransfer(ctx, "USDT", back, okxexec.AcctTrading, okxexec.AcctFunding)
		if err != nil {
			recFail(sec, "transfer_back", err.Error())
		} else {
			parked = back
			rec(sec, "transfer_back", cltid, "ok", fmt.Sprintf("trading→funding 复位 %.2f", back), true)
			time.Sleep(2 * time.Second) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
		}
		fund2, _ := cli.AssetBalances(ctx)
		var fu float64
		for _, b := range fund2 {
			if b.Ccy == "USDT" {
				fu = f64(b.AvailBal)
			}
		}
		pu := fu - 0.5
		if pu > parked {
			pu = parked // 只复购本探针刚停入的额度——存量 Earn 禁碰
		}
		if pu > 1 {
			if err := cli.SavingsPurchase(ctx, "USDT", pu); err != nil {
				rec(sec, "repurchase", err.Error(), "ok", "Earn 复购复位", false)
			} else {
				rec(sec, "repurchase", pu, "ok", "Earn 复购复位", true)
			}
		}
	}
	raw, _ := cli.Get(ctx, "/api/v5/account/positions")
	rec(sec, "final_flat", jget(raw, "pos"), "∅", "终态全平实读", jget(raw, "pos") == "" || jget(raw, "pos") == "∅")
}

// detailAvail 实读 /account/balance details[] 里某币种 availBal（可用余额在
// details 层不在行顶层——统一取件）。
func detailAvail(ctx context.Context, cli *okxexec.Client, ccy string) float64 {
	raw, err := cli.Get(ctx, "/api/v5/account/balance?ccy="+ccy)
	if err != nil {
		return 0
	}
	var rows []struct {
		Details []struct {
			Ccy      string `json:"ccy"`
			AvailBal string `json:"availBal"`
		} `json:"details"`
	}
	if json.Unmarshal(raw, &rows) != nil || len(rows) == 0 {
		return 0
	}
	for _, d := range rows[0].Details {
		if d.Ccy == ccy {
			return f64(d.AvailBal)
		}
	}
	return 0
}

func fmtAmt(f float64) string {
	s := strconv.FormatFloat(f, 'f', 8, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// ledgerCap 台账封顶（复审 860b8c1 钦定形）：可动额度= min(闲余-缓冲, max(本进程引入,0))
// ——引入额度为零=零额度（零摆回/零买回），封顶无条件、无卫分支。
func ledgerCap(availLessBuffer, introduced float64) float64 {
	return min(availLessBuffer, max(introduced, 0))
}

func die(err error) { fmt.Fprintln(os.Stderr, "FAIL:", err); os.Exit(1) }
