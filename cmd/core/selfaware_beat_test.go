package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestPulseState 常驻器官三态判——读败=?/静阈内=动/僵阈内=静/越僵阈=僵。
func TestPulseState(t *testing.T) {
	now := time.Now()
	at := func(d time.Duration) *time.Time { t := now.Add(-d); return &t }
	cases := []struct {
		ts   *time.Time
		err  error
		want string
	}{
		{at(10 * time.Second), nil, "动"},
		{at(2 * time.Minute), nil, "静"},
		{at(10 * time.Minute), nil, "僵"},
		{nil, nil, "?"},
		{nil, errors.New("x"), "?"},
	}
	for i, c := range cases {
		if got := pulseState(c.ts, c.err, time.Minute, 5*time.Minute); got != c.want {
			t.Fatalf("case %d: got %s want %s", i, got, c.want)
		}
	}
}

// TestSpawnlessOrphanGrace 唤起无果在飞宽限（87帧常开实证根修）：
// 新唤<15min未配对=在飞不计；>15min未配对=孤儿照计；配对收口=不计。
func TestSpawnlessOrphanGrace(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE event IN ('唤起','唤起果','唤起败','会话收口') AND ts>now()-interval '2 hours'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	spawn := func(unit string, age time.Duration) {
		// seat=审计机（唤知机×唤起=迁移外直插产线行，测库矩阵只载种子）；
		// 判据不滤 seat——同闸内组合即合法。
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note,ts)
			VALUES('审计机','唤起',$1,now()-$2*interval '1 second')`,
			"unit="+unit+" trigger=t", age.Seconds()); err != nil {
			t.Fatalf("spawn %s: %v", unit, err)
		}
	}
	count := func() int {
		n, err := svc.spawnlessOrphans(ctx)
		if err != nil {
			t.Fatalf("spawnlessOrphans: %v", err)
		}
		return n
	}
	spawn("t-fresh", 5*time.Second) // 在飞——宽限内不计
	if got := count(); got != 0 {
		t.Fatalf("在飞唤误计 got %d want 0", got)
	}
	spawn("t-orphan", 20*time.Minute) // 越宽限未配对=孤儿——计
	spawn("t-paired", 20*time.Minute) // 越宽限有配对——不计
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('唤知','会话收口','unit=t-paired 收口')`); err != nil {
		t.Fatalf("close: %v", err)
	}
	if got := count(); got != 1 {
		t.Fatalf("孤儿唤漏计 got %d want 1", got)
	}
}

// TestTruncRunes 摘要截断——多字节不截半字。
func TestTruncRunes(t *testing.T) {
	if got := truncRunes("工单#1 认领:开修——修码", 6); got != "工单#1 认" {
		t.Fatalf("截断 %q", got)
	}
	if got := truncRunes("短", 9); got != "短" {
		t.Fatalf("短串 %q", got)
	}
}
