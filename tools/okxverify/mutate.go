// mutate.go —— okxverify mutating 实单谱拆件①②③④（尘币自筹资/划转往返/Earn/
// 合约腿——行数棘轮自主拆分，行为与单文件版逐行同源）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"phase/internal/trading/okxexec"
)

func mSelfFund(ctx context.Context, cli *okxexec.Client) (tra float64, dustInst string, soldUSD float64) {
	sec := "mut"
	tra, _ = cli.TradingAvailBal(ctx, "USDT")
	fund, _ := cli.AssetBalances(ctx)
	var fundUSDT float64
	for _, b := range fund {
		if b.Ccy == "USDT" {
			fundUSDT = f64(b.AvailBal)
		}
	}
	fmt.Printf("起始: trading_avail=%.4f funding_avail=%.4f\n", tra, fundUSDT)
	if tra < 2 {
		// 枚举 details 找可卖币（非 USDT、availBal>0、折 USD≥$3）
		raw, _ := cli.Get(ctx, "/api/v5/account/balance")
		var dets []struct {
			Ccy      string `json:"ccy"`
			AvailBal string `json:"availBal"`
		}
		var b0 []struct {
			Details []struct {
				Ccy      string `json:"ccy"`
				AvailBal string `json:"availBal"`
			} `json:"details"`
		}
		if json.Unmarshal(raw, &b0) == nil && len(b0) > 0 {
			for _, d := range b0[0].Details {
				dets = append(dets, d)
			}
		}
		var sellCcy, sellSz string
		for _, d := range dets {
			if d.Ccy == "USDT" || f64(d.AvailBal) <= 0 {
				continue
			}
			inst := d.Ccy + "-USDT"
			m, err := cli.InstrumentMeta(ctx, "SPOT", inst)
			if err != nil || m.State != "live" || f64(d.AvailBal) < m.MinSz {
				continue
			}
			tk, _ := cli.Get(ctx, "/api/v5/market/ticker?instId="+inst)
			if usd := f64(d.AvailBal) * f64(jget(tk, "last")); usd < 3 {
				fmt.Printf("  尘币 %s avail=%s 折$%.2f——太薄跳过\n", d.Ccy, d.AvailBal, usd)
				continue
			}
			sellCcy, sellSz = d.Ccy, d.AvailBal
			break
		}
		if sellCcy != "" {
			inst := sellCcy + "-USDT"
			fmt.Printf("自筹资: 卖 %s %s→USDT（户内尘币造测试金，测毕买回复位）\n", sellSz, sellCcy)
			clid := fmt.Sprintf("vfz%d", time.Now().UnixNano()%1e12)
			sr, err := cli.PlaceOrder(ctx, okxexec.OrderParams{InstID: inst, TdMode: "cash",
				Side: "sell", OrdType: "market", Sz: sellSz, ClOrdID: clid})
			if err != nil {
				recFail(sec, "dust_sell", inst+" "+err.Error())
			} else {
				rec(sec, "dust_sell", jget(sr, "ordId"), "ordId", "尘币卖成 USDT", true)
				time.Sleep(2 * time.Second) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
				st, _ := cli.ResolveOrder(ctx, inst, clid)
				rec(sec, "dust_sell_state", st, "filled", "", st == okxexec.StateFilled)
				traPre := tra
				tra, _ = cli.TradingAvailBal(ctx, "USDT")
				if d := tra - traPre; d > 0 {
					soldUSD = d
				}
				rec(sec, "usdt_after_sell", tra, ">2", "筹资后闲余", tra > 2)
				dustInst = inst // 记给复位段（划转摆回之前先买回——否则无余资）
			}
		} else {
			fmt.Println("无可卖尘币——mutating 谱限于零资项")
		}
	}
	return tra, dustInst, soldUSD
}

