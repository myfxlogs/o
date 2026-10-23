package execution

import (
	"context"
	"fmt"
	"testing"

	"phase/internal/platform/config"
	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

// fakeVenue 可控 venue（Place 行为按 clOrdID 路由；onPlace=调用序钩子——前置赎回序锚用）。
type fakeVenue struct {
	leg2Fail  bool
	rbFail    bool
	resolveTo string
	placed    []string
	onPlace   func()
}

func (f *fakeVenue) Place(ctx context.Context, leg Leg, clOrdID string) (OrderResult, error) {
	f.placed = append(f.placed, clOrdID+":"+leg.Side)
	if f.onPlace != nil {
		f.onPlace()
	}
	if f.leg2Fail && leg.Venue == "leg2" {
		return OrderResult{ClOrdID: clOrdID, State: "unknown"}, fmt.Errorf("transport down")
	}
	if f.rbFail && contains(clOrdID, "-rb-") {
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
	if f.rbFail && contains(clOrdID, "-rb-") {
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
