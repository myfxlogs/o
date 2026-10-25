// trx.go —— TRX 自营盘采集臂装配（T-0108 nox 移植——只读观测面）：
// trx 链上 4 采集器（账户资源/txs/witness/apr）+cmc 3 采集器（prices/account/
// history）+WS 事件流，Poll 周期=app_config trx.poll_seconds（缺 300s fail-closed
// 跳臂）。cmc 凭据=app_secrets 密文列解（业主条件请求未供=臂让位显形不假零）。
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/data/cmc"
	"o/internal/data/trx"
	"o/internal/platform/config"
	"o/internal/platform/roster"
	"o/internal/store/pgstore"
)

// trxArms 双臂发射（T-0108：TRX 链上+CMC 纯读——缺件各自让位显形）。
func trxArms(ctx context.Context, errc chan<- error, pool *pgxpool.Pool, cfg *config.Client) {
	go trxArm(ctx, errc, pool, cfg)
	go cmcArm(ctx, errc, pool, cfg)
}

// trxArm TRX 链上采集臂（地址缺=让位不误报——业主条件请求件）。
func trxArm(ctx context.Context, errc chan<- error, pool *pgxpool.Pool, cfg *config.Client) {
	addr, err := cfg.GetString("trx.stake_address")
	if err != nil || addr == "" {
		log.Printf("[trx] stake_address 键缺/空——链上采集臂让位（业主条件请求件）")
		return
	}
	secs := 300
	if v, e := cfg.GetFloat("trx.poll_seconds"); e == nil && v > 0 {
		secs = int(v)
	}
	st := pgstore.NewStore(pool)
	apiKey := func() string { v, _ := cfg.GetString("trx.trongrid_api_key"); return v }
	accts := trx.NewAccountCollector(addr, st, "https://api.trongrid.io",
		"https://api.tronscan.org/api", apiKey)
	txs := trx.NewTxsCollector(addr, st, "https://apilist.tronscan.org/api",
		apiKey, "https://api.tronscan.org/api", nil, 0.10)
	wit := trx.NewWitnessCollector(addr, st, "https://api.tronscan.org/api", apiKey)
	apr := trx.NewAprCollector(addr, st, "https://api.tronscan.org/api")
	if err := roster.Register(ctx, pool, "TRX 链上采集", "trx",
		"账户资源+txs+witness+APR 采集岗", nil); err != nil {
		errc <- err
		return
	}
	poll := func() {
		if _, err := accts.Poll(ctx); err != nil {
			log.Printf("[trx] account 采集败: %v", err)
		}
		if _, err := txs.Poll(ctx); err != nil {
			log.Printf("[trx] txs 采集败: %v", err)
		}
		if _, err := wit.Poll(ctx); err != nil {
			log.Printf("[trx] witness 采集败: %v", err)
		}
		if _, err := apr.Poll(ctx); err != nil {
			log.Printf("[trx] apr 采集败: %v", err)
		}
	}
	go roster.BeatLoop(ctx, pool, "TRX 链上采集", 30*time.Second, errc)
	log.Printf("[trx] 链上采集臂上岗 addr=%s poll=%ds", addr, secs)
	poll()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(secs) * time.Second): // sched-exempt-ok：外部 venue 采集节拍（非周期环病——Poll 单次真读）
			poll()
		}
	}
}

