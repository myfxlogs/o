package execution

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
	"o/migrations"
)

// fakeVenue 可控 venue（Place 行为按 clOrdID 路由；onPlace=调用序钩子——前置赎回序锚用）。
type fakeVenue struct {
	leg2Fail     bool
	rbFail       bool
	resolveTo    string
	placed       []string
	closed       []bool
	onPlace      func()
	reject       bool              // Place 业务拒（sCode≠0 语义——T-0040 留痕锚）
	apiReject    *okxexec.APIError // envelope 业务拒走 err 径（T-0042 留痕锚）
	resolveCalls int
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
	f.resolveCalls++
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
	// T-0064 追加锚：spot 腿 quote_ccy 径——sz=perp 实名义 USD 额（venue 内换算）
	// +TgtQuote 位；perpUSD=0.03×0.01×80950=$24.285。
	if !legs[1].TgtQuote {
		t.Fatalf("spot 腿应 TgtQuote（quote_ccy 市价径）: %+v", legs[1])
	}
	qsz, _ := strconv.ParseFloat(legs[1].Sz, 64)
	perpUSD, _ := strconv.ParseFloat(legs[0].Sz, 64)
	perpUSD *= 0.01 * 80950
	if qsz < perpUSD*0.999 || qsz > perpUSD*1.001 {
		t.Fatalf("quote 委托额应=perp 实名义: qsz=%.4f perpUSD=%.4f", qsz, perpUSD)
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

// T-0042 返修锚：envelope 拒分码族——51xxx 参数/业务拒族=确定单未建，直接
// absent 留痕、禁查单（Resolve 零调用）。
func TestPlaceAPIError51FamilyNoResolve(t *testing.T) {
	fv := &fakeVenue{apiReject: &okxexec.APIError{Code: "51016", Msg: "Order amount is below minimum"}, resolveTo: "filled"}
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err == nil {
		t.Fatal("51 族拒应失败收场")
	}
	if fv.resolveCalls != 0 {
		t.Fatalf("51 族拒禁走 Resolve（单确定未建）: resolveCalls=%d", fv.resolveCalls)
	}
	var state, code string
	if err := r.pool.QueryRow(ctx, `SELECT state,venue_err_code FROM fin_order_leg
		WHERE execution_id=$1 AND leg_no=1`, eid).Scan(&state, &code); err != nil {
		t.Fatal(err)
	}
	if state != "failed" || code != "51016" {
		t.Fatalf("51 族留痕: state=%s code=%q", state, code)
	}
}

// T-0042 返修锚：非 51 族（50004 超时等）=歧态势——留痕后仍走 Resolve 定真态
// （禁直接 absent），拒码透传进 leg 行。
func TestPlaceAPIErrorNon51FamilyResolves(t *testing.T) {
	fv := &fakeVenue{apiReject: &okxexec.APIError{Code: "50004", Msg: "The request timed out"}, resolveTo: "filled"}
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err != nil {
		t.Fatalf("50004→Resolve=filled 应成功收场: %v", err)
	}
	if fv.resolveCalls == 0 {
		t.Fatal("非 51 族拒必须走 Resolve 定真态（单或已建=歧态势）")
	}
	var state, code string
	if err := r.pool.QueryRow(ctx, `SELECT state,venue_err_code FROM fin_order_leg
		WHERE execution_id=$1 AND leg_no=1`, eid).Scan(&state, &code); err != nil {
		t.Fatal(err)
	}
	if state != "filled" || code != "50004" {
		t.Fatalf("50004 留痕透传: state=%s code=%q（应 filled+50004）", state, code)
	}
}

// T-0046 ③锚：close Intent 的 LEGGED 回滚=重开原向仓（Close=false——恒 true=
// reduceOnly 打空仓拒单→歧态势，exec5 实证 canon rollback-close-direction）。
func TestRollbackCloseIntentReopens(t *testing.T) {
	fv := &fakeVenue{leg2Fail: true}
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	in := mkIntent(did)
	in.Kind = "close"
	in.Legs[0].Close = true // 平仓 Intent 双腿 Close=true（cmd closeLegOf 语义）
	in.Legs[1].Close = true
	eid, err := r.Execute(ctx, in)
	if err == nil {
		t.Fatal("leg2 败应 LEGGED")
	}
	// 回滚腿（clOrdID 含 rb）Close 必为 false（重开非再平）。
	for i, p := range fv.placed {
		if contains(p, "rb") && fv.closed[i] {
			t.Fatalf("close Intent 回滚腿 Close 应 false（重开语义）: %v", p)
		}
		if contains(p, "rb") && !fv.closed[i] {
			// 重开装配核：无 reduceOnly；SWAP 重开 posSide=开仓向（spot 免带）。
			if bp := buildOrderParams(Leg{InstID: "BTC-USDT", Side: "buy"}, "phx", "long_short_mode"); bp.ReduceOnly {
				t.Fatalf("spot 重开腿带 reduceOnly: %+v", bp)
			}
			bp := buildOrderParams(Leg{InstID: "BTC-USDT-SWAP", Side: "buy"}, "phx", "long_short_mode")
			if bp.ReduceOnly || bp.PosSide != "long" {
				t.Fatalf("SWAP 重开腿应无 reduceOnly+开仓 posSide: %+v", bp)
			}
		}
	}
	var state string
	if err := r.pool.QueryRow(ctx, "SELECT state FROM fin_execution WHERE id=$1", eid).Scan(&state); err != nil || state != "rolled_back" {
		t.Fatalf("state=%s err=%v（应 rolled_back——重开回滚成）", state, err)
	}
}

// T-0046 ⑤锚：非 51 族 envelope 拒→Resolve 定态 absent——拒码兜底不丢
// （venue_err_code=50013 留痕，禁留空吞真拒因）。
func TestVenueErrNon51Fallback(t *testing.T) {
	fv := &fakeVenue{apiReject: &okxexec.APIError{Code: "50013", Msg: "System busy"}, resolveTo: "absent"}
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err == nil {
		t.Fatal("应失败收场")
	}
	var code, msg string
	if err := r.pool.QueryRow(ctx, `SELECT venue_err_code,venue_err_msg FROM fin_order_leg
		WHERE execution_id=$1 AND leg_no=1`, eid).Scan(&code, &msg); err != nil {
		t.Fatal(err)
	}
	if code != "50013" || msg != "System busy" {
		t.Fatalf("拒码兜底丢失: code=%q msg=%q", code, msg)
	}
}

// TestLegCashSymmetric T-0083 腿级现金流对称锚：开=现货买腿 spot_buy 流出毛额
// （负）、平=现货卖腿 spot_sell 流入毛额（正）——两腿各记净额自然=真价差；
// 合约腿不触现径；position_ref 归仓（open 仓行后补挂、close 现读归仓）。
func TestLegCashSymmetric(t *testing.T) {
	fv := &fakeVenue{} // 全腿 filled px=1
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err != nil {
		t.Fatalf("开仓: %v", err)
	}
	var pid int64
	if err := r.pool.QueryRow(ctx,
		"SELECT id FROM fin_position WHERE execution_id=$1", eid).Scan(&pid); err != nil {
		t.Fatalf("仓行: %v", err)
	}
	// 开：spot_buy −0.01（sz0.01×px1 毛额）归仓
	var amt float64
	var pref *int64
	if err := r.pool.QueryRow(ctx, `SELECT amount,position_ref FROM fin_cashflow
		WHERE kind='spot_buy' AND source_table='fin_order_leg'`).Scan(&amt, &pref); err != nil {
		t.Fatalf("spot_buy 行: %v", err)
	}
	if amt != -0.01 || pref == nil || *pref != pid {
		t.Fatalf("开仓 spot_buy amount=%.6f position_ref=%v（应 −0.01 归仓#%d）", amt, pref, pid)
	}
	// 平：spot_sell +0.02（sz0.02×px1）归同仓
	did2 := mkToken(t, ctx, r, false, false, "allow")
	cin := Intent{Kind: "close", DecisionID: did2, InstID: "BTC-USDT-SWAP", NotionalUSD: 100,
		Legs: [2]Leg{
			{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: "1", Notional: 50, Close: true},
			{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: "0.02", Notional: 50, Close: true},
		}}
	if _, err := r.Execute(ctx, cin); err != nil {
		t.Fatalf("平仓: %v", err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT amount,position_ref FROM fin_cashflow
		WHERE kind='spot_sell' AND source_table='fin_order_leg'`).Scan(&amt, &pref); err != nil {
		t.Fatalf("spot_sell 行: %v", err)
	}
	if amt != 0.02 || pref == nil || *pref != pid {
		t.Fatalf("平仓 spot_sell amount=%.6f position_ref=%v（应 +0.02 归仓#%d）", amt, pref, pid)
	}
	// 合约腿零现金行——SWAP 腿不产 spot_* 流水
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow c
		JOIN fin_order_leg l ON c.source_ref=l.id AND c.source_table='fin_order_leg'
		WHERE l.inst_id LIKE '%-SWAP'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("合约腿现金行=%d（应 0）", n)
	}
}

// T-0083 返修①②锚：close 径禁建 fin_position 新行（空壳病灶根除）+linkLegCash
// 只挂 open 径——close 卖腿现金行归「开仓仓现读」原行，不被 execution_id 回溯重绑。
func TestCloseNoShellRowBindsOriginal(t *testing.T) {
	fv := &fakeVenue{}
	ctx, r := setup(t, fv)
	did := mkToken(t, ctx, r, false, false, "allow")
	eid, err := r.Execute(ctx, mkIntent(did))
	if err != nil {
		t.Fatalf("开仓: %v", err)
	}
	var pid int64
	if err := r.pool.QueryRow(ctx, "SELECT id FROM fin_position WHERE execution_id=$1", eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	did2 := mkToken(t, ctx, r, false, false, "allow")
	cin := Intent{Kind: "close", DecisionID: did2, InstID: "BTC-USDT-SWAP", NotionalUSD: 100,
		Legs: [2]Leg{
			{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: "1", Notional: 50, Close: true},
			{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: "0.02", Notional: 50, Close: true},
		}}
	if _, err := r.Execute(ctx, cin); err != nil {
		t.Fatalf("平仓: %v", err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM fin_position").Scan(&n); err != nil || n != 1 {
		t.Fatalf("close 径建仓行=%d（应 0——禁空壳）", n)
	}
	var amt float64
	var pref *int64
	if err := r.pool.QueryRow(ctx, `SELECT amount,position_ref FROM fin_cashflow
		WHERE kind='spot_sell' AND source_table='fin_order_leg'`).Scan(&amt, &pref); err != nil {
		t.Fatal(err)
	}
	if amt != 0.02 || pref == nil || *pref != pid {
		t.Fatalf("spot_sell 归仓错: amount=%.4f pref=%v want %d", amt, pref, pid)
	}
}

// T-0083 返修③④锚：0040 数据修复——壳绑现金行重绑「该 inst 最近已开仓」
// 原开仓行；零引用壳行删除。幂等：重跑零变化。
func TestMigration0040ShellRepair(t *testing.T) {
	fv := &fakeVenue{}
	ctx, r := setup(t, fv)
	// 造病灶快照：原开仓 P1 + close exec 壳 P2 + 卖腿 + 壳绑现金行。
	var eidOpen, eidClose int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES(1,'BTC-USDT',100,'closed') RETURNING id`).Scan(&eidOpen); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES(1,'BTC-USDT',100,'closed') RETURNING id`).Scan(&eidClose); err != nil {
		t.Fatal(err)
	}
	var p1, p2 int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,opened_at)
		VALUES($1,'BTC-USDT',100,now()-interval '2 hours') RETURNING id`, eidOpen).Scan(&p1); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,opened_at)
		VALUES($1,'BTC-USDT',100,now()) RETURNING id`, eidClose).Scan(&p2); err != nil {
		t.Fatal(err)
	}
	var legID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state,px)
		VALUES($1,2,'cl-x','okx','sell','BTC-USDT','0.02','filled',100) RETURNING id`, eidClose).Scan(&legID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('spot_sell','okx','USDT',2,$1,$2,'fin_order_leg',now())`, p2, legID); err != nil {
		t.Fatal(err)
	}
	// 跑 0040 本体（Migrate 版本序已含——测试库走手排防版本先记账）。
	body, err := migrations.FS.ReadFile("0040_position_ref_repair.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, string(body)); err != nil {
		t.Fatal(err)
	}
	var pref *int64
	if err := r.pool.QueryRow(ctx, "SELECT position_ref FROM fin_cashflow WHERE source_ref=$1", legID).Scan(&pref); err != nil {
		t.Fatal(err)
	}
	if pref == nil || *pref != p1 {
		t.Fatalf("卖腿现金行应重绑原开仓 %d: got %v", p1, pref)
	}
	var n int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM fin_position WHERE inst_id='BTC-USDT'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("壳行未清: inst 仓行=%d（应 1）", n)
	}
	// 幂等：重跑零变化。
	if _, err := r.pool.Exec(ctx, string(body)); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, "SELECT position_ref FROM fin_cashflow WHERE source_ref=$1", legID).Scan(&pref); err != nil || pref == nil || *pref != p1 {
		t.Fatalf("重跑应零变化: %v %v", pref, err)
	}
}

// T-0107 锚：腿行量纲消歧落档——sz_unit 按 TgtQuote/SWAP 分派；qty_base=
// venue 实填量（accFillSz）唯一真源（请求量禁当成交记；SWAP=contract 无币量）。
func TestLegUnitRecorded(t *testing.T) {
	ctx, r := setup(t, &fakeVenue{})
	did := mkToken(t, ctx, r, false, false, "allow")
	in := mkIntent(did)
	in.Legs[0].TgtQuote = true // T-0064 生产径：spot 市价买 quote_ccy 委托
	if _, err := r.Execute(ctx, in); err != nil {
		t.Fatal(err)
	}
	var execID int64
	if err := r.pool.QueryRow(ctx, `SELECT max(id) FROM fin_execution`).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	var unit1 string
	var qty1 *float64
	if err := r.pool.QueryRow(ctx, `SELECT sz_unit,qty_base FROM fin_order_leg
		WHERE execution_id=$1 AND leg_no=1`, execID).Scan(&unit1, &qty1); err != nil {
		t.Fatal(err)
	}
	if unit1 != "quote" || qty1 == nil || *qty1 != 100 {
		t.Fatalf("spot quote 腿应 quote+实填 100: %s %v", unit1, qty1)
	}
	var unit2 string
	var qty2 *float64
	if err := r.pool.QueryRow(ctx, `SELECT sz_unit,qty_base FROM fin_order_leg
		WHERE execution_id=$1 AND leg_no=2`, execID).Scan(&unit2, &qty2); err != nil {
		t.Fatal(err)
	}
	if unit2 != "contract" || qty2 != nil {
		t.Fatalf("SWAP 腿应 contract 无币量: %s %v", unit2, qty2)
	}
}
