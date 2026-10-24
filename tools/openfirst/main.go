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
	mt5pb "phase/internal/gen/mtapi/mt5"
	"phase/internal/mt5"
	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/risk"
	"phase/internal/trading/execution"
	"phase/internal/trading/mtexec"
	"phase/internal/trading/okxexec"
)

func main() {
	sid := flag.Int64("strategy", 0, "strategy_id（缺省=最新 probe/live 非 retired 行）")
	dry := flag.Bool("dry", false, "干跑——构 Intent+Check 不执行")
	structure := flag.String("structure", "funding", "腿构：funding（同 venue 双腿）|cfd_perp（okx perp 卖+mt5 cfd 买）")
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
	// Earn 赎回件装配同产线（T-0037——交易户 USDT≈0 实勘，不挂=保证金必拒；
	// 两结构通用：赎回只动 OKX Earn，cfd_perp 的 mt5 腿保证金在 MT5 户内）。
	earnW := earn.NewWorker(pool, cfg)
	earnW.AttachVenue(cli)
	var execR *execution.Runner
	var legs [2]execution.Leg
	var spotInst string
	if *structure == "cfd_perp" {
		// cfd_perp：leg1=okx SWAP 卖（ctVal 系）+leg2=mt5 cfd 买（lot 换算
		// SymbolParams→CfdMeta——venue 元数据缺=fail-closed）。
		mt := mtConnect(ctx, pool, cfg, mk)
		cfdInst, err := cfg.GetString("exec.cfd_map." + instID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "exec.cfd_map 无映射 fail-closed:", instID)
			os.Exit(1)
		}
		pm, err := cli.InstrumentMeta(ctx, "SWAP", instID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "perp 元数据:", err)
			os.Exit(1)
		}
		sp, err := mt.MT5().SymbolParams(mt.Authed(ctx), &mt5pb.SymbolParamsRequest{Symbol: cfdInst})
		if err != nil || sp.GetResult() == nil {
			fmt.Fprintln(os.Stderr, "cfd 元数据:", err)
			os.Exit(1)
		}
		sgi, sg := sp.GetResult().GetSymbolInfo(), sp.GetResult().GetSymbolGroup()
		px, err := execution.LatestPx(ctx, pool, spotInstCached(instID))
		if err != nil {
			fmt.Fprintln(os.Stderr, "px:", err)
			os.Exit(1)
		}
		legs, spotInst, err = execution.BuildCfdLegs(instID, cfdInst, notional,
			execution.CfdMeta{ContractSize: sgi.GetContractSize(),
				MinLot: sg.GetMinLots(), LotStep: sg.GetLotsStep()}, px, pm.CtVal)
		if err != nil {
			fmt.Fprintln(os.Stderr, "legs:", err)
			os.Exit(1)
		}
		execR = execution.NewRunner(pool, cfg, execution.RouterExecutor{
			"okx": execution.OkxExecutor{C: cli},
			"mt5": mtexec.Executor{C: mt},
		}).WithEarn(earnW)
	} else {
		execR = execution.NewRunner(pool, cfg, execution.OkxExecutor{C: cli}).WithEarn(earnW)
		// 双腿乐器派生+sz 换算（leg1 SWAP 卖/leg2 派生现货买；minSz 闸 fail-closed）。
		var lerr error
		legs, spotInst, lerr = execution.BuildLegs(ctx, pool, func(ctx context.Context, iid string) (execution.InstMeta, error) {
			it := "SWAP"
			if iid == spotInstCached(instID) {
				it = "SPOT"
			}
			m, err := cli.InstrumentMeta(ctx, it, iid)
			return execution.InstMeta{CtVal: m.CtVal, CtValCcy: m.CtValCcy, LotSz: m.LotSz, MinSz: m.MinSz}, err
		}, venue, instID, notional)
		if lerr != nil {
			fmt.Fprintln(os.Stderr, "legs:", lerr)
			os.Exit(1)
		}
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

// mtConnect mt5 真户会话接线（凭据自 app_secrets mt5.*；gateway_addr=config 键；
// 凭据缺=fail-closed——未连成功不构腿）。
func mtConnect(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client, mk []byte) *mt5.Client {
	get := func(k string) string {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			fmt.Fprintln(os.Stderr, k, err)
			os.Exit(1)
		}
		p, _ := cryptoutil.Open(mk, ct)
		return p
	}
	addr, err := cfg.GetString("mt5.gateway_addr")
	if err != nil {
		fmt.Fprintln(os.Stderr, "mt5.gateway_addr:", err)
		os.Exit(1)
	}
	cli, err := mt5.Dial(addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mt5 dial:", err)
		os.Exit(1)
	}
	var login uint64
	if _, err := fmt.Sscanf(get("mt5.login"), "%d", &login); err != nil {
		fmt.Fprintln(os.Stderr, "mt5.login 非数字:", err)
		os.Exit(1)
	}
	if err := cli.Connect(ctx, mt5.Credentials{Login: login,
		Password: get("mt5.password"), Server: get("mt5.server"), Host: get("mt5.host")}); err != nil {
		fmt.Fprintln(os.Stderr, "mt5 connect:", err)
		os.Exit(1)
	}
	return cli
}
