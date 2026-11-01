package review

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// setupApply 并集库造 Applier（与 setup 同链序——md 先 fin 后）。
func setupApply(t *testing.T) (context.Context, *Applier) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return ctx, NewApplier(pool)
}

// TestVerdictApplyFalsified 判决链不断锚桩测：falsified 判决→同键 armed 提案
// 退役+live 策略 degraded+transition 留痕+游标推进；重放幂等（二度零重复）。
func TestVerdictApplyFalsified(t *testing.T) {
	ctx, a := setupApply(t)
	var eid, prid, sid, execID, pid, rid int64
	_ = a.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('AP-USDT-SWAP','funding-hedge','short_perp_long_spot',1.0,'proposed','{}','{}') RETURNING id`).Scan(&eid)
	_ = a.pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr,state)
		VALUES($1,'AP-USDT-SWAP','short_perp_long_spot',1.0,'fin_armed') RETURNING id`, eid).Scan(&prid)
	_ = a.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,direction)
		VALUES('AP-USDT-SWAP','funding-hedge',$1,'live',100,'short_perp_long_spot') RETURNING id`, prid).Scan(&sid)
	_ = a.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES($1,'AP-USDT-SWAP',100,'open') RETURNING id`, sid).Scan(&execID)
	_ = a.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,closed_at,settled_amount,settled_ccy,settled_at)
		VALUES($1,$2,'AP-USDT-SWAP',100,now(),-0.5,'USDT',now()) RETURNING id`, execID, sid).Scan(&pid)
	_ = a.pool.QueryRow(ctx, `INSERT INTO fin_review(position_ref,verdict,pred_net,actual_net,deviation,attribution,detail)
		VALUES($1,'falsified',1.0,-0.5,1.5,'funding_flip','{}') RETURNING id`, pid).Scan(&rid)
	if err := a.applyDue(ctx); err != nil {
		t.Fatal(err)
	}
	var st string
	_ = a.pool.QueryRow(ctx, "SELECT state FROM fin_proposal WHERE id=$1", prid).Scan(&st)
	if st != "fin_retired" {
		t.Fatalf("armed 提案未退役 state=%s", st)
	}
	_ = a.pool.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE id=$1", sid).Scan(&st)
	if st != "degraded" {
		t.Fatalf("live 策略未降级 state=%s", st)
	}
	var n int
	_ = a.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_strategy_transition WHERE strategy_id=$1 AND to_state='degraded'", sid).Scan(&n)
	if n != 1 {
		t.Fatalf("transition 留痕=%d 行", n)
	}
	var upto int64
	_ = a.pool.QueryRow(ctx, "SELECT applied_upto FROM fin_verdict_cursor WHERE k=1").Scan(&upto)
	if upto < rid {
		t.Fatalf("游标未推进 upto=%d rid=%d", upto, rid)
	}
	// 幂等重放：二度消费零副作用（transition 不翻倍）。
	if err := a.applyDue(ctx); err != nil {
		t.Fatal(err)
	}
	_ = a.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_strategy_transition WHERE strategy_id=$1 AND to_state='degraded'", sid).Scan(&n)
	if n != 1 {
		t.Fatalf("重放后 transition=%d 行——幂等破", n)
	}
}

// TestVerdictApplySkipsNonFalsified 非 falsified 行=游标过顶跳过不应用。
func TestVerdictApplySkipsNonFalsified(t *testing.T) {
	ctx, a := setupApply(t)
	var pid, rid int64
	_ = a.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,closed_at,settled_amount,settled_ccy,settled_at)
		VALUES(0,0,'SK-USDT-SWAP',100,now(),1.0,'USDT',now()) RETURNING id`).Scan(&pid)
	_ = a.pool.QueryRow(ctx, `INSERT INTO fin_review(position_ref,verdict,pred_net,actual_net,deviation,attribution,detail)
		VALUES($1,'strong',1.0,1.0,0,'luck','{}') RETURNING id`, pid).Scan(&rid)
	if err := a.applyDue(ctx); err != nil {
		t.Fatal(err)
	}
	var upto int64
	_ = a.pool.QueryRow(ctx, "SELECT applied_upto FROM fin_verdict_cursor WHERE k=1").Scan(&upto)
	if upto < rid {
		t.Fatalf("非 falsified 行未过游标 upto=%d rid=%d", upto, rid)
	}
}
