// retire_test.go —— 提案队列卫生闸桩锚（T-0161）：eval 翻拒→armed/proposed
// 提案联动退位+溯源行（幂等）；fin_proposed 过窗陈旧回收（幂等）；armed
// 超窗退位落因；scan 序免误开户（eval 已翻拒的提案不再进 paper）。
package lifecycle

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// newRig 测试台：池+全迁移+lifecycle 判据全键（criteria fail-closed 需全量在位）。
func newRig(t *testing.T) *Runner {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for k, v := range map[string]string{
		"invest.paper.notional_usd": "100", "invest.lifecycle.paper_min_hours": "0",
		"invest.lifecycle.paper_min_net_usd": "0.001", "invest.lifecycle.paper_max_dd_usd": "5",
		"invest.lifecycle.live_switch": "false"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return NewRunner(pool, cfg)
}

// fixtureEval+proposal 最小夹具：eval verdict + 提案 state（返 eval_id/proposal id）。
func fixture(t *testing.T, r *Runner, inst, verdict, propState string) (int64, int64) {
	t.Helper()
	ctx := context.Background()
	var eid, pid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES($1,'funding-hedge','short_perp_long_spot',$2,'{}','{}') RETURNING id`, inst, verdict).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr,state)
		VALUES($1,$2,'short_perp_long_spot',0.5,$3) RETURNING id`, eid, inst, propState).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return eid, pid
}

func propState(t *testing.T, r *Runner, pid int64) (string, *string, *time.Time) {
	t.Helper()
	ctx := context.Background()
	var st string
	var reason *string
	var retiredAt *time.Time
	if err := r.pool.QueryRow(ctx, `SELECT state,retire_reason,retired_at FROM fin_proposal WHERE id=$1`, pid).
		Scan(&st, &reason, &retiredAt); err != nil {
		t.Fatal(err)
	}
	return st, reason, retiredAt
}

// 锚①：eval 翻拒→同 eval_id 的 fin_proposed+fin_armed 提案当拍退位+溯源行；
// 幂等（二扫零变化）；他键提案（eval 仍 proposed）零误伤。
func TestEvalRejectedRetiresProposals(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	_, pProp := fixture(t, r, "RJ-A-USDT-SWAP", "proposed", "fin_proposed")
	_, pArmed := fixture(t, r, "RJ-B-USDT-SWAP", "proposed", "fin_armed")
	_, pCtrl := fixture(t, r, "RJ-C-USDT-SWAP", "proposed", "fin_proposed")

	if _, err := r.pool.Exec(ctx, `UPDATE fin_opp_eval SET verdict='rejected'
		WHERE inst_id IN ('RJ-A-USDT-SWAP','RJ-B-USDT-SWAP')`); err != nil {
		t.Fatal(err)
	}
	if err := r.retireEvalRejected(ctx); err != nil {
		t.Fatal(err)
	}
	for _, pid := range []int64{pProp, pArmed} {
		st, reason, retiredAt := propState(t, r, pid)
		if st != "fin_retired" || reason == nil || *reason != "eval_rejected" || retiredAt == nil {
			t.Fatalf("提案 #%d 应 retired+eval_rejected+时点 got %s/%v/%v", pid, st, reason, retiredAt)
		}
	}
	if st, _, _ := propState(t, r, pCtrl); st != "fin_proposed" {
		t.Fatalf("对照提案（eval 仍 proposed）不应动 got %s", st)
	}
	// 幂等：二扫零变化（reason/时点不被覆写漂移）。
	if err := r.retireEvalRejected(ctx); err != nil {
		t.Fatal(err)
	}
	st, reason, _ := propState(t, r, pProp)
	if st != "fin_retired" || reason == nil || *reason != "eval_rejected" {
		t.Fatalf("幂等二扫后应原样 got %s/%v", st, reason)
	}
}

