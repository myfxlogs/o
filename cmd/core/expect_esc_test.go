package main

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// expectProdEscCheck：生产面零实产→expect_<面>升件；产出落账→恢复自闭。
// 量表 beat 为种子面（self_state 6h拍 9h 窗——插鲜行=绿,零行=断供）。
func TestExpectProdEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='expect_self_state' OR (event='量表' AND ticket='self_state')`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}

	svc.expectProdEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='预期产出' AND esc_scope='expect_self_state'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("断供升件 n=%d err=%v", n, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('探测席','量表','self_state','t 自态帧')`); err != nil {
		t.Fatalf("量表行: %v", err)
	}
	svc.expectProdEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%expect_self_state%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("恢复 n=%d", n)
	}
}

// 产活在窗=不升件（健康面无假火）。
func TestExpectProdHealthy(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE esc_scope LIKE 'expect_%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	for _, d := range expectDecls {
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('探测席','量表',$1,'t 帧')`, d.name); err != nil {
			t.Fatalf("播产 %s: %v", d.name, err)
		}
	}
	svc := coreService{pool: pool}
	svc.expectProdEscCheck(ctx, map[string]time.Time{})
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='预期产出'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("健康面假火 n=%d err=%v", n, err)
	}
}
