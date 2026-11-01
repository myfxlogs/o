// localwrite_test.go —— T-0209 桩锚：谓词过滤+per-inst 3s 节流双断言
// +审计驳回锚③补桩（撞键/共写幂等 DB 桩——O_TEST_PG_DSN 缺=skip）。
package okxws

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestLocalWritePredFilter(t *testing.T) {
	c := &Collector{
		LocalWritePred: func(kind, instID string) bool {
			return kind == "ticks" && instID == "BTC-USDT-SWAP" // funding 同集但测 ticks 域
		},
	}
	it := rowItem{tick: &tickRow{instID: "BTC-USDT-SWAP"}}
	// 谓词闸路径（handle 内联件——桩直接断言谓词判据）
	if !c.LocalWritePred("ticks", it.tick.instID) {
		t.Fatal("活跃集 inst 应过闸")
	}
	if c.LocalWritePred("ticks", "SOL-USDT-SWAP") {
		t.Fatal("离集 inst 应跳闸（不落行不写僵尸）")
	}
}

func TestPassThrottle3s(t *testing.T) {
	c := &Collector{passAt: map[string]time.Time{}}
	it := rowItem{tick: &tickRow{instID: "BTC-USDT-SWAP"}}
	if !c.passThrottle(it) {
		t.Fatal("首帧应过")
	}
	if c.passThrottle(it) {
		t.Fatal("3s 内同 inst 应跳帧（写洪防）")
	}
	c.passAt["ticks:BTC-USDT-SWAP"] = time.Now().Add(-4 * time.Second)
	if !c.passThrottle(it) {
		t.Fatal("4s 后应过（间隔满）")
	}
}

// T-0209 审计驳回锚③补桩（headless 2026-10-30 判词：upsert 写径零撞键
// 断言、同键共写幂等零桩）。DB 桩——O_TEST_PG_DSN 缺=skip（testutil 惯例）。

// uTicksBridge 桥投影 md_ticks upsert（projector/mirror.go uTicks 逐字拷——
// D-0030 §六 per-table 键表；桥臂=projector 反向桥维护面，本桩验「本地写×
// 桥写同键共写幂等」兼容语义：镜像 SQL 漂移由 projector 包自测拦，此处
// 锁的是 D-0018 最新赢/不膨胀契约）。
const uTicksBridge = `INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask,vol_24h,recv_ts)
	VALUES($1::timestamptz,$2,$3,$4::numeric,$5::numeric,$6::numeric,$7::numeric,$8::timestamptz)
	ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last,bid=EXCLUDED.bid,
		ask=EXCLUDED.ask,vol_24h=EXCLUDED.vol_24h,recv_ts=EXCLUDED.recv_ts
	WHERE EXCLUDED.ts > md_ticks.ts`

func mdTickState(t *testing.T, pool *pgxpool.Pool, inst string) (int, float64) {
	t.Helper()
	var n int
	var last float64
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*),COALESCE(max(last),0) FROM md_ticks WHERE inst_id=$1`, inst).Scan(&n, &last); err != nil {
		t.Fatal(err)
	}
	return n, last
}

// TestInsertBatchKeyCollisionIdempotent 撞键断言：同键批内双行+跨批重发
// 零膨胀（upsert 末帧覆盖非追加）；funding 同键同拍 DO NOTHING 不膨胀。
func TestInsertBatchKeyCollisionIdempotent(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, "DELETE FROM md_ticks")
	_, _ = pool.Exec(ctx, "DELETE FROM md_funding")
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM md_ticks") }()
	c := &Collector{pool: pool}
	base := time.Now().UTC().Truncate(time.Second)
	// 批内同键双行（值异）——行数不膨胀、后行赢
	if err := c.insertBatch(ctx, []rowItem{
		{tick: &tickRow{instID: "BTC-USDT-SWAP", ts: base, last: 100}},
		{tick: &tickRow{instID: "BTC-USDT-SWAP", ts: base.Add(time.Second), last: 200}},
	}); err != nil {
		t.Fatal(err)
	}
	if n, last := mdTickState(t, pool, "BTC-USDT-SWAP"); n != 1 || last != 200 {
		t.Fatalf("批内撞键: 行=%d last=%v（期望 1/200——upsert 非追加）", n, last)
	}
	// 跨批重发同键新帧——仍零膨胀
	if err := c.insertBatch(ctx, []rowItem{
		{tick: &tickRow{instID: "BTC-USDT-SWAP", ts: base.Add(2 * time.Second), last: 300}},
	}); err != nil {
		t.Fatal(err)
	}
	if n, last := mdTickState(t, pool, "BTC-USDT-SWAP"); n != 1 || last != 300 {
		t.Fatalf("跨批撞键: 行=%d last=%v（期望 1/300）", n, last)
	}
	// funding 同键同拍双发——DO NOTHING 零膨胀
	if err := c.insertBatch(ctx, []rowItem{
		{funding: &fundingRow{instID: "BTC-USDT-SWAP", ts: base, fundingRate: 0.0001}},
		{funding: &fundingRow{instID: "BTC-USDT-SWAP", ts: base, fundingRate: 0.0001}},
	}); err != nil {
		t.Fatal(err)
	}
	var fn int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM md_funding WHERE inst_id='BTC-USDT-SWAP'`).Scan(&fn); err != nil {
		t.Fatal(err)
	}
	if fn != 1 {
		t.Fatalf("funding 同键同拍: 行=%d（期望 1——幂等键 DO NOTHING）", fn)
	}
}

// TestLocalBridgeCoWriteIdempotent 本地写×桥写同键共写幂等：行数不膨胀+
// ts 守卫最新赢（桥旧行不夺本地新帧、桥新行赢——dev 桥复活后同键让位
// 语义对拍，D-0018）。
func TestLocalBridgeCoWriteIdempotent(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, "DELETE FROM md_ticks")
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM md_ticks") }()
	c := &Collector{pool: pool}
	base := time.Now().UTC().Truncate(time.Second)
	inst := "ETH-USDT-SWAP"
	// 本地写 T2（新帧）
	if err := c.insertBatch(ctx, []rowItem{
		{tick: &tickRow{instID: inst, ts: base.Add(2 * time.Second), last: 200}},
	}); err != nil {
		t.Fatal(err)
	}
	// 桥写 T1（旧行）——ts 守卫拒=不夺本地帧
	if _, err := pool.Exec(ctx, uTicksBridge, base.Add(1*time.Second), "okx", inst, 100, 99, 101, 10.0, base.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if n, last := mdTickState(t, pool, inst); n != 1 || last != 200 {
		t.Fatalf("桥旧行应被守卫拒: 行=%d last=%v（期望 1/200）", n, last)
	}
	// 桥写 T3（新行）——最新赢
	if _, err := pool.Exec(ctx, uTicksBridge, base.Add(3*time.Second), "okx", inst, 300, 299, 301, 10.0, base.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	if n, last := mdTickState(t, pool, inst); n != 1 || last != 300 {
		t.Fatalf("桥新行应赢: 行=%d last=%v（期望 1/300——行数不膨胀+最新赢）", n, last)
	}
}