// 锚②：fin_proposed 过窗（缺省 24h）陈旧回收 stale_expired；新鲜行不动；幂等。
func TestStaleProposedRecycled(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	_, pFresh := fixture(t, r, "ST-F-USDT-SWAP", "proposed", "fin_proposed")
	_, pStale := fixture(t, r, "ST-S-USDT-SWAP", "proposed", "fin_proposed")
	if _, err := r.pool.Exec(ctx, `UPDATE fin_proposal SET created_at=now()-interval '25 hours'
		WHERE id=$1`, pStale); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleProposed(ctx); err != nil {
		t.Fatal(err)
	}
	st, reason, retiredAt := propState(t, r, pStale)
	if st != "fin_retired" || reason == nil || *reason != "stale_expired" || retiredAt == nil {
		t.Fatalf("陈旧提案应 retired+stale_expired+时点 got %s/%v/%v", st, reason, retiredAt)
	}
	if st, _, _ := propState(t, r, pFresh); st != "fin_proposed" {
		t.Fatalf("新鲜提案不应动 got %s", st)
	}
	if err := r.retireStaleProposed(ctx); err != nil {
		t.Fatal(err)
	}
	if st, reason, _ := propState(t, r, pStale); st != "fin_retired" || reason == nil || *reason != "stale_expired" {
		t.Fatalf("幂等二扫后应原样 got %s/%v", st, reason)
	}
}

// 锚③：armed 超窗退役落 armed_ttl_expired 溯源（T-0161 施工项③同裁补因）。
func TestArmedTtlTraceReason(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	_, pFresh := fixture(t, r, "AT-F-USDT-SWAP", "proposed", "fin_armed")
	_, pOld := fixture(t, r, "AT-O-USDT-SWAP", "proposed", "fin_armed")
	if _, err := r.pool.Exec(ctx, `UPDATE fin_proposal SET created_at=now()-interval '25 hours'
		WHERE id=$1`, pOld); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleArmed(ctx); err != nil {
		t.Fatal(err)
	}
	st, reason, _ := propState(t, r, pOld)
	if st != "fin_retired" || reason == nil || *reason != "armed_ttl_expired" {
		t.Fatalf("armed 超窗应 retired+armed_ttl_expired got %s/%v", st, reason)
	}
	if st, _, _ := propState(t, r, pFresh); st != "fin_armed" {
		t.Fatalf("新鲜 armed 不应动 got %s", st)
	}
}

