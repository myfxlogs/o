// Package pgstore —— PostgreSQL 访问层（连接池 + 文件序迁移）。
package pgstore

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/pgguard"
)

// Connect 建池并 ping（失败即错——鸡生蛋引导面 fail-closed）。
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	// T-0050 实勘根修（SG 2026-10-24 行情断流）：默认 MaxConns=max(4,NumCPU) 被
	// T-0048 事件脊 7 路 LISTEN 长连接占尽——采集写/巡扫饿死（Recv-Q 积压读循环
	// 堵）。显式扩容：监听面 7+采集写+巡扫并发+心跳。
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgpool parse: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 8
	pgguard.GuardPool(cfg)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgpool new: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pg ping: %w", err)
	}
	return pool, nil
}

// Store 行级封装（T-0108 nox 移植面——trx/cmc 持久化方法挂点；O 面只需 pool 一件）。
type Store struct {
	pool *pgxpool.Pool
}

// NewStore 池封装（cmd 装配层注入）。
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// InsertAlert/InsertAlertDedup 告警写面降级件（T-0108 移植——O 无 alerts 表域；
// 采集面告警降格 log 显形不假静默，真告警径=rej/notify 另行立票）。
func (s *Store) InsertAlert(ctx context.Context, _ *int64, level, msg string) error {
	log.Printf("[trx/%s] %s", level, msg)
	return nil
}

// InsertAlertDedup 去重告警（同窗内同文抑写——log 径同降格）。
func (s *Store) InsertAlertDedup(ctx context.Context, level, msg string, _ time.Duration) error {
	return s.InsertAlert(ctx, nil, level, msg)
}
