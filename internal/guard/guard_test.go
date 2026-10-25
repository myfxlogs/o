package guard

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/risk"
	"o/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *Warden, *fakeExec) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// 测试面隔离：只清本包自有面（fin_position 系 fixture 行按 guard-test 策略打标
	// 级联清——lifecycle/execution 等包并发共用库，无差别清=撞他包在途行）。
	for _, stmt := range []string{
		`DELETE FROM fin_guard_close WHERE position_id IN
		   (SELECT id FROM fin_position WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref='guard-test'))`,
		`DELETE FROM fin_position WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref='guard-test')`,
		`DELETE FROM fin_order_leg WHERE execution_id IN
		   (SELECT id FROM fin_execution WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref='guard-test'))`,
		`DELETE FROM fin_execution WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref='guard-test')`,
		"DELETE FROM fin_strategy WHERE template_ref='guard-test'",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	for k, v := range map[string]string{
		"guard.flip_bps": "5", "guard.max_hold_hours": "72",
		"guard.retry_max_attempts": "3", "guard.retry_backoff_seconds": "0",
		"exec.hedge_band_bps": "20",
		"risk.kill_switch":    "false", "risk.max_notional_usd": "500",
		"risk.max_venue_pct": "0.35", "risk.max_exposure_usd": "1000",
		"risk.token_ttl_seconds": "30",
		"risk.max_dd_30d":        "0.03", "risk.dd_window_days": "30"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	fx := &fakeExec{}
	w := NewWarden(pool, cfg, risk.NewGate(pool, cfg).WithEquity(func(ctx context.Context, v string) (float64, error) { return 1000, nil }), fx)
	w.AttachSizer(fakeSizer{}) // T-0046 ②：现读面必接（缺=平仓 fail-closed）——默认直通量
	return ctx, w, fx
}

// fakeSizer 平仓量现读假件（默认直通：avail 巨量+无网格；可编程 avail/lot/err）。
type fakeSizer struct {
	avail    float64
	contract float64 // 合约实读（缺省随 avail）
	lot      float64
	errOn    string // instID 后缀命中=读败
}

func (f fakeSizer) SpotAvailBal(ctx context.Context, instID string) (float64, error) {
	if f.errOn != "" && strings.HasSuffix(instID, f.errOn) {
		return 0, errors.New("venue read down")
	}
	if f.avail > 0 {
		return f.avail, nil
	}
	return 1e9, nil
}

func (f fakeSizer) ContractPos(ctx context.Context, instID string) (float64, error) {
	if f.contract > 0 {
		return f.contract, nil
	}
	return f.SpotAvailBal(ctx, instID)
}

func (f fakeSizer) LotSz(ctx context.Context, instID string) (float64, error) {
	return f.lot, nil
}

// fakeExec 执行缝假件——录调用、可编程序失败（execID 定值 999 供衍生仓关闭断言；
// failFirstN=首 N 次瞬时败——dd 续平循环锚用）。
type fakeExec struct {
	calls      []CloseSpec
	fail       error
	failFirstN int
}

const fakeCloseExecID int64 = 999

func (f *fakeExec) ExecuteClose(_ context.Context, spec CloseSpec) (int64, error) {
	f.calls = append(f.calls, spec)
	if f.failFirstN > 0 && len(f.calls) <= f.failFirstN {
		return 0, errors.New("transient venue fail")
	}
	return fakeCloseExecID, f.fail
}

// mkPosition 造一仓双腿 fixture（spot 买+perp 卖=正 funding 收方；名义可调），
// 返 position id。
func mkPosition(t *testing.T, ctx context.Context, w *Warden, stratState string,
	perpSz float64, openedAt time.Time) int64 {
	t.Helper()
	var stratID, execID, posID int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue) VALUES('BTC-USDT-SWAP','guard-test',$1,100,'okx') RETURNING id`,
		stratState).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state) VALUES($1,'BTC-USDT-SWAP',100,'open') RETURNING id`,
		stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	for i, lg := range []struct {
		inst, side string
		sz         float64
	}{
		{"BTC-USDT", "buy", 1}, {"BTC-USDT-SWAP", "sell", perpSz}} {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
			VALUES($1,$2,$3,'okx',$4,$5,$6,'filled')`,
			execID, i+1, fmt.Sprintf("cl-%d-%s", execID, lg.inst), lg.side, lg.inst, lg.sz); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		VALUES($1,$2,'BTC-USDT-SWAP',100,$3) RETURNING id`, execID, stratID, openedAt).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	return posID
}

