package pgstore

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"o/migrations"
)

type migEntry struct {
	Version int
	Name    string
}

// migFileRe 迁移文件唯一合法名形（O-41367 派生化消根——目录=唯一清单：
// migrations/ 入库即执行，登记表手写并写面物理消失，两支在途分支同改
// 文件尾的结构性必撞根修）。
var migFileRe = regexp.MustCompile(`^(fin|md)_(\d+)_.*\.sql$`)

// deriveMigrationTable embed FS 目录枚举→文件名解析版本→版本号升序构建
// 迁移链。fail-closed：*.sql 名不匹配/版本号链内重复=panic——脏文件名
// =编译期等价物宁炸不静默（登记漏执行/撞号 silent-skip 失败类同消）。
// 非 .sql 件（domain.tsv/migrations.go 等册面）不入链。
func deriveMigrationTable() []migEntry {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		panic(fmt.Sprintf("migrations FS 枚举败: %v", err))
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return buildMigrationTable(names)
}

// buildMigrationTable 纯函数面（桩测注入文件名集——embed 无子目录，
// 名字集即目录面全集）。
func buildMigrationTable(names []string) []migEntry {
	var out []migEntry
	seen := map[string]string{} // chain/版本号 → 文件名（撞号即炸）
	for _, name := range names {
		if !strings.HasSuffix(name, ".sql") {
			continue // 非迁移件（domain.tsv/migrations.go/README）不入链
		}
		m := migFileRe.FindStringSubmatch(name)
		if m == nil {
			panic(fmt.Sprintf("迁移文件名非法 %q——唯一合法形 fin|md_NNNN_*.sql（目录=清单，脏名=编译期等价炸）", name))
		}
		ver, err := strconv.Atoi(m[2])
		if err != nil {
			panic(fmt.Sprintf("迁移版本号解析败 %q: %v", name, err))
		}
		key := m[1] + "/" + m[2]
		if prev, dup := seen[key]; dup {
			panic(fmt.Sprintf("迁移版本撞号 %s：%s 与 %s——同号后登记者被 silent-skip", key, prev, name))
		}
		seen[key] = name
		out = append(out, migEntry{Version: ver, Name: name})
	}
	sort.Slice(out, func(i, j int) bool {
		ci, cj := out[i].Name[:3], out[j].Name[:3]
		if ci != cj {
			return ci < cj // 链前缀序 fin<md
		}
		return out[i].Version < out[j].Version
	})
	return out
}

// migrationTable 迁移链全表（派生件——包级惰性即 init 期求值：脏名/撞号
// 在测试与生产 boot 同炸，不留运行期才发现面）。
var migrationTable = deriveMigrationTable()

// Migrate 按域选链（md|fin——T-0167 分域：PG-md 走 md 链、SG 走 fin 链），
// 按 migrationTable（=migrations/ 目录派生，O-41367）序进应用
// migrations/<domain>_*.sql；schema_migrations 记已应用版本（基线制，库级
// 一库一链）。幂等：已应用版本跳过；目录=唯一合法迁移清单（入库即执行）。
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
