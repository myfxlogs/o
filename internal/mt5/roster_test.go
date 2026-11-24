// roster_test.go —— O-37448 桩测锚（类型化不变式验收面）：
//
//	① 第三户零码接入——icmarkets（桩 venue）仅 roster 行+<cred_ns>.* 凭据族
//	   落库即被 syncRoster 拾起会话（码面零 icmarkets 字面量=CFD 框架非
//	   「XM 集成」判据）；
//	② 双户各持会话 token——两户 ConnectEx 各建 <venue>.session_token 共享
//	   行互不覆写，roster state 双双 active 自报；
//	③ 一户 ConnectEx 败不连坐——败户 degraded+last_error+「venue接入」
//	   ops_event 行（esc_scope=venue:login 首入判词），他户照常 active+token
//	   在册，败户会话环仍存活续试；
//	④ Exness 回归——venue='mt5' 键形全存量锚（SessionKey/SymbolsKey/
//	   channel='mt5_boot'/venue='mt5' 写键/md_ticks 键域）。
package mt5

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

var rosterTestMK = []byte("0123456789abcdef0123456789abcdef")

// seedMK 测试主权钥——O_MASTER_KEY_FILE env 白名单件（readCreds 同径读）。
func seedMK(t *testing.T) {
	t.Helper()
	f := filepath.Join(t.TempDir(), "o_master_key")
	if err := os.WriteFile(f, rosterTestMK, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("O_MASTER_KEY_FILE", f)
}

// seedSecret <key>=plaintext Seal 入 app_secrets（幂等覆写——跨测残留同键
// 不污染本测判词）。
func seedSecret(t *testing.T, ctx context.Context, pool *pgxpool.Pool, key, plain string) {
	t.Helper()
	ct, err := cryptoutil.Seal(rosterTestMK, plain)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)
		 ON CONFLICT (key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext`, key, ct); err != nil {
		t.Fatal(err)
	}
}

// seedAcctCreds <cred_ns>.{login,password,server} 凭据族落库——host 行缺席
// =server 径户合法形（XM 无 host 行实证——readCreds 空串容忍面）。
func seedAcctCreds(t *testing.T, ctx context.Context, pool *pgxpool.Pool, credNS, login string) {
	t.Helper()
	seedSecret(t, ctx, pool, credNS+".login", login)
	seedSecret(t, ctx, pool, credNS+".password", "pw-"+login)
	seedSecret(t, ctx, pool, credNS+".server", "Test-Server-"+credNS)
}

// seedAcct roster 户行 upsert（registered 态——会话环即拾即试，state 由
// 会话自报推进）。
func seedAcct(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	venue, credNS, login string) {
	t.Helper()
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_venue_roster(venue,proto,cred_ns,login,roles,state)
		 VALUES($1,'mt5',$2,$3,'{collect,execute}','registered')
		 ON CONFLICT (venue) DO UPDATE SET cred_ns=EXCLUDED.cred_ns,
		 login=EXCLUDED.login, state='registered', last_error=NULL`,
		venue, credNS, login); err != nil {
		t.Fatal(err)
	}
}