func seedFacts(t *testing.T, ctx context.Context, w *Warden, rate float64) {
	t.Helper()
	for _, f := range []struct{ inst, last string }{
		{"BTC-USDT", "100"}, {"BTC-USDT-SWAP", "100"}} {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'okx',$1,$2)`, f.inst, f.last); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_funding(ts,venue,inst_id,funding_rate) VALUES(now(),'okx','BTC-USDT-SWAP',$1)`, rate); err != nil {
		t.Fatal(err)
	}
	// 乐器面值现读件（T-0046 ①：SWAP 名义=sz×ctVal×px——perp 腿分派依赖；
	// T-0048 后 facts 表废，注入假件同产线语义）。
	w.AttachCtVal(func(ctx context.Context, venue, instID string) (float64, bool) {
		if v, ok := instrCtVal[instID]; ok {
			return v, true
		}
		return 0, false
	})
}

// instrCtVal 面值假件表（测试可写——面值注入件测试源）。
var instrCtVal = map[string]float64{"BTC-USDT-SWAP": 1}

func seedInstr(t *testing.T, ctx context.Context, w *Warden, ctVal float64) {
	t.Helper()
	instrCtVal["BTC-USDT-SWAP"] = ctVal
}

func posClosed(t *testing.T, ctx context.Context, w *Warden, posID int64) (bool, string, string) {
	t.Helper()
	var closedAt *time.Time
	var state, trig *string
	if err := w.pool.QueryRow(ctx,
		`SELECT p.closed_at, g.state::text, g.trigger_kind::text
		 FROM fin_position p LEFT JOIN fin_guard_close g ON g.position_id=p.id
		 WHERE p.id=$1`, posID).Scan(&closedAt, &state, &trig); err != nil {
		t.Fatal(err)
	}
	if state == nil {
		return false, "open", "" // 无 guard 行=在仓未触发
	}
	return closedAt != nil, *state, *trig
}

func reloadCfg(t *testing.T, ctx context.Context, w *Warden) {
	t.Helper()
	if err := w.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
}

// 锚①：funding 翻面——rate=−10bps 与 sell 腿（收方 +1）异号且 |10bps|>flip_bps(5)。
func TestFundingFlipTrigger(t *testing.T) {
	ctx, w, fx := setup(t)
	seedFacts(t, ctx, w, -0.001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fx.calls) != 1 {
		t.Fatalf("期望一次平仓执行，got %d", len(fx.calls))
	}
	// 反向腿锚：spot buy→sell、perp sell→buy，inst/sz 沿原仓。
	if fx.calls[0].Legs[0].Side != "sell" || fx.calls[0].Legs[1].Side != "buy" ||
		fx.calls[0].Legs[0].InstID != "BTC-USDT" || fx.calls[0].Legs[1].InstID != "BTC-USDT-SWAP" {
		t.Fatalf("双腿反向规格不符: %+v", fx.calls[0].Legs)
	}
	closed, state, trig := posClosed(t, ctx, w, posID)
	if !closed || state != "closed" || trig != "funding_flip" {
		t.Fatalf("仓应 closed/funding_flip，got closed=%v state=%q trig=%q", closed, state, trig)
	}
	var kind, decision string
	if err := w.pool.QueryRow(ctx,
		"SELECT kind::text,decision::text FROM fin_risk_decision ORDER BY id DESC LIMIT 1").Scan(&kind, &decision); err != nil {
		t.Fatal(err)
	}
	if kind != "close" || decision != "allow" {
		t.Fatalf("决策行应 close/allow，got %s/%s", kind, decision)
	}
}

