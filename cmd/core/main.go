// cmd/core —— O 单二进制（cmd/core，架构约束 §1 装配钉）。
// boot 序：DB_DSN 连库 → 迁移 → config client（Load+Watch）→ 花名册 Register+Beat
// （运行时本体=1 号员工 o-core）→ :50051 ConnectRPC+健康端点。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"o/internal/data/okxpriv"
	"o/internal/data/okxws"
	"o/internal/earn"
	"o/internal/guard"
	"o/internal/ledger"
	"o/internal/ledger/holding"
	"o/internal/ledger/report"
	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/platform/netmeter"
	"o/internal/platform/notify"
	"o/internal/platform/pgguard"
	"o/internal/platform/roster"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/store/pgstore"
	"o/internal/trading/execution"
	"o/internal/trading/okxexec"
)

const (
	employeeName  = "o-core"
	employeeDept  = "platform"
	employeeTitle = "运行时本体（1 号员工）"
	beatEvery     = 30 * time.Second
	listenAddr    = ":50051"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("core boot FAIL: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return errors.New("DB_DSN 未设（唯一 bootstrap env——缺件 fail-closed）")
	}
	pool, err := pgstore.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil { // SG=钱径域（D-0030 §三）选 fin 链
		return err
	}

	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		return fmt.Errorf("config 首载：%w", err)
	}
	errc := make(chan error, 8)
	go func() {
		if err := cfg.Watch(ctx); err != nil && ctx.Err() == nil {
			errc <- fmt.Errorf("config watch: %w", err)
		}
	}()

	if err := roster.Register(ctx, pool, employeeName, employeeDept, employeeTitle, nil); err != nil {
		return err
	}
	go roster.BeatLoop(ctx, pool, employeeName, beatEvery, errc)
	go func() {
		for err := range errc {
			log.Printf("WARN: %v", err)
		}
	}()

	// P2 首岗：OKX 公开行情采集（员工「行情巡」）——标的集自 app_config 热启，
	// 键缺席=采集岗待命自轮询不炸 boot（配置驱动装配，非 env 驱动）。
	// 带宽计量件（T-0057：net.budget_kbps 键缺=闸缺席计量仍行——fail-open 计量面；
	// alerter 声明后置位——闭包延迟调 nil 安全）。
	var meter *netmeter.Meter
	var okcli *okxexec.Client
	collector := okxws.NewCollector(pool, cfg)
	collector.ExpandSymbols = universeExpander(func() *okxexec.Client { return okcli }, pool)
	go collector.Run(ctx, errc)

	// 私流守：OKX 私有 WS（凭据自 app_secrets 密文列解；缺件 fail-closed
	// 不炸 boot——E-1 未落位期岗位缺席留痕非死码）。
	// 凭据缺席期仓巡平仓缝降级 stub（拒执 fail-closed）——凭据落位重启即全链。
	var closer guard.Executor
	// Earn 停泊态+feishu 告警通道（凭据就位挂接——boot_secrets.go）。（boot_secrets.go——app_secrets 密文列解，缺件=nil 降级；
	// typed-nil 主修 T-0032：凭据缺席传真 nil 接口非 (*FeishuClient)(nil)）。
	var sender notify.Sender
	if feishuCli := feishuSender(ctx, pool); feishuCli != nil {
		sender = feishuCli
	}
	alerter := notify.NewAlerter(sender, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.enabled"); return v })
	{
		budgetKB, _ := cfg.GetInt("net.budget_kbps")
		meter = netmeter.New(int(budgetKB), func(ctx context.Context, kind, msg string) {
			alerter.Alert(ctx, kind, msg)
		})
		// T-0130 序位修：meter 造成后才挂（原 AttachMeter(nil)——计量面死挂实证）。
		collector.AttachMeter(meter)
	}
	startEventPush(ctx, errc, pool, cfg, sender)                                   // 经营事件报（T-0061——event_push.go）
	collector.AttachAlerter(alerter)                                               // T-0128 返修②：面死告警 humanize kind
	collector.AttachSpotFeed(spotFeedFor(func() *okxexec.Client { return okcli })) // T-0134 现货 REST 轮询读件
	collector.AttachMarkFeed(markFeedFor(func() *okxexec.Client { return okcli })) // T-0135 mark-price REST 轮询读件
	go killSwitchNotifier(ctx, pool, cfg, alerter)
	go startRetention(ctx, errc, pool, alerter) // T-0153 清道夫日拍
	riskGate := risk.NewGate(pool, cfg)
	// T-0104/D-0013：节点角色闸（noderole.go）——只读镜像写径不可达。
	ro := nodeReadonly()
	newCli := cliFactory(ro)
	earnW := earn.NewWorker(pool, cfg)
	if !ro {
		go earnW.Run(ctx, errc)
	}
	var closeSizer guard.VenueSizer // 平仓量现读面（priv 域接线 T-0046②）
	var liveOpener lifecycle.Opener // live 开仓缝（priv 域接线 T-0050）
	if priv, err := okxpriv.NewCollector(ctx, pool, false); err != nil {
		log.Printf("WARN: 私流守缺席（%v）——凭据落位后重启即上岗", err)
	} else {
		okcli = newCli(priv.Credentials())
		if ro { // 只读件已注——mutating 接线全撤：归口/执事/开平仓缝不挂。
		} else {
			earnW.AttachVenue(okcli)
			earnW.AttachAlerter(alerter)
			// venue 现读件（T-0048：决策/判定/对账时点实读——快照行全域出局）。
			// T-0048 返修②③：闲余=account 帧 details availBal（交易户口径——
			// depositIdle 自 AcctTrading 摆钱；push-on-change 帧龄不判陈）。
			earnW.AttachIdle(func(ctx context.Context) (float64, error) {
				return okxAvailFromEvents(ctx, pool, okcli)
			})
			earnW.AttachIdleFunding(func(ctx context.Context) (float64, error) { // T-0149 ②
				return okxFundingAvailUSDT(ctx, okcli)
			})
			earnW.AttachCoins(okxCoinsFor(pool, okcli)) // T-0095 v3 残币归口判据源
			// 交易域首岗：执事（Executor=okxexec 唯一 venue 径——凭据与私流守同源解出）。
			execR := execution.NewRunner(pool, cfg,
				&execution.OkxExecutor{C: okxexec.NewClient(priv.Credentials(), false)}).WithEarn(earnW).WithAlerter(alerter)
			go execution.Run(ctx, errc, execR)
			closer = closeExec{r: execR}
			// live 开仓缝（T-0050——gate+执事+元数据件同源接线）。
			liveOpener = hedgeOpener{pool: pool, cfg: cfg, gate: riskGate, r: execR, // T-0050
				cli: newCli(priv.Credentials())}
		} // !ro mutating 接线域收
		// 读径面（只读镜像全通）：boot 首读播种+平仓量现读+funding 记账+私流守。
		creds := priv.Credentials()
		priv.SetBootFn(func(ctx context.Context) (map[string]float64, error) {
			return okxexec.BalanceEquity(ctx, creds, false)
		})
		// 平仓量现读面（T-0046 ②——priv 凭据域内接线）。
		closeSizer = okxSizer{newCli(priv.Credentials())}
		// funding 记账（T-0044）：余额帧事件驱动 bills→fin_cashflow（bills.go）。
		priv.SetBillsHook(ledgerHook(pool, func() *okxexec.Client { return newCli(priv.Credentials()) }))
		go priv.Run(ctx, errc)

		// 台账同步：资产端点采（okxexec 装配注入——账本域不反向依赖交易域；
		// /asset/balances+earn 面=钦定 REST 保留面非轮询病灶）。
		okcli := newCli(priv.Credentials())
		hs := holding.NewSyncer(pool, func(ctx context.Context) ([]holding.AssetRow, error) {
			var out []holding.AssetRow
			bals, err := okcli.AssetBalances(ctx)
			if err != nil {
				return nil, err
			}
			for _, b := range bals {
				amt := okxexec.ParseF(b.Bal)
				if amt == 0 {
					continue
				}
				out = append(out, holding.AssetRow{Ccy: b.Ccy, Amount: amt,
					Kind: "spot_dust", VenueRef: "asset",
					ValRef: map[string]any{"avail": b.AvailBal, "frozen": b.FrozenBal}})
			}
			if sav, err := okcli.SavingsBalance(ctx); err == nil {
				var rows []struct {
					Ccy string `json:"ccy"`
					Amt string `json:"amt"`
				}
				if json.Unmarshal(sav, &rows) == nil {
					for _, r2 := range rows {
						out = append(out, holding.AssetRow{Ccy: r2.Ccy,
							Amount: okxexec.ParseF(r2.Amt), Kind: "earn_flexible",
							VenueRef: "savings", Product: "savings"})
					}
				}
			}
			return out, nil
		})
		go hs.Run(ctx, errc)

	}

	// MT5 数据面只读接入（员工 汇市巡——T-0031）：凭据缺席 fail-closed WARN 不炸 boot。
	if m5, err := mt5.NewCollector(ctx, pool, cfg); err != nil {
		log.Printf("WARN: 汇市巡缺席（%v）——mt5 凭据落位后重启即上岗", err)
	} else {
		m5.AttachMeter(meter)
		go m5.Run(ctx, errc)
	}

	// T-0165 返修①②：研究臂外置 cmd/research；xmin 哨兵留本机（守 PG 非守臂）。
	xs := pgguard.NewSentinel(pool)
	xs.AttachAlerter(alerter)
	go xs.Run(ctx)
	trxW := trxArms(ctx, errc, pool, cfg) // T-0110 TRX/CMC 采集臂+读面装配（trx.go）

	// 投资域首岗：命官——paper→probe→live 开仓消费（T-0050 Opener 缝）。
	lr := lifecycle.NewRunner(pool, cfg)
	if liveOpener != nil {
		lr.AttachOpener(liveOpener)
	}
	go lr.Run(ctx, errc)

	// T-0165：复盘域随迁开发机（o-research——SG 零研究行）。

	// 账本域：司账+核账（入账结算+对账环——消费事实行不触发钱动作）。
	// T-0165 返修③：report 留 SG（钱径读写面 app_secrets/master key/
	// fin_cashflow——票面第5条明令只迁纯研究面）。
	// 采集观测源束（T-0170——collector/meter 方法值闭包，GetSystemHealth 供数）。
	md := &mdStats{frames: collector.FrameCount, lastFrame: collector.LastFrame,
		rejects: collector.Stats.Snapshot, netKbps: meter.RateKbps, netChannels: meter.Channels}
	svc, navFn := assemble(pool, fundingLiveFor(okcli), earnLiveFor(okcli),
		mt5LiveFor(pool, cfg), dcdLiveFor(okcli), trxW, md)
	var pusher *report.Pusher
	{
		mk, _ := config.ReadMasterKey()
		pusher = report.NewPusher(pool, cfg, mk)
		go report.NewRunner(pool, pusher).WithConfig(cfg).WithNav(navFn).Run(ctx, errc)
	}
	ledgerR := ledger.NewRunner(pool, cfg).WithAlerter(alerter)
	go ledgerR.Run(ctx, errc)

	// 风控域首岗：门官——唯一交易前闸，只认放行令牌。
	go risk.Run(ctx, errc, riskGate)

	// 看护域首岗：仓巡（G8）——三触发+联动，close 免拦径，双腿平走 G7
	//（凭据缺席=缝降级 stub 拒执 fail-closed）。
	wd := guard.NewWarden(pool, cfg, riskGate, closer)
	// 平仓量 venue 现读面（T-0046 ②——availBal/positions 实读，禁抄腿行委托量；
	// 凭据缺席=不挂=平仓 fail-closed 不假量）。
	if closeSizer != nil {
		wd.AttachSizer(closeSizer) // T-0046②：平仓量现读件
	}
	wd.AttachCtVal(ctValReader(okcli)) // T-0046①：面值现读件（venue_readers.go）
	wd.AttachAlerter(alerter)
	// venue 现读件（T-0048：快照行全域出局——venue_readers.go 装配）。
	attachVenueReaders(riskGate, wd, ledgerR, pool, okcli,
		func(ctx context.Context) (*mt5.Client, error) { return mt5.OpenSession(ctx, pool, cfg) })
	go wd.Run(ctx, errc)

	// 总览活页（T-0067：service.go mountUI——SPA 静态面；数据面唯一入口=
	// CoreService Connect，活推=Watch 流帧——T-0156 REST/SSE 双轨面撤）。
	mux := http.NewServeMux()
	mountUI(mux)
	// T-0086 ①④：资金户/Earn 活读件注入（venue_readers.go——活读件注入）。
	// T-0096：MT5 户活读件同注（OpenSession 现读——凭据缺席=读件缺不假零）。
	// T-0110：TRX 自营盘读面同 server 注册（trxW 同源单例）。
	mountCore(mux, svc)

	srv := &http.Server{Addr: listenAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
	}()
	log.Printf("o-core up :%s（employee=%s 已注册，config gen=%d）", listenAddr, employeeName, cfg.Generation())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
