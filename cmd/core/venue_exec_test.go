// venue_exec_test.go —— O-37449 名册驱动执行路由桩测：三户路由键/
// disabled 即拍拒单零 Exness 调用/名册缺席静态对兜底/方言缺映射
// esc_scope=venue 上浮/Exness 回归。
package main

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/okxexec"
)

// xvSeedRoster 名册夹具（三户：exness active/xmglobal active/icldemo disabled
// +ghost 未在册对照——execute 授权面语义全覆）。
func xvSeedRoster(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	for _, r := range []struct{ v, p, st, login string }{
		{"exness", "cfd", "active", "3001157"}, {"xmglobal", "cfd", "active", "110660304"},
		{"icldemo", "cfd", "disabled", "555001"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_roster
			(venue,proto,cred_ns,login,roles,state,alias)
			VALUES($1,$2,$3,$4,'{collect,execute}',$5,'t')
			ON CONFLICT (venue) DO UPDATE SET state=EXCLUDED.state,roles=EXCLUDED.roles,login=EXCLUDED.login`,
			r.v, r.p, r.v+".mt5", r.login, r.st); err != nil {
			t.Fatal("名册夹具:", err)
		}
	}
}

// xvCountCli xvenueSess 桩（account 调用计数——「零 Exness 调用」断言面）。
type xvCountCli struct {
	acct   string
	calls  *int32
	mt5Cli *xvMT5Fake
}

func (f *xvCountCli) account(context.Context) (string, error) {
	atomic.AddInt32(f.calls, 1)
	return f.acct, nil
}
func (f *xvCountCli) Authed(ctx context.Context) context.Context { return ctx }
func (f *xvCountCli) Conn() *grpc.ClientConn                     { return nil }
func (f *xvCountCli) MT5() mt5pb.MT5Client {
	if f.mt5Cli != nil {
		return f.mt5Cli
	}
	return xvMT5Fake{}
}
func (f *xvCountCli) Suspect() {}

// xvRecDial 方言桩（VenueSymbol 调用记录——XM 户译名径断言面）。
type xvRecDial struct {
	got *[]string
	sym string
}

func (d xvRecDial) VenueSymbol(_ context.Context, canonical string) (string, error) {
	*d.got = append(*d.got, canonical)
	return d.sym, nil
}
func (xvRecDial) Escalate(context.Context, string, string) {}

func xvFleet(t *testing.T, ctx context.Context, pool *pgxpool.Pool, cfg *config.Client,
	deps map[string]*venueExecDeps) *xvenueFleet {
	t.Helper()
	f := newXvenueFleet(pool, cfg, okxCredsZero, &venueExecDeps{
		venue: "exness", cli: &xvSessFake{mt5Cli: xvMT5Fake{}, acct: "3001157"},
		dial: xvDialFake{}, ds: nil})
	f.ds = mt5.NewDialectStore(pool)
	f.newDep = func(v string) *venueExecDeps { return deps[v] }
	return f
}

var okxCredsZero = okxexec.Credentials{}

// 锚①：三户路由键——depsFor 按 venue 各取各件（XM 注假件不碰 Exness 会话）。
func TestXvenueFleetThreeVenueRouting(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	xvSeedRoster(t, ctx, pool)
	var xmDialCalls, mt5Calls int32
	var dialSeen []string
	xmCli := &xvCountCli{acct: "110660304", calls: &xmDialCalls}
	f := xvFleet(t, ctx, pool, cfg, map[string]*venueExecDeps{
		"xmglobal": {venue: "xmglobal", cli: xmCli,
			dial: xvRecDial{got: &dialSeen, sym: "BTCUSD#"}, ds: mt5.NewDialectStore(pool)},
	})
	// xmglobal→注假三件；mt5→预挂存量件
	d1, err := f.depsFor(ctx, "xmglobal")
	if err != nil || d1.cli != xmCli {
		t.Fatalf("xmglobal deps 路由错: %v", err)
	}
	if _, err := f.depsFor(ctx, "exness"); err != nil {
		t.Fatal("mt5 deps:", err)
	}
	// executorFor：xmglobal→mtexec 懒连件；icldemo disabled→即拍拒；
	// ghost 未在册+名册活→fail-closed 不回退 mt5（替户下单禁线）
	if _, err := f.executorFor(ctx, "icldemo"); err == nil ||
		!strings.Contains(err.Error(), "未授权执行") {
		t.Fatalf("disabled 户未拒: %v", err)
	}
	if _, err := f.executorFor(ctx, "ghostvenue"); err == nil ||
		!strings.Contains(err.Error(), "不在册") {
		t.Fatalf("未在册 venue 未拒（禁回退 mt5）: %v", err)
	}
	if ex, err := f.executorFor(ctx, "xmglobal"); err != nil || ex == nil {
		t.Fatalf("xmglobal executor 缺: %v", err)
	}
	_ = mt5Calls // mt5 计数面留待 e2e 锚
	_ = dialSeen
}

// 锚②：disabled 户 Place=零 Exness 调用（拒单发生在 depsFor 前——
// 会话件 account() 计数恒零=替户下单物理不可达）。
func TestXvenueFleetDisabledZeroExnessCall(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	xvSeedRoster(t, ctx, pool)
	var mt5N, xmN int32
	primary := &xvCountCli{acct: "3001157", calls: &mt5N, mt5Cli: &xvMT5Fake{}}
	f := newXvenueFleet(pool, cfg, okxCredsZero, &venueExecDeps{
		venue: "exness", cli: primary, dial: xvDialFake{}, ds: mt5.NewDialectStore(pool)})
	f.newDep = func(v string) *venueExecDeps {
		return &venueExecDeps{venue: v, cli: &xvCountCli{acct: "9", calls: &xmN},
			dial: xvRecDial{got: &[]string{}, sym: "X"}, ds: f.ds}
	}
	leg := execution.Leg{Venue: "icldemo", InstID: "XAUUSD#", Side: "buy", Sz: "0.01"}
	if _, err := f.Place(ctx, leg, "ph1x1x1"); err == nil {
		t.Fatal("disabled 户 Place 未拒")
	}
	if n := atomic.LoadInt32(&mt5N) + atomic.LoadInt32(&xmN); n != 0 {
		t.Fatalf("disabled 拒单仍触会话件（mt5=%d xm=%d——替户下单险敞开）", mt5N, xmN)
	}
}

// 锚③：名册读败/授权面零行=静态对兜底（RENAME 模拟迁移窗——
// mt5/okx 照旧路由，xmglobal 无静态件=err 不兜底造假）。
func TestXvenueFleetRosterAbsentFallback(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	if _, err := pool.Exec(ctx, `ALTER TABLE fin_venue_roster RENAME TO fin_venue_roster_bak`); err != nil {
		t.Skip("roster 表缺席不可测（他测序依赖）:", err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, `ALTER TABLE fin_venue_roster_bak RENAME TO fin_venue_roster`); err != nil {
			t.Log("roster 复原败（后续测或受影响）:", err)
		}
	}()
	var mt5N int32
	f := newXvenueFleet(pool, cfg, okxCredsZero, &venueExecDeps{
		venue: "exness", cli: &xvCountCli{acct: "3001157", calls: &mt5N},
		dial: xvDialFake{}, ds: mt5.NewDialectStore(pool)})
	if _, err := f.executorFor(ctx, "exness"); err != nil {
		t.Fatalf("名册缺席 mt5 未兜底: %v", err)
	}
	if _, err := f.executorFor(ctx, "okx"); err != nil {
		t.Fatalf("名册缺席 okx 未兜底: %v", err)
	}
	if _, err := f.executorFor(ctx, "xmglobal"); err == nil {
		t.Fatal("名册缺席 xmglobal 竟兜底成功——造假执行件")
	}
}

// 锚④：方言缺映射 esc_scope=venue 上浮（XM 户 Escalate 落账
// esc_scope='xmglobal' 非 'mt5'——替户归因不串户）。
func TestMt5LiveDialectVenueEscalateScope(t *testing.T) {
	ctx, pool, _ := setupDB(t)
	d := mt5LiveDialect{S: mt5.NewDialectStore(pool), Seat: "o-core", venue: "xmglobal",
		cli: &mt5LazyCli{pool: pool, cfg: nil, venue: "xmglobal"}}
	d.Escalate(ctx, "BTCUSD#", "查无映射")
	var scope string
	if err := pool.QueryRow(ctx, `SELECT esc_scope FROM ops_event
		WHERE seat='o-core' AND event='方言缺映射' ORDER BY id DESC LIMIT 1`).Scan(&scope); err != nil {
		t.Fatal("升件行缺:", err)
	}
	if scope != "xmglobal" {
		t.Fatalf("esc_scope=%q 应 xmglobal（替户下单归因串户险）", scope)
	}
	// 'exness' 兼容锚——空 venue 译件 esc_scope='exness' 回归（O-39440①）
	d2 := mt5LiveDialect{S: mt5.NewDialectStore(pool), Seat: "o-core"}
	d2.Escalate(ctx, "BTCUSDm", "x")
	if err := pool.QueryRow(ctx, `SELECT esc_scope FROM ops_event
		WHERE seat='o-core' AND event='方言缺映射' ORDER BY id DESC LIMIT 1`).Scan(&scope); err != nil {
		t.Fatal(err)
	}
	if scope != "exness" {
		t.Fatalf("exness 兼容锚 esc_scope=%q", scope)
	}
}

// 锚⑤：xvenueOpen XM 腿 venue 透传——pair 副腿 venue='xmglobal'→
// 腿构 Venue='xmglobal'+RouterExecutor 按键路由 XM fake（Exness fake
// 零调用=替户下单禁线端到端锚）。
func TestXvenueOpenXMRoute(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	xvSeedRoster(t, ctx, pool)
	for k, v := range map[string]string{
		"exec.xvenue_open": "true", "exec.probe_notional_usd": "50",
		"exec.reopen_cooldown_sec": "0", "exec.hedge_band_bps": "20",
		"invest.lifecycle.live_switch": "true", "risk.max_notional_usd": "5000",
		"risk.max_venue_pct": "0.9", "risk.max_exposure_usd": "100000",
		"risk.token_ttl_seconds": "60",
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO app_config(key,value)
			VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, k, v); err != nil {
			t.Fatal("闸键夹具:", err)
		}
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// pair：主行 okx+副腿 xmglobal（mkXvenuePairV 变体——venue 参数化）
	var eid, pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('BTC-USDT-SWAP','xvenue-carry','long_perp_short_cfd',0.08,'proposed','[]','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,state,net_apr)
		VALUES($1,'BTC-USDT-SWAP','long_perp_short_cfd','fin_armed',0) RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	var mainID, mateID int64
	for _, row := range []struct{ inst, venue string }{{"BTC-USDT-SWAP", "okx"}, {"BTCUSD#", "xmglobal"}} {
		var sid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
			VALUES($1,'xvenue-carry',$2,'long_perp_short_cfd',50,'live',$3,'{}') RETURNING id`,
			row.inst, pid, row.venue).Scan(&sid); err != nil {
			t.Fatal("策略夹具:", err)
		}
		if row.venue == "okx" {
			mainID = sid
		} else {
			mateID = sid
		}
	}
	// XM 方言行=fin_0249 迁移已落（exec 径 canonical=cfd venue 符号自指形
	// ——迁移断言即夹具，重 INSERT 撞 PK 不另铺）。
	for _, inst := range []string{"BTC-USDT", "BTC-USDT-SWAP"} {
		if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(venue,inst_id,last,ts)
			VALUES('okx',$1,100,now()) ON CONFLICT (venue,inst_id) DO UPDATE SET last=EXCLUDED.last,ts=EXCLUDED.ts`, inst); err != nil {
			t.Fatal("价夹具:", err)
		}
	}
	// 路由观察面：RouterExecutor 双 fake 按 venue 键分派
	var seq []string
	var xmN, mt5N int32
	xmExec := &xvExecFake{seq: &seq, name: "xmglobal", ticket: 888}
	mt5Exec := &xvExecFake{seq: &seq, name: "mt5", ticket: 777}
	r := execution.NewRunner(pool, cfg, execution.RouterExecutor{
		"exness": mt5Exec, "xmglobal": xmExec,
		"okx": &xvExecFake{seq: &seq, name: "okx", ticket: 666}})
	gate := risk.NewGate(pool, cfg).WithEquity(func(context.Context, string) (float64, error) {
		return 1e6, nil
	})
	// fleet：xmglobal 注假三件（acct/dial/方言仓——SymbolParams 活读桩）
	var dialSeen []string
	xmCli := &xvCountCli{acct: "110660304", calls: &xmN, mt5Cli: &xvMT5Fake{}}
	fleet := newXvenueFleet(pool, cfg, okxCredsZero, &venueExecDeps{
		venue: "exness", cli: &xvCountCli{acct: "3001157", calls: &mt5N, mt5Cli: &xvMT5Fake{}},
		dial: xvDialFake{}, ds: mt5.NewDialectStore(pool)})
	fleet.newDep = func(v string) *venueExecDeps {
		return &venueExecDeps{venue: v, cli: xmCli,
			dial: xvRecDial{got: &dialSeen, sym: "BTCUSD#"}, ds: mt5.NewDialectStore(pool)}
	}
	h := hedgeOpener{pool: pool, cfg: cfg, gate: gate, r: r, xf: fleet,
		xmeta: func(context.Context, string) (execution.InstMeta, error) {
			return execution.InstMeta{CtVal: 0.01, CtValCcy: "BTC", LotSz: 1, MinSz: 0.1}, nil
		}}
	if _, err := h.xvenueOpen(ctx, xvSpec(mainID, "BTC-USDT-SWAP")); err != nil {
		t.Fatal("xvenueOpen XM 径:", err)
	}
	// 断言：XM fake 收 CFD 腿 place（Venue='xmglobal' 路由键命中）；
	// Exness fake 零调用=替户下单禁线端到端锚。
	var xmPlace, mt5Place bool
	for _, s := range seq {
		if strings.HasPrefix(s, "xmglobal:place") {
			xmPlace = true
		}
		if strings.HasPrefix(s, "mt5:place") {
			mt5Place = true
		}
	}
	if !xmPlace || mt5Place {
		t.Fatalf("路由错：seq=%v（xm 收腿=%v mt5 收腿=%v）", seq, xmPlace, mt5Place)
	}
	if len(dialSeen) == 0 || dialSeen[0] != "BTCUSD#" {
		t.Fatalf("方言译名径错: %v", dialSeen)
	}
	_ = mateID
}
