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

// museDueEscCheck muse 醒探:自问未答题在案+队列轻+距末想>8h∨零史→
// 升件;答题/重队列/频闸内皆不唤;muse 行产=自复绿。
func TestMuseDueEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='muse' OR event='自问' OR ticket='muse' OR event='升件'`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	fireN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_scope='muse' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	svc.museDueEscCheck(ctx, deadSince) // 零题不唤
	if n := fireN(); n != 0 {
		t.Fatalf("零题误唤: n=%d", n)
	}

	var qid int64
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席','自问','self_question','armed 哑弹为何 launch 前无验证? 源=勘面') RETURNING id`).
		Scan(&qid); err != nil {
		t.Fatal(err)
	}
	svc.museDueEscCheck(ctx, deadSince) // 题在+队列空+零史→唤
	if n := fireN(); n != 1 {
		t.Fatalf("该唤未唤: n=%d", n)
	}

	// 答题销题(muse 行+处置引)→复绿
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('唤知席','muse','muse',$1)`,
		fmt.Sprintf("自问#%d armed哑弹为何无launch验证→落=工单", qid)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("自问#%d 答毕——muse 行在案", qid)); err != nil {
		t.Fatal(err)
	}
	svc.museDueEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%/muse%'`).Scan(&recN); err != nil || recN != 1 {
		t.Fatalf("答题未复绿: recN=%d err=%v", recN, err)
	}
}

// learnGapEscCheck 求知滞探:OPEN 求知票龄>2h 零触→升件;
// 新票(<2h)/活在认领/已处置皆静默;销票处置=复绿。
func TestLearnGapEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='learn_gap' OR event='工单' OR event='认领'`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	fireN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_scope='learn_gap' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	svc.learnGapEscCheck(ctx, deadSince) // 零票静默
	if n := fireN(); n != 0 {
		t.Fatalf("零票误唤: n=%d", n)
	}

	// 新票(<2h)不唤
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','工单','求知:okx earn 活期产品面——期限=24h,源=自问#1')`); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("新票误唤: n=%d", n)
	}

	// 陈票(3h)唤
	var tid int64
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,ts)
		VALUES('处置席','工单','求知:mtapi 还有哪些已授信未探读面——期限=48h,源=自问#2',
			now()-interval '3 hours') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	if n := fireN(); n != 1 {
		t.Fatalf("陈票未唤: n=%d", n)
	}

	// 处置销票→复绿
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d digest 交付毕——canon 条目落位", tid)); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%/learn_gap%'`).Scan(&recN); err != nil || recN != 1 {
		t.Fatalf("销票未复绿: recN=%d err=%v", recN, err)
	}
}

// museClaimEscCheck muse 落实探:muse 行缺「落=」∧无核行→升件;
// 载落=静默;处置核行=复绿。
func TestMuseClaimEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='muse_claim' OR event='muse' OR event='处置'`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	fireN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_scope='muse_claim' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// 合规 muse 行(载落=)不唤
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('唤知席','muse','muse','自问#9 armed哑弹为何无launch验证→需venue闸;落=工单#555')`); err != nil {
		t.Fatal(err)
	}
	svc.museClaimEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("合规行误唤: n=%d", n)
	}

	// 缺落=行唤
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('唤知席','muse','muse','自问#10 想了一圈感觉挺好的')`); err != nil {
		t.Fatal(err)
	}
	svc.museClaimEscCheck(ctx, deadSince)
	if n := fireN(); n != 1 {
		t.Fatalf("缺落=未唤: n=%d", n)
	}

	// 核行处置→复绿
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置','muse 行核毕:自问#10 判词不实已追立工单#556')`); err != nil {
		t.Fatal(err)
	}
	svc.museClaimEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%/muse_claim%'`).Scan(&recN); err != nil || recN != 1 {
		t.Fatalf("核行未复绿: recN=%d err=%v", recN, err)
	}
}

// selfq_gap 自问字段 drift 核:锚(首条载 自问= 收口行)前安静,
// 锚后缺字段行→唤知/selfq_gap;同条件不重复火(escEdgeUV dedup)。
func TestSelfqGapEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='selfq_gap' OR (event='会话收口' AND seat='唤知')`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	fireN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_scope='selfq_gap' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// 锚前行(契未上线)不计
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('唤知','会话收口','unit=t-a 收口:面净 用帧=自态')`); err != nil {
		t.Fatal(err)
	}
	svc.frameUseEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("锚前误唤: n=%d", n)
	}

	// 锚行(契上线证)
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('唤知','会话收口','unit=t-b 收口:处置 用帧=自态 自问=9')`); err != nil {
		t.Fatal(err)
	}
	svc.frameUseEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("锚行即唤: n=%d", n)
	}

	// 锚后缺字段行→火
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('唤知','会话收口','unit=t-c 收口:处置 用帧=叙事')`); err != nil {
		t.Fatal(err)
	}
	svc.frameUseEscCheck(ctx, deadSince)
	if n := fireN(); n != 1 {
		t.Fatalf("缺字段未唤: n=%d", n)
	}

	// 条件在不再火(dedup)
	svc.frameUseEscCheck(ctx, deadSince)
	if n := fireN(); n != 1 {
		t.Fatalf("同条件重火: n=%d", n)
	}
}