func mTransfers(ctx context.Context, cli *okxexec.Client, mvusd, tra float64) {
	sec := "mut"
	mv := mvusd / 4
	if tra >= mv+1 {
		cltid, err := cli.AssetTransfer(ctx, "USDT", mv, okxexec.AcctTrading, okxexec.AcctFunding)
		if err != nil {
			rec(sec, "transfer_out", "clTId="+cltid, "ok", "歧态势——查 transfer-state", false)
			st, _ := cli.AssetTransferState(ctx, "USDT", cltid)
			rec(sec, "transfer_state", st, "success/failed", "歧态势仲裁（真 clTId）", st == "success" || st == "failed")
		} else {
			rec(sec, "transfer_out", cltid, "ok", "trading→funding", cltid != "")
			time.Sleep(1500 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
			st, _ := cli.AssetTransferState(ctx, "USDT", cltid)
			rec(sec, "transfer_state", st, "success", "clTId 定态实证", st == "success")
		}
		cltid2, err := cli.AssetTransfer(ctx, "USDT", mv, okxexec.AcctFunding, okxexec.AcctTrading)
		if err != nil {
			recFail(sec, "transfer_in", "clTId="+cltid2+" "+err.Error())
		} else {
			rec(sec, "transfer_in", cltid2, "ok", "funding→trading 回摆", true)
		}
	} else {
		rec(sec, "transfer_rt", "skip", "avail>mv", "trading 闲余不足以安全往返", false)
	}
}

func mEarn(ctx context.Context, cli *okxexec.Client) {
	sec := "mut"
	if raw, err := cli.SavingsBalance(ctx); err == nil {
		var sv []map[string]any
		_ = json.Unmarshal(raw, &sv)
		var samt float64
		for _, d := range sv {
			if fmt.Sprint(d["ccy"]) == "USDT" {
				samt = f64(fmt.Sprint(d["amt"]))
			}
		}
		if samt >= 2 {
			if err := cli.SavingsRedeem(ctx, "USDT", 1); err != nil {
				recFail(sec, "savings_redeem", err.Error())
			} else {
				rec(sec, "savings_redeem", 1, "ok", "Earn 赎回 1U→funding", true)
				time.Sleep(3 * time.Second) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
				if err := cli.SavingsPurchase(ctx, "USDT", 1); err != nil {
					rec(sec, "savings_repurchase", err.Error(), "ok", "复购复位", false)
				} else {
					rec(sec, "savings_repurchase", 1, "ok", "复购复位", true)
				}
			}
		} else {
			rec(sec, "savings_cycle", "USDT amt="+fmtAmt(samt), "USDT≥2", "Earn 无可赎 USDT——记实跳测", false)
		}
	} else {
		recFail(sec, "savings_read", err.Error())
	}
}

func mPerp(ctx context.Context, cli *okxexec.Client, tra float64, posMode string, priv chan string) {
	sec := "mut"
	lever := 3.0
	if raw, err := cli.Get(ctx, "/api/v5/account/leverage-info?instId=BTC-USDT-SWAP&mgnMode=cross"); err == nil {
		if lv := f64(jget(raw, "lever")); lv > 0 {
			lever = lv
		}
	}
	var perpSym string
	var perpSz float64
	for _, cand := range []string{"BTC-USDT-SWAP", "ETH-USDT-SWAP", "SOL-USDT-SWAP", "DOGE-USDT-SWAP", "XRP-USDT-SWAP"} {
		m, err := cli.InstrumentMeta(ctx, "SWAP", cand)
		if err != nil || m.CtVal <= 0 || m.State != "live" {
			continue
		}
		tk, _ := cli.Get(ctx, "/api/v5/market/ticker?instId="+cand)
		px := f64(jget(tk, "last"))
		sz := m.MinSz
		if sz < m.LotSz {
			sz = m.LotSz
		}
		notional := sz * m.CtVal * px
		margin := notional / lever
		fmt.Printf("候选 %s: ctVal=%g minSz=%g notional≈%.2f USD 保证金≈%.2f（%.0fx）\n",
			cand, m.CtVal, sz, notional, margin, lever)
		if margin < tra*0.7 {
			perpSym, perpSz = cand, sz
			break
		}
	}
	if perpSym != "" {
		ps := ""
		if posMode == "long_short_mode" {
			ps = "long"
		}
		clid := fmt.Sprintf("vfy%d", time.Now().UnixNano()%1e12)
		or, err := cli.PlaceOrder(ctx, okxexec.OrderParams{InstID: perpSym, TdMode: "cross",
			Side: "buy", OrdType: "market", Sz: fmtAmt(perpSz), ClOrdID: clid, PosSide: ps})
		if err != nil {
			recFail(sec, "perp_open", perpSym+" "+err.Error())
		} else {
			rec(sec, "perp_open", jget(or, "ordId"), "ordId", perpSym+" 市价开 BUY", jget(or, "ordId") != "")
		}
		time.Sleep(1500 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
		st, _ := cli.ResolveOrder(ctx, perpSym, clid)
		rec(sec, "perp_open_state", st, "filled", "clOrdId 定态", st == okxexec.StateFilled)
		drainPriv(priv, "@开仓帧")
		raw, _ := cli.Get(ctx, "/api/v5/account/positions?instId="+perpSym)
		posSz := jget(raw, "pos")
		rec(sec, "perp_position", posSz, ">0", "持仓实读", f64(posSz) > 0)
		cps := ""
		if posMode == "long_short_mode" {
			cps = "long"
		}
		clid2 := fmt.Sprintf("vfx%d", time.Now().UnixNano()%1e12)
		cr, err := cli.PlaceOrder(ctx, okxexec.OrderParams{InstID: perpSym, TdMode: "cross",
			Side: "sell", OrdType: "market", Sz: fmtAmt(perpSz), ClOrdID: clid2, PosSide: cps, ReduceOnly: true})
		if err != nil {
			recFail(sec, "perp_close", err.Error())
		} else {
			rec(sec, "perp_close", jget(cr, "ordId"), "ordId", "reduceOnly 平", true)
		}
		time.Sleep(1500 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
		st2, _ := cli.ResolveOrder(ctx, perpSym, clid2)
		rec(sec, "perp_close_state", st2, "filled", "", st2 == okxexec.StateFilled)
		raw, _ = cli.Get(ctx, "/api/v5/account/positions?instId="+perpSym)
		rem := jget(raw, "pos")
		rec(sec, "perp_flat", rem, "∅/0", "平仓后实读", rem == "" || rem == "∅" || f64(rem) == 0)
		drainPriv(priv, "@平仓帧")
	} else {
		rec(sec, "perp_leg", "skip", "可付标的", "perp 腿无可付标的（候选全超闲余）", false)
	}
}
