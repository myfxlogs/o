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

// hotDomainEscCheck 热域探：同 scope 30d 内被 ≥5 张 stuck_esc 行点名→升件；
// 域下阈=复绿撤态补发；stuck_esc 外行不计入。
func TestHotDomainEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE esc_scope IN ('stuck_esc','hotdom_pendbr','hotdom_rare')`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	seed := func(scope string, n int) { // 种 n 张点名该域的 stuck_esc 行
		for i := 0; i < n; i++ {
			note := fmt.Sprintf("v=%s#%d 唤知/stuck_esc: 销案后复燃", scope, 1000+i)
			if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,esc_class,esc_scope,note)
				VALUES('探测席','升件','唤知','stuck_esc',$1)`, note); err != nil {
				t.Fatal(err)
			}
		}
	}
	svc.hotDomainEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope LIKE 'hotdom_%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("零数据误火: n=%d err=%v", n, err)
	}
	seed("pendbr", hotDomainMinRefires) // 达阈
	seed("rare", hotDomainMinRefires-1) // 下阈
	svc.hotDomainEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='hotdom_pendbr'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("热域升件缺: n=%d err=%v", n, err)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE esc_scope='hotdom_pendbr'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "pendbr") || !strings.Contains(note, "复燃") {
		t.Fatalf("note 缺域/数载: %s", note)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='hotdom_rare'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("下阈误火: n=%d err=%v", n, err)
	}
	// 同纹重拍不暴（fp=域 30d 记忆期）
	svc.hotDomainEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='hotdom_pendbr'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("同纹暴报: n=%d err=%v", n, err)
	}
	// 域下阈→复绿撤态（stuck_esc 行老化出窗模拟——删行即下阈）
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE esc_scope='stuck_esc'`); err != nil {
		t.Fatal(err)
	}
	svc.hotDomainEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%hotdom_pendbr%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("复绿行缺: n=%d err=%v", n, err)
	}
}

// dreamDueEscCheck 梦探:判毕素材在案+回放>7d∨零史→升件;有梦复绿;
// 素材尽不唤。
func TestDreamDueEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM ops_event WHERE esc_scope='dream_due' OR ticket='dream_replay'`,
		`DELETE FROM fin_econ_lesson`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	fireN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_scope='dream_due' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	svc.dreamDueEscCheck(ctx, deadSince) // 零素材不唤
	if n := fireN(); n != 0 {
		t.Fatalf("零素材误唤: n=%d", n)
	}

	// 判毕素材入案→唤
	if _, err := pool.Exec(ctx, `INSERT INTO fin_econ_lesson(kind,entity_ref,status,verdict_ref)
		VALUES('falsified_verdict','goal:test','verdicted','弃——验毕')`); err != nil {
		t.Fatal(err)
	}
	svc.dreamDueEscCheck(ctx, deadSince)
	if n := fireN(); n != 1 {
		t.Fatalf("到时未唤: n=%d", n)
	}

	// 产回放行→复绿
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('唤知席','回放','dream_replay','反事实#1-test: 实态…')`); err != nil {
		t.Fatal(err)
	}
	svc.dreamDueEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%dream_due%'`).Scan(&recN); err != nil || recN != 1 {
		t.Fatalf("有梦未复绿: recN=%d err=%v", recN, err)
	}
}
