// Package testutil —— 测试基建（独立测试实例 D-021；缺 DSN=skip 语义）。
// T-0021 隔离硬化：每测试包独占 schema（test_<pkg>_<rand>+search_path）——
// 共享库并发下 app_config 键/快照行/策略行互不干扰（guard 翻 kill_switch
// 不再误伤 risk 包，他包 fixture 行不顶阈值）。Migrate 非限定名随 schema 落位。
package testutil

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/pgguard"
)

// Pool 接 O_TEST_PG_DSN 独立测试实例+包级 schema 隔离；
// 未设 DSN=Skip（CI 5434/本机覆写端口同一口径）。
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("O_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("O_TEST_PG_DSN 未设——独立测试实例缺席，skip（不打生产实例）")
	}
	// 包级 schema 名（caller 包尾段+随机尾——同包多测共享 schema 无碍，包间零干扰）
	pkg := "x"
	if _, file, _, ok := runtime.Caller(1); ok {
		parts := strings.Split(file, "/")
		if len(parts) >= 2 {
			pkg = strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
					return r
				}
				return '_'
			}, parts[len(parts)-2])
		}
	}
	schema := fmt.Sprintf("t_%s_%d", pkg, time.Now().UnixNano()%1e8)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("dsn parse: %v", err)
	}
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, err := c.Exec(ctx, "SET search_path TO "+schema+",public")
		return err
	}
	pgguard.GuardPool(cfg) // T-0154 事务残留守卫（测试面同守——桩锚同池验）
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("test pool: %v", err)
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+schema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
