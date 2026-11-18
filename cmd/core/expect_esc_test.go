package main

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// expectProdEscCheck（降级契约#22578①）：生产面零实产=死态日计一火不落
// 升件;产出落账=复绿清态零恢复行。量表 beat 为种子面（self_state 6h拍
// 9h 窗——插鲜行=绿,零行=断供）。
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
	var n int
	resetSelfMeter()

	svc.expectProdEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='预期产出' AND esc_scope='expect_self_state'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("降级件竟升件 n=%d err=%v", n, err)
	}
	if selfEscMeter.counts["预期产出/expect_self_state"] != 1 {
		t.Fatalf("断供未入日计: %v", selfEscMeter.counts)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('探测席','量表','self_state','t 自态帧')`); err != nil {
		t.Fatalf("量表行: %v", err)
	}
	svc.expectProdEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%expect_self_state%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("降级件竟写恢复 n=%d err=%v", n, err)
	}
}

// TestExpectProdDeployGrace 部署冷启宽免（工单#13830——升件#13763
// self_aware 死窗1m39s 复绿实证）：部署龄<10min 零窗行不计;部署陈化
// 后同零行照计（宽免不掩真断供——降级契约计数语义同）。
func TestExpectProdDeployGrace(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='expect_self_state' OR (event='量表' AND ticket='self_state')
		OR (seat='DEPLOY' AND event='部署')`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	var n int
	resetSelfMeter()
	// 部署行 1min 前→宽免域：零产不计
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,ts)
		VALUES('DEPLOY','部署','','deploy aabbccddee rc=0 image=o-core:aabbccddee',
		now()-interval '1 minutes')`); err != nil {
		t.Fatalf("部署行: %v", err)
	}
	svc.expectProdEscCheck(ctx, map[string]time.Time{})
	if selfEscMeter.counts["预期产出/expect_self_state"] != 0 {
		t.Fatalf("宽免窗内误计: %v", selfEscMeter.counts)
	}
	// 更新部署行（11min 前成为最新）→宽免撤,零产照计
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,ts)
		VALUES('DEPLOY','部署','','deploy 1122334455 rc=0 image=o-core:1122334455',
		now()-interval '11 minutes')`); err != nil {
		t.Fatalf("部署行②: %v", err)
	}
	svc.expectProdEscCheck(ctx, map[string]time.Time{})
	if selfEscMeter.counts["预期产出/expect_self_state"] != 1 {
		t.Fatalf("宽免撤后真断供未计: %v", selfEscMeter.counts)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='预期产出'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("降级件竟升件 n=%d err=%v", n, err)
	}
}

// 产活在窗=不计（健康面无假火）。
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
	resetSelfMeter()
	svc.expectProdEscCheck(ctx, map[string]time.Time{})
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='预期产出'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("健康面假火 n=%d err=%v", n, err)
	}
	for k, v := range selfEscMeter.counts {
		if v != 0 {
			t.Fatalf("健康面误计 %s=%d", k, v)
		}
	}
}
