// cmd/core —— PHASE 单二进制（cmd/core，架构约束 §1 装配钉）。
// boot 序：DB_DSN 连库 → 迁移 → config client（Load+Watch）→ 花名册 Register+Beat
// （运行时本体=1 号员工 phase-core）→ :50051 ConnectRPC+健康端点。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"phase/internal/platform/config"
	"phase/internal/platform/roster"
	"phase/internal/store/pgstore"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
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
