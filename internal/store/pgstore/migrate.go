package pgstore

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"o/migrations"
)

// Migrate 按域选链（md|fin——T-0167 分域：PG-md 走 md 链、SG 走 fin 链），按登记表
// 序进应用 migrations/<domain>_*.sql；schema_migrations 记已应用版本（基线制，库级
// 一库一链）。幂等：已应用版本跳过；未登记文件不执行（登记表=唯一合法迁移清单）。
func Migrate(ctx context.Context, pool *pgxpool.Pool, domain string) error {
	if domain != "md" && domain != "fin" {
		return fmt.Errorf("migrate domain %q 非法（合法域=md|fin——分域选链 fail-closed）", domain)
	}
	// 迁移串行化：并发 Migrate（多测试包/多实例 boot）在目录层撞 pg_type 唯一索引——
	// advisory lock 互斥（会话级，连接断开自放）。
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("migrate lock acquire: %w", err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(860223)"); err != nil {
		return fmt.Errorf("migrate advisory lock: %w", err)
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock(860223)")
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version int PRIMARY KEY,
		name text NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("schema_migrations bootstrap: %w", err)
	}
	type entry = struct {
		Version int
		Name    string
	}
	chain := make([]entry, 0, len(migrationTable))
	for _, m := range migrationTable {
		if strings.HasPrefix(m.Name, domain+"_") {
			chain = append(chain, m)
		}
	}
	sort.Slice(chain, func(i, j int) bool { return chain[i].Version < chain[j].Version })
	for _, m := range chain {
		var applied bool
		if err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", m.Version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("migration %d probe: %w", m.Version, err)
		}
		if applied {
			continue
		}
		body, err := migrations.FS.ReadFile(m.Name)
		if err != nil {
			return fmt.Errorf("migration %d read %s: %w", m.Version, m.Name, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migration %d begin: %w", m.Version, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %d %s: %w", m.Version, m.Name, err)
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations(version,name) VALUES($1,$2)", m.Version, m.Name); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %d record: %w", m.Version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migration %d commit: %w", m.Version, err)
		}
	}
	return nil
}
