// Package pgstore —— PostgreSQL 访问层（连接池 + 文件序迁移）。
package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
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
