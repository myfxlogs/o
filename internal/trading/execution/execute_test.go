package execution

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"phase/internal/platform/config"
	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
	"phase/internal/trading/okxexec"
)

// fakeVenue 可控 venue（Place 行为按 clOrdID 路由；onPlace=调用序钩子——前置赎回序锚用）。
type fakeVenue struct {
	leg2Fail  bool
	rbFail    bool
	resolveTo string
	placed    []string
	closed    []bool
	onPlace   func()
	reject    bool              // Place 业务拒（sCode≠0 语义——T-0040 留痕锚）
	apiReject *okxexec.APIError // envelope 业务拒走 err 径（T-0042 留痕锚）
}

func (f *fakeVenue) Place(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error) {
	f.placed = append(f.placed, clOrdID+":"+leg.Side)
	f.closed = append(f.closed, leg.Close)
	if f.onPlace != nil {
		f.onPlace()
	}
	if f.apiReject != nil {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, f.apiReject
	}
	if f.reject {
		return OrderResult{ClOrdID: clOrdID, State: "absent", VenueCode: "51000", VenueMsg: "Parameter clOrdId error"}, nil
	}
	if f.leg2Fail && leg.Venue == "leg2" {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, fmt.Errorf("transport down")
	}
	if f.rbFail && contains(clOrdID, "rb") {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, fmt.Errorf("rb transport down")
	}
	return OrderResult{ClOrdID: clOrdID, State: "filled", FillPx: 1, FillSz: 100}, nil
}
func (f *fakeVenue) Resolve(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error) {
	if f.resolveTo != "" {
		return OrderResult{ClOrdID: clOrdID, State: f.resolveTo}, nil
	}
	if f.leg2Fail && leg.Venue == "leg2" {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, fmt.Errorf("resolve down")
	}
	if f.rbFail && contains(clOrdID, "rb") {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, fmt.Errorf("rb resolve down")
	}
	return OrderResult{ClOrdID: clOrdID, State: "filled"}, nil
}
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func setup(t *testing.T, fv Executor) (context.Context, *Runner) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"exec.hedge_band_bps": "20", "invest.lifecycle.live_switch": "true"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, NewRunner(pool, cfg, fv)
}

