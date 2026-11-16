package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestOppSurfaceStallEsc 机会面断供探——曾活族 48h 零评估=升件;
// 活族不误发;销后（族回活）复绿自闭。
func TestOppSurfaceStallEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_opp_eval WHERE inst_id LIKE 'T-STALL%';
		DELETE FROM md_eval_pulse WHERE template LIKE 't_%fam';
		DELETE FROM ops_event WHERE esc_scope='opp_eval_stall'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	for i := 0; i < 20; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence,ts)
			VALUES($1,'t_deadfam','dcd_put','rejected','[]','{}'::jsonb,now()-interval '3 days')`,
			fmt.Sprintf("T-STALL-d%d", i)); err != nil {
			t.Fatalf("eval%d: %v", i, err)
		}
	}
	for i := 0; i < 20; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence,ts)
			VALUES($1,'t_livefam','dcd_put','rejected','[]','{}'::jsonb,now()-interval '3 days')`,
			fmt.Sprintf("T-STALL-l%d", i)); err != nil {
			t.Fatalf("eval%d: %v", i, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('T-STALL-fresh','t_livefam','dcd_put','rejected','[]','{}'::jsonb)`); err != nil {
		t.Fatalf("fresh: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_eval_pulse(template,last_eval_ts,eval_n_24h)
		VALUES('t_deadfam',now()-interval '3 days',20),('t_livefam',now(),21)`); err != nil {
		t.Fatalf("pulse: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	svc.oppSurfaceStallEscCheck(ctx, deadSince)
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_class='数据缺' AND esc_scope='opp_eval_stall'`).Scan(&note); err != nil {
		t.Fatalf("断供升件未落: %v", err)
	}
	if note == "" || !strings.Contains(note, "t_deadfam") || strings.Contains(note, "t_livefam") {
		t.Fatalf("升件 note 族谱错: %s", note)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('T-STALL-revive','t_deadfam','dcd_put','rejected','[]','{}'::jsonb)`); err != nil {
		t.Fatalf("revive: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE md_eval_pulse SET last_eval_ts=now(),eval_n_24h=21
		WHERE template='t_deadfam'`); err != nil {
		t.Fatalf("revive pulse: %v", err)
	}
	svc.oppSurfaceStallEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%opp_eval_stall%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("复绿恢复行 n=%d err=%v", n, err)
	}
}

// TestLiveDudEsc live 哑弹探——live≥7d 零仓位=升件;
// 有仓 wired/新生 young 不误发;哑弹开仓后复绿自闭。
func TestLiveDudEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_position WHERE inst_id LIKE 'T-DUD%';
		DELETE FROM fin_execution WHERE inst_id LIKE 'T-DUD%';
		DELETE FROM fin_strategy WHERE inst_id LIKE 'T-DUD%';
		DELETE FROM ops_event WHERE esc_scope='live_dud'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	// 哑弹：live 8d 零仓位
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,entered_at)
		VALUES('T-DUD-dud','t_tpl','live',50,now()-interval '8 days')`); err != nil {
		t.Fatalf("dud: %v", err)
	}
	// wired：live 8d 但开过仓
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,entered_at)
		VALUES('T-DUD-wired','t_tpl','live',50,now()-interval '8 days')`); err != nil {
		t.Fatalf("wired: %v", err)
	}
	var wiredID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM fin_strategy WHERE inst_id='T-DUD-wired'`).Scan(&wiredID); err != nil {
		t.Fatalf("wiredID: %v", err)
	}
	var execID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('T-DUD-wired',50,'closed') RETURNING id`).Scan(&execID); err != nil {
		t.Fatalf("exec: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES($1,$2,'T-DUD-wired',50)`, execID, wiredID); err != nil {
		t.Fatalf("pos: %v", err)
	}
	// young：live 2d 零仓位——未到判龄
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,entered_at)
		VALUES('T-DUD-young','t_tpl','live',50,now()-interval '2 days')`); err != nil {
		t.Fatalf("young: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	svc.liveDudEscCheck(ctx, deadSince)
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_class='商业产出' AND esc_scope='live_dud'`).Scan(&note); err != nil {
		t.Fatalf("哑弹升件未落: %v", err)
	}
	if !strings.Contains(note, "T-DUD-dud") || strings.Contains(note, "T-DUD-wired") || strings.Contains(note, "T-DUD-young") {
		t.Fatalf("升件 note 谱错: %s", note)
	}
	// 哑弹开仓→复绿自闭
	var dudID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM fin_strategy WHERE inst_id='T-DUD-dud'`).Scan(&dudID); err != nil {
		t.Fatalf("dudID: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('T-DUD-dud',50,'open') RETURNING id`).Scan(&execID); err != nil {
		t.Fatalf("exec2: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES($1,$2,'T-DUD-dud',50)`, execID, dudID); err != nil {
		t.Fatalf("pos2: %v", err)
	}
	svc.liveDudEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%live_dud%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("开仓复绿 n=%d err=%v", n, err)
	}
}