// cmcArm CMC 纯读采集臂（凭据未供=让位显形——业主条件请求件 fin_condition_gap）。
func cmcArm(ctx context.Context, errc chan<- error, pool *pgxpool.Pool, cfg *config.Client) {
	// 凭据密文列解（凭据归 pg 律——app_config 禁明文凭据，T-0108 返修）。
	sec := func(k string) string {
		b, err := cfg.Secret(ctx, k)
		if err != nil {
			return ""
		}
		return string(b)
	}
	cmc.SetCreateTplHex(sec("cmc.create_tpl")) // 重登模板密件注入（nox wire_cmc.go:22 同位）
	email, pass := sec("cmc.email"), sec("cmc.password")
	acctS, _ := cfg.GetString("cmc.account_id") // 非密装配键（账户号非凭据）
	acct, _ := strconv.ParseInt(acctS, 10, 64)
	if email == "" || pass == "" || acct == 0 {
		log.Printf("[cmc] 凭据键缺——采集臂让位（业主条件请求件——对冲腿显读件缺）")
		return
	}
	st := pgstore.NewStore(pool)
	auth := cmc.NewAuthManager(email, pass, acct, cmc.DefaultPlatformBase)
	go auth.StartRefresh(ctx)
	if err := roster.Register(ctx, pool, "CMC CFD 采集", "cmc", "prices/account/history 纯读采集岗", nil); err != nil {
		errc <- err
		return
	}
	go roster.BeatLoop(ctx, pool, "CMC CFD 采集", 30*time.Second, errc)
	prices := cmc.NewPricesCollector(auth, "X-AFQUL", st, cmc.DefaultRestBase)
	accts := cmc.NewAccountCollector(auth, acct, st, cmc.DefaultRestBase)
	hist := cmc.NewHistoryCollector(auth, acct, st, cmc.DefaultRestBase)
	poll := func() {
		if err := auth.Login(ctx); err != nil {
			log.Printf("[cmc] login 败（本轮采集让位）: %v", err)
			return
		}
		if _, err := prices.Poll(ctx); err != nil {
			log.Printf("[cmc] prices 采集败: %v", err)
		}
		if _, err := accts.Poll(ctx); err != nil {
			log.Printf("[cmc] account 采集败: %v", err)
		}
		if _, err := hist.Poll(ctx); err != nil {
			log.Printf("[cmc] history 采集败: %v", err)
		}
	}
	log.Printf("[cmc] 纯读采集臂上岗 acct=%d", acct)
	poll()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute): // sched-exempt-ok：CMC REST 采集节拍（venue 限频礼貌）
			poll()
		}
	}
}

// mountTrxAPI /api/trx/* 只读端点（D-629：自营盘业绩外——数据面独挂不塞
// Overview/nav_total）。读件缺=null 显形不造零。
func mountTrxAPI(mux *http.ServeMux, pool *pgxpool.Pool) {
	get := func(path string, q string) {
		mux.HandleFunc("/api/trx/"+path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var raw json.RawMessage
			if err := pool.QueryRow(r.Context(), q).Scan(&raw); err != nil || raw == nil {
				raw = json.RawMessage("null")
			}
			_, _ = w.Write(raw)
		})
	}
	get("account", `SELECT row_to_json(t) FROM (SELECT address,votes_total,sr_address,
		balance_sun,allowance_sun,energy_limit,energy_used,net_limit,net_used,
		free_net_used,free_net_limit,energy_delegated,bandwidth_staked,
		tron_power,reward_num_sun,fetched_at FROM fin_trx_account
		ORDER BY fetched_at DESC LIMIT 1) t`)
	get("txs", `SELECT COALESCE(json_agg(t ORDER BY t.ts DESC),'[]') FROM
		(SELECT tx_id,ts,kind,amount_sun,counterparty FROM fin_trx_txs
		ORDER BY ts DESC LIMIT 30) t`)
	get("radar", `SELECT COALESCE(json_agg(t ORDER BY t.apr DESC),'[]') FROM
		(SELECT sr_address,apr FROM fin_trx_vote_apr_top ORDER BY apr DESC LIMIT 27) t`)
	get("vote", `SELECT row_to_json(t) FROM (SELECT current_rate,current_sr,
		max_rate,max_sr,spread,min_rate FROM fin_trx_vote_apr
		ORDER BY fetched_at DESC LIMIT 1) t`)
	get("cmc", `SELECT row_to_json(t) FROM (SELECT account_id,total_account_value,
		total_cash_balance,total_unrealised_pnl,distance_to_closeout_pct,fetched_at
		FROM fin_cmc_account_summary ORDER BY fetched_at DESC LIMIT 1) t`)
	get("cmc_positions", `SELECT COALESCE(json_agg(t),'[]') FROM
		(SELECT instrument_id,direction,quantity,entry_price,unrealised_pnl,
		margin_percentage,fetched_at FROM fin_cmc_positions
		ORDER BY fetched_at DESC LIMIT 20) t`)
}
