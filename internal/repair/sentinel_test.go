package repair

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// O-38640 哨兵桩：md_klines venue='mt5' 收帧/收线双面 + 器官产出账。
// 形谱：①mt5 收帧陈→RECV 升件 ②mt5 确认棒陈（写触碰掩盖）→CLOSE 升件
// ③mt5 双面鲜→静默 ④器官产出陈→ORGAN 升件 ⑤器官产出鲜/名单空→静默
// ⑥systemd 定位形走注入缝。

func sentinelFixture(t *testing.T) (*Sentinel, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`DELETE FROM md_klines`,
		`DELETE FROM app_config WHERE key IN ('ops.cron_manifest','ops.organ_max_age_h')`,
		`DELETE FROM ops_event WHERE esc_scope LIKE 'KLINE-MT5-%' OR esc_scope LIKE 'ORGAN-%'`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	return New(pool, pool, cfg), pool
}

// seedMt5 md_klines venue='mt5' 行（barAge=开点龄 h,recvAge=收帧龄 h,confirm=收线位）。
func seedMt5(t *testing.T, pool *pgxpool.Pool, inst string,
	barAge, recvAge float64, confirm bool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm,recv_ts)
		 VALUES('mt5',$1,'1D',now()-make_interval(hours=>$2),1,1,1,1,1,1,$3,
		 now()-make_interval(hours=>$4))`,
		inst, barAge, confirm, recvAge); err != nil {
		t.Fatal(err)
	}
}

// sentEsc 面级升件行数（esc_scope 键级断言）。
func sentEsc(t *testing.T, pool *pgxpool.Pool, scope string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND esc_scope=$1`, scope).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSentinelMt5RecvStale(t *testing.T) {
	s, pool := sentinelFixture(t)
	ctx := context.Background()
	// 收帧 10h>8h 阈——RECV 升件；确认棒开点 50h（收线 26h<96h）——CLOSE 静默。
	seedMt5(t, pool, "EURUSD", 50, 10, true)
	s.beat(ctx)
	if n := sentEsc(t, pool, "KLINE-MT5-RECV"); n != 1 {
		t.Fatalf("mt5 收帧陈 RECV 应升件, got %d", n)
	}
	if n := sentEsc(t, pool, "KLINE-MT5-CLOSE"); n != 0 {
		t.Fatalf("mt5 收线鲜 CLOSE 不应升件, got %d", n)
	}
}

func TestSentinelMt5CloseStale(t *testing.T) {
	s, pool := sentinelFixture(t)
	ctx := context.Background()
	// 周棒污染形：确认棒开点 220h（收线≈8d>96h）+收帧 1h 前被触碰
	// ——写触碰掩盖下 CLOSE 必须升件（同 O-38640 语义律）。
	seedMt5(t, pool, "EURUSD", 220, 1, true)
	s.beat(ctx)
	if n := sentEsc(t, pool, "KLINE-MT5-CLOSE"); n != 1 {
		t.Fatalf("mt5 收线陈 CLOSE 应升件, got %d", n)
	}
	if n := sentEsc(t, pool, "KLINE-MT5-RECV"); n != 0 {
		t.Fatalf("mt5 收帧鲜 RECV 不应升件, got %d", n)
	}
}

func TestSentinelMt5FreshQuiet(t *testing.T) {
	s, pool := sentinelFixture(t)
	ctx := context.Background()
	// 双面鲜：收帧 1h<8h、确认棒开点 30h（收线 6h 前<96h）——静默。
	seedMt5(t, pool, "EURUSD", 30, 1, true)
	s.beat(ctx)
	if n := sentEsc(t, pool, "KLINE-MT5-RECV") + sentEsc(t, pool, "KLINE-MT5-CLOSE"); n != 0 {
		t.Fatalf("mt5 双面鲜应静默, got %d", n)
	}
}

// setOrganManifest app_config 写名单+重载快照（投影面同键径）。
func setOrganManifest(t *testing.T, s *Sentinel, pool *pgxpool.Pool, manifest string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO app_config(key,value)
		VALUES('ops.cron_manifest',$1::jsonb)
		ON CONFLICT(key) DO UPDATE SET value=$1::jsonb`, manifest); err != nil {
		t.Fatal(err)
	}
	if err := s.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSentinelOrganFileFaces(t *testing.T) {
	s, pool := sentinelFixture(t)
	ctx := context.Background()
	dir := t.TempDir()
	stale := filepath.Join(dir, "stale.txt")
	fresh := filepath.Join(dir, "fresh.txt")
	for _, f := range []string{stale, fresh} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	setOrganManifest(t, s, pool,
		`["oldjob:`+stale+`","newjob:`+fresh+`","badline"]`)
	s.sysLast = func(context.Context, string) (*time.Time, error) { return nil, nil }
	s.beat(ctx)
	if n := sentEsc(t, pool, "ORGAN-OLDJOB"); n != 1 {
		t.Fatalf("器官产出陈 ORGAN-OLDJOB 应升件, got %d", n)
	}
	if n := sentEsc(t, pool, "ORGAN-NEWJOB"); n != 0 {
		t.Fatalf("器官产出鲜 ORGAN-NEWJOB 不应升件, got %d", n)
	}
}

func TestSentinelOrganEmptyManifestQuiet(t *testing.T) {
	s, pool := sentinelFixture(t)
	ctx := context.Background()
	// 名单空=不探（防误报钦定）——无任何 ORGAN 升件。
	setOrganManifest(t, s, pool, `[]`)
	s.beat(ctx)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope LIKE 'ORGAN-%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("空名单不应升件, got %d", n)
	}
}

func TestSentinelOrganSystemdFace(t *testing.T) {
	s, pool := sentinelFixture(t)
	ctx := context.Background()
	// systemd 定位形走注入缝——72h 前触发>36h 阈→升件。
	stale := time.Now().Add(-72 * time.Hour)
	s.sysLast = func(_ context.Context, unit string) (*time.Time, error) {
		if unit != "o-backup-md" {
			t.Fatalf("systemd 单元名错: %s", unit)
		}
		return &stale, nil
	}
	setOrganManifest(t, s, pool, `["o-backup-md:systemd"]`)
	s.beat(ctx)
	if n := sentEsc(t, pool, "ORGAN-O_BACKUP_MD"); n != 1 {
		t.Fatalf("systemd 器官陈应升件, got %d", n)
	}
}

func TestParseOrganEntry(t *testing.T) {
	name, loc, err := parseOrganEntry(" arch_daily : /opt/x/* ")
	if err != nil || name != "arch_daily" || loc != "/opt/x/*" {
		t.Fatalf("解析错: %q %q %v", name, loc, err)
	}
	if _, _, err := parseOrganEntry("noColon"); err == nil {
		t.Fatal("缺冒号应拒")
	}
	if _, _, err := parseOrganEntry(":/x"); err == nil {
		t.Fatal("空名应拒")
	}
	// 未知定位形 face 必报错不静默。
	f := organFace(context.Background(), "x", "bogus", nil)
	if _, err := f(context.Background()); err == nil {
		t.Fatal("未知定位形应读败")
	}
}
