// ownermodel_beat_test.go —— 懂你模型:请托谱帧落账/reask_denied
// 重复问已否件升件/无否史 pending 不升。
package main

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func ownerModelSetup(t *testing.T) (context.Context, coreService) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM fin_condition_gap`,
		`DELETE FROM ops_event WHERE ticket='owner_model' OR esc_class='懂你'`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	return ctx, coreService{pool: pool}
}

func gapRow(t *testing.T, ctx context.Context, svc coreService, state string) {
	if _, err := svc.pool.Exec(ctx, `INSERT INTO fin_condition_gap(kind,venue,surface,state,resolved_at)
		VALUES('credential','mt5','acct',$1,
			CASE WHEN $1='pending' THEN NULL ELSE now() END)`, state); err != nil {
		t.Fatalf("gap %s: %v", state, err)
	}
}

func TestOwnerModelFrame(t *testing.T) {
	ctx, svc := ownerModelSetup(t)
	gapRow(t, ctx, svc, "fulfilled")
	gapRow(t, ctx, svc, "rejected")
	svc.ownerModelBeat(ctx, map[string]time.Time{})
	var note string
	if err := svc.pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='量表' AND ticket='owner_model' ORDER BY id DESC LIMIT 1`).
		Scan(&note); err != nil {
		t.Fatalf("懂你帧未落: %v", err)
	}
	t.Logf("frame: %s", note)
}

func TestOwnerModelReaskDenied(t *testing.T) {
	ctx, svc := ownerModelSetup(t)
	gapRow(t, ctx, svc, "rejected")
	gapRow(t, ctx, svc, "dismissed")
	gapRow(t, ctx, svc, "pending") // 同键 pending+2否史=重复问已否
	svc.ownerModelBeat(ctx, map[string]time.Time{})
	var n int
	svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='升件'
		AND esc_class='懂你' AND esc_scope='reask_denied'`).Scan(&n)
	if n != 1 {
		t.Fatalf("reask_denied 未升件: %d", n)
	}
}

func TestOwnerModelNoDenialNoEsc(t *testing.T) {
	ctx, svc := ownerModelSetup(t)
	gapRow(t, ctx, svc, "pending") // 零否史 pending 不升
	svc.ownerModelBeat(ctx, map[string]time.Time{})
	var n int
	svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='升件'
		AND esc_class='懂你'`).Scan(&n)
	if n != 0 {
		t.Fatalf("零否史竟升件: %d", n)
	}
}