// rosterClean 桩户清场（共享 schema 跨测——本测 venue 域行/密钥/共享行
// 定向清，ops_event append-only 只按 esc_scope 谓词计不清）。
func rosterClean(t *testing.T, ctx context.Context, pool *pgxpool.Pool, venues ...string) {
	t.Helper()
	for _, v := range venues {
		if _, err := pool.Exec(ctx, `DELETE FROM fin_venue_roster WHERE venue=$1`, v); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM app_config WHERE key=$1`, v+".session_token"); err != nil {
			t.Fatal(err)
		}
	}
}

// stubConnEx ConnectionClient 桩——ConnectEx 按 User(login) 分档：
// failLogin>0 且=User→Error 败（败户隔离面）；余→Result="tok-<user>"
// （token 按户可辨=「各持 token」断言源）。CheckConnect 恒活（复挂径
// CheckAlive 通过）；byLogin=per-login 计数（败户续试断言面，mutex 护）。
type stubConnEx struct {
	mt5pb.ConnectionClient
	failLogin uint64
	mu        sync.Mutex
	byLogin   map[uint64]int
}

func (s *stubConnEx) noteLogin(u uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byLogin == nil {
		s.byLogin = map[uint64]int{}
	}
	s.byLogin[u]++
}

func (s *stubConnEx) loginCount(u uint64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.byLogin[u]
}

func (s *stubConnEx) ConnectEx(ctx context.Context, r *mt5pb.ConnectExRequest,
	opts ...grpc.CallOption) (*mt5pb.ConnectExReply, error) {
	s.noteLogin(r.User)
	if s.failLogin != 0 && r.User == s.failLogin {
		return &mt5pb.ConnectExReply{Error: &mt5pb.Error{
			Message: fmt.Sprintf("refused login %d", r.User)}}, nil
	}
	return &mt5pb.ConnectExReply{Result: fmt.Sprintf("tok-%d", r.User)}, nil
}

func (s *stubConnEx) Connect(ctx context.Context, r *mt5pb.ConnectRequest,
	opts ...grpc.CallOption) (*mt5pb.ConnectReply, error) {
	s.noteLogin(r.User)
	return &mt5pb.ConnectReply{Result: fmt.Sprintf("tok-%d", r.User)}, nil
}

func (s *stubConnEx) CheckConnect(ctx context.Context, r *mt5pb.CheckConnectRequest,
	opts ...grpc.CallOption) (*mt5pb.CheckConnectReply, error) {
	return &mt5pb.CheckConnectReply{Result: "ok"}, nil
}

// stubMT5Full MT5Client 桩——AccountSummary 恒活/SymbolParams 按 known 集
// 回确（venue 判存形）/SymbolList 回混合方言名/SymbolConfirm 由 Client 逐调
// 本面/OrderHistory 空窗（balanceOpsLoop 首拍零行即收）。
type stubMT5Full struct {
	mt5pb.MT5Client
	known map[string]bool
}

func (s *stubMT5Full) AccountSummary(ctx context.Context, r *mt5pb.AccountSummaryRequest,
	opts ...grpc.CallOption) (*mt5pb.AccountSummaryReply, error) {
	return &mt5pb.AccountSummaryReply{Result: &mt5pb.AccountSummary{
		Equity: 1000, FreeMargin: 900, Balance: 1000}}, nil
}

func (s *stubMT5Full) SymbolParams(ctx context.Context, r *mt5pb.SymbolParamsRequest,
	opts ...grpc.CallOption) (*mt5pb.SymbolParamsReply, error) {
	if s.known[r.Symbol] {
		return &mt5pb.SymbolParamsReply{Result: &mt5pb.SymbolParams{
			Symbol: r.Symbol, SymbolInfo: &mt5pb.SymbolInfo{},
			SymbolGroup: &mt5pb.SymGroup{}}}, nil
	}
	return &mt5pb.SymbolParamsReply{Error: &mt5pb.Error{
		Message: "Symbol not found"}}, nil
}

func (s *stubMT5Full) SymbolList(ctx context.Context, r *mt5pb.SymbolListRequest,
	opts ...grpc.CallOption) (*mt5pb.SymbolListReply, error) {
	return &mt5pb.SymbolListReply{Result: []string{
		"BTCUSD#", "ETHUSD#", "XAUUSD#", "BTCUSDm", "DE30", "OIL"}}, nil
}

func (s *stubMT5Full) OrderHistory(ctx context.Context, r *mt5pb.OrderHistoryRequest,
	opts ...grpc.CallOption) (*mt5pb.OrderHistoryReply, error) {
	return &mt5pb.OrderHistoryReply{}, nil
}

// stubSubs SubscriptionsClient 桩——Subscribe/SubscribeOrderProfit 恒回执。
type stubSubs struct{ mt5pb.SubscriptionsClient }

func (s *stubSubs) Subscribe(ctx context.Context, r *mt5pb.SubscribeRequest,
	opts ...grpc.CallOption) (*mt5pb.SubscribeReply, error) {
	return &mt5pb.SubscribeReply{}, nil
}

func (s *stubSubs) SubscribeOrderProfit(ctx context.Context, r *mt5pb.SubscribeOrderProfitRequest,
	opts ...grpc.CallOption) (*mt5pb.SubscribeOrderProfitReply, error) {
	return &mt5pb.SubscribeOrderProfitReply{}, nil
}

// stubStrm StreamsClient 桩——OnQuote/OnOrderProfit 返阻塞流（ctx 终解锁——
// session() 全链走通后流消费驻留面，测试只须会话建立+写键证据）。
type stubStrm struct{ mt5pb.StreamsClient }

func (s *stubStrm) OnQuote(ctx context.Context, r *mt5pb.OnQuoteRequest,
	opts ...grpc.CallOption) (grpc.ServerStreamingClient[mt5pb.OnQuoteReply], error) {
	return &fakeQuoteStream{ctx: ctx, mode: "block"}, nil
}

func (s *stubStrm) OnOrderProfit(ctx context.Context, r *mt5pb.OnOrderProfitRequest,
	opts ...grpc.CallOption) (grpc.ServerStreamingClient[mt5pb.OnOrderProfitReply], error) {
	return &fakeProfitStream{ctx: ctx}, nil
}

// fakeProfitStream OnOrderProfit 阻塞流桩（同 fakeQuoteStream block 形——
// Recv 驻留至 sessCtx 取消返 ctx err=会话终收尸径）。
type fakeProfitStream struct {
	ctx context.Context
}

func (f *fakeProfitStream) Recv() (*mt5pb.OnOrderProfitReply, error) {
	<-f.ctx.Done()
	return nil, f.ctx.Err()
}
func (f *fakeProfitStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeProfitStream) Trailer() metadata.MD         { return nil }
func (f *fakeProfitStream) CloseSend() error             { return nil }
func (f *fakeProfitStream) Context() context.Context     { return f.ctx }
func (f *fakeProfitStream) SendMsg(any) error            { return nil }
func (f *fakeProfitStream) RecvMsg(any) error            { return nil }

// rosterFixture 名册+凭据+网关址+桩会话面装配——返回 collector 与共享
// connM 桩（failLogin 分档断言面）。known=SymbolParams 回确集。
func rosterFixture(t *testing.T, failLogin uint64) (*Collector, context.Context,
	context.CancelFunc, *stubConnEx) {
	t.Helper()
	seedMK(t)
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO app_config(key,value) VALUES('mt5.gateway_addr','"gw.test:443"')
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	conn := &stubConnEx{failLogin: failLogin}
	mt5s := &stubMT5Full{known: map[string]bool{
		"BTCUSD#": true, "ETHUSD#": true, "XAUUSD#": true,
		"BTCUSDm": true, "ETHUSDm": true, "XAUUSDm": true}}
	dial := func(addr string) (*Client, error) {
		return &Client{connM: conn, mt5: mt5s, subs: &stubSubs{}, strm: &stubStrm{}}, nil
	}
	c := &Collector{pool: pool, cfg: cfg, dial: dial, runs: map[string]*acctRun{},
		rej: rejects.New("mt5", "connect_failed", "subscribe_failed", "stream_dead",
			"symbols_read_failed", "snapshot_write_failed", "tick_write_err", "write_err",
			"balance_ops_err")}
	rctx, cancel := context.WithCancel(ctx)
	return c, rctx, cancel, conn
}

// waitSessionKey 共享行出现等待（会话建立异步——stub 全同步，5s 富余窗）。
func waitSessionKey(t *testing.T, ctx context.Context, pool *pgxpool.Pool, key string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var tok string
		if err := pool.QueryRow(ctx,
			`SELECT value::text FROM app_config WHERE key=$1`, key).Scan(&tok); err == nil &&
			strings.Contains(tok, "token") {
			return
		}
		time.Sleep(20 * time.Millisecond) // sched-exempt-ok 测试轮询拍（有界 deadline 等待非生产调度原语）
	}
	t.Fatalf("共享行 %s 未立（会话未建）", key)
}

func sessionToken(t *testing.T, ctx context.Context, pool *pgxpool.Pool, key string) string {
	t.Helper()
	var raw string
	if err := pool.QueryRow(ctx,
		`SELECT value::text FROM app_config WHERE key=$1`, key).Scan(&raw); err != nil {
		t.Fatalf("共享行 %s 缺: %v", key, err)
	}
	if !strings.Contains(raw, `"token"`) {
		t.Fatalf("共享行 %s 坏体: %s", key, raw)
	}
	return raw
}

// rosterState 户 state 读。
func rosterState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, venue string) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(ctx,
		`SELECT state FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&s); err != nil {
		t.Fatalf("roster %s 读败: %v", venue, err)
	}
	return s
}