// 锚②：delta 漂移——腿名义 100 vs 200（中差 6667bps>band 20bps），funding 正常不触发翻面。
func TestDeltaDriftTrigger(t *testing.T) {
	ctx, w, fx := setup(t)
	seedFacts(t, ctx, w, 0.0001)
	posID := mkPosition(t, ctx, w, "live", 2, time.Now())
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	_, state, trig := posClosed(t, ctx, w, posID)
	if state != "closed" || trig != "delta_drift" {
		t.Fatalf("仓应 closed/delta_drift，got state=%q trig=%q calls=%d", state, trig, len(fx.calls))
	}
}

// 锚③：持仓超龄——100h>max_hold(72h)，facts 正常不先触发。
func TestMaxHoldTrigger(t *testing.T) {
	ctx, w, _ := setup(t)
	seedFacts(t, ctx, w, 0.0001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now().Add(-100*time.Hour))
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	_, state, trig := posClosed(t, ctx, w, posID)
	if state != "closed" || trig != "max_hold" {
		t.Fatalf("仓应 closed/max_hold，got state=%q trig=%q", state, trig)
	}
}

// 锚④：策略降级联动——degraded 必平（facts 全正常也平）。
func TestStrategyDegradedTrigger(t *testing.T) {
	ctx, w, _ := setup(t)
	seedFacts(t, ctx, w, 0.0001)
	posID := mkPosition(t, ctx, w, "degraded", 1, time.Now())
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	_, state, trig := posClosed(t, ctx, w, posID)
	if state != "closed" || trig != "strategy_degraded" {
		t.Fatalf("仓应 closed/strategy_degraded，got state=%q trig=%q", state, trig)
	}
}

