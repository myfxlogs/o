package hunger

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// 工单#25803 锚：「DEPLOY 心跳末行龄>45min(3×拍距)→病灶行+观测盲升件
// esc_scope=deploy_watch_silent」——timer死/服务崩/锁卡死=心跳停三因同覆；
// 零心跳行史=部署前史/写径全断同报；鲜行=在拍零病灶；同窗 24h 去重。
// 判/去重/升件归 selfprobe 公共模子（feedstale_test 同款四态）。

// seedDeployHeartbeat DEPLOY 心跳行定龄（age=末行龄——>45min 即病灶）。
func seedDeployHeartbeat(t *testing.T, pool *pgxpool.Pool, ageMin float64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ops_event(seat,event,ticket,note,ts)
		 VALUES('DEPLOY','心跳','','心跳 已同步 sha=abc123 base=def456',
		        now()-make_interval(mins=>$1))`, ageMin); err != nil {
		t.Fatal(err)
	}
}

// dwRows 看门狗两形行数（feed断更病灶行/升件行——断言公共面）。
func dwRows(t *testing.T, pool *pgxpool.Pool) (lesion, esc int) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='feed断更' AND ticket='#25803'`).Scan(&lesion); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND esc_scope='deploy_watch_silent'`).Scan(&esc); err != nil {
		t.Fatal(err)
	}
	return lesion, esc
}

// TestProbeDeployWatchLesionOnStaleHeartbeat 形①：末行龄 90min>45min→病灶行+升件落账。
func TestProbeDeployWatchLesionOnStaleHeartbeat(t *testing.T) {
	s, pool := feedFixture(t)
	ctx := context.Background()
	seedDeployHeartbeat(t, pool, 90)
	if err := s.ProbeDeployWatch(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	lesion, esc := dwRows(t, pool)
	if lesion != 1 || esc != 1 {
		t.Fatalf("病灶行缺失: feed断更=%d 升件=%d", lesion, esc)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_scope='deploy_watch_silent'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "静默") || !strings.Contains(note, "45min") {
		t.Fatalf("升件 note 缺静默/阈锚: %s", note)
	}
}

// TestProbeDeployWatchQuietOnFreshHeartbeat 形②：末行鲜=零行（15min 拍在案非病灶）。
func TestProbeDeployWatchQuietOnFreshHeartbeat(t *testing.T) {
	s, pool := feedFixture(t)
	ctx := context.Background()
	seedDeployHeartbeat(t, pool, 10)
	if err := s.ProbeDeployWatch(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if lesion, esc := dwRows(t, pool); lesion != 0 || esc != 0 {
		t.Fatalf("鲜行误报: feed断更=%d 升件=%d", lesion, esc)
	}
}

// TestProbeDeployWatchEdgeAtThreshold 形②b：末行龄恰 44min<45min=鲜（阈界不误伤）。
func TestProbeDeployWatchEdgeAtThreshold(t *testing.T) {
	s, pool := feedFixture(t)
	ctx := context.Background()
	seedDeployHeartbeat(t, pool, 44)
	if err := s.ProbeDeployWatch(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if lesion, esc := dwRows(t, pool); lesion != 0 || esc != 0 {
		t.Fatalf("阈界误报: feed断更=%d 升件=%d", lesion, esc)
	}
}

// TestProbeDeployWatchEmptyHistory 形③：零心跳行史=静默极端形同报
// （timer 从未跑成/写径全断——与断更同语义）。
func TestProbeDeployWatchEmptyHistory(t *testing.T) {
	s, pool := feedFixture(t)
	ctx := context.Background()
	if err := s.ProbeDeployWatch(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	lesion, esc := dwRows(t, pool)
	if lesion != 1 || esc != 1 {
		t.Fatalf("零行史病灶缺失: feed断更=%d 升件=%d", lesion, esc)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='feed断更'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "零心跳行史") {
		t.Fatalf("零行史 note 未显形: %s", note)
	}
}

// TestProbeDeployWatchDedup24h 同窗去重：病灶在而再拍——24h 内至多一报。
func TestProbeDeployWatchDedup24h(t *testing.T) {
	s, pool := feedFixture(t)
	ctx := context.Background()
	seedDeployHeartbeat(t, pool, 90)
	for i := 0; i < 2; i++ {
		if err := s.ProbeDeployWatch(ctx); err != nil {
			t.Fatalf("probe %d: %v", i, err)
		}
	}
	if lesion, esc := dwRows(t, pool); lesion != 1 || esc != 1 {
		t.Fatalf("同窗去重失效: feed断更=%d 升件=%d", lesion, esc)
	}
}