// TestTicketScopeSuppress 票抑——同 scope 开工单在案=探针静默:
// OPEN 票抑火;销票(非挂起处置)→抑撤重火;挂票期限内仍抑;
// 挂票过期→抑撤重火。「票管催」对「探针重火」的分流实证。
func TestTicketScopeSuppress(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_strategy WHERE inst_id='T-SUP-dud';
		DELETE FROM ops_event WHERE esc_scope='live_dud' OR note LIKE '%t-sup%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,entered_at)
		VALUES('T-SUP-dud','t_tpl','live',50,now()-interval '8 days')`); err != nil {
		t.Fatalf("dud: %v", err)
	}
	svc := coreService{pool: pool}
	ds := map[string]time.Time{}
	cnt := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE event='升件' AND esc_scope='live_dud'`).Scan(&n); err != nil {
			t.Fatalf("cnt: %v", err)
		}
		return n
	}
	closeEsc := func() {
		var id int64
		if err := pool.QueryRow(ctx, `SELECT max(id) FROM ops_event
			WHERE event='升件' AND esc_scope='live_dud'`).Scan(&id); err != nil {
			t.Fatalf("esc id: %v", err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
			VALUES('处置席','处置',$1)`, fmt.Sprintf("升件#%d 销——t-sup 定性转票", id)); err != nil {
			t.Fatalf("closeEsc: %v", err)
		}
	}
	mkTicket := func(note, age string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,ts)
			VALUES('处置席','工单',$1,now()-$2::interval) RETURNING id`, note, age).Scan(&id); err != nil {
			t.Fatalf("ticket: %v", err)
		}
		return id
	}
	mkDisp := func(tid int64, note string) {
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
			VALUES('处置席','处置',$1)`, fmt.Sprintf("工单#%d %s", tid, note)); err != nil {
			t.Fatalf("disp: %v", err)
		}
	}

	// a: 升件→销→OPEN 票抑火(条件仍真,重火被票接管)
	svc.liveDudEscCheck(ctx, ds)
	if n := cnt(); n != 1 {
		t.Fatalf("首火 n=%d", n)
	}
	closeEsc()
	t1 := mkTicket("工单/dud: t-sup 哑弹退役面 ——期限=72h,源=升件#1,scope=live_dud", "1 hour")
	svc.liveDudEscCheck(ctx, ds)
	if n := cnt(); n != 1 {
		t.Fatalf("OPEN 票抑火 n=%d（应 1——重火被票接管）", n)
	}

	// b: 销票(非挂起)→抑撤重火
	mkDisp(t1, "销——判词修毕(非挂起)")
	svc.liveDudEscCheck(ctx, ds)
	if n := cnt(); n != 2 {
		t.Fatalf("销票抑撤 n=%d（应 2——重火复岗）", n)
	}
	closeEsc()

	// c: 挂票期限内仍抑
	t2 := mkTicket("工单/dud3: t-sup 挂起 ——期限=72h,源=升件#1,scope=live_dud", "1 hour")
	mkDisp(t2, "挂起条件=等业主凭据")
	svc.liveDudEscCheck(ctx, ds)
	if n := cnt(); n != 2 {
		t.Fatalf("挂票抑火 n=%d（应 2——挂起在期限内仍管催）", n)
	}

	// d: 挂票过期→抑撤重火
	if _, err := pool.Exec(ctx, `UPDATE ops_event SET ts=now()-interval '80 hours'
		WHERE id=$1`, t2); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	svc.liveDudEscCheck(ctx, ds)
	if n := cnt(); n != 3 {
		t.Fatalf("挂票过期重火 n=%d（应 3——抑撤自然复验）", n)
	}
}

// TestLessonBacklogEsc 矿积压探——pending≥5 且最陈>24h=升件;
// 不足阈/不够陈不误发。
func TestLessonBacklogEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_eng_lesson WHERE signature LIKE 't-supply%';
		DELETE FROM ops_event WHERE esc_scope='lesson_backlog'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	svc.lessonBacklogEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='lesson_backlog'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("空矿误发 n=%d", n)
	}
	for i := 0; i < 5; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_eng_lesson(signature,cluster_n,numbers,status,ts)
			VALUES($1,0,'[]'::jsonb,'pending',now()-interval '2 days')`,
			fmt.Sprintf("t-supply-%d", i)); err != nil {
			t.Fatalf("lesson%d: %v", i, err)
		}
	}
	svc.lessonBacklogEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='唤知' AND esc_scope='lesson_backlog'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("积压升件 n=%d err=%v", n, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_eng_lesson SET status='canon' WHERE signature LIKE 't-supply%'`); err != nil {
		t.Fatalf("销: %v", err)
	}
	svc.lessonBacklogEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%lesson_backlog%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("清矿复绿 n=%d err=%v", n, err)
	}
}
