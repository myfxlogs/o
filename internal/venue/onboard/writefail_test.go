// writefail_test.go —— T-0402 锚③桩测：ops_event 写拒权不静默——
// 撤权角色写径 → trans 返 err + WARN 级日志断言（升件兜底同被拒，WARN 仍出）。
package onboard

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
)

func TestOpsEventWriteDeniedWarns(t *testing.T) {
	pool := fixturePool(t)
	ctx := context.Background()
	var schema string
	if err := pool.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	const role = "t0402_deny"
	for _, q := range []string{
		`DROP ROLE IF EXISTS ` + role,
		`CREATE ROLE ` + role + ` LOGIN PASSWORD 't0402'`,
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO %s`, schema, role),
		fmt.Sprintf(`GRANT SELECT ON %s.ops_event TO %s`, schema, role),
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DROP ROLE IF EXISTS `+role) //nolint:errcheck
	})

	cfg, err := pgxpool.ParseConfig(os.Getenv("O_TEST_PG_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = testutil.ShardDatabase() // T-0468：deny 池与 fixturePool 同分库——schema/授权在分库
	cfg.ConnConfig.User = role
	cfg.ConnConfig.Password = "t0402"
	cfg.AfterConnect = func(c context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(c, "SET search_path TO "+schema+",public")
		return err
	}
	deny, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer deny.Close()

	r := &Runner{md: pool, fin: deny,
		rej: rejects.New("onboard", "write_err"), now: time.Now}

	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	err = r.trans(ctx, "v-deny", "0", "probed", "探测", map[string]any{}, "n", false)
	if err == nil {
		t.Fatal("撤权写应返 err")
	}
	if !strings.Contains(buf.String(), "WARN [onboard] ops_event 写败") {
		t.Fatalf("写败静默——WARN 缺: %q", buf.String())
	}
}

// fixturePool 取迁移后测试池（复用 fixture 建链面——md+fin 双链迁齐）。
func fixturePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	_, pool, _ := fixture(t)
	return pool
}
