package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestOppSurfaceStallEsc/TestLiveDudEsc 随减法 B 批退役——两探判据
// 平移入 ops_state 视件（fin_0202,「不升件不唤醒」钦定）。
// TestTicketScopeSuppress 票抑——同 scope 开工单在案=探针静默:
// OPEN 票抑火;销票(非挂起处置)→抑撤重火;挂票期限内仍抑;
// 挂票过期→抑撤重火。「票管催」对「探针重火」的分流实证。
func TestTicketScopeSuppress(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_eng_lesson WHERE signature LIKE 't-sup%';
		DELETE FROM ops_event WHERE esc_scope='lesson_backlog' OR note LIKE '%t-sup%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_eng_lesson(signature,cluster_n,numbers,status,ts)
		SELECT 't-sup-'||g,0,'[]'::jsonb,'pending',now()-interval '2 days'
		FROM generate_series(1,5) g`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	svc := coreService{pool: pool}
	ds := map[string]time.Time{}
	cnt := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE event='升件' AND esc_scope='lesson_backlog'`).Scan(&n); err != nil {
			t.Fatalf("cnt: %v", err)
		}
		return n
	}
	closeEsc := func() {
		var id int64
		if err := pool.QueryRow(ctx, `SELECT max(id) FROM ops_event
			WHERE event='升件' AND esc_scope='lesson_backlog'`).Scan(&id); err != nil {
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
	svc.lessonBacklogEscCheck(ctx, ds)
	if n := cnt(); n != 1 {
		t.Fatalf("首火 n=%d", n)
	}
	closeEsc()
	t1 := mkTicket("工单/sup: t-sup 矿积压清理面 ——期限=72h,源=升件#1,scope=lesson_backlog", "1 hour")
	svc.lessonBacklogEscCheck(ctx, ds)
	if n := cnt(); n != 1 {
		t.Fatalf("OPEN 票抑火 n=%d（应 1——重火被票接管）", n)
	}

	// b: 销票(非挂起)→抑撤重火
	mkDisp(t1, "销——判词修毕(非挂起)")
	svc.lessonBacklogEscCheck(ctx, ds)
	if n := cnt(); n != 2 {
		t.Fatalf("销票抑撤 n=%d（应 2——重火复岗）", n)
	}
	closeEsc()

	// c: 挂票期限内仍抑
	t2 := mkTicket("工单/sup3: t-sup 挂起 ——期限=72h,源=升件#1,scope=lesson_backlog", "1 hour")
	mkDisp(t2, "挂起条件=等业主凭据")
	svc.lessonBacklogEscCheck(ctx, ds)
	if n := cnt(); n != 2 {
		t.Fatalf("挂票抑火 n=%d（应 2——挂起在期限内仍管催）", n)
	}

	// d: 挂票过期→抑撤重火
	if _, err := pool.Exec(ctx, `UPDATE ops_event SET ts=now()-interval '80 hours'
		WHERE id=$1`, t2); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	svc.lessonBacklogEscCheck(ctx, ds)
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
