package main

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 升件环：死→恰一升件（24h 同窗去重）；复绿→恢复行携升件#id 自闭。
func TestFeedEscLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='t_dead' OR note LIKE '%t_dead%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	svc.escalate(ctx, "断流", "t_dead", "面断流 t_dead：零行——超阈。t_fix")
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='断流' AND esc_scope='t_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件 n=%d err=%v", n, err)
	}
	svc.escalate(ctx, "断流", "t_dead", "重报") // 同窗重报不暴
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='t_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("去重 n=%d err=%v", n, err)
	}
	svc.recoverNote(ctx, "断流", "t_dead", time.Now().Add(-10*time.Minute))
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE seat='探测席' AND event='恢复' AND note LIKE '%t_dead%'`).Scan(&note); err != nil {
		t.Fatalf("恢复行缺: %v", err)
	}
	var id int64
	_ = pool.QueryRow(ctx, `SELECT id FROM ops_event WHERE esc_scope='t_dead'`).Scan(&id)
	if want := "升件#" + strconv.FormatInt(id, 10); !strings.Contains(note, want) {
		t.Fatalf("恢复行未携升件引: %s（望含 %s）", note, want)
	}
}

// escEdge 沿触发：非死→死恰一报,持续死不重报,复绿自闭。
func TestEscEdge(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE esc_scope='t_edge'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	ds := map[string]time.Time{}
	fire := func(d bool) {
		svc.escEdge(ctx, ds, "资金闸", "t_edge", d, func() string { return "t_edge note" })
	}
	fire(false)
	fire(true)
	fire(true) // 持续死不重报
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_scope='t_edge'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("沿触发 n=%d err=%v", n, err)
	}
	fire(false) // 复绿自闭
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%t_edge%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("自闭 n=%d err=%v", n, err)
	}
}
