// cmd/core —— PHASE 单二进制（cmd/core，架构约束 §1 装配钉）。
// boot 序：DB_DSN 连库 → 迁移 → config client（Load+Watch）→ 花名册 Register+Beat
// （运行时本体=1 号员工 phase-core）→ :50051 ConnectRPC+健康端点。
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

	"phase/internal/data/okxpriv"
	"phase/internal/data/okxws"
	"phase/internal/earn"
	"phase/internal/guard"
	"phase/internal/ledger"
	legfunding "phase/internal/ledger/funding"
	"phase/internal/ledger/holding"
	"phase/internal/ledger/report"
	"phase/internal/mt5"
	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/platform/notify"
	"phase/internal/platform/roster"
	"phase/internal/portfolio/lifecycle"
	"phase/internal/research/funding"
	"phase/internal/review"
	"phase/internal/risk"
	"phase/internal/store/pgstore"
	"phase/internal/trading/execution"
	"phase/internal/trading/okxexec"
)

const (
	employeeName  = "phase-core"
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

	if err := pgstore.Migrate(ctx, pool); err != nil {
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
	collector := okxws.NewCollector(pool, cfg)
	go collector.Run(ctx, errc)

	// 私流守：OKX 私有 WS（凭据自 app_secrets 密文列解；缺件 fail-closed
	// 不炸 boot——E-1 未落位期岗位缺席留痕非死码）。
	// 凭据缺席期仓巡平仓缝降级 stub（拒执 fail-closed）——凭据落位重启即全链。
	var closer guard.Executor
	// Earn 停泊态（员工 配管）：现率采集无凭据径先行；停泊钱动面凭据就位挂接。
	mkA, _ := config.ReadMasterKey()
	var feishuCli *notify.FeishuClient
	{
		var idCt, secCt []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishu.app_id'").Scan(&idCt); err == nil {
			if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishu.app_secret'").Scan(&secCt); err == nil {
				id, e1 := cryptoutil.Open(mkA, idCt)
				sec, e2 := cryptoutil.Open(mkA, secCt)
				if e1 == nil && e2 == nil {
					feishuCli = notify.NewFeishu(id, sec)
				}
			}
		}
	}
	// typed-nil 主修（T-0032）：凭据缺席传 nil 接口非 (*FeishuClient)(nil)——接口装
	// 类型化 nil 指针时 ==nil 判 false，SendText 调 nil receiver 即 panic（SG 崩环实证）。
	var sender notify.Sender
	if feishuCli != nil {
		sender = feishuCli
	}
	alerter := notify.NewAlerter(sender, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.enabled"); return v })
	// kill_switch 状态变迁推（T-0048 事件化：LISTEN config_changed 醒后查现值——
	// true/false 各一报不重复；周期轮询拆）。
	go func() {
		prev, _ := cfg.GetBool("risk.kill_switch")
		for {
			conn, err := pool.Acquire(ctx)
			if err != nil {
				return
			}
			if _, err := conn.Exec(ctx, "LISTEN config_changed"); err != nil {
				conn.Release()
				return
			}
			n, err := conn.Conn().WaitForNotification(ctx)
			conn.Release()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				continue // 断线重听（事件化=真事件脊非轮询）
			}
			_ = n
			v, err := cfg.GetBool("risk.kill_switch")
			if err == nil && v != prev {
				alerter.Alert(ctx, "kill_switch", fmt.Sprintf("kill_switch %v→%v", prev, v))
				prev = v
			}
		}
	}()
	var okcli *okxexec.Client
	earnW := earn.NewWorker(pool, cfg)
	go earnW.Run(ctx, errc)
	if priv, err := okxpriv.NewCollector(ctx, pool, false); err != nil {
		log.Printf("WARN: 私流守缺席（%v）——凭据落位后重启即上岗", err)
	} else {
		okcli = okxexec.NewClient(priv.Credentials(), false)
		earnW.AttachVenue(okcli)
		earnW.AttachAlerter(alerter)
		// venue 现读件（T-0048：决策/判定/对账时点实读——快照行全域出局）。
		earnW.AttachIdle(func(ctx context.Context) (float64, error) {
			b, err := okcli.AssetBalances(ctx)
			if err != nil {
				return 0, err
			}
			for _, x := range b {
				if x.Ccy == "USDT" {
					return okxexec.ParseF(x.AvailBal), nil
				}
			}
			return 0, nil
		})
		// 交易域首岗：执事（Executor=okxexec 唯一 venue 径——凭据与私流守同源解出）。
		execR := execution.NewRunner(pool, cfg,
			&execution.OkxExecutor{C: okxexec.NewClient(priv.Credentials(), false)}).WithEarn(earnW).WithAlerter(alerter)
		go execution.Run(ctx, errc, execR)
		closer = closeExec{r: execR}
		// boot 首读播种器=okxexec /account/balance（事件锚定形一次性拉取——
		// cmd 装配层接线，数据域不反向依赖交易域）。
		creds := priv.Credentials()
		priv.SetBootFn(func(ctx context.Context) (map[string]float64, error) {
			return okxexec.BalanceEquity(ctx, creds, false)
		})
		// funding 实收记账（T-0044）：余额帧事件驱动 bills→fin_cashflow（bills.go）。
		priv.SetBillsHook(legfunding.NewIngester(pool,
			billsSrc{okxexec.NewClient(priv.Credentials(), false)}).Hook())
		go priv.Run(ctx, errc)

		// 台账同步：资产端点采（okxexec 装配注入——账本域不反向依赖交易域；
		// /asset/balances+earn 面=钦定 REST 保留面非轮询病灶）。
		okcli := okxexec.NewClient(priv.Credentials(), false)
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
		go m5.Run(ctx, errc)
	}

	// 研究域首岗：funding 枚举+评估卡（员工 探员/验证官）——判据全 app_config 机读。
	go funding.NewRunner(pool, cfg).Run(ctx, errc)

	// 投资域首岗：生命周期官（员工 命官）——提案→paper 沙箱→probe 晋升闸。
	go lifecycle.NewRunner(pool, cfg).Run(ctx, errc)

	// 复盘域：评史官（一仓一判决书+远期验证写回环——G10 收口）。
	go review.NewRunner(pool).Run(ctx, errc)

	// 账本域：司账+核账（入账结算+对账环——消费事实行不触发钱动作）。
	{
		mk, _ := config.ReadMasterKey()
		go report.NewRunner(pool, report.NewPusher(pool, cfg, mk)).WithConfig(cfg).Run(ctx, errc)
	}
	ledgerR := ledger.NewRunner(pool, cfg).WithAlerter(alerter)
	go ledgerR.Run(ctx, errc)

	// 风控域首岗：唯一交易前闸（员工 门官）——G7 消费件就绪后只认放行令牌。
	riskGate := risk.NewGate(pool, cfg)
	go risk.Run(ctx, errc, riskGate)

	// 看护域首岗：仓巡（G8）——持仓哨戒三触发+策略联动，close intent 过闸免拦径，
	// 反向双腿平走执事 G7 径（凭据缺席=缝降级 stub 拒执 fail-closed）。
	wd := guard.NewWarden(pool, cfg, riskGate, closer)
	wd.AttachAlerter(alerter)
	// venue 现读件（T-0048：快照行全域出局——venue_readers.go 装配）。
	attachVenueReaders(riskGate, wd, ledgerR, okcli)
	go wd.Run(ctx, errc)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// 产出窗=行情新鲜度+rejects 计数门观测端点（判定面可观测 D-652-A）。
	mux.HandleFunc("/report", report.Handler(pool))
	mux.HandleFunc("/md/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"frames":     collector.FrameCount(),
			"last_frame": collector.LastFrame(),
			"rejects":    collector.Stats.Snapshot(),
		})
	})
	mountCore(mux)

	srv := &http.Server{Addr: listenAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
	}()
	log.Printf("phase-core up :%s（employee=%s 已注册，config gen=%d）", listenAddr, employeeName, cfg.Generation())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
