package repair

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 册外单元永不触+日配额闸——两闸在 systemctl 前拦截（闸过路径
// 不触真重启,故可单测）。
func TestWatchGuards(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM md_repair_log`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	st := map[string]*unitState{}
	// 永陈面：测试表一行陈 ts——max(ts) 返真值过零行闸,陈度超阈。
	for _, q := range []string{
		`DROP TABLE IF EXISTS md_repair_test_feed`,
		`CREATE TABLE md_repair_test_feed(ts timestamptz)`,
		`INSERT INTO md_repair_test_feed VALUES(now()-interval '2 days')`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatalf("feed seed: %v", err)
		}
	}
	deadFeed := func(unit string) feedReg {
		return feedReg{Name: "t", Unit: unit, Table: "md_repair_test_feed",
			Where: "true", DeadSec: 1}
	}
	check(ctx, pool, deadFeed("o-core"), defGuard, st) // 册外拒
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM md_repair_log`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("册外落账 n=%d err=%v", n, err)
	}
	// 配额闸：预置 3 条 restart 账——再判=quota_burned 非 restart。
	for i := 0; i < 3; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO md_repair_log(unit,feed,action)
			VALUES('o-collecto','x','restart')`); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	check(ctx, pool, deadFeed("o-collecto"), defGuard, st)
	var act string
	if err := pool.QueryRow(ctx, `SELECT action FROM md_repair_log
		WHERE unit='o-collecto' AND feed='t'`).Scan(&act); err != nil || act != "quota_burned" {
		t.Fatalf("配额闸 act=%q err=%v", act, err)
	}
}