// 锚⑤：kill_switch=true 下 close 仍放行（E2E 过仓巡径）；失败→close_failed→节拍重试→closed。
func TestCloseRetryAfterFailure(t *testing.T) {
	ctx, w, fx := setup(t)
	seedFacts(t, ctx, w, -0.001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	if _, err := w.pool.Exec(ctx, `UPDATE app_config SET value='"true"' WHERE key='risk.kill_switch'`); err != nil {
		t.Fatal(err)
	}
	reloadCfg(t, ctx, w)
	fx.fail = errors.New("venue 歧义")
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	closed, state, _ := posClosed(t, ctx, w, posID)
	if closed || state != "close_failed" {
		t.Fatalf("执行败应 close_failed，got closed=%v state=%q", closed, state)
	}
	var attempts int
	if err := w.pool.QueryRow(ctx,
		"SELECT attempts FROM fin_guard_close WHERE position_id=$1", posID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempts=1 期望，got %d", attempts)
	}
	fx.fail = nil
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	closed, state, trig := posClosed(t, ctx, w, posID)
	if !closed || state != "closed" || trig != "funding_flip" {
		t.Fatalf("重试应 closed，got closed=%v state=%q trig=%q", closed, state, trig)
	}
	if len(fx.calls) != 2 {
		t.Fatalf("两次执行调用期望，got %d", len(fx.calls))
	}
}

// 锚⑥：重试耗尽——attempts 达 guard.retry_max_attempts 后不再扫（升级不沉默）。
func TestCloseGiveUp(t *testing.T) {
	ctx, w, fx := setup(t)
	seedFacts(t, ctx, w, -0.001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	fx.fail = errors.New("venue 持续败")
	for i := 0; i < 3; i++ {
		if err := w.scanOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if len(fx.calls) != 3 {
		t.Fatalf("三次重试期望，got %d", len(fx.calls))
	}
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fx.calls) != 3 {
		t.Fatalf("耗尽后应停扫，got %d 次调用", len(fx.calls))
	}
	_, state, _ := posClosed(t, ctx, w, posID)
	if state != "close_failed" {
		t.Fatalf("耗尽仓应停留 close_failed 待人工，got %q", state)
	}
	if n := w.rej.Snapshot()["guard/close_gave_up"]; n != 1 {
		t.Fatalf("close_gave_up 计数=1 期望，got %d", n)
	}
}

// 锚⑦：判据键缺=fail-closed 跳过该触发面（仓不误平、循环不炸）。
func TestMissingKeyFailClosed(t *testing.T) {
	ctx, w, fx := setup(t)
	seedFacts(t, ctx, w, -0.001) // 翻面条件成立
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	if _, err := w.pool.Exec(ctx, `DELETE FROM app_config WHERE key='guard.flip_bps'`); err != nil {
		t.Fatal(err)
	}
	reloadCfg(t, ctx, w)
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fx.calls) != 0 {
		t.Fatalf("flip_bps 缺键应跳过触发不误平，got %+v", fx.calls)
	}
	closed, _, _ := posClosed(t, ctx, w, posID)
	if closed {
		t.Fatal("仓应仍在")
	}
	// 节拍键缺=整轮 fail-closed 报错不炸。
	if _, err := w.pool.Exec(ctx, `DELETE FROM app_config WHERE key='guard.retry_backoff_seconds'`); err != nil {
		t.Fatal(err)
	}
	reloadCfg(t, ctx, w)
	if err := w.scanOnce(ctx); err == nil {
		t.Fatal("retry 键缺应整轮报错 fail-closed")
	}
}

// T-0046 ①锚：SWAP 名义=sz×ctVal×px（0.03 张 ctVal=0.01 → ~$24 级非 $2421）；
// 两腿均衡名义（$25 vs $25）漂移 < band 不误触发。
func TestNotionalCtValDispatch(t *testing.T) {
	ctx, w, _ := setup(t)
	seedFacts(t, ctx, w, 0.0001) // tick=100
	seedInstr(t, ctx, w, 0.01)   // ctVal=0.01 BTC/张
	n, ok := w.notionalNow(ctx, leg{inst: "BTC-USDT-SWAP", venue: "okx", sz: 0.03})
	if !ok || n != 0.03*0.01*100 {
		t.Fatalf("SWAP 名义 sz×ctVal×px: got %v ok=%v want %v", n, ok, 0.03*0.01*100)
	}
	ns, ok := w.notionalNow(ctx, leg{inst: "BTC-USDT", venue: "okx", sz: 0.25})
	if !ok || ns != 25 {
		t.Fatalf("SPOT 名义 sz×px: got %v ok=%v want 25", ns, ok)
	}
	// 均衡仓（spot 0.25 BTC=$25 vs perp 25 张×0.01×100=$25）——漂移 0 不触发。
	fx := &fakeExec{}
	w2 := NewWarden(w.pool, w.cfg, w.gate, fx)
	w2.AttachSizer(fakeSizer{})
	var stratID, execID, posID int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue) VALUES('BTC-USDT-SWAP','guard-test','live',25,'okx') RETURNING id`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state) VALUES($1,'BTC-USDT-SWAP',25,'open') RETURNING id`, stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	for i, lg := range []struct {
		inst, side string
		sz         float64
	}{{"BTC-USDT", "buy", 0.25}, {"BTC-USDT-SWAP", "sell", 25}} {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
			VALUES($1,$2,$3,'okx',$4,$5,$6,'filled')`, execID, i+1, fmt.Sprintf("cl-x-%d", i), lg.side, lg.inst, lg.sz); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		VALUES($1,$2,'BTC-USDT-SWAP',25,now()) RETURNING id`, execID, stratID).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	if err := w2.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fx.calls) != 0 {
		t.Fatalf("均衡仓误触发平仓（量纲病灶体征）: %+v", fx.calls)
	}
	// ctVal 读不到=fail-closed 不误判（facts_missing 计数不触发）。
	w.AttachCtVal(nil)
	if n, ok := w.notionalNow(ctx, leg{inst: "BTC-USDT-SWAP", venue: "okx", sz: 1}); ok || n != 0 {
		t.Fatal("ctVal 缺行应 false")
	}
}