// mkToken 产 allow 决策+令牌（直连 fin_risk_*——execution 不 import risk）。
func mkToken(t *testing.T, ctx context.Context, r *Runner, used bool, expired bool, dec string) int64 {
	var did int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{}',$1,CASE WHEN $1='allow' THEN 'allow' ELSE 'notional_cap' END,1) RETURNING id`, dec).Scan(&did); err != nil {
		t.Fatal(err)
	}
	exp := "now()+interval '60 seconds'"
	if expired {
		exp = "now()-interval '1 second'"
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_risk_token(decision_id,intent_hash,expires_at,used_at)
		VALUES($1,'x',`+exp+`,CASE WHEN $2 THEN now() ELSE NULL END)`, did, used); err != nil {
		t.Fatal(err)
	}
	return did
}

var twoLegs = [2]Leg{
	{Venue: "okx", InstID: "BTC-USDT", Side: "buy", Sz: "0.01", Notional: 100},
	{Venue: "leg2", InstID: "BTC-USDT-SWAP", Side: "sell", Sz: "1", Notional: 100},
}

func mkIntent(did int64) Intent {
	return Intent{DecisionID: did, StrategyRef: 1, InstID: "BTC-USDT-SWAP", NotionalUSD: 100, Legs: twoLegs}
}

func TestTokenFourRejects(t *testing.T) {
	ctx, r := setup(t, &fakeVenue{})
	for _, tc := range []struct {
		used, expired bool
		dec, want     string
	}{
		{false, false, "deny", "token_not_allow"},
		{true, false, "allow", "token_used"},
		{false, true, "allow", "token_expired"},
	} {
		did := mkToken(t, ctx, r, tc.used, tc.expired, tc.dec)
		if _, err := r.Execute(ctx, mkIntent(did)); err == nil || err.Error() != tc.want {
			t.Fatalf("%s: err=%v", tc.want, err)
		}
	}
	if _, err := r.Execute(ctx, mkIntent(99999)); err == nil || err.Error() != "token_missing" {
		t.Fatalf("token_missing: %v", err)
	}
}

func TestLiveSwitchOff(t *testing.T) {
	ctx, r := setup(t, &fakeVenue{})
	if _, err := r.pool.Exec(ctx,
		"UPDATE app_config SET value='\"false\"' WHERE key='invest.lifecycle.live_switch'"); err != nil {
		t.Fatal(err)
	}
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	did := mkToken(t, ctx, r, false, false, "allow")
	if _, err := r.Execute(ctx, mkIntent(did)); err == nil || err.Error() != "live_switch_off" {
		t.Fatalf("live off: %v", err)
	}
}

func TestHedgeBand(t *testing.T) {
	ctx, r := setup(t, &fakeVenue{})
	did := mkToken(t, ctx, r, false, false, "allow")
	in := mkIntent(did)
	in.Legs[1].Notional = 200 // 差 50% > 20bps
	if _, err := r.Execute(ctx, in); err == nil {
		t.Fatal("对冲带越限未拒")
	}
	did2 := mkToken(t, ctx, r, false, false, "allow")
	if _, err := r.Execute(ctx, mkIntent(did2)); err != nil {
		t.Fatalf("正常带内应过: %v", err)
	}
}

func TestLeggedRollback(t *testing.T) {
	t.Cleanup(func() {})
	fv := &fakeVenue{leg2Fail: true}
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err == nil {
		t.Fatal("leg2 败应 LEGGED")
	}
	var state string
	if err := r.pool.QueryRow(ctx, "SELECT state FROM fin_execution WHERE id=$1", eid).Scan(&state); err != nil || state != "rolled_back" {
		t.Fatalf("state=%s err=%v（应 rolled_back）", state, err)
	}
	var n int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_order_leg WHERE execution_id=$1 AND state='rolled_back'", eid).Scan(&n); err != nil || n != 1 {
		t.Fatalf("回滚腿行=%d", n)
	}
	// position 不开（全收回）
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_position WHERE execution_id=$1", eid).Scan(&n); err != nil || n != 0 {
		t.Fatalf("回滚后仓行=%d", n)
	}
}

func TestRollbackFailedEscalates(t *testing.T) {
	fv := &fakeVenue{leg2Fail: true, rbFail: true}
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, _ := r.Execute(ctx, mkIntent(did))
	var state string
	if err := r.pool.QueryRow(ctx, "SELECT state FROM fin_execution WHERE id=$1", eid).Scan(&state); err != nil || state != "rollback_failed" {
		t.Fatalf("state=%s（应 rollback_failed 升级留痕）", state)
	}
	for i, c := range fv.closed {
		if c && fv.placed[i] != "" && !strings.Contains(fv.placed[i], "rb") {
			t.Fatalf("close 腿应只在 rb 径: %v", fv.placed[i])
		}
	}
	var rbSeen bool
	for i, p := range fv.placed {
		if strings.Contains(p, "rb") && fv.closed[i] {
			rbSeen = true
		}
	}
	if !rbSeen {
		t.Fatal("回滚腿未携 Close 标记——hedge 户反向新开险")
	}
	if r.rej.Snapshot()["execution/rollback_failed"] == 0 {
		t.Fatal("rollback_failed 计数门未计")
	}
}

func TestSingleUseAtomic(t *testing.T) {
	ctx, r := setup(t, &fakeVenue{})
	did := mkToken(t, ctx, r, false, false, "allow")
	if _, err := r.Execute(ctx, mkIntent(did)); err != nil {
		t.Fatalf("首用应成: %v", err)
	}
	if _, err := r.Execute(ctx, mkIntent(did)); err == nil || err.Error() != "token_used" {
		t.Fatalf("复用应拒 token_used: %v", err)
	}
}

// close 径 live_switch 免检锚（T-0021：开仓闸非退仓闸——关闸期仓必须退得掉）。
func TestCloseSkipsLiveSwitch(t *testing.T) {
	ctx, r := setup(t, &fakeVenue{})
	if _, err := r.pool.Exec(ctx,
		"UPDATE app_config SET value='\"false\"' WHERE key='invest.lifecycle.live_switch'"); err != nil {
		t.Fatal(err)
	}
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// open 原锚不破：live off=拒执
	did := mkToken(t, ctx, r, false, false, "allow")
	if _, err := r.Execute(ctx, mkIntent(did)); err == nil || err.Error() != "live_switch_off" {
		t.Fatalf("open live off: %v", err)
	}
	// close 免检：live off 仍放行（令牌照常验——机械防绕过不松）
	did2 := mkToken(t, ctx, r, false, false, "allow")
	in := mkIntent(did2)
	in.Kind = "close"
	if _, err := r.Execute(ctx, in); err != nil {
		t.Fatalf("close 应免检放行: %v", err)
	}
}

// T-0033 返修锚：双腿异乐器断言+sz 非空且≥minSz；派生败 fail-closed。
func TestLegsDerivationAndSz(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// 真 meta（OKX 实勘 2026-10-24）：SWAP ctVal=0.01 **BTC 本位**（ctValCcy=BTC——
	// T-0037 病灶位：旧锚把它当 USD 面值，$25 腿算出 2500 张≈$2M）；px=80950。
	if _, err := pool.Exec(ctx,
		"INSERT INTO md_ticks(ts,inst_id,last) VALUES(now(),'BTC-USDT',90000),(now(),'BTC-USDT-SWAP',80950)"); err != nil {
		t.Fatal(err)
	}
	load := func(ctx context.Context, iid string) (InstMeta, error) {
		if iid == "BTC-USDT-SWAP" {
			return InstMeta{CtVal: 0.01, CtValCcy: "BTC", LotSz: 0.01, MinSz: 0.01}, nil
		}
		return InstMeta{MinSz: 0.0001, LotSz: 0.0001}, nil
	}
	legs, spot, err := BuildLegs(ctx, pool, load, "okx", "BTC-USDT-SWAP", 50)
	if err != nil {
		t.Fatal(err)
	}
	// T-0037 验收锚：$50 双腿→perp 张 ≈25/(0.01×80950)=0.0309→lot 取整 0.03 ∈[0.02,0.04]
	sz, _ := strconv.ParseFloat(legs[0].Sz, 64)
	if sz < 0.02 || sz > 0.04 {
		t.Fatalf("perp 张数=%.4f（应∈[0.02,0.04]——币本位换算未生效，2500=旧病灶）", sz)
	}
	if legs[0].InstID == legs[1].InstID {
		t.Fatal("双腿同乐器——派生失败")
	}
	if spot != "BTC-USDT" || legs[1].InstID != "BTC-USDT" {
		t.Fatalf("派生错: %s", spot)
	}
	if legs[0].Sz == "" || legs[1].Sz == "" {
		t.Fatal("sz 空")
	}
	// 派生败 fail-closed 锚
	if _, _, err := BuildLegs(ctx, pool, load, "okx", "BTC-USDT", 100); err == nil {
		t.Fatal("非 SWAP 乐器应 fail-closed")
	}
	// minSz 闸锚：notional 过小 → perp sz<minSz 拒
	if _, _, err := BuildLegs(ctx, pool, load, "okx", "BTC-USDT-SWAP", 0.5); err == nil {
		t.Fatal("最小张起不来应拒")
	}
	// ctValCcy=USD 分流锚（字段实读不假设）：面值=10 USD → 张=25/10=2.5→floor lot 0.01=2.5
	loadUSD := func(ctx context.Context, iid string) (InstMeta, error) {
		if iid == "BTC-USDT-SWAP" {
			return InstMeta{CtVal: 10, CtValCcy: "USD", LotSz: 0.01, MinSz: 0.01}, nil
		}
		return InstMeta{MinSz: 0.0001, LotSz: 0.0001}, nil
	}
	legs2, _, err := BuildLegs(ctx, pool, loadUSD, "okx", "BTC-USDT-SWAP", 50)
	if err != nil {
		t.Fatal(err)
	}
	if legs2[0].Sz != "2.5" {
		t.Fatalf("USD 面值直除锚: sz=%s（应 2.5）", legs2[0].Sz)
	}
}

// T-0035 锚：路由断言（按 leg.Venue 分派互不串）+cfd 腿换算闸（lot 步长/min 闸）。
func TestRouterAndCfdLegs(t *testing.T) {
	var hit []string
	mk := func(tag string) Executor {
		return fakeVenueTag{tag, &hit}
	}
	r := RouterExecutor{"okx": mk("okx"), "mt5": mk("mt5")}
	_, err := r.Place(context.Background(), Leg{Venue: "mt5", InstID: "BTCUSDm"}, "x1")
	if err != nil || hit[0] != "mt5" {
		t.Fatalf("路由错 hit=%v", hit)
	}
	_, err = r.Place(context.Background(), Leg{Venue: "okx", InstID: "BTC-USDT-SWAP"}, "x2")
	if err != nil || hit[1] != "okx" {
		t.Fatalf("路由错 hit=%v", hit)
	}
	if _, err := r.Place(context.Background(), Leg{Venue: "nope", InstID: "X"}, "x3"); err == nil {
		t.Fatal("无 venue 执行件应拒")
	}
	// cfd 腿：lot 换算+minLot 闸+异 venue 断言+perp 币本位换算锚（T-0037：
	// half=1000 → 1000/(0.01×90000)=1.111→floor=1 张——旧病灶 1000/0.01=10 万张）
	legs, inst, err := BuildCfdLegs("BTC-USDT-SWAP", "BTCUSDm", 2000,
		CfdMeta{ContractSize: 1, MinLot: 0.01, LotStep: 0.01, TradeMode: 4}, 90000, 0.01)
	if err != nil || legs[0].Venue != "okx" || legs[1].Venue != "mt5" || inst != "BTCUSDm" {
		t.Fatalf("cfd 腿构 err=%v inst=%s", err, inst)
	}
	if legs[0].Sz != "1" {
		t.Fatalf("perp 张=%s（应 1——币本位 ×px 换算未生效）", legs[0].Sz)
	}
	if legs[1].Sz == "" {
		t.Fatal("sz 空")
	}
	if _, _, err := BuildCfdLegs("BTC-USDT-SWAP", "", 100, CfdMeta{1, 0.01, 0.01, 4}, 90000, 0.01); err == nil {
		t.Fatal("cfd_map 缺映射应拒")
	}
	if _, _, err := BuildCfdLegs("BTC-USDT-SWAP", "BTCUSDm", 0.001, CfdMeta{1, 0.01, 0.01, 4}, 90000, 0.01); err == nil {
		t.Fatal("minLot 起不来应拒")
	}
}

type fakeVenueTag struct {
	tag string
	hit *[]string
}

func (f fakeVenueTag) Place(ctx context.Context, leg Leg, id string) (OrderResult, error) {
	*f.hit = append(*f.hit, f.tag)
	return OrderResult{ClOrdID: id, State: "filled"}, nil
}
func (f fakeVenueTag) Resolve(ctx context.Context, leg Leg, id string) (OrderResult, error) {
	return OrderResult{ClOrdID: id, State: "filled"}, nil
}

// TestCfdTradeModeGate T-0039 可开性预检锚：TradeMode==FullAccess(4) 才放行；
// CloseOnly(3)/Disabled(0)/单向限(1/2) 拒构（文案带实测值——Exness 动态限防线）。
func TestCfdTradeModeGate(t *testing.T) {
	for tm, label := range map[int32]string{
		0: "Disabled", 1: "LongOnly", 2: "ShortOnly", 3: "CloseOnly",
	} {
		_, _, err := BuildCfdLegs("BTC-USDT-SWAP", "BTCUSDm", 2000,
			CfdMeta{ContractSize: 1, MinLot: 0.01, LotStep: 0.01, TradeMode: tm}, 90000, 0.01)
		if err == nil {
			t.Fatalf("TradeMode=%d(%s) 应拒构", tm, label)
		}
		if !strings.Contains(err.Error(), fmt.Sprintf("TradeMode=%d", tm)) {
			t.Fatalf("拒构文案缺实测值: %v", err)
		}
	}
	// FullAccess 放行（既有换算锚面复证）
	if _, _, err := BuildCfdLegs("BTC-USDT-SWAP", "BTCUSDm", 2000,
		CfdMeta{ContractSize: 1, MinLot: 0.01, LotStep: 0.01, TradeMode: 4}, 90000, 0.01); err != nil {
		t.Fatalf("FullAccess 应放行: %v", err)
	}
}

// T-0040 锚：Place 业务拒（sCode≠0）真拒因进 fin_order_leg.state=failed+拒码留痕。
func TestPlaceRejectCodeLeavesTrace(t *testing.T) {
	fv := &fakeVenue{reject: true}
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err == nil {
		t.Fatal("业务拒应失败收场")
	}
	var state, code, msg string
	if err := r.pool.QueryRow(ctx, `SELECT state,venue_err_code,venue_err_msg FROM fin_order_leg
		WHERE execution_id=$1 AND leg_no=1`, eid).Scan(&state, &code, &msg); err != nil {
		t.Fatal(err)
	}
	if state != "failed" || code != "51000" || msg == "" {
		t.Fatalf("留痕缺失: state=%s code=%q msg=%q（应 failed+51000+真拒因）", state, code, msg)
	}
}

// T-0042 锚：envelope 业务拒（*APIError 走 err 径）拒码进 fin_order_leg.venue_err_*。
func TestPlaceAPIErrorLeavesTrace(t *testing.T) {
	fv := &fakeVenue{apiReject: &okxexec.APIError{Code: "51016", Msg: "Order amount is below minimum"}}
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err == nil {
		t.Fatal("envelope 拒应失败收场")
	}
	var state, code, msg string
	if err := r.pool.QueryRow(ctx, `SELECT state,venue_err_code,venue_err_msg FROM fin_order_leg
		WHERE execution_id=$1 AND leg_no=1`, eid).Scan(&state, &code, &msg); err != nil {
		t.Fatal(err)
	}
	if state != "failed" || code != "51016" || msg == "" {
		t.Fatalf("envelope 拒留痕缺失: state=%s code=%q msg=%q", state, code, msg)
	}
}
