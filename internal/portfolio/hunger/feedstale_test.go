package hunger

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0410 锚①红桩：「feed 陈>阈+零升件行」修前红——md_calendar_push 末行
// sent_at 超阈（24h——feed 本应日更）零新行 → ops_event `feed断更` 行+
// 观测盲升件（esc_scope='CALENDAR-FEED'，24h 同窗去重）。开关关态=feed
// 应停非病灶不探；零推送行史（max=NULL）=断更极端形同报。

func feedFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
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
	return New(pool, cfg), pool
}

// calSwitch 日历推送开关置位（fin_0035 种 "false"——病灶案生产被开，桩按态设；
// cfg=一次性快照——UPDATE 后须 Load 重载新值才可见）。
func calSwitch(t *testing.T, s *Sensor, pool *pgxpool.Pool, on bool) {
	t.Helper()
	v := "false"
	if on {
		v = "true"
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`UPDATE app_config SET value=$1::jsonb WHERE key='notify.feishu.calendar.enabled'`, v); err != nil {
		t.Fatal(err)
	}
	if err := s.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
}

// seedPush md_calendar_push 行定龄（age=推送行龄——>24h 即陈）。
func seedPush(t *testing.T, pool *pgxpool.Pool, ageHours float64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO md_calendar_push(event_key,sent_at) VALUES('k|d|t',now()-make_interval(hours=>$1))`,
		ageHours); err != nil {
		t.Fatal(err)
	}
}

// feedRows 病灶两形行数（feed断更行/升件行——断言公共面）。
func feedRows(t *testing.T, pool *pgxpool.Pool) (lesion, esc int) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='feed断更' AND ticket='T-0410'`).Scan(&lesion); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND esc_scope='CALENDAR-FEED'`).Scan(&esc); err != nil {
		t.Fatal(err)
	}
	return lesion, esc
}

// TestProbeFeedStaleLesionOnStaleRow 形①：开关开+末行龄 30h>阈→病灶行+升件落账。
func TestProbeFeedStaleLesionOnStaleRow(t *testing.T) {
	s, pool := feedFixture(t)
	ctx := context.Background()
	calSwitch(t, s, pool, true)
	seedPush(t, pool, 30)
	if err := s.ProbeFeedStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	lesion, esc := feedRows(t, pool)
	if lesion != 1 || esc != 1 {
		t.Fatalf("病灶行缺失: feed断更=%d 升件=%d", lesion, esc)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_scope='CALENDAR-FEED'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "断更") || !strings.Contains(note, "sent_at") {
		t.Fatalf("升件 note 缺断更/sent_at 锚: %s", note)
	}
}

// TestProbeFeedStaleQuietOnFreshRow 形②：末行鲜=零行（feed 日更在拍非病灶）。
func TestProbeFeedStaleQuietOnFreshRow(t *testing.T) {
	s, pool := feedFixture(t)
	ctx := context.Background()
	calSwitch(t, s, pool, true)
	seedPush(t, pool, 2)
	if err := s.ProbeFeedStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if lesion, esc := feedRows(t, pool); lesion != 0 || esc != 0 {
		t.Fatalf("鲜行误报: feed断更=%d 升件=%d", lesion, esc)
	}
}

// TestProbeFeedStaleSkipOnSwitchOff 形③：开关关=feed 应停非病灶——陈行不探。
func TestProbeFeedStaleSkipOnSwitchOff(t *testing.T) {
	s, pool := feedFixture(t)
	ctx := context.Background()
	calSwitch(t, s, pool, false) // fin_0035 种缺省同——显式钉形
	seedPush(t, pool, 30)
	if err := s.ProbeFeedStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if lesion, esc := feedRows(t, pool); lesion != 0 || esc != 0 {
		t.Fatalf("关态误报: feed断更=%d 升件=%d", lesion, esc)
	}
}

// TestProbeFeedStaleEmptyTable 形④：开关开+零推送行史=断更极端形同报
// （2026-11-06 病灶案=开关开而两日零行——零行史同语义）。
func TestProbeFeedStaleEmptyTable(t *testing.T) {
	s, pool := feedFixture(t)
	ctx := context.Background()
	calSwitch(t, s, pool, true)
	if err := s.ProbeFeedStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	lesion, esc := feedRows(t, pool)
	if lesion != 1 || esc != 1 {
		t.Fatalf("零行史病灶缺失: feed断更=%d 升件=%d", lesion, esc)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='feed断更'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "零推送行史") {
		t.Fatalf("零行史 note 未显形: %s", note)
	}
}

// TestProbeFeedStaleDedup24h 同窗去重：病灶在而再拍——24h 内至多一报。
func TestProbeFeedStaleDedup24h(t *testing.T) {
	s, pool := feedFixture(t)
	ctx := context.Background()
	calSwitch(t, s, pool, true)
	seedPush(t, pool, 30)
	for i := 0; i < 2; i++ {
		if err := s.ProbeFeedStale(ctx); err != nil {
			t.Fatalf("probe %d: %v", i, err)
		}
	}
	if lesion, esc := feedRows(t, pool); lesion != 1 || esc != 1 {
		t.Fatalf("同窗去重失效: feed断更=%d 升件=%d", lesion, esc)
	}
}
