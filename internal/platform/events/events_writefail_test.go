// events_writefail_test.go —— 工单#22873 锚：事件拍 fn 败→writefail
// 升件行（esc_class='写败'、esc_scope=EVENT-BEAT/<name> 24h 同窗去重）——
// 42P10 类写拒连刷 18h 升件零发静默面的机械对治实证。
package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/platform/writefail"
)

// sigPool writefail.Pool 桩——Exec 透传真池+首调关 called 信号位
// （自报语句执行毕=行已落，同步可读断言零轮询）。
type sigPool struct {
	inner     *pgxpool.Pool
	called    chan struct{}
	closeOnce sync.Once
}

func (s *sigPool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	tag, err := s.inner.Exec(ctx, sql, arguments...)
	s.closeOnce.Do(func() { close(s.called) })
	return tag, err
}

func TestListenLoopChanBeatFailEscalates(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sp := &sigPool{inner: pool, called: make(chan struct{})}
	writefail.SetPool(sp)
	defer writefail.SetPool(nil)

	name := fmt.Sprintf("test-beat-%d", time.Now().UnixNano())
	fn := func(context.Context, string) error { return errors.New("injected write fail") }
	done := make(chan struct{})
	go func() {
		ListenLoopChan(ctx, pool, name, []string{"ev_never_" + name}, fn)
		close(done)
	}()

	select {
	case <-sp.called: // 自报 Exec 毕=升件行已落（同步可读断言）
	case <-time.After(10 * time.Second): // sched-exempt-ok 测试收束期界——等 boot 拍自报异步落账上界，非周期调度
		t.Fatal("拍败自报升件未发（writefail 径断）")
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='写败' AND esc_scope=$1`,
		"EVENT-BEAT/"+name).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("升件行数=%d want 1（同窗去重库侧封顶——>1 即双发病灶）", n)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second): // sched-exempt-ok 测试收束期界——等环收摊上界
		t.Fatal("ListenLoopChan 收摊悬挂")
	}
}

func TestListenLoopChanBeatFailNoPoolSilent(t *testing.T) {
	// 池未装配=守位不报（core 等他进程天然免疫——本包测试防回归）。
	writefail.SetPool(nil)
	pool := testutil.Pool(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	name := fmt.Sprintf("test-beat-nopool-%d", time.Now().UnixNano())
	fnCalled := make(chan struct{})
	var fnOnce sync.Once
	fn := func(context.Context, string) error {
		fnOnce.Do(func() { close(fnCalled) })
		return errors.New("x")
	}
	done := make(chan struct{})
	go func() {
		ListenLoopChan(ctx, pool, name, []string{"ev_never_" + name}, fn)
		close(done)
	}()
	select {
	case <-fnCalled: // boot 拍已跑——fn 返后 Report=nil 池瞬时 no-op（无可写面）
	case <-time.After(10 * time.Second): // sched-exempt-ok 测试收束期界——等 boot 拍上界
		t.Fatal("boot 拍未跑")
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope=$1`, "EVENT-BEAT/"+name).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("未装配池仍落升件行 n=%d", n)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second): // sched-exempt-ok 测试收束期界——等环收摊上界
		t.Fatal("ListenLoopChan 收摊悬挂")
	}
}
