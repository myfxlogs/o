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
		fmt.Sprintf("工单#%d digest 交付毕——canon 条目落位", tid)); err != nil {
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
		fmt.Sprintf("工单#%d digest 交付毕", tid)); err != nil {
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
