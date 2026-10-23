package roster

import (
	"context"
	"testing"
	"time"

	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

func TestRegisterBeat(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM fin_employee_registry WHERE name='test-emp'"); err != nil {
		t.Fatal(err)
	}
	if err := Register(ctx, pool, "test-emp", "platform", "测试员工", nil); err != nil {
		t.Fatalf("register: %v", err)
	}
	var status string
	var beat1 time.Time
	if err := pool.QueryRow(ctx,
		"SELECT status, last_beat FROM fin_employee_registry WHERE name='test-emp'").Scan(&status, &beat1); err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Fatalf("status=%s", status)
	}
	time.Sleep(1100 * time.Millisecond) // timestamptz 秒级精度外余量
	if err := Beat(ctx, pool, "test-emp"); err != nil {
		t.Fatalf("beat: %v", err)
	}
	var beat2 time.Time
	if err := pool.QueryRow(ctx,
		"SELECT last_beat FROM fin_employee_registry WHERE name='test-emp'").Scan(&beat2); err != nil {
		t.Fatal(err)
	}
	if !beat2.After(beat1) {
		t.Fatalf("beat 未推进: %v -> %v", beat1, beat2)
	}
	if err := Beat(ctx, pool, "ghost-emp"); err == nil {
		t.Fatal("未注册行 Beat 应 fail-closed")
	}
	// 再注册=幂等
	if err := Register(ctx, pool, "test-emp", "platform", "测试员工", nil); err != nil {
		t.Fatalf("re-register: %v", err)
	}
}