// T-0046 ②锚：平仓量 venue 实读对齐——avail<腿记录取 avail 落网格；读败=fail-closed。
func TestSizedCloseLegs(t *testing.T) {
	ctx, w, _ := setup(t)
	cls := [2]CloseLeg{
		{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: 0.00030897},
		{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: 3}}
	// avail=0.00030866（taker 费内扣实持）、lot=0.0000001——0.00030897→0.0003086。
	w.AttachSizer(fakeSizer{avail: 0.00030866, contract: 1e9, lot: 0.0000001})
	got, ok := w.sizedCloseLegs(ctx, cls)
	if !ok || math.Abs(got[0].Sz-0.0003086) > 1e-12 {
		t.Fatalf("spot 平仓量应对齐 avail 网格: ok=%v sz=%.8f", ok, got[0].Sz)
	}
	if got[1].Sz != 3 { // perp avail 直通（同 sizer avail>3）
		t.Fatalf("perp 平仓量应沿实读: %v", got[1].Sz)
	}
	w.AttachSizer(fakeSizer{errOn: "-USDT"})
	if _, ok := w.sizedCloseLegs(ctx, cls); ok {
		t.Fatal("venue 读败应 fail-closed 拒平")
	}
	w.AttachSizer(fakeSizer{avail: 0.000000001, lot: 0.0000001})
	if _, ok := w.sizedCloseLegs(ctx, cls); ok {
		t.Fatal("对齐后零量应 fail-closed 拒平（禁发残量单）")
	}
}

// T-0046 ④锚：getF 双型——20 / "20" / "20.5" 三形同值（裸数字废闸根修）。
func TestGetFDualType(t *testing.T) {
	ctx, w, _ := setup(t)
	for i, v := range []string{"20", "20.5", "0.5"} {
		k := "guard.test_dual"
		// 裸数字形（jsonb 原生 number）与字符串形交替写。
		raw := v
		if i == 0 {
			raw = v // to_jsonb text→"20"；裸数字走下面 exec
		}
		_ = raw
		if i == 0 {
			if _, err := w.pool.Exec(ctx,
				`INSERT INTO app_config(key,value) VALUES('guard.test_dual','20'::jsonb)
				 ON CONFLICT(key) DO UPDATE SET value='20'::jsonb`); err != nil {
				t.Fatal(err)
			}
		} else {
			if _, err := w.pool.Exec(ctx,
				`INSERT INTO app_config(key,value) VALUES('guard.test_dual',to_jsonb($1::text))
				 ON CONFLICT(key) DO UPDATE SET value=to_jsonb($1::text)`, v); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.cfg.Load(ctx); err != nil {
			t.Fatal(err)
		}
		want := 20.0
		if v == "20.5" {
			want = 20.5
		}
		if v == "0.5" {
			want = 0.5
		}
		got, err := w.getF(k)
		if err != nil || got != want {
			t.Fatalf("形 %q: getF=%v err=%v want %v（单型废闸病灶）", v, got, err, want)
		}
	}
}

// T-0046 ⑥锚：markClosed 同关 fin_execution（原仓 exec+平仓 exec 行 state=closed）。
func TestMarkClosedClosesExecution(t *testing.T) {
	ctx, w, _ := setup(t)
	seedFacts(t, ctx, w, -0.001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	// 平仓 exec fixture（fakeExec 定值 999——markClosed 应同关此行）。
	var stratID int64
	if err := w.pool.QueryRow(ctx,
		"SELECT strategy_ref FROM fin_position WHERE id=$1", posID).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO fin_execution(id,strategy_ref,inst_id,notional_usd,state)
		 VALUES($1,$2,'BTC-USDT-SWAP',100,'open')`, fakeCloseExecID, stratID); err != nil {
		t.Fatal(err)
	}
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM fin_execution e
		JOIN fin_position p ON p.execution_id=e.id
		WHERE p.id=$1::bigint AND e.state='closed'`, posID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("原仓 exec 行未关（sweepIdle 闸堵病灶）: n=%d err=%v", n, err)
	}
	var cn int
	if err := w.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_execution WHERE id=$1 AND state='closed'", fakeCloseExecID).Scan(&cn); err != nil || cn != 1 {
		t.Fatalf("平仓 exec 行未关: n=%d err=%v", cn, err)
	}
}

