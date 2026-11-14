// reflex_esc_test.go —— 反射编译器:裁定行 1:1 回执+物化/lethal 记号放大/
// 放行撤闸/最新胜序(陈记号不复活已解禁规)/非法形拒回执/停摆探。
package main

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func reflexSetup(t *testing.T) (context.Context, coreService) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM fin_econ_rule`,
		`DELETE FROM ops_event WHERE event IN ('裁定','反射','处置','升件')`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	return ctx, coreService{pool: pool}
}

func ruleState(t *testing.T, ctx context.Context, svc coreService, canon string) (bool, bool) {
	var active bool
	err := svc.pool.QueryRow(ctx, `SELECT active FROM fin_econ_rule
		WHERE kind='admit_block' AND match=$1::jsonb`, canon).Scan(&active)
	if err != nil {
		return false, false
	}
	return active, true
}

func insertOps(t *testing.T, ctx context.Context, svc coreService, event, note string) int64 {
	var id int64
	if err := svc.pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席',$1,$2,$3) RETURNING id`, event, event, note).Scan(&id); err != nil {
		t.Fatalf("insert %s: %v", event, err)
	}
	return id
}

func TestReflexVerdictMaterializes(t *testing.T) {
	ctx, svc := reflexSetup(t)
	insertOps(t, ctx, svc, "裁定", `admit_block {"template_ref":"funding-hedge"}`)
	svc.reflexCompileCheck(ctx, map[string]time.Time{})
	act, ok := ruleState(t, ctx, svc, `{"template_ref":"funding-hedge"}`)
	if !ok || !act {
		t.Fatalf("裁定未物化为激活规约: exists=%v active=%v", ok, act)
	}
	var n int
	svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='反射'
		AND note ~ '裁定#[0-9]+→rule#[0-9]+'`).Scan(&n)
	if n != 1 {
		t.Fatalf("物化反射回执缺失: %d", n)
	}
}

func TestReflexLethalTokenAmplifies(t *testing.T) {
	ctx, svc := reflexSetup(t)
	insertOps(t, ctx, svc, "处置", "升件#9 销——信念失真标 lethal:dcd_call 断粮")
	svc.reflexCompileCheck(ctx, map[string]time.Time{})
	act, ok := ruleState(t, ctx, svc, `{"template_ref":"dcd_call"}`)
	if !ok || !act {
		t.Fatalf("lethal 记号未放大成闸: exists=%v active=%v", ok, act)
	}
}

func TestReflexReleaseDeactivates(t *testing.T) {
	ctx, svc := reflexSetup(t)
	insertOps(t, ctx, svc, "处置", "升件#9 销——lethal:dcd_call")
	svc.reflexCompileCheck(ctx, map[string]time.Time{})
	insertOps(t, ctx, svc, "处置", "升件#10 销——校准后 放行:dcd_call")
	svc.reflexCompileCheck(ctx, map[string]time.Time{})
	act, ok := ruleState(t, ctx, svc, `{"template_ref":"dcd_call"}`)
	if !ok || act {
		t.Fatalf("放行记号未撤闸: exists=%v active=%v", ok, act)
	}
}

func TestReflexLatestWins(t *testing.T) {
	ctx, svc := reflexSetup(t)
	insertOps(t, ctx, svc, "处置", "升件#9 销——lethal:x_tpl")                    // 陈 lethal
	insertOps(t, ctx, svc, "裁定", `解禁 admit_block {"template_ref":"x_tpl"}`) // 后件解禁
	svc.reflexCompileCheck(ctx, map[string]time.Time{})
	act, ok := ruleState(t, ctx, svc, `{"template_ref":"x_tpl"}`)
	if !ok || act {
		t.Fatalf("陈 lethal 复活了已解禁规约: exists=%v active=%v", ok, act)
	}
}

func TestReflexInvalidVerdictReceipt(t *testing.T) {
	ctx, svc := reflexSetup(t)
	insertOps(t, ctx, svc, "裁定", `admit_block {"venue":"okx"}`) // 越闭合集键
	svc.reflexCompileCheck(ctx, map[string]time.Time{})
	var n int
	svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='反射'
		AND note LIKE '%→拒:%'`).Scan(&n)
	if n != 1 {
		t.Fatalf("非法裁定无拒回执: %d", n)
	}
	svc.pool.QueryRow(ctx, `SELECT count(*) FROM fin_econ_rule`).Scan(&n)
	if n != 0 {
		t.Fatalf("非法裁定竟物化: %d", n)
	}
}

func TestReflexStallEscalates(t *testing.T) {
	ctx, svc := reflexSetup(t)
	id := insertOps(t, ctx, svc, "裁定", `admit_block {"template_ref":"y_tpl"}`)
	if _, err := svc.pool.Exec(ctx, `UPDATE ops_event SET ts=now()-interval '2 hours' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	// 不跑编译——模拟编译器死:未编裁定陈化>1h
	dead := map[string]time.Time{}
	var uncompiled int
	svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event e WHERE e.event='裁定'
		AND e.ts<now()-interval '1 hour'
		AND NOT EXISTS(SELECT 1 FROM ops_event r WHERE r.event='反射'
			AND r.note ~ ('裁定#'||e.id||'([^0-9]|$)'))`).Scan(&uncompiled)
	if uncompiled != 1 {
		t.Fatalf("未编裁定计数错: %d", uncompiled)
	}
	// 编译跑通后 stall 条件应消——跑一拍再核
	svc.reflexCompileCheck(ctx, dead)
	svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='升件'
		AND esc_class='反射'`).Scan(&uncompiled)
	if uncompiled != 0 {
		t.Fatalf("编译后仍升 stall 件: %d", uncompiled)
	}
}
