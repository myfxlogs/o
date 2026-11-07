package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0385 接线桩测三件：minerLoop 逐拍调 MineLessons（独立 12h 慢拍槽+
// 拉起即首挖）；败径隔离（假败→主拍不炸+升件自报行落+同窗 24h 去重）。

// TestMinerLoopFirstMineOnStartAndEveryBeat 拉起即首挖（部署即补当前窗
// 非干等 12h——T-0381 首侦同例）；ticker 拍逐拍续挖；ctx 退即净退。
func TestMinerLoopFirstMineOnStartAndEveryBeat(t *testing.T) {
	stub := &stubSensor{mineN: 2}
	r := &Runner{sensor: stub}
	old := minerBeatEvery
	minerBeatEvery = 5 * time.Millisecond // 测试面覆写（生产值钦定不动）
	defer func() { minerBeatEvery = old }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { r.minerLoop(ctx); close(done) }()
	deadline := time.After(2 * time.Second) // sched-exempt-ok 桩测期界等待（测试面拍锚非生产调度）
	for stub.mines.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("MineLessons 未逐拍连调（含拉起首挖）: n=%d", stub.mines.Load())
		case <-time.After(10 * time.Millisecond): // sched-exempt-ok 桩测期界等待（测试面拍锚非生产调度）
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second): // sched-exempt-ok 桩测期界等待（净退界非生产调度）
		t.Fatal("minerLoop 未随 ctx 净退")
	}
}

// TestMinerTickFailIsolatedSelfReported 败径隔离：MineLessons 假败→tick
// 正常返+升件自报行落（写败/LIFECYCLE-MINER）+同窗 24h 去重（T-0369
// 自报率限 hungerEscalate 同构）。
func TestMinerTickFailIsolatedSelfReported(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	stub := &stubSensor{merr: errors.New("假败——扫窗读败")}
	r := &Runner{pool: pool, sensor: stub}
	r.minerTick(ctx)
	const esc = `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='LIFECYCLE-MINER' AND note LIKE 'minertick 败%'`
	var n int
	if err := pool.QueryRow(ctx, esc).Scan(&n); err != nil || n != 1 {
		t.Fatalf("自报行缺: n=%d err=%v", n, err)
	}
	r.minerTick(ctx) // 同窗再败——去重不暴
	if err := pool.QueryRow(ctx, esc).Scan(&n); err != nil || n != 1 {
		t.Fatalf("同窗去重失效: n=%d err=%v", n, err)
	}
}
