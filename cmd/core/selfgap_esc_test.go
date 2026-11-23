package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// learnGapEscCheck 求知滞探（分级契约#22578①+O-32295）:OPEN 求知票龄>2h
// 分级输出——计数带(单票≤12h∧<3张)死态日计一火不落升件;热带(任一票>12h
// ∨滞票≥3张)转真升件;新票(<2h)/活在认领/已处置皆不死;计数带销票=
// 复绿清态零恢复行,升件带销票=恢复行自闭。
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
	resetSelfMeter()
	fireN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_scope='learn_gap' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	meterN := func() int { return selfEscMeter.counts["唤知/learn_gap"] }

	svc.learnGapEscCheck(ctx, deadSince) // 零票静默
	if n := fireN(); n != 0 {
		t.Fatalf("零票误唤: n=%d", n)
	}
	if meterN() != 0 {
		t.Fatalf("零票误计: %v", selfEscMeter.counts)
	}

	// 新票(<2h)不死
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','工单','求知:okx earn 活期产品面——期限=24h,源=自问#1')`); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("新票误唤: n=%d", n)
	}
	if meterN() != 0 {
		t.Fatalf("新票误计: %v", selfEscMeter.counts)
	}

	// 陈票(3h)死态——日计一火无升件
	var tid int64
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,ts)
		VALUES('处置席','工单','求知:mtapi 还有哪些已授信未探读面——期限=48h,源=自问#2',
			now()-interval '3 hours') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("降级件竟升件: n=%d", n)
	}
	if meterN() != 1 {
		t.Fatalf("陈票死态未入日计: %v", selfEscMeter.counts)
	}

	// 处置销票→复绿清态零恢复行
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d 销——digest 交付毕 canon 条目落位", tid)); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%/learn_gap%'`).Scan(&recN); err != nil || recN != 0 {
		t.Fatalf("降级件竟写恢复: recN=%d err=%v", recN, err)
	}
}

// TestLearnGapEscCheckHot 单票滞龄>12h=热带真升件（O-32295 案②）——
// 升件落账不占日计;销票复绿=恢复行自闭（升件态与计数带分键实证）。
func TestLearnGapEscCheckHot(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='learn_gap' OR event='工单' OR event='认领' OR event='恢复'`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	resetSelfMeter()

	var tid int64
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,ts)
		VALUES('处置席','工单','求知:hyperliquid 清算流历史面——期限=72h,源=自问#9',
			now()-interval '13 hours') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='learn_gap' AND event='升件'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("热带票未升件: n=%d err=%v", n, err)
	}
	if m := selfEscMeter.counts["唤知/learn_gap"]; m != 0 {
		t.Fatalf("升件带误占日计: %v", selfEscMeter.counts)
	}

	// 销票→复绿恢复行（升件态 deadSince 在案→recoverNote 自闭）
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d 销——digest 交付毕", tid)); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	var recN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%/learn_gap%'`).Scan(&recN); err != nil || recN != 1 {
		t.Fatalf("升件带复绿无恢复行: recN=%d err=%v", recN, err)
	}
}

// TestLearnGapEscCheckTicketN 滞票≥3张=簇级热带升件（O-32295 案③）——
// 单票皆在计数带(3h)但张数过阈→升件;日计不占。
func TestLearnGapEscCheckTicketN(t *testing.T) {
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
	resetSelfMeter()

	for i := 0; i < learnGapEscTicketN; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note,ts)
			VALUES('处置席','工单',$1,now()-interval '3 hours')`,
			fmt.Sprintf("求知:venue-%d 普查面续——期限=24h,源=自问#%d", i, i)); err != nil {
			t.Fatal(err)
		}
	}
	svc.learnGapEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='learn_gap' AND event='升件'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("滞票过阈未升件: n=%d err=%v", n, err)
	}
	if m := selfEscMeter.counts["唤知/learn_gap"]; m != 0 {
		t.Fatalf("升件带误占日计: %v", selfEscMeter.counts)
	}
}

// TestLearnGapEscCheckAnchor 锚位契约（工单O-32638 词面闸普查B档②）：
// learn_gap 消费谓词与 dispatch_research 研究票消费锚同形——句首
// ^求知[:：]|研究席[:：]；「工单/slug:…文中求知：…」句中提及非研究票，
// 不得计滞留（产消谓词不对称=colon 案同族未爆面）。
func TestLearnGapEscCheckAnchor(t *testing.T) {
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
	resetSelfMeter()

	// 句中「求知：」提及（非句首）+陈票龄——锚位拒，不入日计不升件
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note,ts)
		VALUES('处置席','工单','工单/probe-x: 本件提及求知：子题派发——期限=24h',
			now()-interval '3 hours')`); err != nil {
		t.Fatal(err)
	}
	// 对照：句首「研究席：」前缀票=消费锚同形，陈票应入日计
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note,ts)
		VALUES('处置席','工单','研究席：venue-y 深研面——期限=48h',
			now()-interval '3 hours')`); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	if m := selfEscMeter.counts["唤知/learn_gap"]; m != 1 {
		t.Fatalf("锚位判错: 句中票误计或研究席票漏计 meter=%v", selfEscMeter.counts)
	}
}

// TestLearnGapEscCheckActCloseSource 闭票谓词同源化（工单O-35491④/源票
// #35525）：旧谓词只认 event='处置'∧note~工单#N——研究席会话收口
// （event='会话收口' sem={"act":"close","rel":N} 显式销契，产线实证
// #35082/#32652/#32274 等）闭票被计零触，已交付票必复火（实证#35417
// 五票全闭照火）。新谓词吃 ops_event_act act='close'（rel=t.id∧ts>t.ts，
// 与 ops_ticket_desk closed 同源——desk 定义改自动跟随）。
func TestLearnGapEscCheckActCloseSource(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='learn_gap' OR event='工单' OR event='认领' OR event='会话收口' OR event='处置'`); err != nil {
		t.Fatal(err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	resetSelfMeter()

	// 五票求知票（3h 陈票龄=计数带内）——四票走会话收口销契、一票走处置销词
	ids := make([]int64, 0, 5)
	for i := 0; i < 5; i++ {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,ts)
			VALUES('处置席','工单',$1,now()-interval '3 hours') RETURNING id`,
			fmt.Sprintf("求知:venue-z%d 深研面——期限=24h,源=自问#%d", i, i)).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	for i, id := range ids[:4] {
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note,sem)
			VALUES('研究席','会话收口',$1,$2)`,
			fmt.Sprintf("unit=o-research-t 触发=tk#%d 产出=digest#%d", id, id),
			fmt.Sprintf(`{"act":"close","rel":%d}`, id)); err != nil {
			t.Fatalf("会话收口销行 #%d: %v", i, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`,
		fmt.Sprintf("工单#%d 销——digest 交付毕", ids[4])); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='learn_gap' AND event='升件'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("五闭票误升件: n=%d err=%v", n, err)
	}
	if m := selfEscMeter.counts["唤知/learn_gap"]; m != 0 {
		t.Fatalf("五闭票误入日计: %v", selfEscMeter.counts)
	}

	// 对照：真滞票（13h 热带）仍检——升件落账
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note,ts)
		VALUES('处置席','工单','求知:venue-w 真滞面——期限=24h,源=自问#z',
			now()-interval '13 hours')`); err != nil {
		t.Fatal(err)
	}
	svc.learnGapEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='learn_gap' AND event='升件'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("真滞票漏检: n=%d err=%v", n, err)
	}
}