// 锚④：scan 序免误开户——eval 已翻拒的 fin_proposed 提案先退位、不建策略行。
func TestScanSkipsEvalRejectedProposal(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	_, pid := fixture(t, r, "SK-A-USDT-SWAP", "rejected", "fin_proposed")
	if err := r.scan(ctx); err != nil {
		t.Fatalf("scan: %v", err)
	}
	st, reason, _ := propState(t, r, pid)
	if st != "fin_retired" || reason == nil || *reason != "eval_rejected" {
		t.Fatalf("scan 应联动退位 got %s/%v", st, reason)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy WHERE proposal_ref=$1`, pid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("eval 已翻拒提案不得开户 got %d 策略行", n)
	}
}

// stratFixture 策略行最小夹具（T-0258 armed_ttl 桩）——state/entered_at 直给。
func stratFixture(t *testing.T, r *Runner, inst, state, tmpl string, enteredAgo string) int64 {
	t.Helper()
	ctx := context.Background()
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction,entered_at)
		VALUES($1,$2,$3,100,'short_perp_long_spot',now()-$4::interval) RETURNING id`,
		inst, tmpl, state, enteredAgo).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

func stratState(t *testing.T, r *Runner, sid int64) string {
	t.Helper()
	var st string
	if err := r.pool.QueryRow(context.Background(),
		"SELECT state FROM fin_strategy WHERE id=$1", sid).Scan(&st); err != nil {
		t.Fatal(err)
	}
	return st
}

// T-0258 锚①+②：tag3 活动锚=执行侧证据——ttl 窗内有证据者不退役
// （execution 尝试行/eval_reject 闸拒行/未平仓各一桩），零证据者退役且
// 落 fin_strategy_transition 行（reason=armed_ttl_expired）——两向断言。
func TestArmedTtlActivityAnchor(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	sExec := stratFixture(t, r, "AA-ACT-USDT-SWAP", "live", "funding-hedge", "25 hours")
	sRej := stratFixture(t, r, "AA-REJ-USDT-SWAP", "live", "funding-hedge", "25 hours")
	sPos := stratFixture(t, r, "AA-POS-USDT-SWAP", "live", "funding-hedge", "25 hours")
	sDead := stratFixture(t, r, "AA-DED-USDT-SWAP", "live", "funding-hedge", "25 hours")
	sProbe := stratFixture(t, r, "AA-PRB-USDT-SWAP", "probe", "funding-hedge", "25 hours")
	// sExec：执行尝试证据（败行同计——执行侧活动不分成败）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state,created_at)
		VALUES($1,'AA-ACT-USDT-SWAP',100,'failed',now()-interval '1 hour')`, sExec); err != nil {
		t.Fatal(err)
	}
	// sRej：开仓闸拒留痕（全拒≠不活——拦在开仓前的策略拒卡即活动证据）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_eval_reject(inst_id,template,direction,reason,ts)
		VALUES('AA-REJ-USDT-SWAP','funding-hedge','short_perp_long_spot','lifetime_below_min',now()-interval '1 hour')`); err != nil {
		t.Fatal(err)
	}
	// sPos：未平仓行——无仓谓词不命中（有仓永不退役原语义保留）。
	var eid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state,created_at)
		VALUES($1,'AA-POS-USDT-SWAP',100,'open',now()-interval '20 hours') RETURNING id`, sPos).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		VALUES($1,$2,'AA-POS-USDT-SWAP',100,now()-interval '20 hours')`, eid, sPos); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleArmed(ctx); err != nil {
		t.Fatal(err)
	}
	for sid, want := range map[int64]string{
		sExec: "live", sRej: "live", sPos: "live", sDead: "retired", sProbe: "retired"} {
		if got := stratState(t, r, sid); got != want {
			t.Fatalf("策略 #%d 应 %s got %s", sid, want, got)
		}
	}
	// 锚②：transition 留痕 from/to/reason 全落——零行=裸 UPDATE 逃逸回潮。
	for sid, wantFrom := range map[int64]string{sDead: "live", sProbe: "probe"} {
		var from, to, reason string
		if err := r.pool.QueryRow(ctx, `SELECT from_state,to_state,reason FROM fin_strategy_transition
			WHERE strategy_id=$1 ORDER BY id DESC LIMIT 1`, sid).Scan(&from, &to, &reason); err != nil {
			t.Fatalf("策略 #%d 退役零 transition 行（裸 UPDATE 逃逸回潮）: %v", sid, err)
		}
		if from != wantFrom || to != "retired" || reason != "armed_ttl_expired" {
			t.Fatalf("策略 #%d transition 应 %s→retired/armed_ttl_expired got %s→%s/%s",
				sid, wantFrom, from, to, reason)
		}
	}
	// 他域模板不受 funding 闸拒证据误喂：dcd_call 同 inst+direction 零自家
	// 证据——funding-hedge 拒行不得续命。
	sDcd := stratFixture(t, r, "AA-DCD-USDT-SWAP", "live", "dcd_call", "25 hours")
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_eval_reject(inst_id,template,direction,reason,ts)
		VALUES('AA-DCD-USDT-SWAP','funding-hedge','short_perp_long_spot','lifetime_below_min',now()-interval '1 hour')`); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleArmed(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stratState(t, r, sDcd); got != "retired" {
		t.Fatalf("dcd_call 策略被他族拒行续命 got %s", got)
	}
}

// T-0258 锚③ 现犯回放：live funding-hedge 被开仓闸持续拒（拒因行 ts 逐轮
// 刷新——rejectEvalReason ON CONFLICT 刷 ts 同构）超 ttl 仍存活——对应
// 10-30 7 条 funding-hedge 误杀实例形（零仓零执行纯闸拒）。
func TestArmedTtlGateRejectedSurvives(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	sid := stratFixture(t, r, "BARD-USDT-SWAP", "live", "funding-hedge", "25 hours")
	for _, ago := range []string{"20 hours", "12 hours", "1 hours"} {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_eval_reject(inst_id,template,direction,reason,ts)
			VALUES('BARD-USDT-SWAP','funding-hedge','short_perp_long_spot','breakeven_beyond_hold',now()-$1::interval)
			ON CONFLICT (inst_id,template,direction,reason) DO UPDATE SET ts=EXCLUDED.ts`, ago); err != nil {
			t.Fatal(err)
		}
		if err := r.retireStaleArmed(ctx); err != nil {
			t.Fatal(err)
		}
		if got := stratState(t, r, sid); got != "live" {
			t.Fatalf("闸拒回放（拒因 ts=%s 前）后策略应仍 live got %s", ago, got)
		}
	}
}
