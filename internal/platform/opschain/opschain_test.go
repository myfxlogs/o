package opschain

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0386 验收锚桩测：①插3行链连续+篡改中间行→验链检出+自报行；
// ②REVOKE 断言=应用角色 UPDATE/DELETE 被拒（INSERT 照旧——不破坏写面）；
// ③创世锚+链起点断言。

func fixture(t *testing.T) (*Verifier, context.Context) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM ops_event"); err != nil {
		t.Fatal(err)
	}
	return New(pool), ctx
}

type chainRow struct {
	ID       int64
	PrevHash *string
	Hash     *string
}

func chained(t *testing.T, v *Verifier, ctx context.Context) []chainRow {
	t.Helper()
	rows, err := v.pool.Query(ctx,
		`SELECT id,prev_hash,hash FROM ops_event WHERE hash IS NOT NULL ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []chainRow
	for rows.Next() {
		var r chainRow
		if err := rows.Scan(&r.ID, &r.PrevHash, &r.Hash); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestGenesisAnchorAndLinkage 锚③：迁移后首个链行=创世锚 prev_hash NULL；
// 链起点后逐行 prev_hash=前行 hash；写者显式供伪 hash 被 trigger 覆写。
func TestGenesisAnchorAndLinkage(t *testing.T) {
	v, ctx := fixture(t)
	put := func(note string) {
		if _, err := v.pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note) VALUES('桩席','桩事','T-9Z01',$1)`,
			note); err != nil {
			t.Fatal(err)
		}
	}
	put("创世行")
	put("第二行")
	// 显式供伪戳——trigger 覆写（伪戳入库不可能）。
	if _, err := v.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,prev_hash,hash)
		VALUES('桩席','桩事','T-9Z01','伪戳行','forged_prev','forged_hash')`); err != nil {
		t.Fatal(err)
	}
	cs := chained(t, v, ctx)
	if len(cs) != 3 {
		t.Fatalf("链行数=%d 应=3", len(cs))
	}
	if cs[0].PrevHash != nil {
		t.Fatalf("创世锚破：首链行 prev_hash=%v 应 NULL", *cs[0].PrevHash)
	}
	if cs[0].Hash == nil || len(*cs[0].Hash) != 64 {
		t.Fatalf("创世行 hash=%v 应 64 hex", cs[0].Hash)
	}
	for i := 1; i < len(cs); i++ {
		if cs[i].PrevHash == nil || *cs[i].PrevHash != *cs[i-1].Hash {
			t.Fatalf("链断@i=%d：prev_hash≠前行 hash", i)
		}
	}
	if *cs[2].Hash == "forged_hash" || (cs[2].PrevHash != nil && *cs[2].PrevHash == "forged_prev") {
		t.Fatal("显式伪戳未被 trigger 覆写")
	}
	rep, err := v.Verify(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK() {
		t.Fatalf("完好链误报: %+v", rep.Findings)
	}
}

// TestTamperDetectedAndReported 锚①：插3行链连续→篡改中间行（owner 面
// 模拟高权追改）→验链检出伪戳+断链→自报双行（账链断+升件）落账。
func TestTamperDetectedAndReported(t *testing.T) {
	v, ctx := fixture(t)
	var ids [3]int64
	for i := range ids {
		if err := v.pool.QueryRow(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note) VALUES('桩席','桩事','T-9Z02',$1)
			RETURNING id`, fmt.Sprintf("链行%d", i+1)).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	if rep, err := v.Verify(ctx); err != nil || !rep.OK() {
		t.Fatalf("3行链未完: %+v err=%v", rep, err)
	}
	// 篡改中间行内容——hash 不随行重算→伪戳检出。
	if _, err := v.pool.Exec(ctx,
		`UPDATE ops_event SET note='追改后内容' WHERE id=$1`, ids[1]); err != nil {
		t.Fatal(err)
	}
	rep, err := v.Verify(ctx)
	if err != nil {
		t.Fatal(err)
	}
	kind := map[string]bool{}
	for _, f := range rep.Findings {
		kind[f.Kind] = true
	}
	if !kind["伪戳"] {
		t.Fatalf("中间行篡改未检出伪戳: %+v", rep.Findings)
	}
	// 篡改链针——断链检出（prev_hash 与前链行 hash 不等）。
	if _, err := v.pool.Exec(ctx,
		`UPDATE ops_event SET prev_hash='deadbeef' WHERE id=$1`, ids[2]); err != nil {
		t.Fatal(err)
	}
	rep, err = v.Verify(ctx)
	if err != nil {
		t.Fatal(err)
	}
	kind = map[string]bool{}
	for _, f := range rep.Findings {
		kind[f.Kind] = true
	}
	if !kind["断链"] {
		t.Fatalf("链针篡改未检出断链: %+v", rep.Findings)
	}
	// 自报双行：账链断+升件（账链断类/ops_event 域）。
	if err := v.ReportBreak(ctx, rep); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := v.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='账链断' AND ticket='T-0386'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("账链断行: n=%d err=%v", n, err)
	}
	if err := v.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='账链断' AND esc_scope='ops_event'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件行: n=%d err=%v", n, err)
	}
	// 同窗 24h 去重——断链未修复持续检出亦不爆量。
	if err := v.ReportBreak(ctx, rep); err != nil {
		t.Fatal(err)
	}
	if err := v.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='账链断' AND ticket='T-0386'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("同窗去重失效: n=%d err=%v", n, err)
	}
	// 自报行自身入链（自报可自证——链写触发器同覆）。
	cs := chained(t, v, ctx)
	if len(cs) != 3+2 { // 3 桩行+账链断+升件（第二拍去重零行）
		t.Fatalf("自报行未入链: 链行=%d", len(cs))
	}
}

