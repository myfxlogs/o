// calibDriftEscCheck 校准失真探:族实测后代≥10 且 |pred−real|>5pp→
// 升件;失真族销号(实测修平)复绿;校准族/小样本静默。
package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestCalibDriftEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM ops_event WHERE esc_scope='calib_drift'`,
		`DELETE FROM fin_paper_fill`, `DELETE FROM fin_strategy`,
		`DELETE FROM fin_proposal`, `DELETE FROM fin_opp_eval`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	fireN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_scope='calib_drift' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	seed := func(tpl string, n int, pred, netPer float64) {
		for i := 0; i < n; i++ {
			inst := fmt.Sprintf("INST-%s-%d", tpl, i)
			var eid, pid, sid int64
			if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval
				(inst_id,template,direction,net_apr,verdict,legs,evidence)
				VALUES($3,$1,'basis_quarter',$2,'proposed','[]','{}') RETURNING id`,
				tpl, pred, inst).Scan(&eid); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal
				(eval_id,inst_id,direction,net_apr,state)
				VALUES($1,$3,'basis_quarter',$2,'fin_proposed') RETURNING id`,
				eid, pred, inst).Scan(&pid); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy
				(inst_id,template_ref,proposal_ref,state,notional_usd)
				VALUES($3,$1,$2,'retired',100) RETURNING id`,
				tpl, pid, inst).Scan(&sid); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO fin_paper_fill
				(strategy_id,kind,inst_id,amount_usd) VALUES($1,'funding',$3,$2)`,
				sid, netPer, inst); err != nil {
				t.Fatal(err)
			}
		}
	}

	svc.calibDriftEscCheck(ctx, deadSince) // 零族静默
	if n := fireN(); n != 0 {
		t.Fatalf("零族误唤: n=%d", n)
	}

	seed("calm-fam", 10, 0.10, 0.0274) // 校准族:pred0.10≈real0.10
	svc.calibDriftEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("校准族误唤: n=%d", n)
	}

	// 时代守卫:eval 开户后改写(e.ts>pr.created_at)=事后重估非入场
	// 信念——失真幅相同但全簇被剔,探沉默(工单#15866 funding-hedge
	// 幻影失真同构:负簇 eval.ts 均开户后18d)
	seed("ghost-fam", 10, 0.10, -10)
	if _, err := pool.Exec(ctx, `UPDATE fin_opp_eval e SET ts=now()+interval '2 days'
		WHERE e.id IN (SELECT p.eval_id FROM fin_proposal p
			JOIN fin_strategy s ON s.proposal_ref=p.id
			WHERE s.template_ref='ghost-fam')`); err != nil {
		t.Fatal(err)
	}
	svc.calibDriftEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("改写簇误唤(时代守卫失效): n=%d", n)
	}

	seed("wild-fam", 10, 0.10, -10) // 失真族:pred+10% vs real -3650%
	svc.calibDriftEscCheck(ctx, deadSince)
	if n := fireN(); n != 1 {
		t.Fatalf("失真族未唤: n=%d", n)
	}

	// 实测修平(测试库直接改种值)→复绿
	if _, err := pool.Exec(ctx, `UPDATE fin_paper_fill f SET amount_usd=0.0274
		WHERE f.strategy_id IN (SELECT id FROM fin_strategy WHERE template_ref='wild-fam')`); err != nil {
		t.Fatal(err)
	}
	svc.calibDriftEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%calib_drift%'`).Scan(&recN); err != nil || recN != 1 {
		t.Fatalf("修平未复绿: recN=%d err=%v", recN, err)
	}
}
