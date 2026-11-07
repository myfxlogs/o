package hostmetrics

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0434 四断言红桩（缺一不收）：
// ① pool nil 启动→恢复后自报环必活（原 pool==nil→return 一次永退红桩）；
// ② Upsert 注入连败→ops_event 升件行落（原仅 log 静默死红桩）；
// ③ fin_host_metrics.ts 手工回拨>阈→停拍升件（selfprobe 第三实例）；
// ④ 前端 stale 渲染断言归 web/scripts/hostmetrics-domtest.mjs（渲染契约）。

// fixture 测试池（包级 schema 隔离——testutil 契约）。
func fixture(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return pool
}

// cleanEsc 清 HOSTMETRICS 升件痕（同 schema 跨测隔离——升件去重窗以
// class+scope 判，跨测残留会误抑制本测升件）。
func cleanEsc(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`DELETE FROM ops_event WHERE esc_class='观测盲' AND esc_scope LIKE 'HOSTMETRICS-%'`); err != nil {
		t.Fatal(err)
	}
}

// waitRow 轮询断言（环异步——行现或超时）。
func waitRow(t *testing.T, pool *pgxpool.Pool, query string, args ...any) bool {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			return true
		}
		time.Sleep(10 * time.Millisecond) // sched-exempt-ok：桩测轮询等待（测试期界非生产调度）
	}
	return false
}

// TestRunPoolNilThenRecover 断言①：poolFn 先 nil 后真池——自报环必活
// （修前形：pool==nil→return 一次永退，恢复后永不自报）。
func TestRunPoolNilThenRecover(t *testing.T) {
	pool := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	var mu sync.Mutex
	poolFn := func() DB {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls <= 2 {
			return nil // 未就绪形——退避重试环必须兜住
		}
		return pool
	}
	go Run(ctx, poolFn, "sg", "/", 5*time.Millisecond)
	if !waitRow(t, pool, `SELECT count(*) FROM fin_host_metrics WHERE host='sg'`) {
		t.Fatal("pool 恢复后自报环未活——sg 行未落（永退病灶复现）")
	}
}

// errInjected 连败注入错（failHostDB 写面全败标记）。
var errInjected = errors.New("injected: fin_host_metrics 写面连败")

// failHostDB 连败注入缝：fin_host_metrics 写面全败、ops_event 面透传真池
// （连败升件的同库写径=真实径——fin_host_metrics 锁/权坏不拖 ops_event）。
type failHostDB struct{ real *pgxpool.Pool }

func (f failHostDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "fin_host_metrics") {
		return pgconn.CommandTag{}, errInjected
	}
	return f.real.Exec(ctx, sql, args...)
}

func (f failHostDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return f.real.QueryRow(ctx, sql, args...)
}

// TestRunUpsertFailEscalation 断言②：Upsert 连败≥阈→ops_event 升件行落
// （修前形：失败仅 log——观测断供无感 8 天）。
func TestRunUpsertFailEscalation(t *testing.T) {
	pool := fixture(t)
	cleanEsc(t, pool)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go Run(ctx, func() DB { return failHostDB{real: pool} }, "sg", "/", 5*time.Millisecond)
	if !waitRow(t, pool, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND esc_scope='HOSTMETRICS-SG'
		AND note LIKE '%连败%'`) {
		t.Fatal("连败升件行未落——静默死病灶复现")
	}
}

// TestProbeStaleEscalationOnRolledBackTs 断言③：ts 手工回拨>阈→停拍升件
// （鲜行零报/零行史同报/24h 同窗去重全形覆盖）。
func TestProbeStaleEscalationOnRolledBackTs(t *testing.T) {
	pool := fixture(t)
	cleanEsc(t, pool)
	ctx := context.Background()
	// 形⓪ 鲜行（sg+dev 在拍）→零报。
	if err := Upsert(ctx, pool, "sg", Sample{Cores: 1}); err != nil {
		t.Fatal(err)
	}
	if err := Upsert(ctx, pool, "dev", Sample{Cores: 1}); err != nil {
		t.Fatal(err)
	}
	if err := ProbeStale(ctx, pool, time.Now); err != nil {
		t.Fatalf("probe: %v", err)
	}
	assertEscRows(t, pool, 0, "鲜行误报")
	// 形① ts 回拨 1h>阈 10m→sg 停拍升件落账。
	if _, err := pool.Exec(ctx,
		`UPDATE fin_host_metrics SET ts=now()-interval '1 hour' WHERE host='sg'`); err != nil {
		t.Fatal(err)
	}
	if err := ProbeStale(ctx, pool, time.Now); err != nil {
		t.Fatalf("probe: %v", err)
	}
	assertEscRows(t, pool, 1, "回拨停拍升件行未落")
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_scope='HOSTMETRICS-SG'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "停拍") || !strings.Contains(note, "ts=") {
		t.Fatalf("升件 note 缺停拍/ts 锚: %s", note)
	}
	// 形② 再拍→24h 同窗去重（升件行不增）。
	if err := ProbeStale(ctx, pool, time.Now); err != nil {
		t.Fatalf("probe: %v", err)
	}
	assertEscRows(t, pool, 1, "同窗去重失效")
	// 形③ 零行史（dev 行删）→断供极端形同报。
	if _, err := pool.Exec(ctx, `DELETE FROM fin_host_metrics WHERE host='dev'`); err != nil {
		t.Fatal(err)
	}
	if err := ProbeStale(ctx, pool, time.Now); err != nil {
		t.Fatalf("probe: %v", err)
	}
	assertEscRows(t, pool, 2, "零行史升件行未落")
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_scope='HOSTMETRICS-DEV'
		AND note LIKE '%零行史%'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
}

// assertEscRows HOSTMETRICS-* 升件行数断言（阶段间公共面）。
func assertEscRows(t *testing.T, pool *pgxpool.Pool, want int, msg string) {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope LIKE 'HOSTMETRICS-%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != want {
		t.Fatalf("%s: 升件行=%d（欲=%d）", msg, n, want)
	}
}