// TestResidualSuppressesForgeryOnly 豁免册（fin_0169/工单#11419）：
// 册内行仅抑「伪戳」重报——断链拓扑检不豁免。
func TestResidualSuppressesForgeryOnly(t *testing.T) {
	v, ctx := fixture(t)
	var id int64
	if err := v.pool.QueryRow(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('桩席','桩事','T-9Z03','豁免桩行')
		RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := v.pool.Exec(ctx,
		`UPDATE ops_event SET note='追改后内容' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	rep, err := v.Verify(ctx)
	if err != nil || rep.OK() {
		t.Fatalf("篡改未检出: %+v err=%v", rep, err)
	}
	if _, err := v.pool.Exec(ctx,
		`INSERT INTO ops_chain_residual(id,reason,ticket_ref) VALUES($1,'桩豁免','工单#11419')`, id); err != nil {
		t.Fatal(err)
	}
	rep, err = v.Verify(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rep.Findings {
		if f.ID == id && f.Kind == "伪戳" {
			t.Fatalf("册内行伪戳未抑: %+v", rep.Findings)
		}
	}
	// 册内行断链仍检——豁免=伪戳一档非拓扑免检。
	if _, err := v.pool.Exec(ctx,
		`UPDATE ops_event SET prev_hash='deadbeef' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	rep, err = v.Verify(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var dang bool
	for _, f := range rep.Findings {
		if f.ID == id && f.Kind == "断链" {
			dang = true
		}
	}
	if !dang {
		t.Fatalf("册内行断链被误豁免: %+v", rep.Findings)
	}
}

// TestLockdownRevokesUD 锚②：应用角色 UPDATE/DELETE/TRUNCATE 被拒
// （锁权幂等——授权漂移重锁同调一式）；INSERT 照旧=写面不破坏。
func TestLockdownRevokesUD(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 桩角色=锁权册在册名（roster_rw——缺席则建；角色=库级全局件非 schema 域）。
	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='roster_rw')`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		if _, err := pool.Exec(ctx, `CREATE ROLE roster_rw`); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = pool.Exec(ctx2, `DROP ROLE roster_rw`)
		})
	}
	// 测试 schema USAGE 补授——prod 写面落 public（PUBLIC 天生持 USAGE），
	// 包级隔离 schema 无 PUBLIC USAGE，不补=名解析不可见非权拒。
	var schema string
	if err := pool.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `GRANT USAGE ON SCHEMA "`+schema+`" TO roster_rw;
		GRANT SELECT,INSERT,UPDATE,DELETE,TRUNCATE ON ops_event TO roster_rw;
		GRANT USAGE ON SEQUENCE ops_event_id_seq TO roster_rw;
		GRANT SELECT,UPDATE ON ops_headless TO roster_rw`); err != nil { // fin_0088 touch 触发器 INVOKER 面——WHERE 读列须 SELECT+UPDATE（prod 写角色同款在册权）
		t.Fatal(err)
	}
	// 授权漂移景→重锁——UD/TRUNCATE 收编、INSERT/SELECT 照旧。
	if _, err := pool.Exec(ctx, `SELECT ops_event_lockdown()`); err != nil {
		t.Fatal(err)
	}
	for _, priv := range []struct {
		p    string
		want bool
	}{
		{"UPDATE", false}, {"DELETE", false}, {"TRUNCATE", false},
		{"INSERT", true}, {"SELECT", true},
	} {
		var has bool
		if err := pool.QueryRow(ctx,
			`SELECT has_table_privilege('roster_rw','ops_event',$1)`, priv.p).Scan(&has); err != nil {
			t.Fatal(err)
		}
		if has != priv.want {
			t.Fatalf("roster_rw %s=%v 应=%v（锁权面漂移）", priv.p, has, priv.want)
		}
	}
	// 功能断言：应用角色面 UPDATE/DELETE 实拒、INSERT 实通（写面不破坏）。
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SET ROLE roster_rw`); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = conn.Exec(context.Background(), `RESET ROLE`) }()
	if _, err := conn.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('桩席','锁后写','T-9Z03','锁后插行')`); err != nil {
		t.Fatalf("锁后 INSERT 被拒——写面破坏: %v", err)
	}
	if _, err := conn.Exec(ctx, `UPDATE ops_event SET note='x' WHERE true`); err == nil ||
		!strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("锁后 UPDATE 未拒: %v", err)
	}
	if _, err := conn.Exec(ctx, `DELETE FROM ops_event WHERE true`); err == nil ||
		!strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("锁后 DELETE 未拒: %v", err)
	}
}
