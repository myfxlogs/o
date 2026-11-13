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
