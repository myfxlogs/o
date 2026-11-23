package main

// O-34618 DB 读行径桩——fileRows 桩面不触滞后窗/水位/seen 集，
// 本件直走 pending()+processBatch() 真库径（testutil.Pool，
// O_TEST_PG_DSN 缺=skip）：单行+静默限期踹、补扫合并、晚提交
// 小 id 行不丢、水位只推进到已处理行。隔离手法：wm 锚 max(id)+1
// +LOOKBACK_ID=1 收窄补扫窗，防存量行混入断言。

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// kickRec 踹件记录器（kickFn 注入缝——断言单元去重/逐踹）。
type kickRec struct {
	mu   sync.Mutex
	unit []string
}

func (k *kickRec) fn(_ context.Context, u string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.unit = append(k.unit, u)
}

func (k *kickRec) count(u string) int {
	k.mu.Lock()
	defer k.mu.Unlock()
	n := 0
	for _, x := range k.unit {
		if x == u {
			n++
		}
	}
	return n
}

func maxEventID(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int64 {
	t.Helper()
	var m int64
	if err := pool.QueryRow(ctx, `SELECT COALESCE(max(id),0) FROM ops_event`).Scan(&m); err != nil {
		t.Fatalf("max id: %v", err)
	}
	return m
}

func insRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, event, note, tsExpr string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO ops_event(seat,event,note,ts) VALUES('wakebus-test',$1,$2,`+tsExpr+`) RETURNING id`,
		event, note).Scan(&id); err != nil {
		t.Fatalf("insert %s: %v", note, err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `DELETE FROM ops_event WHERE id=$1`, id) })
	return id
}

func TestPendingLagAndSweepDelivery(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Setenv("WAKEBUS_LAG_SEC", "1")     // 滞后窗压 1s——真库径可秒级验
	t.Setenv("WAKEBUS_LOOKBACK_ID", "1") // 补扫窗收 1——存量行不入断言域
	wm0 := maxEventID(t, ctx, pool) + 1
	id := insRow(t, ctx, pool, "工单", "求知:lagcase", "now()")

	st := &batchState{wm: wm0, seen: map[int64]struct{}{}}
	// 新鲜行（ts=now，龄<lag）——pending 必空（水位不动=读漏不越）
	if n, err := pending(ctx, pool, st.wm); err != nil || len(n) != 0 {
		t.Fatalf("fresh row pending=%d err=%v want 0", len(n), err)
	}
	// 静默期=补扫径再进（等价 sweepLoop 拍点）：行龄过 lag 后 pending 见
	time.Sleep(1200 * time.Millisecond) // sched-exempt-ok 滞后窗老化等待——被测谓词语义本体非调度原语
	kick := &kickRec{}
	if err := processBatch(ctx, pool, st, kick.fn); err != nil {
		t.Fatalf("batch: %v", err)
	}
	if kick.count("o-research-dispatch.service") != 1 {
		t.Fatalf("单行静默未踹 research unit=%v", kick.unit)
	}
	if st.wm != id {
		t.Fatalf("wm=%d want %d", st.wm, id)
	}
}

func TestStragglerSmallIDNotLost(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Setenv("WAKEBUS_LAG_SEC", "1")
	// 晚提交小 id 行：显式 id=maxid+5500（空位必无碰撞）、ts 老过滞后窗；
	// wm=S+50 越过它——默认 lookback 5000 使 pending 下界=S-4950 只收它
	base := maxEventID(t, ctx, pool)
	sid := base + 5500
	var lo int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO ops_event(id,seat,event,note,ts) VALUES($1,'wakebus-test','工单','求知:straggler',now()-interval '2 hours') RETURNING id`,
		sid,
	).Scan(&lo); err != nil {
		t.Fatalf("insert straggler: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `DELETE FROM ops_event WHERE id=$1`, lo) })

	st := &batchState{wm: sid + 50, seen: map[int64]struct{}{}}
	kick := &kickRec{}
	if err := processBatch(ctx, pool, st, kick.fn); err != nil {
		t.Fatalf("batch1: %v", err)
	}
	if kick.count("o-research-dispatch.service") != 1 {
		t.Fatalf("晚提交小 id 行被水位越过丢失 kick=%v", kick.unit)
	}
	if st.wm != sid+50 {
		t.Fatalf("补扫行动了水位 wm=%d", st.wm)
	}
	// 再跑一批——seen 去重不得重踹
	kick2 := &kickRec{}
	if err := processBatch(ctx, pool, st, kick2.fn); err != nil {
		t.Fatalf("batch2: %v", err)
	}
	if len(kick2.unit) != 0 {
		t.Fatalf("seen 去重失效重踹=%v", kick2.unit)
	}
}

func TestCoalesceAndWatermarkOnlyProcessed(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Setenv("WAKEBUS_LAG_SEC", "1")
	t.Setenv("WAKEBUS_LOOKBACK_ID", "1")
	wm0 := maxEventID(t, ctx, pool) + 1
	// 连珠两老行+一新鲜行——合并踹+水位不越未处理行
	insRow(t, ctx, pool, "工单", "求知:burst1", "now()-interval '1 hour'")
	id2 := insRow(t, ctx, pool, "工单", "求知:burst2", "now()-interval '1 hour'")
	id3 := insRow(t, ctx, pool, "工单", "求知:burst3", "now()")

	st := &batchState{wm: wm0, seen: map[int64]struct{}{}}
	kick := &kickRec{}
	if err := processBatch(ctx, pool, st, kick.fn); err != nil {
		t.Fatalf("batch1: %v", err)
	}
	// 老两行过窗踹一发合并；新鲜行滞后窗外不入水位
	if n := kick.count("o-research-dispatch.service"); n != 1 {
		t.Fatalf("合并失效 kicks=%d want 1", n)
	}
	if st.wm != id2 {
		t.Fatalf("wm=%d want %d（新鲜行 id=%d 不得入水位）", st.wm, id2, id3)
	}
	// 新鲜行龄过窗后下拍（=补扫径）见行踹+水位推进
	time.Sleep(1200 * time.Millisecond) // sched-exempt-ok 滞后窗老化等待——被测谓词语义本体非调度原语
	kick2 := &kickRec{}
	if err := processBatch(ctx, pool, st, kick2.fn); err != nil {
		t.Fatalf("batch2: %v", err)
	}
	if st.wm != id3 {
		t.Fatalf("二拍 wm=%d want %d", st.wm, id3)
	}
}

func TestSweepLoopFires(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fires := make(chan struct{}, 8)
	go sweepLoop(ctx, 50*time.Millisecond, func() error { // sched-exempt-ok 桩测节拍=被测件本体周期
		fires <- struct{}{}
		return nil
	})
	select {
	case <-fires:
	case <-time.After(2 * time.Second): // sched-exempt-ok 桩测看门狗超时——非周期调度
		t.Fatal("sweepLoop 2s 内未拍")
	}
	select {
	case <-fires:
	case <-time.After(2 * time.Second): // sched-exempt-ok 桩测看门狗超时——非周期调度
		t.Fatal("sweepLoop 未周期续拍")
	}
}
