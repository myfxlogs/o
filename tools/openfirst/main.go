// openfirst —— T-0033 运维触发件：人工显式触发单次开仓流（probe/live 策略
// →open Intent→risk.Check→token→Execute）。全程走既有闸径不绕一件
// （live_switch/hedge_band/Earn 赎回/双腿原子——与自动径同码）。
// 用法：DB_DSN=… PHASE_MASTER_KEY_FILE=… openfirst [-strategy <id>] [-dry]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/earn"
	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/risk"
	"phase/internal/trading/execution"
	"phase/internal/trading/okxexec"
)

func main() {
	sid := flag.Int64("strategy", 0, "strategy_id（缺省=最新 probe/live 非 retired 行）")
	dry := flag.Bool("dry", false, "干跑——构 Intent+Check 不执行")
	flag.Parse()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	mk, err := config.ReadMasterKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "cfg:", err)
		os.Exit(1)
	}
	// 策略行：-strategy 显式 or 最新 probe/live（非 retired）。
	var instID string
	var notional float64
	var venue string
	q := `SELECT inst_id,notional_usd,venue FROM fin_strategy WHERE id=$1 AND state IN ('probe','live')`
	args := []any{*sid}
	if *sid == 0 {
		q = `SELECT inst_id,notional_usd,venue FROM fin_strategy
			WHERE state IN ('probe','live') ORDER BY id DESC LIMIT 1`
		args = nil
	}
	if err := pool.QueryRow(ctx, q, args...).Scan(&instID, &notional, &venue); err != nil {
		fmt.Fprintln(os.Stderr, "strategy:", err)
		os.Exit(1)
	}
	// 双帽取小：min(strategy.notional, probe_notional, max_notional)。
	cap1, e1 := cfg.GetFloat("exec.probe_notional_usd")
	cap2, e2 := cfg.GetFloat("risk.max_notional_usd")
	if e1 != nil || e2 != nil {
		fmt.Fprintln(os.Stderr, "cap 键缺 fail-closed:", e1, e2)
		os.Exit(1)
	}
	if cap1 < notional {
		notional = cap1
	}
	if cap2 < notional {
		notional = cap2
	}
	if notional <= 0 {
		fmt.Fprintln(os.Stderr, "notional=0 拒")
		os.Exit(1)
	}
	gate := risk.NewGate(pool, cfg)
	in := risk.Intent{StrategyRef: *sid, Venue: venue, InstID: instID,
		Legs: []string{"perp_short", "spot_long"}, NotionalUSD: notional, Leverage: 1, Kind: "open"}
	dec := gate.Check(ctx, in)
	fmt.Printf("decision=%d allow=%v reason=%s\n", dec.DecisionID, dec.Allow, dec.Reason)
	if !dec.Allow || *dry {
		return
	}
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
	cli := okxexec.NewClient(cred, false)
	// T-0037 追加：Earn 赎回件装配同产线（cmd/core WithEarn 径）——交易户实勘
	// USDT≈0（资金全泊 Earn），不挂=首单保证金必拒；赎回额=notional（buffer 键面
	// 归 earn 包，fail-closed 律同产线）。
	earnW := earn.NewWorker(pool, cfg)
	earnW.AttachVenue(cli)
	execR := execution.NewRunner(pool, cfg, execution.OkxExecutor{C: cli}).WithEarn(earnW)
	fmt.Printf("earn 赎回径装配 OK（WithEarn——首单前 RedeemToTrading 生效）\n")
	// 双腿乐器派生+sz 换算（leg1 SWAP 卖/leg2 派生现货买；minSz 闸 fail-closed）。
	legs, spotInst, err := execution.BuildLegs(ctx, pool, func(ctx context.Context, iid string) (execution.InstMeta, error) {
		it := "SWAP"
		if iid == spotInstCached(instID) {
			it = "SPOT"
		}
		m, err := cli.InstrumentMeta(ctx, it, iid)
		return execution.InstMeta{CtVal: m.CtVal, LotSz: m.LotSz, MinSz: m.MinSz}, err
	}, venue, instID, notional)
	if err != nil {
		fmt.Fprintln(os.Stderr, "legs:", err)
		os.Exit(1)
	}
	fmt.Printf("legs: %s sell %s + %s buy %s\n", instID, legs[0].Sz, spotInst, legs[1].Sz)
	eid, err := execR.Execute(ctx, execution.Intent{
		DecisionID: dec.DecisionID, StrategyRef: *sid, InstID: instID, NotionalUSD: notional,
		Legs: legs})
	if err != nil {
		fmt.Fprintln(os.Stderr, "execute:", err, "exec_id=", eid)
		os.Exit(1)
	}
	fmt.Printf("EXEC-OK exec_id=%d notional=%.2f inst=%s\n", eid, notional, instID)
}

func spotInstCached(instID string) string {
	spot, _ := execution.DeriveSpotInst(instID)
	return spot
}
