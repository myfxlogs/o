// Package testutil —— 测试基建（独立测试实例 D-021；缺 DSN=skip 语义）。
package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool 接 PHASE_TEST_PG_DSN 独立测试实例；未设=Skip（CI 5434/本机覆写端口同一口径）。
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("PHASE_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("PHASE_TEST_PG_DSN 未设——独立测试实例缺席，skip（不打生产实例）")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("test pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
