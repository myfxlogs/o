// trx.go —— TRX 自营盘采集臂装配（T-0108 nox 移植；T-0110 整品升级——
// 独立页+全功能读面+trxvote 引擎旗歇）。trx 链上 4 采集器+cmc 采集+WS 事件流，
// Poll 周期=app_config trx.poll_seconds（缺 300s fail-closed 跳臂）。凭据=
// app_secrets 密文列解（业主条件请求未供=臂让位显形不假零）。D-629：全面
// 「业绩外·自营盘」——数据面独挂不塞 Overview/nav_total。
package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/data/cmc"
	"o/internal/data/trx"
	"o/internal/platform/condreq"
	"o/internal/platform/config"
	"o/internal/platform/roster"
	"o/internal/store/pgstore"
)

// trxWiring 采集件共享装配（T-0110——采集臂与读面/刷新踢/投票引擎同源单例；
// nil 件=让位显形不假零）。
type trxWiring struct {
	pool        *pgxpool.Pool
	cfg         *config.Client
	st          *pgstore.Store
	accts       *trx.AccountCollector
	txs         *trx.TxsCollector
	wit         *trx.WitnessCollector
	apr         *trx.AprCollector
	cmcLive     func() cmc.LiveState // nil=WS 未接线（freshLive DB 兜底径）
	cmcWsHealth func() bool          // nil=false（REST 兜底显形）
}

// trxArms 双臂发射+装配返回（T-0110：wiring 供读面/刷新踢/引擎同源消费）。
func trxArms(ctx context.Context, errc chan<- error, pool *pgxpool.Pool, cfg *config.Client) *trxWiring {
	w := &trxWiring{pool: pool, cfg: cfg, st: pgstore.NewStore(pool)}
	go trxArm(ctx, errc, w)
	go cmcArm(ctx, errc, w)
	go trxvoteArm(ctx, errc, w) // T-0110 旗歇引擎（trx.vote_enabled=false 停驶待令）
	return w
}

// trxArm TRX 链上采集臂（地址缺=让位不误报——业主条件请求件）。
func trxArm(ctx context.Context, errc chan<- error, w *trxWiring) {
	addr, err := w.cfg.GetString("trx.stake_address")
	if err != nil || addr == "" {
		log.Printf("[trx] stake_address 键缺/空——链上采集臂让位（业主条件请求件）")
		return
	}
	secs := 300
	if v, e := w.cfg.GetFloat("trx.poll_seconds"); e == nil && v > 0 {
		secs = int(v)
	}
	apiKey := func() string { v, _ := w.cfg.GetString("trx.trongrid_api_key"); return v }
	w.accts = trx.NewAccountCollector(addr, w.st, "https://api.trongrid.io",
		"https://apilist.tronscan.org", apiKey)
	// T-0182 凭据语义错配根修：/v1//wallet 径=TronGrid 面（TRON-PRO-API-KEY
	// 成对）；tronscan /api/ 径=公开面零凭据（trongrid key 灌 tronscan=401 间歇
	// 实证病灶——凭据语义错配典型坑）。witness/apr 基址同律纠。
	w.txs = trx.NewTxsCollector(addr, w.st, "https://api.trongrid.io",
		apiKey, "https://apilist.tronscan.org", nil, 0.10)
	w.wit = trx.NewWitnessCollector(addr, w.st, "https://api.trongrid.io", apiKey)
	w.apr = trx.NewAprCollector(addr, w.st, "https://apilist.tronscan.org")
	if err := roster.Register(ctx, w.pool, "TRX 链上采集", "trx",
		"账户资源+txs+witness+APR 采集岗", nil); err != nil {
		errc <- err
		return
	}
	poll := func() {
		if _, err := w.accts.Poll(ctx); err != nil {
			log.Printf("[trx] account 采集败: %v", err)
		}
		if _, err := w.txs.Poll(ctx); err != nil {
			log.Printf("[trx] txs 采集败: %v", err)
		}
		if _, err := w.wit.Poll(ctx); err != nil {
			log.Printf("[trx] witness 采集败: %v", err)
		}
		if _, err := w.apr.Poll(ctx); err != nil {
			log.Printf("[trx] apr 采集败: %v", err)
		}
		trxRollup(ctx, w) // T-0110：日终 rollup（幂等 ON CONFLICT(day)——随拍补当日）
	}
	go roster.BeatLoop(ctx, w.pool, "TRX 链上采集", 30*time.Second, errc)
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
func cmcArm(ctx context.Context, errc chan<- error, w *trxWiring) {
	// 凭据密文列解（凭据归 pg 律——app_config 禁明文凭据，T-0108 返修）。
	sec := func(k string) string {
		b, err := w.cfg.Secret(ctx, k)
		if err != nil {
			return ""
		}
		return string(b)
	}
	cmc.SetCreateTplHex(sec("cmc.create_tpl")) // 重登模板密件注入（nox wire_cmc.go:22 同位）
	email, pass := sec("cmc.email"), sec("cmc.password")
	acctS, _ := w.cfg.GetString("cmc.account_id") // 非密装配键（账户号非凭据）
	acct, _ := strconv.ParseInt(acctS, 10, 64)
	if email == "" || pass == "" || acct == 0 {
		log.Printf("[cmc] 凭据键缺——采集臂让位（业主条件请求件——对冲腿显读件缺）")
		_ = condreq.Request(ctx, w.pool, "credential", "cmc", "cmc.email+cmc.password+cmc.account_id",
			`{"missing":"凭据/账户号三件","impact":"CMC 对冲腿采集臂停摆"}`)
		return
	}
	pool, st := w.pool, w.st
	auth := cmc.NewAuthManager(email, pass, acct, cmc.DefaultPlatformBase)
	go auth.StartRefresh(ctx)
	// WS 事件流（T-0110：LiveState 内存态=对冲结构 freshLive 主源；ws_disabled
	// 热启回退全 REST——nox wire_cmc 同语义）。
	wsOff, _ := w.cfg.GetBool("cmc.ws_disabled")
	if !wsOff {
		ws := cmc.NewWsClient(auth, acct, st, cmc.DefaultWsBase, cmc.DefaultRestBase)
		w.cmcLive = ws.LiveState
		w.cmcWsHealth = ws.IsHealthy
		go ws.Run(ctx) // sched-exempt-ok：WS 事件流（有流用流——断流自愈循环）
		log.Printf("[cmc] WS 事件流接线（LiveState 内存态供数）")
	}
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