// 案①②：双户零码接入+各持 token——mt5（Exness 桩）+icmarkets（第三户桩，
// cred_ns='icmarkets.mt5'）仅名册+凭据族落库即双会话齐起；两户共享行
// token=ConnectEx 回执按 login 可辨互不覆写；state 双双 active 自报。
func TestRosterMultiAccountSessions(t *testing.T) {
	c, rctx, cancel, _ := rosterFixture(t, 0)
	defer cancel()
	ctx := context.Background()
	pool := c.pool
	venues := []string{"mt5", "icmarkets"}
	rosterClean(t, ctx, pool, venues...)
	defer rosterClean(t, ctx, pool, venues...)
	// mt5 存量单键凭据族（cred_ns='mt5'——<cred_ns>.login 形同构）。
	seedAcctCreds(t, ctx, pool, "mt5", "3001157")
	seedAcctCreds(t, ctx, pool, "icmarkets.mt5", "777001")
	seedAcct(t, ctx, pool, "mt5", "mt5", "3001157")
	seedAcct(t, ctx, pool, "icmarkets", "icmarkets.mt5", "777001")

	c.syncRoster(rctx, func(addr string) (*Client, error) {
		return c.dial(addr)
	})
	c.mu.Lock()
	if len(c.runs) != 2 {
		c.mu.Unlock()
		t.Fatalf("runs=%d want 2（双户各持会话环）", len(c.runs))
	}
	if c.runs["mt5"].acct.CredNS != "mt5" || c.runs["icmarkets"].acct.CredNS != "icmarkets.mt5" {
		c.mu.Unlock()
		t.Fatalf("cred_ns 投影错: %+v / %+v", c.runs["mt5"].acct, c.runs["icmarkets"].acct)
	}
	c.mu.Unlock()

	waitSessionKey(t, ctx, pool, "mt5.session_token")
	waitSessionKey(t, ctx, pool, "icmarkets.session_token")
	tokA := sessionToken(t, ctx, pool, "mt5.session_token")
	tokB := sessionToken(t, ctx, pool, "icmarkets.session_token")
	if !strings.Contains(tokA, "tok-3001157") || !strings.Contains(tokB, "tok-777001") {
		t.Fatalf("token 户归属错: mt5=%s icmarkets=%s", tokA, tokB)
	}
	// state 自报——双户 active（registered→connecting→active 迁移链在案）。
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if rosterState(t, ctx, pool, "mt5") == "active" &&
			rosterState(t, ctx, pool, "icmarkets") == "active" {
			break
		}
		time.Sleep(50 * time.Millisecond) // sched-exempt-ok 测试轮询拍（有界 deadline 等待非生产调度原语）
	}
	if s := rosterState(t, ctx, pool, "mt5"); s != "active" {
		t.Fatalf("mt5 state=%s want active", s)
	}
	if s := rosterState(t, ctx, pool, "icmarkets"); s != "active" {
		t.Fatalf("icmarkets state=%s want active", s)
	}
	// 写面分键实证——boot 快照 venue 列各持户 token（md_account_events）。
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		pool.QueryRow(ctx, `SELECT count(*) FROM md_account_events
			WHERE venue IN ('mt5','icmarkets') AND channel LIKE '%_boot'`).Scan(&n)
		if n >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond) // sched-exempt-ok 测试轮询拍（有界 deadline 等待非生产调度原语）
	}
	var venuesSeen int
	pool.QueryRow(ctx, `SELECT count(DISTINCT venue) FROM md_account_events
		WHERE venue IN ('mt5','icmarkets') AND channel LIKE '%_boot'`).Scan(&venuesSeen)
	if venuesSeen != 2 {
		t.Fatalf("boot 事件 venue 分键=%d want 2（venue 列写面未按户）", venuesSeen)
	}
}

