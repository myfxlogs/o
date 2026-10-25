// trxvote.go —— TRX 投票收益优化器装配（T-0110 nox wire_trxvote 移植——
// **旗歇态**：trx.vote_enabled 默认 false→构造即让位显形（「投票引擎已迁移·
// 停驶待令」），翻 true 方上链（配置翻位=业主显式授权点，留账）；私钥=
// app_secrets trx.vote_private_key 眠置（旗未启零触零读）。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"o/internal/platform/safego"
	"o/internal/portfolio/trxvote"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// trxvoteArm 投票引擎臂（旗歇守卫第一行——禁旗未启读私钥/连链）。
func trxvoteArm(ctx context.Context, errc chan<- error, w *trxWiring) {
	enabled, err := w.cfg.GetBool("trx.vote_enabled")
	if err != nil || !enabled {
		log.Printf("[trxvote] 投票引擎已迁移·停驶待令（trx.vote_enabled=false——翻 true 方上链，私钥眠置零触）")
		return
	}
	log.Printf("[trxvote] trx.vote_enabled=true——引擎启驶（业主显式授权点已留账）")
	sched := buildTrxVoteOptimizer(ctx, w)
	if sched == nil {
		return
	}
	// D-137①④：reward 差分 → KickBoundary + APR 重取；自发 tx 确认 → 重取。
	if w.accts != nil {
		w.accts.SetOnRewardDelta(func(ctx context.Context, rewardNumSun, delta int64) {
			log.Printf("[trxvote] reward delta %+d sun = maintenance boundary → kick", delta)
			sched.KickBoundary("reward_delta")
			if w.apr != nil {
				w.apr.TriggerNow(ctx)
			}
		})
		w.accts.SetOnAccountFetched(sched.ClaimCheck)
	}
	sched.SetOnEvaluated(func(ctx context.Context) {
		if w.apr != nil {
			w.apr.TriggerNow(ctx)
		}
	})
	sched.SetOnSelfTxConfirmed(func(ctx context.Context) {
		if w.accts != nil {
			_, _ = w.accts.Poll(ctx)
		}
		if w.apr != nil {
			w.apr.TriggerNow(ctx)
		}
		if w.txs != nil {
			_, _ = w.txs.Poll(ctx)
		}
	})
	safego.Go("trxvote", func() { sched.Run(ctx) })
}

// buildTrxVoteOptimizer 引擎装配（凭据缺/私钥坏/连接败=降级 nil 显形）。
func buildTrxVoteOptimizer(ctx context.Context, w *trxWiring) *trxvote.Scheduler {
	degrade := func(msg string) *trxvote.Scheduler {
		log.Printf("ALERT %s", msg)
		return nil
	}
	privKeyHex := ""
	if b, err := w.cfg.Secret(ctx, "trx.vote_private_key"); err == nil {
		privKeyHex = string(b)
	}
	addr, _ := w.cfg.GetString("trx.stake_address")
	switch {
	case privKeyHex == "":
		return degrade("[trxvote] trx.vote_private_key 未设置——引擎不启")
	case addr == "":
		return degrade("[trxvote] trx.stake_address 未设置——引擎不启")
	}
	privKey, err := keys.GetPrivateKeyFromHex(privKeyHex)
	if err != nil {
		return degrade(fmt.Sprintf("[trxvote] 私钥解析失败: %v（引擎不启——键值永不落日志）", err))
	}
	grpcEndpoint := "grpc.trongrid.io:50051"
	if v, err := w.cfg.GetString("trx.grpc_endpoint"); err == nil && v != "" {
		grpcEndpoint = v
	}
	conn := client.NewGrpcClient(grpcEndpoint)
	if err := conn.Start(grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		return degrade(fmt.Sprintf("[trxvote] TRON gRPC 连接败: %v（引擎不启）", err))
	}
	if apiKey, _ := w.cfg.GetString("trx.trongrid_api_key"); apiKey != "" {
		conn.SetAPIKey(apiKey)
	}
	executor := trxvote.NewExecutor(conn, privKey, addr, func(level, msg string) {
		log.Printf("ALERT [trxvote/%s] %s", level, msg)
	})
	tronscanBase := "https://api.tronscan.org/api"
	rebalanceCfg := trxvote.RebalanceConfig{
		MinSRs:             cfgInt(w, "trx.vote_min_srs", 3),
		MaxSRs:             cfgInt(w, "trx.vote_max_srs", 3),
		MaxBrokerage:       cfgInt(w, "trx.vote_max_brokerage", 20),
		MinRank:            127,
		YieldDropPct:       cfgFloat(w, "trx.vote_yield_drop_pct", 15.0),
		BetterOppPct:       cfgFloat(w, "trx.vote_better_opp_pct", 0.0),
		MaxRebalancePerDay: cfgInt(w, "trx.vote_max_rebal_per_day", 1),
		MinInterval:        time.Duration(cfgFloat(w, "trx.vote_min_interval_min", 120)) * time.Minute,
		SpreadThreshold:    cfgFloat(w, "trx.vote_spread_threshold", 0.001),
	}
	sched := trxvote.NewScheduler(executor, &trxvote.PgStoreAdapter{Store: w.st}, addr, tronscanBase,
		rebalanceCfg, trxvote.NewFallbackAccountFetcher(
			trxvote.NewTronscanAccountFetcher(tronscanBase),
			trxvote.NewGRPCAccountFetcher(conn)))
	sched.SetNextMaintenanceSource(func(ctx context.Context) (time.Time, error) {
		nm, err := conn.GetNextMaintenanceTimeCtx(ctx)
		if err != nil {
			return time.Time{}, err
		}
		return time.UnixMilli(nm.GetNum()), nil
	})
	return sched
}

func cfgInt(w *trxWiring, k string, def int) int {
	if v, err := w.cfg.GetFloat(k); err == nil && v > 0 {
		return int(v)
	}
	return def
}

func cfgFloat(w *trxWiring, k string, def float64) float64 {
	if v, err := w.cfg.GetFloat(k); err == nil && v > 0 {
		return v
	}
	return def
}
