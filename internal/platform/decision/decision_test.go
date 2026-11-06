package decision

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0415 桩测面：①fin 写径单行落账+载荷；②seat 镜像触发器三词+COS 升件
// 打包口；③裁词裸词 RAISE（红桩②——「通过」裸行绝迹：镜像败令事件行
// 整行不入账）；④裁定席 verdict 镜像（lesson→verdicted）；⑤LostEsc
// 写败升件+同窗去重。

func fixture(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return pool, ctx
}

func decCount(t *testing.T, pool *pgxpool.Pool, ctx context.Context, where string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_decision WHERE `+where, args...).Scan(&n); err != nil {
		t.Fatalf("decision count: %v", err)
	}
	return n
}

func TestWriteFinRow(t *testing.T) {
	pool, ctx := fixture(t)
	r := Row{Domain: "fin", Actor: "guard", RefID: "42", Action: "close",
		Reason:  "funding_flip：latest=-0.0001×sign=+1 |率|10bps>flip_bps=5",
		Payload: map[string]any{"position_id": 7, "expected_net_apr": 0.08, "realized_usd": -1.2}}
	if err := Write(ctx, pool, r); err != nil {
		t.Fatalf("fin 行写败: %v", err)
	}
	var dom, act, ref, action, reason, payload string
	if err := pool.QueryRow(ctx, `SELECT domain,actor,ref_id,action,reason,payload::text
		FROM ops_decision ORDER BY id DESC LIMIT 1`).Scan(
		&dom, &act, &ref, &action, &reason, &payload); err != nil {
		t.Fatal(err)
	}
	if dom != "fin" || act != "guard" || ref != "42" || action != "close" ||
		!strings.Contains(reason, "funding_flip") {
		t.Fatalf("行面漂移: %s/%s/%s/%s %s", dom, act, ref, action, reason)
	}
	if !strings.Contains(payload, "realized_usd") {
		t.Fatalf("payload 缺实现件: %s", payload)
	}
}

func TestSeatMirrorVerdict(t *testing.T) {
	pool, ctx := fixture(t)
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('决策席','裁决','T-9000','通过：验收锚齐 全链绿戳 sha=abc123')`); err != nil {
		t.Fatal(err)
	}
	var actor, ref, action, reason string
	if err := pool.QueryRow(ctx, `SELECT actor,ref_id,action,reason FROM ops_decision
		WHERE domain='seat' ORDER BY id DESC LIMIT 1`).Scan(&actor, &ref, &action, &reason); err != nil {
		t.Fatal(err)
	}
	if actor != "决策席" || ref != "T-9000" || action != "裁决" || !strings.Contains(reason, "验收锚齐") {
		t.Fatalf("seat 镜像行漂移: %s %s %s %s", actor, ref, action, reason)
	}
}

func TestSeatMirrorParamAdjustAndCOS(t *testing.T) {
	pool, ctx := fixture(t)
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('决策席-DM写径','参数调整','T-0376','key=guard.flip_bps 旧=5 新=6 证据=x')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('COS','COS上行','','打包件 docs/cos/packs/20261107-x.md——红线三件呈业主')`); err != nil {
		t.Fatal(err)
	}
	if n := decCount(t, pool, ctx, `action='参数调整'`); n != 1 {
		t.Fatalf("参数调整镜像缺: %d", n)
	}
	var action, actor string
	if err := pool.QueryRow(ctx, `SELECT action,actor FROM ops_decision
		WHERE action='升件裁量'`).Scan(&action, &actor); err != nil || actor != "COS" {
		t.Fatalf("COS 升件打包镜像缺/漂: %v %s", err, actor)
	}
	// 非 COS 升件行=机器自报不镜像（升件裁量只收席位裁量面）。
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('META-DM','升件','','队列推导败——容错必报')`); err != nil {
		t.Fatal(err)
	}
	if n := decCount(t, pool, ctx, `action='升件裁量'`); n != 1 {
		t.Fatalf("机器升件误镜像: %d", n)
	}
}

func TestBareVerdictRejected(t *testing.T) {
	pool, ctx := fixture(t)
	// 红桩②：裸词裁词在任意写径不可落——镜像 RAISE 令 ops_event 行整行败退。
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('决策席','裁决','T-9001','通过')`); err == nil {
		t.Fatal("裸词裁决行竟入账——红桩②失守")
	}
	if n := decCount(t, pool, ctx, `ref_id='T-9001'`); n != 0 {
		t.Fatalf("裸词裁词镜像竟落账: %d", n)
	}
	var en int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE ticket='T-9001'`).Scan(&en); err != nil || en != 0 {
		t.Fatalf("裸词事件行竟落账: %d %v", en, err)
	}
	// 直写口同守（psql/壳径）。
	if err := Write(ctx, pool, Row{Domain: "seat", Actor: "决策席", RefID: "T-9002",
		Action: "裁决", Reason: "通过"}); err == nil {
		t.Fatal("写口裸词竟放行")
	}
}

func TestLessonVerdictMirror(t *testing.T) {
	pool, ctx := fixture(t)
	var lid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_econ_lesson(kind,entity_ref,numbers)
		VALUES('settled_loss','position:9','{}') RETURNING id`).Scan(&lid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_econ_lesson SET status='verdicted',
		verdict_ref='canon:semantic:cmc-trx-vs-company:锚对拍成立' WHERE id=$1`, lid); err != nil {
		t.Fatal(err)
	}
	var actor, ref, reason string
	if err := pool.QueryRow(ctx, `SELECT actor,ref_id,reason FROM ops_decision
		WHERE domain='seat' AND action='裁决'`).Scan(&actor, &ref, &reason); err != nil {
		t.Fatal(err)
	}
	if actor != "裁定席" || ref != "fin_econ_lesson#"+strconv.FormatInt(lid, 10) || !strings.Contains(reason, "canon:") {
		t.Fatalf("裁定席镜像行漂移: %s %s %s", actor, ref, reason)
	}
	// 裸 verdict_ref=裁词缺席——pending→verdicted 首迁即 RAISE 整行败退（禁静默）。
	var lid2 int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_econ_lesson(kind,entity_ref,numbers)
		VALUES('falsified_verdict','position:11','{}') RETURNING id`).Scan(&lid2); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_econ_lesson SET status='verdicted',
		verdict_ref='' WHERE id=$1`, lid2); err == nil {
		t.Fatal("裸 verdict_ref 竟过镜像——裁词缺席失守")
	}
	var st string
	if err := pool.QueryRow(ctx, `SELECT status FROM fin_econ_lesson WHERE id=$1`, lid2).Scan(&st); err != nil || st != "pending" {
		t.Fatalf("裸裁词 verdict 竟落账: %s %v", st, err)
	}
}

func TestLostEscDedup(t *testing.T) {
	pool, ctx := fixture(t)
	bad := Row{Domain: "fin", Actor: "lifecycle", RefID: "77", Action: "bogus", Reason: "x"}
	if err := Write(ctx, pool, bad); err == nil {
		t.Fatal("词集外行竟放行——CHECK 失守")
	}
	WriteEsc(ctx, pool, bad) // 写败→升件
	WriteEsc(ctx, pool, bad) // 同窗去重——至多一报
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='写败' AND esc_scope='ops_decision'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("写败升件行缺/重复: %d %v", n, err)
	}
}
