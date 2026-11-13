package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// goalIdleEscCheck 目标票怠探:OPEN goal 票票龄>4h 且近 8h 无处置触→
// 升件;新票宽限/在场触/已销票皆不火;触出现=复绿撤态。
func TestGoalIdleEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE note LIKE '%goal-%' OR esc_scope='goal_idle'`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}

	seedGoal := func(slug string, hoursAgo float64) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,ts)
			VALUES('处置席','工单',$1,now()-$2::float*interval '1 hour') RETURNING id`,
			fmt.Sprintf("工单/goal-%s: 目标=X 期限=24h,scope=hunger_gap", slug), hoursAgo).
			Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	idleN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_scope='goal_idle' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	fresh := seedGoal("fresh", 2)  // 票龄<4h 宽限
	stale := seedGoal("stale", 10) // OPEN 10h 零触——该火
	touched := seedGoal("touch", 10)
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d 接续:分解中", touched)); err != nil {
		t.Fatal(err)
	}
	closed := seedGoal("done", 10)
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d 销——证伪", closed)); err != nil {
		t.Fatal(err)
	}
	_ = fresh

	svc.goalIdleEscCheck(ctx, deadSince)
	if n := idleN(); n != 1 {
		t.Fatalf("该火未火或误火: 升件数=%d(应=1,仅 stale 票)", n)
	}

	svc.goalIdleEscCheck(ctx, deadSince) // 同值未销不暴
	if n := idleN(); n != 1 {
		t.Fatalf("未销在案暴升: n=%d", n)
	}

	// 触出现→复绿撤态补发恢复行
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d 推进:账面分解完成", stale)); err != nil {
		t.Fatal(err)
	}
	svc.goalIdleEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%goal_idle%'`).Scan(&recN); err != nil || recN != 1 {
		t.Fatalf("触后未复绿: recN=%d err=%v", recN, err)
	}
}