// 案③：一户 ConnectEx 败不连坐——icmarkets login=777001 被桩拒：
// 该户 degraded+last_error+「venue接入」ops_event 首入判词行
// （esc_scope=venue:login）；mt5 户照常 active+session_token 在册；
// 败户 runs 环仍在（退避续试非拆除）。
func TestRosterFailureIsolation(t *testing.T) {
	c, rctx, cancel, conn := rosterFixture(t, 777001)
	defer cancel()
	ctx := context.Background()
	pool := c.pool
	venues := []string{"mt5", "icmarkets-fail"}
	rosterClean(t, ctx, pool, venues...)
	defer rosterClean(t, ctx, pool, venues...)
	seedAcctCreds(t, ctx, pool, "mt5", "3001157")
	seedAcctCreds(t, ctx, pool, "icx.mt5", "777001")
	seedAcct(t, ctx, pool, "mt5", "mt5", "3001157")
	seedAcct(t, ctx, pool, "icmarkets-fail", "icx.mt5", "777001")
	// 败户专用键域清场（venue 名独立——共享行键=venue 寻址）。
	if _, err := pool.Exec(ctx,
		`DELETE FROM app_config WHERE key='icmarkets-fail.session_token'`); err != nil {
		t.Fatal(err)
	}

	c.syncRoster(rctx, func(addr string) (*Client, error) { return c.dial(addr) })
	waitSessionKey(t, ctx, pool, "mt5.session_token")

	// 败户 degraded+last_error 落名册（自报径）。
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if rosterState(t, ctx, pool, "icmarkets-fail") == "degraded" {
			break
		}
		time.Sleep(50 * time.Millisecond) // sched-exempt-ok 测试轮询拍（有界 deadline 等待非生产调度原语）
	}
	if s := rosterState(t, ctx, pool, "icmarkets-fail"); s != "degraded" {
		t.Fatalf("败户 state=%s want degraded", s)
	}
	var le string
	pool.QueryRow(ctx, `SELECT coalesce(last_error,'') FROM fin_venue_roster
		WHERE venue='icmarkets-fail'`).Scan(&le)
	if !strings.Contains(le, "refused") {
		t.Fatalf("last_error 未载败因: %q", le)
	}
	// 「venue接入」首入判词行（esc_scope=venue:login——broker_face 拍注面）。
	// state UPDATE 与 ops_event INSERT 是 reportVenueFail 两拍独立写——
	// degraded 可见≠判词行已落，轮询收敛（败户恒 degraded 不再迁移→行数恒 1）。
	deadline = time.Now().Add(4 * time.Second)
	var n int
	for time.Now().Before(deadline) {
		pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE event='venue接入' AND esc_scope='icmarkets-fail:777001'`).Scan(&n)
		if n >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond) // sched-exempt-ok 测试轮询拍（有界 deadline 等待非生产调度原语）
	}
	if n != 1 {
		t.Fatalf("venue接入判词行=%d want 1（首入 degraded 补行幂等）", n)
	}
	// 他户不连坐——mt5 须达 active（token 落行与 active 自报是两拍写，
	// 单发查=竞态窗——轮询同 degraded 径语义）；败户共享行缺席；
	// 败户环仍存活。
	deadline = time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if rosterState(t, ctx, pool, "mt5") == "active" {
			break
		}
		time.Sleep(50 * time.Millisecond) // sched-exempt-ok 测试轮询拍（有界 deadline 等待非生产调度原语）
	}
	if s := rosterState(t, ctx, pool, "mt5"); s != "active" {
		t.Fatalf("mt5 被连坐 state=%s want active", s)
	}
	var hasTok bool
	pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app_config
		WHERE key='icmarkets-fail.session_token')`).Scan(&hasTok)
	if hasTok {
		t.Fatal("败户不应有 session_token 行（ConnectEx 未过）")
	}
	c.mu.Lock()
	_, mt5Alive := c.runs["mt5"]
	_, failAlive := c.runs["icmarkets-fail"]
	c.mu.Unlock()
	if !mt5Alive || !failAlive {
		t.Fatalf("会话环缺失 mt5=%v fail=%v（败户应续试不拆除）", mt5Alive, failAlive)
	}
	// 败户退避续试实证——首拍败后 reconMin=2s 重试窗内 ConnectEx 应≥2 次
	// （他户 login 计数不混——per-login 键断言隔离面）。
	time.Sleep(2300 * time.Millisecond) // sched-exempt-ok 测试桩退避窗等待（非生产调度原语）
	if n := conn.loginCount(777001); n < 2 {
		t.Fatalf("败户 ConnectEx 续试=%d want ≥2（会话环活性）", n)
	}
	// 判词行幂等——多次 degraded 自报不重复补行（prev=='degraded' 闸）。
	pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='venue接入' AND esc_scope='icmarkets-fail:777001'`).Scan(&n)
	if n != 1 {
		t.Fatalf("venue接入判词行=%d want 1（重试不重复补行）", n)
	}
}

// 案④：Exness 回归锚——venue='mt5' 全键形存量不变（零迁移窗实证）。
func TestExnessKeyCompatibility(t *testing.T) {
	a := VenueAcct{Venue: "mt5", CredNS: "mt5", Login: "3001157"}
	if a.SessionKey() != "mt5.session_token" {
		t.Fatalf("SessionKey=%q want 存量 mt5.session_token", a.SessionKey())
	}
	if a.SymbolsKey() != "mt5.symbols" {
		t.Fatalf("SymbolsKey=%q want 存量 mt5.symbols", a.SymbolsKey())
	}
	if sharedSessionKey != "mt5.session_token" {
		t.Fatalf("sharedSessionKey=%q", sharedSessionKey)
	}
	// CredNSFor 兜底——roster 无行/'mt5' venue=存量单键凭据族语义不变。
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	pool.Exec(ctx, `DELETE FROM fin_venue_roster WHERE venue='mt5'`)
	defer seedAcct(t, ctx, pool, "mt5", "mt5", "3001157") // 共享 schema 复原——后续测不继承缺行态
	ns, err := CredNSFor(ctx, pool, "mt5")
	if err != nil || ns != "mt5" {
		t.Fatalf("CredNSFor mt5 兜底 ns=%q err=%v", ns, err)
	}
	if _, err := CredNSFor(ctx, pool, "nosuchvenue"); err == nil {
		t.Fatal("未知 venue 应 err 显形禁猜键前缀")
	}
	// snapshotOnce 写面——venue='mt5'/channel='mt5_boot' 存量形不变。
	c := &Collector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5", "connect_failed", "subscribe_failed", "stream_dead",
			"symbols_read_failed", "snapshot_write_failed", "tick_write_err", "write_err",
			"balance_ops_err")}
	cl := &Client{mt5: &stubMT5Full{}}
	ar := &acctRun{acct: VenueAcct{Venue: "mt5", CredNS: "mt5"}, cr: Credentials{Login: 3001157}}
	c.snapshotOnce(ctx, cl, ar.acct.Venue+"_boot", ar)
	var ch, vn string
	if err := pool.QueryRow(ctx, `SELECT channel,venue FROM md_account_events
		WHERE channel='mt5_boot' ORDER BY ts DESC LIMIT 1`).Scan(&ch, &vn); err != nil {
		t.Fatal("mt5_boot 事件行缺:", err)
	}
	if vn != "mt5" {
		t.Fatalf("venue=%q want mt5（Exness 写键回归）", vn)
	}
}
