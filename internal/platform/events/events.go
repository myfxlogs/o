// Package events —— 事件脊（T-0048：周期扫描环全拆事件化）。PG LISTEN/NOTIFY
// 为跨进程真事件脊（同事务同 channel+payload 天然去重=批量 flush 一拍一醒）。
// 每环允许「期界型活性兜底」（liveness 心跳——失联自检非工作扫描，注释钉死）。
package events

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// livenessMax 活性兜底窗（事件失联>10min→一拍自检——健康探针非工作扫描）。
const livenessMax = 10 * time.Minute

// listenSession 一次听会话：conn+专读环合流面（T-0154 返修判据①解耦件）。
// read 独占套接字排干通知，worker 经 wake（cap1 塌陷）取拍；还池前必经
// teardown 等读环退出（读与还池互斥）。reaped/rerr 只在 worker goroutine
// 触碰——无锁。
type listenSession struct {
	conn    *pgxpool.Conn
	wake    chan string        // 通知合流唤醒位（cap1——积压塌陷一醒；携 channel 名）
	readErr chan error         // 读环败因回传（cap1——断线重连径）
	cancel  context.CancelFunc // 停读环（rctx 派生自环 ctx——收摊中断读）
	reaped  bool
	rerr    error
}

// startListener 借 conn→事务态自净→LISTEN 注册→注册毕断言 idle→起专读环。
func startListener(ctx context.Context, pool *pgxpool.Pool, channels []string) (*listenSession, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	ls := &listenSession{conn: conn, wake: make(chan string, 1), readErr: make(chan error, 1)}
	drop := func(err error) (*listenSession, error) { // 读环未起——UNLISTEN 清频弃还（T-0131）
		_, _ = conn.Exec(context.Background(), "UNLISTEN *")
		conn.Release()
		return nil, err
	}
	// 返修判据②：借到事务态连接（守卫漏网形）→ROLLBACK 清态；清不动=弃连
	// 不 proceed——带病注册=聋哑+xmin 双实害源（首修吞错洞实证堵死）。
	if st := conn.Conn().PgConn().TxStatus(); st != 'I' {
		if _, rbErr := conn.Exec(ctx, "ROLLBACK"); rbErr != nil {
			return drop(fmt.Errorf("events: 借到事务态连接（TxStatus=%q）自净回滚败——弃连: %w", st, rbErr))
		}
		log.Printf("[events] WARN 借到事务态连接已自净回滚（TxStatus=%q）", st)
	}
	for _, ch := range channels {
		if _, err := conn.Exec(ctx, "LISTEN "+ch); err != nil {
			return drop(err)
		}
	}
	// 返修判据③：注册毕断言 idle——autocommit LISTEN 必落 'I'；事务内注册
	// （'T'/'E'）不 COMMIT 永不生效=全聋（T-0154 实害②），拒启走重连。
	if st := conn.Conn().PgConn().TxStatus(); st != 'I' {
		return drop(fmt.Errorf("events: LISTEN 注册毕 TxStatus=%q≠'I'——事务内注册聋哑形拒启", st))
	}
	rctx, cancel := context.WithCancel(ctx)
	ls.cancel = cancel
	go ls.read(rctx)
	return ls, nil
}

// read 专读环——独占套接字持续排干（fn 长拍期间不歇）；通知塌陷进 wake
// （cap1：积压合流一醒）。读败经 readErr 回传 worker（断线重连径）。
func (ls *listenSession) read(ctx context.Context) {
	for {
		n, err := ls.conn.Conn().WaitForNotification(ctx)
		if err != nil {
			ls.readErr <- err
			return
		}
		select {
		case ls.wake <- n.Channel:
		default: // 已有未取唤醒位=塌陷（「一醒一拍」背压；新 channel 名覆盖旧）
		}
	}
}

// teardown 收摊——cancel 停读→等读环退出（读与还池互斥）→UNLISTEN 清频
// （频道残留禁带还池——T-0131；ctx 已取消用 Background 兜底）→还池。返回
// 读环败因（cancel 自停之败由调用方按语义取舍——期界拍=正常 nil）。
func (ls *listenSession) teardown() error {
	ls.cancel()
	if !ls.reaped {
		ls.rerr = <-ls.readErr
		ls.reaped = true
	}
	_, _ = ls.conn.Exec(context.Background(), "UNLISTEN *")
	ls.conn.Release()
	return ls.rerr
}

