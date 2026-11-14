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

// goalIdleEscCheck 目标票怠探:OPEN goal 票票龄>4h 且近 8h 无处置触→
// 升件;新票宽限/在场触/已销票皆不火;触出现=复绿撤态。
func TestGoalIdleEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE note LIKE '%goal-%' OR esc_scope='goal_idle'`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}

	seedGoal := func(slug string, hoursAgo float64) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,ts)
			VALUES('处置席','工单',$1,now()-$2::float*interval '1 hour') RETURNING id`,
			fmt.Sprintf("工单/goal-%s: 目标=X 期限=24h,scope=hunger_gap", slug), hoursAgo).
			Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	idleN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_scope='goal_idle' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	fresh := seedGoal("fresh", 2)  // 票龄<4h 宽限
	stale := seedGoal("stale", 10) // OPEN 10h 零触——该火
	touched := seedGoal("touch", 10)
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d 接续:分解中", touched)); err != nil {
		t.Fatal(err)
	}
	closed := seedGoal("done", 10)
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d 销——证伪", closed)); err != nil {
		t.Fatal(err)
	}
	_ = fresh

	svc.goalIdleEscCheck(ctx, deadSince)
	if n := idleN(); n != 1 {
		t.Fatalf("该火未火或误火: 升件数=%d(应=1,仅 stale 票)", n)
	}

	svc.goalIdleEscCheck(ctx, deadSince) // 同值未销不暴
	if n := idleN(); n != 1 {
		t.Fatalf("未销在案暴升: n=%d", n)
	}

	// 触出现→复绿撤态补发恢复行
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d 推进:账面分解完成", stale)); err != nil {
		t.Fatal(err)
	}
	svc.goalIdleEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%goal_idle%'`).Scan(&recN); err != nil || recN != 1 {
		t.Fatalf("触后未复绿: recN=%d err=%v", recN, err)
	}
}

// breedDryEscCheck 育种旱探:7d 零 proposed 评估→升件,note 载 proven 种数;
// 有新评估=不旱。proven 判定=template 后代 realized+paper 净>0。
func TestBreedDryEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM ops_event WHERE esc_scope='breed_dry'`,
		`DELETE FROM fin_paper_fill`, `DELETE FROM fin_strategy`,
		`DELETE FROM fin_proposal`, `DELETE FROM fin_opp_eval`,
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
			WHERE esc_scope='breed_dry' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// proven 种=live 策略+正 paper 净
	var sid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd)
		VALUES('BTC-USDT-SWAP','funding-hedge','live',100) RETURNING id`).
		Scan(&sid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_paper_fill
		(strategy_id,kind,inst_id,amount_usd) VALUES($1,'funding','BTC-USDT-SWAP',5)`, sid); err != nil {
		t.Fatal(err)
	}

	svc.breedDryEscCheck(ctx, deadSince) // 零评估+有种→火
	if n := fireN(); n != 1 {
		t.Fatalf("旱未火: n=%d", n)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE esc_scope='breed_dry' AND event='升件'`).Scan(&note); err != nil ||
		!strings.Contains(note, "1 个 proven") {
		t.Fatalf("note 缺 proven 数: %q", note)
	}

	// 新评估入水→复绿
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval
		(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('ETH-USDT-SWAP','funding-hedge','short_perp_long_spot',0.1,'proposed','[]','{}')`); err != nil {
		t.Fatal(err)
	}
	svc.breedDryEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%breed_dry%'`).Scan(&recN); err != nil || recN != 1 {
		t.Fatalf("进水未复绿: recN=%d err=%v", recN, err)
	}
}
