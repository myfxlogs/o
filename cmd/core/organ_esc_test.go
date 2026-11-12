package main

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// organOrphanEscCheck：立项无票→organ_<名>升件；开工单载词=覆盖抑火；
// 完工态=非悬置域不探。
func seedOrganFact(t *testing.T, ctx context.Context, svc coreService, note string) {
	t.Helper()
	if _, err := svc.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('处置席','处置','',$1)`,
		note); err != nil {
		t.Fatalf("seed fact: %v", err)
	}
}

func organEscFired(t *testing.T, ctx context.Context, svc coreService, scope string) bool {
	t.Helper()
	var n int
	if err := svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='孤儿账' AND esc_scope=$1`, scope).
		Scan(&n); err != nil {
		t.Fatalf("fired: %v", err)
	}
	return n > 0
}

func TestOrganOrphanPinNoTicket(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	seedOrganFact(t, ctx, svc, "举一反三器官立项——等认领施工")
	svc.organOrphanEscCheck(ctx, map[string]time.Time{})
	if !organEscFired(t, ctx, svc, "organ_举一反三") {
		t.Fatal("立项无票未升件")
	}
	if !organEscFired(t, ctx, svc, "organ_做梦") {
		t.Fatal("无事实器官未按毛坯升件")
	}
}

func TestOrganOrphanCoveredByTicket(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	seedOrganFact(t, ctx, svc, "举一反三器官立项——等认领施工")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('业主','工单','期限=168h 举一反三器官施工')`); err != nil {
		t.Fatalf("ticket: %v", err)
	}
	svc.organOrphanEscCheck(ctx, map[string]time.Time{})
	if organEscFired(t, ctx, svc, "organ_举一反三") {
		t.Fatal("开工单覆盖仍升件")
	}
	if !organEscFired(t, ctx, svc, "organ_懂你") {
		t.Fatal("无票器官漏升件")
	}
}

func TestOrganOrphanDoneSilent(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	seedOrganFact(t, ctx, svc, "举一反三器官认领→施工→部署至产线——销案毕")
	svc.organOrphanEscCheck(ctx, map[string]time.Time{})
	if organEscFired(t, ctx, svc, "organ_举一反三") {
		t.Fatal("完工器官误升件")
	}
}

func TestOrganOrphanAssertAuthority(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	// 断言 done 盖散文立项——断言行是唯一态权威。
	seedOrganFact(t, ctx, svc, "举一反三器官立项——等认领施工")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席','工位','','工位:举一反三:done 据#1——竣工销案')`); err != nil {
		t.Fatalf("assert: %v", err)
	}
	svc.organOrphanEscCheck(ctx, map[string]time.Time{})
	if organEscFired(t, ctx, svc, "organ_举一反三") {
		t.Fatal("断言完工仍升件")
	}
	// 无断言论件=散文推导+自举补断言。
	var n int
	if err := svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='工位' AND note LIKE '工位:做梦:%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("自举断言未落 n=%d err=%v", n, err)
	}
}
