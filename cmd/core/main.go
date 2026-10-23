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
	"phase/internal/ledger"
	"phase/internal/platform/config"
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
	// Earn 停泊态（员工 配管）：现率采集无凭据径先行；停泊钱动面凭据就位挂接。
	earnW := earn.NewWorker(pool, cfg)
	go earnW.Run(ctx, errc)
	if priv, err := okxpriv.NewCollector(ctx, pool, false); err != nil {
		log.Printf("WARN: 私流守缺席（%v）——凭据落位后重启即上岗", err)
	} else {
		earnW.AttachVenue(okxexec.NewClient(priv.Credentials(), false))
		// 交易域首岗：执事（Executor=okxexec 唯一 venue 径——凭据与私流守同源解出）。
		execR := execution.NewRunner(pool, cfg,
			execution.OkxExecutor{C: okxexec.NewClient(priv.Credentials(), false)}).WithEarn(earnW)
		go execution.Run(ctx, errc, execR)
		// boot 首读播种器=okxexec /account/balance（事件锚定形一次性拉取——
		// cmd 装配层接线，数据域不反向依赖交易域）。
		creds := priv.Credentials()
		priv.SetBootFn(func(ctx context.Context) (map[string]float64, error) {
			return okxexec.BalanceEquity(ctx, creds, false)
		})
		go priv.Run(ctx, errc)
	}

	// 研究域首岗：funding 枚举+评估卡（员工 探员/验证官）——判据全 app_config 机读。
	go funding.NewRunner(pool, cfg).Run(ctx, errc)

	// 投资域首岗：生命周期官（员工 命官）——提案→paper 沙箱→probe 晋升闸。
	go lifecycle.NewRunner(pool, cfg).Run(ctx, errc)

	// 复盘域：评史官（一仓一判决书+远期验证写回环——G10 收口）。
	go review.NewRunner(pool).Run(ctx, errc)

	// 账本域：司账+核账（入账结算+对账环——消费事实行不触发钱动作）。
	go ledger.NewRunner(pool, cfg).Run(ctx, errc)

	// 风控域首岗：唯一交易前闸（员工 门官）——G7 消费件就绪后只认放行令牌。
	riskGate := risk.NewGate(pool, cfg)
	go risk.Run(ctx, errc, riskGate)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// 产出窗=行情新鲜度+rejects 计数门观测端点（判定面可观测 D-652-A）。
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
