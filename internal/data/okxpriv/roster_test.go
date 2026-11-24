package okxpriv

// roster_test.go —— O-38013 桩测锚（Supervisor respawn 面）：
//
//	① 凭据缺件户 spawn 前置败——acctLoop 早退自清 kids 残行（O-38013
//	   病灶：goroutine 死 map 占位=syncRoster 永不重 spawn）+credMiss/
//	   retryAt 退避账（2s→60s 顶格按拍续试不放弃）；
//	② 退避窗内 syncRoster 不重复 spawn（credMiss 计数不增），窗过重试；
//	③ 凭据补位后退避窗内重 spawn 即活——fakePrivWS 全链 login+subscribe
//	   →roster state=active 自报；
//	④ disabled/removed 永不复活——出 want 即收会话+退避账清零，后续拍
//	   零 spawn（名册 state 过滤构造性保证）。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/cryptoutil"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

var supTestMK = []byte("fedcba9876543210fedcba9876543210")

// seedSupMK 测试主权钥（readCreds 同径 O_MASTER_KEY_FILE 白名单件）。
func seedSupMK(t *testing.T) {
	t.Helper()
	f := filepath.Join(t.TempDir(), "o_master_key")
	if err := os.WriteFile(f, supTestMK, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("O_MASTER_KEY_FILE", f)
}

// seedCexSecret <key>=plaintext Seal 入 app_secrets（幂等覆写）。
func seedCexSecret(t *testing.T, ctx context.Context, pool *pgxpool.Pool, key, plain string) {
	t.Helper()
	ct, err := cryptoutil.Seal(supTestMK, plain)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)
		 ON CONFLICT (key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext`, key, ct); err != nil {
		t.Fatal(err)
	}
}

// seedCexAcct roster CEX 户行 upsert（proto='cex' collect 授权——state 由
// 会话环自报推进）。
func seedCexAcct(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	venue, credNS, state string) {
	t.Helper()
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_venue_roster(venue,proto,cred_ns,roles,state)
		 VALUES($1,'cex',$2,'{collect}',$3)
		 ON CONFLICT (venue) DO UPDATE SET cred_ns=EXCLUDED.cred_ns,
		 state=EXCLUDED.state, last_error=NULL`, venue, credNS, state); err != nil {
		t.Fatal(err)
	}
}

// supKidsCount kids map 在册数（白盒断言面——同 mt5 runs 先例）。
func supKidsCount(s *Supervisor, venue string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.kids[venue]
	return ok
}

func supCredMiss(s *Supervisor, venue string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.credMiss[venue]
}

// pollUntil 有界轮询收敛（异步 acctLoop 写面——就绪条件断言非固定睡眠）。
func pollUntil(t *testing.T, what string, d time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(30 * time.Millisecond) // sched-exempt-ok 测试轮询拍（有界 deadline 等待非生产调度原语）
	}
	t.Fatalf("%s 超时未收敛", what)
}

