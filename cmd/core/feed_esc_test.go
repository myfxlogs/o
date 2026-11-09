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

// 鲜度死升件环：死→恰一升件（24h 同窗去重）；复绿→恢复行携升件#id 自闭。
func TestFeedEscLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='t_dead' OR esc_scope='t_live' OR note LIKE '%t_dead%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	dead := compBeat{name: "t_dead", dead: time.Minute,
		query: `SELECT now()-interval '1 hour'`, fix: "t_fix"}
	svc.feedEscalate(ctx, dead, nil)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='断流' AND esc_scope='t_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件 n=%d err=%v", n, err)
	}
	svc.feedEscalate(ctx, dead, nil) // 同窗重报不暴
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='t_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("去重 n=%d err=%v", n, err)
	}
	svc.feedRecover(ctx, dead, time.Now().Add(-10*time.Minute))
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
	// 复绿后再死→新升件不受旧同窗阻（新事件=新键同 scope 仍同窗——
	// 设计面：同窗内同面再死不重报，恢复行已供处置席证据）。
}
