// events_test.go —— 读饥饿钉死根修桩锚（T-0154 返修判据①，pool-tx-residue
// 姊妹病灶）：fn 长拍阻塞期间灌通知→专读环持续排干套接字（通知方交付零
// ClientWrite 反压、listener 会话 xmin 零钉）+积压塌陷=一醒一拍。
package events

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"o/internal/platform/testutil"
)

// 锚：fn 阻塞 ~30s 期间灌通知（7KB payload×15ms 节拍，足量溢满内核缓冲）——
// ①通知方 Exec 全程不卡（套接字被专读环持续排空；旧码此形=服务端交付隐式
// 事务卡 ClientWrite→发射超时实锤）；②listener 会话 backend_xmin 全程零钉、
// state=idle/wait_event=ClientRead（排空指纹）；③放行后积压塌陷——额外恰
// 一拍（一醒一拍，非每通知一拍）；④cancel 收摊不悬挂、还池干净。
func TestListenSocketStarvationRootFix(t *testing.T) {
	pool := testutil.Pool(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := fmt.Sprintf("ev_starve_%d", time.Now().UnixNano())

	notifier, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer notifier.Release()

	gate := make(chan struct{})
	var ticks atomic.Int64
	fn := func(context.Context) error {
		if ticks.Add(1) == 1 {
			<-gate // 首拍长拍——模拟 20min+ 扫拍卡读环实害形（本测 30s 量级）
		}
		return nil
	}
	done := make(chan error, 1)
	go func() { done <- listenOnce(ctx, pool, []string{ch}, fn) }()

	// 等 LISTEN 注册落位（pg_stat_activity query 列=末语句 LISTEN ch）。
	var pid int32
	for t0 := time.Now(); ; {
		err := notifier.QueryRow(ctx,
			`SELECT pid FROM pg_stat_activity WHERE query = $1`, "LISTEN "+ch).Scan(&pid)
		if err == nil {
			break
		}
		if time.Since(t0) > 10*time.Second {
			t.Fatalf("listener 注册 10s 未落位: %v", err)
		}
		time.Sleep(50 * time.Millisecond) // sched-exempt-ok 桩测注册轮询（夹具非生产调度）
	}

	// 灌通知（独立连接独占——与采样连接分离防并发用连）。
	payload := strings.Repeat("x", 7000)
	var sent, maxLatencyMs atomic.Int64
	floodCtx, floodStop := context.WithCancel(ctx)
	floodErr := make(chan error, 1)
	go func() {
		tick := time.NewTicker(15 * time.Millisecond) // sched-exempt-ok 桩测灌装节拍（夹具非生产调度）
		defer tick.Stop()
		for {
			select {
			case <-floodCtx.Done():
				floodErr <- nil
				return
			case <-tick.C:
			}
			t0 := time.Now()
			ectx, ecancel := context.WithTimeout(floodCtx, 5*time.Second)
			_, err := notifier.Exec(ectx, "SELECT pg_notify($1, $2)", ch, payload)
			ecancel()
			if err != nil {
				floodErr <- fmt.Errorf("通知发射被反压（套接字未被排空→服务端交付 ClientWrite 钉死实锤）: %w", err)
				return
			}
			if d := time.Since(t0).Milliseconds(); d > maxLatencyMs.Load() {
				maxLatencyMs.Store(d)
			}
			sent.Add(1)
		}
	}()

	// 首拍起（首通知唤醒 fn 卡 gate）。
	for t0 := time.Now(); ticks.Load() == 0; {
		if time.Since(t0) > 5*time.Second {
			t.Fatal("首通知 5s 未唤醒 fn——注册/唤醒链断")
		}
		time.Sleep(20 * time.Millisecond) // sched-exempt-ok 桩测轮询采样（夹具非生产调度）
	}

	// 长拍期间三采样：xmin 零钉+idle 全程；末采样加排空指纹（ClientRead）。
	sampler, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer sampler.Release()
	for i, gap := range []time.Duration{10 * time.Second, 10 * time.Second, 9 * time.Second} {
		time.Sleep(gap) // sched-exempt-ok 桩测采样间距（夹具非生产调度）
		var state, waitEvent string
		var xminNull bool
		if err := sampler.QueryRow(ctx, `SELECT state, coalesce(wait_event,''), backend_xmin IS NULL
			FROM pg_stat_activity WHERE pid = $1`, pid).Scan(&state, &waitEvent, &xminNull); err != nil {
			t.Fatalf("采样 %d: %v", i, err)
		}
		if !xminNull {
			t.Fatalf("采样 %d：listener 会话 backend_xmin 非空——长拍期间事务钉死（根修失效）", i)
		}
		if state != "idle" {
			t.Fatalf("采样 %d：listener 会话 state=%q 非 idle——套接字未被排空", i, state)
		}
		if i == 2 { // 排空指纹：服务端闲等客户端输入（ClientWrite=交付卡死反证）
			for t0 := time.Now(); waitEvent != "ClientRead"; {
				if time.Since(t0) > 3*time.Second {
					t.Fatalf("采样 2：wait_event=%q 非 ClientRead——交付排空指纹未落", waitEvent)
				}
				time.Sleep(100 * time.Millisecond) // sched-exempt-ok 桩测指纹复采（夹具非生产调度）
				if err := sampler.QueryRow(ctx, `SELECT coalesce(wait_event,'')
					FROM pg_stat_activity WHERE pid = $1`, pid).Scan(&waitEvent); err != nil {
					t.Fatalf("指纹复采: %v", err)
				}
			}
		}
	}

	// 停灌→排残→放行长拍：积压塌陷=额外恰一拍。
	floodStop()
	if err := <-floodErr; err != nil {
		t.Fatal(err)
	}
	if n := sent.Load(); n < 500 {
		t.Fatalf("灌通知量 %d 不足 500——灌装面失效，锚不成立", n)
	}
	if ms := maxLatencyMs.Load(); ms >= 5000 {
		t.Fatalf("通知发射最大延时 %dms——反压残留", ms)
	}
	time.Sleep(2 * time.Second) // sched-exempt-ok 桩测排残等待（夹具非生产调度；读速≫灌速余量足）
	close(gate)
	for t0 := time.Now(); ticks.Load() < 2; {
		if time.Since(t0) > 5*time.Second {
			t.Fatalf("放行 5s 未收到塌陷补拍（ticks=%d）", ticks.Load())
		}
		time.Sleep(20 * time.Millisecond) // sched-exempt-ok 桩测轮询采样（夹具非生产调度）
	}
	time.Sleep(2 * time.Second) // sched-exempt-ok 桩测塌陷静置复核（夹具非生产调度）
	if got := ticks.Load(); got != 2 {
		t.Fatalf("「一醒一拍」破——积压 %d 通知塌陷后应恰 2 拍（首拍+补拍），got %d", sent.Load(), got)
	}

	// 收摊不悬挂+还池干净。
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second): // sched-exempt-ok 桩测收摊悬挂看门狗（夹具非生产调度）
		t.Fatal("cancel 5s listenOnce 未返回——teardown 悬挂（读环未等退出）")
	}
	chk, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer chk.Release()
	if st := chk.Conn().PgConn().TxStatus(); st != 'I' {
		t.Fatalf("收摊后池连接应干净 idle got %q", st)
	}
}
