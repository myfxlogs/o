package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// learnGapEscCheck 求知滞探:OPEN 求知票龄>2h 零触→升件;
// 新票(<2h)/活在认领/已处置皆静默;销票处置=复绿。
func TestLearnGapEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='learn_gap' OR event='工单' OR event='认领'`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	fireN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_scope='learn_gap' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	svc.learnGapEscCheck(ctx, deadSince) // 零票静默
	if n := fireN(); n != 0 {
		t.Fatalf("零票误唤: n=%d", n)
	}

	// 新票(<2h)不唤
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','工单','求知:okx earn 活期产品面——期限=24h,源=自问#1')`); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("新票误唤: n=%d", n)
	}

	// 陈票(3h)唤
	var tid int64
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,ts)
		VALUES('处置席','工单','求知:mtapi 还有哪些已授信未探读面——期限=48h,源=自问#2',
			now()-interval '3 hours') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	if n := fireN(); n != 1 {
		t.Fatalf("陈票未唤: n=%d", n)
	}

	// 处置销票→复绿
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d digest 交付毕——canon 条目落位", tid)); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%/learn_gap%'`).Scan(&recN); err != nil || recN != 1 {
		t.Fatalf("销票未复绿: recN=%d err=%v", recN, err)
	}
}
