package main

import (
	"context"
	"strconv"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// ops_esc_pending 惧值视图：scope 专载>class 基线>默认 50 三层解析;
// 单引销+范围引销皆除名;未决按 valence 序可读。
func TestEscPendingValence(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE event IN ('升件','处置','恢复')`); err != nil {
		t.Fatal(err)
	}
	mk := func(class, scope, note string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,esc_class,esc_scope)
			VALUES('探测席','升件',$1,$2,$3) RETURNING id`, note, class, scope).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}

	defN := mk("无名类", "x", "默认惧值件")            // 表内无档→50
	clsN := mk("对账", "x", "class 基线件")         // class 档→85
	scpN := mk("唤知", "muse", "scope 专载档")      // scope>class→15
	lowN := mk("唤知", "其他scope", "class 基线回落件") // 无 scope 档→唤知 class=50
	_ = lowN

	var w1, w2, w3 int
	if err := pool.QueryRow(ctx, `SELECT valence FROM ops_esc_pending WHERE id=$1`, defN).Scan(&w1); err != nil {
		t.Fatalf("默认档未出: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT valence FROM ops_esc_pending WHERE id=$1`, clsN).Scan(&w2); err != nil {
		t.Fatalf("class 档未出: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT valence FROM ops_esc_pending WHERE id=$1`, scpN).Scan(&w3); err != nil {
		t.Fatalf("scope 档未出: %v", err)
	}
	if w1 != 50 || w2 != 85 || w3 != 15 {
		t.Fatalf("惧值解析错: 默认=%d(望50) class=%d(望85) scope=%d(望15)", w1, w2, w3)
	}

	// 排序:对账(85)应压默认(50)与 muse(15)
	var top string
	if err := pool.QueryRow(ctx, `SELECT esc_class||'/'||esc_scope FROM ops_esc_pending
		ORDER BY valence DESC, id LIMIT 1`).Scan(&top); err != nil {
		t.Fatal(err)
	}
	if top != "对账/x" {
		t.Fatalf("惧值序首件错: %s（望 对账/x）", top)
	}

	// 单引销→除名
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`, "升件#"+strconv.FormatInt(clsN, 10)+" 销——账差销案"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_esc_pending WHERE id=$1`, clsN).Scan(&n); err != nil || n != 0 {
		t.Fatalf("单引销件未除名: n=%d err=%v", n, err)
	}

	// 范围引销→除名（defN 落入区间；defN+1=clsN 已单销同批不碍）
	lo, hi := defN, defN+1
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		"升件#"+strconv.FormatInt(lo, 10)+"-"+strconv.FormatInt(hi, 10)+" 批销——同批件"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_esc_pending WHERE id=$1`, defN).Scan(&n); err != nil || n != 0 {
		t.Fatalf("范围引销件未除名: n=%d err=%v", n, err)
	}

	// 未销件仍在
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_esc_pending WHERE id IN ($1,$2)`, scpN, lowN).Scan(&n); err != nil || n != 2 {
		t.Fatalf("未决件误除名: n=%d err=%v", n, err)
	}
}
