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

// TestStuckEsc 卡壳探——「修」类处置后同 scope 复燃=升件;
// 销类处置复燃/无处置连续火不误发。
func TestStuckEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE esc_scope LIKE 't-stuck%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	mk := func(scope string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
			VALUES('探测席','升件','测试',$1,'probe') RETURNING id`, scope).Scan(&id); err != nil {
			t.Fatalf("esc: %v", err)
		}
		return id
	}
	// e1: 修类处置后复燃——该火
	e1 := mk("t-stuck-fix")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`, fmt.Sprintf("升件#%d 销——已修部署 o-core:abc123", e1)); err != nil {
		t.Fatalf("disp1: %v", err)
	}
	mk("t-stuck-fix")
	// e2: 销类处置后复燃——不该火(条件存续=正常复验)
	e2 := mk("t-stuck-dismiss")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`, fmt.Sprintf("升件#%d 销——误发,触发器假", e2)); err != nil {
		t.Fatalf("disp2: %v", err)
	}
	mk("t-stuck-dismiss")
	// e3: 愈类处置后复燃——该火(宣称已愈=假愈)
	e3 := mk("t-stuck-heal")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`, fmt.Sprintf("升件#%d 销——窗口内已愈", e3)); err != nil {
		t.Fatalf("disp3: %v", err)
	}
	mk("t-stuck-heal")
	// e4: 「要码修」待办语处置后复燃——不该火(待修非已修宣称;
	// 裸「修」误中曾滚出 13 代自指链 2026-11-10 夜)
	e4 := mk("t-stuck-todo")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`, fmt.Sprintf("升件#%d 销——探针修要码修在案", e4)); err != nil {
		t.Fatalf("disp4: %v", err)
	}
	mk("t-stuck-todo")
	// e5: 无处置连续火——不该火
	mk("t-stuck-open")
	mk("t-stuck-open")
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	svc.stuckEscCheck(ctx, deadSince)
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_scope='stuck_esc'`).Scan(&note); err != nil {
		t.Fatalf("卡壳升件未落: %v", err)
	}
	if !strings.Contains(note, "t-stuck-fix") || !strings.Contains(note, "t-stuck-heal") || strings.Contains(note, "t-stuck-dismiss") || strings.Contains(note, "t-stuck-todo") || strings.Contains(note, "t-stuck-open") {
		t.Fatalf("升件 note 谱错: %s", note)
	}
}

// TestCodeworkStallEsc 工单滞留探——双臂:①工单真行过期未销=升件
// (期限自载,默认24h);②旧形非探测席升件陈>24h未销同火。
// 已销/期内/探测席件不误发。
func TestCodeworkStallEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE note LIKE '%t-cw%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	ins := func(event, seat, note, age string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,ts)
			VALUES($1,$2,$3,now()-$4::interval) RETURNING id`, seat, event, note, age).Scan(&id); err != nil {
			t.Fatalf("ins %s: %v", note, err)
		}
		return id
	}
	// 工单臂 a: 期限=1h 陈 2h 未销——该火
	ins("工单", "处置席", "工单/risk: t-cw-tix-overdue ——期限=1h,源=升件#1", "2 hours")
	// 工单臂 b: 期限=72h 陈 48h——不该火(期内)
	ins("工单", "处置席", "工单/risk: t-cw-tix-open ——期限=72h,源=升件#1", "48 hours")
	// 工单臂 c: 期限=1h 陈 2h 已销——不该火
	c := ins("工单", "处置席", "工单/risk: t-cw-tix-closed ——期限=1h,源=升件#1", "2 hours")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`, fmt.Sprintf("工单#%d 修——已部署 o-core:abc", c)); err != nil {
		t.Fatalf("disp c: %v", err)
	}
	// 旧形臂 a: 处置席工单陈 48h 未销——该火
	ins("升件", "处置席", "复证自审计 t-cw-old-open", "48 hours")
	// 旧形臂 b: 处置席工单陈 48h 已销——不该火
	b := ins("升件", "处置席", "复证自审计 t-cw-old-closed", "48 hours")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`, fmt.Sprintf("升件#%d 修——已部署 o-core:abc", b)); err != nil {
		t.Fatalf("disp b: %v", err)
	}
	// 旧形臂 c: 探测席升件陈 48h——不该火(探针件非工单)
	ins("升件", "探测席", "商业产出/t-cw-probe", "48 hours")
	// 旧形臂 d: 处置席工单新 2h——不该火(未达龄)
	ins("升件", "处置席", "复证自审计 t-cw-fresh", "2 hours")
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	svc.codeworkStallEscCheck(ctx, deadSince)
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_scope='codework_stall'`).Scan(&note); err != nil {
		t.Fatalf("工单滞留升件未落: %v", err)
	}
	for _, want := range []string{"t-cw-tix-overdue", "t-cw-old-open"} {
		if !strings.Contains(note, want) {
			t.Fatalf("该火未火 %s: %s", want, note)
		}
	}
	for _, not := range []string{"t-cw-tix-open", "t-cw-tix-closed", "t-cw-old-closed", "t-cw-probe", "t-cw-fresh"} {
		if strings.Contains(note, not) {
			t.Fatalf("误发火 %s: %s", not, note)
		}
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