// ListenLoop 订阅 channels→每事件 fn()。读/worker 解耦（T-0154 返修判据①——
// 读饥饿钉死根修，pool-tx-residue 姊妹病灶）：专读 goroutine 持续排干
// WaitForNotification，fn 长拍不再饿死套接字（否则服务端 notify 交付的隐式
// 事务卡 ClientWrite 收不出→通知方 xmin 钉死 vacuum 水平线）；通知塌陷进
// cap1 唤醒位（积压合流一醒），fn 同步串行一拍——「一醒一拍」背压语义不变。
// 断线=重连重听。boot 首拍先跑一次（事件化≠丢首拍——起步态须一次对齐）。
func ListenLoop(ctx context.Context, pool *pgxpool.Pool, name string, channels []string, fn func(context.Context) error) {
	ListenLoopChan(ctx, pool, name, channels, func(ctx context.Context, _ string) error { return fn(ctx) })
}

// ListenLoopChan ListenLoop 的 channel 感知变体（T-0168 projector——单 loop 多
// channel 按 channel 名分派，免全表族过水印）。塌陷期多 channel 合流：唤醒位
// 只携末到 channel 名——余者由调用方水印/对账径兜（幂等 upsert 语义下无损）。
func ListenLoopChan(ctx context.Context, pool *pgxpool.Pool, name string, channels []string, fn func(context.Context, string) error) {
	if err := fn(ctx, ""); err != nil {
		log.Printf("[%s] boot 首拍: %v", name, err)
	}
	backoff := time.Second
	for {
		err := listenOnce(ctx, pool, channels, fn)
		if ctx.Err() != nil {
			return
		}
		log.Printf("[%s] 事件听断（%.0fs 重连）: %v", name, backoff.Seconds(), err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff): // sched-exempt-ok 事件驱动指数退避（断线重连梯——钦定豁免类）
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func listenOnce(ctx context.Context, pool *pgxpool.Pool, channels []string, fn func(context.Context, string) error) error {
	ls, err := startListener(ctx, pool, channels)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			ls.teardown()
			return ctx.Err()
		case err := <-ls.readErr:
			ls.reaped, ls.rerr = true, err
			return ls.teardown()
		case ch := <-ls.wake:
			if err := fn(ctx, ch); err != nil {
				log.Printf("事件拍 fn 败: %v", err)
			}
		}
	}
}

// ListenOrDeadline 订阅 channels 或期界（review 判决到期语义——期界等待=钦定豁免；
// fn 每事件一拍，期界到=正常听断弃连重连拍（期界拍不跑 fn——T-0048 原语义不动）；
// fallback=期界空时 livenessMax 自检）。读/worker 解耦同 ListenLoop（判据①——
// 期界由 worker 侧 timer 承接，专读环全程不中断、套接字持续排干）。
func ListenOrDeadline(ctx context.Context, pool *pgxpool.Pool, name string, channels []string,
	nextDue func(context.Context) time.Duration, fn func(context.Context) error) {
	if err := fn(ctx); err != nil {
		log.Printf("[%s] boot 首拍: %v", name, err)
	}
	backoff := time.Second
	for {
		err := listenOnceDeadline(ctx, pool, name, channels, nextDue, fn)
		if ctx.Err() != nil {
			return
		}
		log.Printf("[%s] 事件听断（%.0fs 重连）: %v", name, backoff.Seconds(), err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff): // sched-exempt-ok 事件驱动指数退避（断线重连梯——钦定豁免类）
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func listenOnceDeadline(ctx context.Context, pool *pgxpool.Pool, name string, channels []string,
	nextDue func(context.Context) time.Duration, fn func(context.Context) error) error {
	ls, err := startListener(ctx, pool, channels)
	if err != nil {
		return err
	}
	for {
		d := nextDue(ctx)
		if d <= 0 {
			d = livenessMax // 期界空=活性兜底自检（失联探针非工作扫描）
		}
		t := time.NewTimer(d)
		select {
		case <-ctx.Done():
			t.Stop()
			ls.teardown()
			return ctx.Err()
		case err := <-ls.readErr:
			t.Stop()
			ls.reaped, ls.rerr = true, err
			return ls.teardown()
		case <-ls.wake:
			t.Stop()
			if err := fn(ctx); err != nil {
				log.Printf("事件拍 fn 败: %v", err)
			}
		case <-t.C:
			ls.teardown()
			return nil // 期界到/兜底醒=正常一拍（弃连重连——原语义不动）
		}
	}
}
