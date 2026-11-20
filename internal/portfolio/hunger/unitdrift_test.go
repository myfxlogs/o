package hunger

// unitdrift_test.go —— 建制单元差分探针锚（工单#26444）：
// ①注册表缺→unit_registry_missing fail-closed；
// ②beat陈/缺→unit_watch_dead；
// ③单态异→unit_state_drift+dedup；配态→静默；
// ④产出行陈→<unit>_heartbeat_stale；鲜行→静默。

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const unitDriftRegJSON = `{
  "host": "models",
  "watch": {"period_sec": 300, "beat_ticket": "UNITW",
            "fact_seat": "建制机", "fact_event": "单态"},
  "units": [
    {"unit": "o-self.timer", "expect_enabled": "enabled", "expect_active": "active",
     "prod_seat": "自我", "prod_event": "判词", "prod_period_min": 60},
    {"unit": "o-x.timer", "expect_enabled": "enabled"}
  ]
}`

func seedUnitReg(t *testing.T, s *Sensor, pool *pgxpool.Pool, regJSON string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO app_config(key,value) VALUES('ops.unit_registry',$1::jsonb)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, regJSON); err != nil {
		t.Fatal(err)
	}
	// cfg 快照径（Load 重灌——Watch LISTEN 测试面不起）。
	if err := s.cfg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func seedUnitBeat(t *testing.T, pool *pgxpool.Pool, ageMin int) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ops_headless(ticket,seat,state,last_activity)
		 VALUES('UNITW','建制机','alive',now()-make_interval(mins=>$1))
		 ON CONFLICT (ticket) DO UPDATE SET last_activity=EXCLUDED.last_activity`, ageMin); err != nil {
		t.Fatal(err)
	}
}

func unitEscCount(t *testing.T, pool *pgxpool.Pool, scope string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ops_event WHERE event='升件' AND esc_scope=$1`,
		scope).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// ①注册表空（units[] 空——fin_0222 种行在案径下空表=fail-closed 同义）。
func TestUnitDriftRegistryMissing(t *testing.T) {
	s, pool := droughtFixture(t)
	seedUnitReg(t, s, pool, `{"host":"models","units":[]}`)
	if err := s.ProbeUnitDrift(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := unitEscCount(t, pool, unitRegistryMissScope); n != 1 {
		t.Fatalf("注册表空未 fail-closed 升件 n=%d", n)
	}
}

// ②beat 缺→unit_watch_dead（判一盲面全开一等事件）。
func TestUnitDriftWatchDead(t *testing.T) {
	s, pool := droughtFixture(t)
	seedUnitReg(t, s, pool, unitDriftRegJSON)
	if err := s.ProbeUnitDrift(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := unitEscCount(t, pool, unitWatchDeadScope); n != 1 {
		t.Fatalf("beat缺未升件 n=%d", n)
	}
	// dedup 二拍零新行。
	if err := s.ProbeUnitDrift(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := unitEscCount(t, pool, unitWatchDeadScope); n != 1 {
		t.Fatalf("二拍重复升件 n=%d", n)
	}
}

// ③a 单态异→unit_state_drift（disable 实测 vs enabled 期望——
// o-self.timer 躺尸实证形）。
func TestUnitDriftStateDrift(t *testing.T) {
	s, pool := droughtFixture(t)
	seedUnitReg(t, s, pool, unitDriftRegJSON)
	seedUnitBeat(t, pool, 2)
	seedOpsEvent(t, pool, 10, "建制机", "单态",
		"unit=o-self.timer enabled=disabled active=inactive want_en=enabled want_act=active drift=1")
	if err := s.ProbeUnitDrift(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := unitEscCount(t, pool, unitDriftEscScope); n != 1 {
		t.Fatalf("态漂移未升件 n=%d", n)
	}
	var note string
	if err := pool.QueryRow(context.Background(),
		`SELECT note FROM ops_event WHERE event='升件' AND esc_scope=$1`,
		unitDriftEscScope).Scan(&note); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"o-self.timer", "enabled=disabled(期望enabled)"} {
		if !strings.Contains(note, want) {
			t.Fatalf("归因段缺 %q —— note=%s", want, note)
		}
	}
}

// ③b 配态/复常→静默（末行态合期望=零写契覆盖面）。
func TestUnitDriftSilent(t *testing.T) {
	s, pool := droughtFixture(t)
	seedUnitReg(t, s, pool, unitDriftRegJSON)
	seedUnitBeat(t, pool, 2)
	seedOpsEvent(t, pool, 10, "建制机", "单态",
		"unit=o-self.timer enabled=enabled active=active want_en=enabled want_act=active drift=0")
	seedOpsEvent(t, pool, 30, "自我", "判词", "第N拍判词")
	if err := s.ProbeUnitDrift(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, sc := range []string{unitDriftEscScope, unitWatchDeadScope,
		"o-self.timer_heartbeat_stale"} {
		if n := unitEscCount(t, pool, sc); n != 0 {
			t.Fatalf("配态误升件 scope=%s n=%d", sc, n)
		}
	}
}

// ④产出行陈→heartbeat_stale（单元活着拍子不在产面）。
func TestUnitDriftHeartbeatStale(t *testing.T) {
	s, pool := droughtFixture(t)
	seedUnitReg(t, s, pool, unitDriftRegJSON)
	seedUnitBeat(t, pool, 2)
	// 判词产出行阈=180min——陈 400min 必升。
	seedOpsEvent(t, pool, 400, "自我", "判词", "陈旧判词行")
	if err := s.ProbeUnitDrift(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := unitEscCount(t, pool, "o-self.timer_heartbeat_stale"); n != 1 {
		t.Fatalf("产出行陈未升件 n=%d", n)
	}
}
