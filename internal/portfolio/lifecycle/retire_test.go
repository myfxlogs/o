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