// mkPosLegs 造指定腿量仓（T-0064 漂移基线锚用——返回 posID）。
func mkPosLegs(t *testing.T, ctx context.Context, w *Warden, spotSz, perpSz float64) int64 {
	t.Helper()
	var stratID, execID, posID int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue) VALUES('ETH-USDT-SWAP','guard-test','live',50,'okx') RETURNING id`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state) VALUES($1,'ETH-USDT-SWAP',50,'open') RETURNING id`, stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	for i, lg := range []struct {
		inst, side string
		sz         float64
	}{{"ETH-USDT", "buy", spotSz}, {"ETH-USDT-SWAP", "sell", perpSz}} {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
			VALUES($1,$2,$3,'okx',$4,$5,$6,'filled')`, execID, i+1, fmt.Sprintf("imb-%d-%s", execID, lg.inst), lg.side, lg.inst, lg.sz); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		VALUES($1,$2,'ETH-USDT-SWAP',50,now()) RETURNING id`, execID, stratID).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	return posID
}

func seedEthFacts(t *testing.T, ctx context.Context, w *Warden, spotPx, perpPx string) {
	t.Helper()
	for inst, px := range map[string]string{"ETH-USDT": spotPx, "ETH-USDT-SWAP": perpPx} {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'okx',$1,$2)`, inst, px); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_funding(ts,venue,inst_id,funding_rate) VALUES(now(),'okx','ETH-USDT-SWAP',0.0001)`); err != nil {
		t.Fatal(err)
	}
	instrCtVal["ETH-USDT-SWAP"] = 0.01
	w.AttachCtVal(func(ctx context.Context, venue, instID string) (float64, bool) {
		if v, ok := instrCtVal[instID]; ok {
			return v, true
		}
		return 0, false
	})
}

// T-0064 锚①：出生粒度残差（~300bps）入基线不杀——两拍零触发+基线落列。
func TestDriftBaselineNoKill(t *testing.T) {
	ctx, w, fx := setup(t)
	posID := mkPosLegs(t, ctx, w, 0.2425, 25) // imb≈(24.25−25)/24.625≈−304bps
	seedEthFacts(t, ctx, w, "100", "100")
	for i := 0; i < 2; i++ {
		if err := w.scanOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if len(fx.calls) != 0 {
		t.Fatalf("基线残差不应杀: %+v", fx.calls)
	}
	var imb *float64
	if err := w.pool.QueryRow(ctx,
		"SELECT open_imbalance_bps FROM fin_position WHERE id=$1", posID).Scan(&imb); err != nil || imb == nil || *imb > -250 || *imb < -350 {
		t.Fatalf("基线未落列/值错: %v err=%v", imb, err)
	}
	_ = posID
}

// T-0064 锚②：基线外偏移 >band 照杀（spot 价走 100→100.5，imb 偏移>20bps）。
func TestDriftOffsetKills(t *testing.T) {
	ctx, w, fx := setup(t)
	mkPosLegs(t, ctx, w, 0.2425, 25)
	seedEthFacts(t, ctx, w, "100", "100")
	if err := w.scanOnce(ctx); err != nil { // 基线播种拍
		t.Fatal(err)
	}
	seedEthFacts(t, ctx, w, "100.5", "100") // spot 腿走强——imb 偏移≈200bps>20
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fx.calls) == 0 {
		t.Fatal("基线外真漂移应杀")
	}
}

// T-0064 锚③：绝对兜底闸——imb>500bps 无条件杀（基线也拦不住）。
func TestDriftAbsCap(t *testing.T) {
	ctx, w, fx := setup(t)
	mkPosLegs(t, ctx, w, 0.2350, 25) // imb≈(23.5−25)/24.25≈−619bps>500
	seedEthFacts(t, ctx, w, "100", "100")
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fx.calls) == 0 {
		t.Fatal("绝对兜底闸应杀（裸腿级失衡）")
	}
}
