package main

// T-0352 venue 读败连败升件闸——合成断言（真库独立实例 per-call schema 隔离）：
// ①合成连败跨阈→升件行恰一落 ops_event（venue/败因摘要/连败拍数三要素在行）
// ②日限 1/venue/日（复常清零后再一轮同日不加行）
// ③未跨阈零行+复常清零（ok 后旧计数不延续）
// ④持续阈径（慢拍面等价锚——时长参量测试覆写）。

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func venueFailRows(t *testing.T, pool *pgxpool.Pool, scope string) (int, string) {
	t.Helper()
	var n int
	var note string
	if err := pool.QueryRow(context.Background(), `SELECT count(*),
		COALESCE(max(note),'') FROM ops_event
		WHERE seat='o-core' AND event='升件' AND esc_class='读败' AND esc_scope=$1`, scope).Scan(&n, &note); err != nil {
		t.Fatal(err)
	}
	return n, note
}

// 锚①②：合成连败跨阈恰一升件行（要素全）+日限 1/venue/日。
func TestVenueFailGateEscalateAndDailyCap(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	g := newVenueFailGate(pool)
	why := errors.New("mt5 facts/summary: account 空体 / <nil>")
	for i := 0; i < venueFailThrBeats-1; i++ {
		g.fail(ctx, "mt5", why)
	}
	if n, _ := venueFailRows(t, pool, "mt5"); n != 0 {
		t.Fatalf("未跨阈误升: %d 行", n)
	}
	g.fail(ctx, "mt5", why) // 第 venueFailThrBeats 拍=跨阈
	n, note := venueFailRows(t, pool, "mt5")
	if n != 1 {
		t.Fatalf("跨阈升件行缺席: %d 行", n)
	}
	for _, frag := range []string{"mt5", "连败 10 拍", "account 空体"} {
		if !strings.Contains(note, frag) {
			t.Fatalf("升件行缺要素 %q: %s", frag, note)
		}
	}
	// 日限 1：复常清零后再一轮连败——同日不加行（DB 侧 NOT EXISTS 去重实证）。
	g.ok("mt5")
	for i := 0; i < venueFailThrBeats; i++ {
		g.fail(ctx, "mt5", why)
	}
	if n, _ := venueFailRows(t, pool, "mt5"); n != 1 {
		t.Fatalf("日限破功: %d 行", n)
	}
}

// 锚③：复常清零——ok 后旧连败计数不延续（9 败+ok+1 败≠跨阈）。
func TestVenueFailGateHysteresisReset(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	g := newVenueFailGate(pool)
	why := errors.New("trading 读败: 帧缺+REST 兜底败")
	for i := 0; i < venueFailThrBeats-1; i++ {
		g.fail(ctx, "okx", why)
	}
	g.ok("okx") // 复常清零
	g.fail(ctx, "okx", why)
	if n, _ := venueFailRows(t, pool, "okx"); n != 0 {
		t.Fatalf("复常未清零——旧计数延续误升: %d 行", n)
	}
}

// 锚④：持续阈径（慢拍面等价锚）——连败 2 拍+时长超阈即升，venue 键参数化非 mt5 特例。
func TestVenueFailGateDurationPath(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	old := venueFailThrDur // 测试覆写锚（var 钦定测试面可覆写）
	venueFailThrDur = -time.Second
	defer func() { venueFailThrDur = old }()
	ctx := context.Background()
	g := newVenueFailGate(pool)
	why := errors.New("venue 现读: savings 读败")
	g.fail(ctx, "okx", why) // 1 拍未跨（n<2 持续径不判）
	if n, _ := venueFailRows(t, pool, "okx"); n != 0 {
		t.Fatalf("单拍误升: %d 行", n)
	}
	g.fail(ctx, "okx", why) // 2 拍+时长超阈→升
	if n, note := venueFailRows(t, pool, "okx"); n != 1 || !strings.Contains(note, "okx") {
		t.Fatalf("持续径升件缺席: %d 行 %s", n, note)
	}
}
