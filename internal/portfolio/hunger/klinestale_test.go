package hunger

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 工单#13065 切片②桩：日K死闸双面——RECV=收帧龄闸（WS 臂活性）/
// CLOSE=收线龄闸（confirm 语义活性,到期门控防冷启误报）。
// 形谱:①零行史→RECV 升件 ②收帧鲜+无到期 bar→双静默(冷启窗)
// ③收帧陈>30m→RECV 升件 ④到期 bar+零收线→CLOSE 升件
// ⑤到期 bar+收线鲜→静默。

func klineFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM md_klines`); err != nil {
		t.Fatal(err)
	}
	return New(pool, cfg), pool
}

// seedKline md_klines 行（barAge=开点龄 h,recvAge=收帧龄 h,confirm=收线位）。
func seedKline(t *testing.T, pool *pgxpool.Pool, inst string,
	barAge, recvAge float64, confirm bool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm,recv_ts)
		 VALUES('okx',$1,'1D',now()-make_interval(hours=>$2),1,1,1,1,1,1,$3,
		 now()-make_interval(hours=>$4))`,
		inst, barAge, confirm, recvAge); err != nil {
		t.Fatal(err)
	}
}

// klineEsc 面级升件行数（esc_scope 键级断言）。
func klineEsc(t *testing.T, pool *pgxpool.Pool, scope string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND esc_scope=$1`, scope).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestProbeKlineStaleZeroHistory(t *testing.T) {
	s, pool := klineFixture(t)
	ctx := context.Background()
	if err := s.ProbeKlineStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if n := klineEsc(t, pool, "KLINE-DAILY-RECV"); n != 1 {
		t.Fatalf("零行史 RECV 应升件, got %d", n)
	}
}

func TestProbeKlineStaleColdStartQuiet(t *testing.T) {
	s, pool := klineFixture(t)
	ctx := context.Background()
	// 冷启窗：收帧鲜（recv 1m）+开点龄 5h<25h 未到期——双面皆静默。
	seedKline(t, pool, "BTC-USDT-SWAP", 5, 0.02, false)
	if err := s.ProbeKlineStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if n := klineEsc(t, pool, "KLINE-DAILY-RECV") + klineEsc(t, pool, "KLINE-DAILY-CLOSE"); n != 0 {
		t.Fatalf("冷启窗应静默, got %d", n)
	}
}

func TestProbeKlineStaleRecvStale(t *testing.T) {
	s, pool := klineFixture(t)
	ctx := context.Background()
	// 收帧龄 1h>30m 阈——RECV 升件（开点 5h 未到期,CLOSE 面仍静默）。
	seedKline(t, pool, "BTC-USDT-SWAP", 5, 1, false)
	if err := s.ProbeKlineStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if n := klineEsc(t, pool, "KLINE-DAILY-RECV"); n != 1 {
		t.Fatalf("收帧陈 RECV 应升件, got %d", n)
	}
	if n := klineEsc(t, pool, "KLINE-DAILY-CLOSE"); n != 0 {
		t.Fatalf("未到期 CLOSE 不应升件, got %d", n)
	}
}

func TestProbeKlineStaleCloseMissing(t *testing.T) {
	s, pool := klineFixture(t)
	ctx := context.Background()
	// 到期 bar(开点 30h)+收帧鲜但零收线行——CLOSE 升件
	// (收帧鲜=臂活,RECV 静默;confirm 永零=收线语义死)。
	seedKline(t, pool, "BTC-USDT-SWAP", 30, 0.02, false)
	if err := s.ProbeKlineStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if n := klineEsc(t, pool, "KLINE-DAILY-RECV"); n != 0 {
		t.Fatalf("收帧鲜 RECV 不应升件, got %d", n)
	}
	if n := klineEsc(t, pool, "KLINE-DAILY-CLOSE"); n != 1 {
		t.Fatalf("到期零收线 CLOSE 应升件, got %d", n)
	}
}

func TestProbeKlineStaleCloseFreshQuiet(t *testing.T) {
	s, pool := klineFixture(t)
	ctx := context.Background()
	// 到期 bar+收线行鲜(recv 2h<30h)——静默。
	seedKline(t, pool, "BTC-USDT-SWAP", 30, 0.02, false)
	seedKline(t, pool, "ETH-USDT-SWAP", 26, 2, true)
	if err := s.ProbeKlineStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if n := klineEsc(t, pool, "KLINE-DAILY-RECV") + klineEsc(t, pool, "KLINE-DAILY-CLOSE"); n != 0 {
		t.Fatalf("收线鲜应静默, got %d", n)
	}
}