// 案①②③④全链：缺件→残行自清+退避→窗内不重试→窗过重试→凭据补位
// spawn 成功 active→disabled 永不复活。
func TestSupervisorCredMissRespawn(t *testing.T) {
	seedSupMK(t)
	pool := testutil.Pool(t)
	ctx := context.Background()
	for _, d := range []string{"fin", "md"} {
		if err := pgstore.Migrate(ctx, pool, d); err != nil {
			t.Fatal(err)
		}
	}
	// 退避覆写——桩测不真等 2s 首档（credMissBackoff var 覆写锚先例）。
	origBackoff := credMissBackoff
	credMissBackoff = func(int) time.Duration { return 400 * time.Millisecond }
	defer func() { credMissBackoff = origBackoff }()
	// 会话址指桩——spawn 成功径不触真 venue。
	srv := fakePrivWS(t, true, nil)
	defer srv.Close()
	origURL := privateWSURL
	privateWSURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	defer func() { privateWSURL = origURL }()

	venue, credNS := "okx_sup1", "okxsup.cex"
	seedCexAcct(t, ctx, pool, venue, credNS, "registered")
	defer func() {
		pool.Exec(ctx, `DELETE FROM fin_venue_roster WHERE venue=$1`, venue)
		pool.Exec(ctx, `DELETE FROM app_secrets WHERE key LIKE $1||'.%'`, credNS)
	}()
	// 清场——上测残留 creds/退避态不污染本测判词。
	pool.Exec(ctx, `DELETE FROM app_secrets WHERE key LIKE $1||'.%'`, credNS)

	s := NewSupervisor(pool, false)
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// ①缺件 spawn 前置败——acctLoop 异步：等 credMiss 记账落账。
	s.syncRoster(rctx)
	pollUntil(t, "首拍 spawn 前置败记账", 3*time.Second,
		func() bool { return supCredMiss(s, venue) == 1 })
	// 病灶断言：kids 残行须自清（goroutine 已退 map 不得占位）。
	pollUntil(t, "kids 残行自清", 2*time.Second,
		func() bool { return !supKidsCount(s, venue) })
	// roster degraded 自报在案（凭据面病灶显形不静默）。
	var st string
	pool.QueryRow(ctx, `SELECT state FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&st)
	if st != "degraded" {
		t.Fatalf("缺件户 state=%s want degraded", st)
	}

	// ②退避窗内 syncRoster 不重复 spawn——窗内拍 credMiss 不增。
	s.syncRoster(rctx)
	time.Sleep(150 * time.Millisecond) // sched-exempt-ok 测试窗内拍等待（非生产调度原语）
	if n := supCredMiss(s, venue); n != 1 {
		t.Fatalf("退避窗内 credMiss=%d want 1（窗内不重试）", n)
	}
	// 窗过重试——syncRoster 逐拍调用（产线=Run ticker 驱动），窗过后
	// 下拍重 spawn→前置再败→credMiss=2（按拍续试不放弃）。
	deadline := time.Now().Add(4 * time.Second)
	for supCredMiss(s, venue) != 2 {
		if time.Now().After(deadline) {
			t.Fatalf("退避窗过重 spawn 未收敛 credMiss=%d want 2", supCredMiss(s, venue))
		}
		s.syncRoster(rctx)
		time.Sleep(120 * time.Millisecond) // sched-exempt-ok 测试拍距（非生产调度原语）
	}

	// ③凭据补位——三键 Seal 落库，退避账清后 syncRoster 重 spawn：
	// fakePrivWS login+subscribe 全链→active 自报+kids 在册。
	for _, f := range []string{"api_key", "api_secret", "passphrase"} {
		seedCexSecret(t, ctx, pool, credNS+"."+f, "k-"+f)
	}
	s.mu.Lock()
	delete(s.retryAt, venue) // 测试加速——等价退避窗自然过
	s.mu.Unlock()
	s.syncRoster(rctx)
	pollUntil(t, "凭据补位重 spawn", 3*time.Second,
		func() bool { return supKidsCount(s, venue) })
	pollUntil(t, "会话 active 自报", 5*time.Second, func() bool {
		pool.QueryRow(ctx, `SELECT state FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&st)
		return st == "active"
	})
	if n := supCredMiss(s, venue); n != 0 {
		t.Fatalf("spawn 成功后 credMiss=%d want 0（退避账清零）", n)
	}

	// ④disabled 永不复活——出 want 即收会话+退避账清零，后续拍零 spawn。
	if _, err := pool.Exec(ctx,
		`UPDATE fin_venue_roster SET state='disabled' WHERE venue=$1`, venue); err != nil {
		t.Fatal(err)
	}
	s.syncRoster(rctx)
	pollUntil(t, "disabled 收会话", 2*time.Second,
		func() bool { return !supKidsCount(s, venue) })
	for i := 0; i < 3; i++ { // 三拍复核——disabled 户零 spawn 零退避账
		s.syncRoster(rctx)
		time.Sleep(50 * time.Millisecond) // sched-exempt-ok 测试拍间隔（非生产调度原语）
	}
	if supKidsCount(s, venue) || supCredMiss(s, venue) != 0 {
		t.Fatal("disabled 户不得 spawn/记退避账（永不复活闸）")
	}
}
