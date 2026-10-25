// Package events —— 事件脊（T-0048：周期扫描环全拆事件化）。PG LISTEN/NOTIFY
// 为跨进程真事件脊（同事务同 channel+payload 天然去重=批量 flush 一拍一醒）。
// 每环允许「期界型活性兜底」（liveness 心跳——失联自检非工作扫描，注释钉死）。
package events

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// livenessMax 活性兜底窗（事件失联>10min→一拍自检——健康探针非工作扫描）。
const livenessMax = 10 * time.Minute

// ListenLoop 订阅 channels→每事件 fn()；fn 同步串行执行（事件即节拍——慢一拍
// 后续通知在通知队列里堆积但 select 一醒一次，天然背压）。断线=重连重听。
// boot 首拍先跑一次（事件化≠丢首拍——起步态须一次对齐）。
func ListenLoop(ctx context.Context, pool *pgxpool.Pool, name string, channels []string, fn func(context.Context) error) {
	if err := fn(ctx); err != nil {
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

func listenOnce(ctx context.Context, pool *pgxpool.Pool, channels []string, fn func(context.Context) error) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	// LISTEN 会话级常驻——还池前 UNLISTEN 全清（T-0131 根修：池连接频道残留
	// 污染下任借用者=自激洪泛源；ctx 已取消时用 Background 兜底执行）。
	defer func() {
		_, _ = conn.Exec(context.Background(), "UNLISTEN *")
		conn.Release()
	}()
	for _, ch := range channels {
		if _, err := conn.Exec(ctx, "LISTEN "+ch); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		_ = n
		if err := fn(ctx); err != nil {
			log.Printf("事件拍 fn 败: %v", err)
		}
	}
}

// ListenOrDeadline 订阅 channels 或期界（review 判决到期语义——期界等待=钦定豁免；
// fn 每事件/到期各一拍；fallback=期界空时 livenessMax 自检）。
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
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer func() { // 同 listenOnce——频道残留禁带还池（T-0131）
		_, _ = conn.Exec(context.Background(), "UNLISTEN *")
		conn.Release()
	}()
	for _, ch := range channels {
		if _, err := conn.Exec(ctx, "LISTEN "+ch); err != nil {
			return err
		}
	}
	for {
		d := nextDue(ctx)
		if d <= 0 {
			d = livenessMax // 期界空=活性兜底自检（失联探针非工作扫描）
		}
		ctx2, cancel := context.WithTimeout(ctx, d)
		n, err := conn.Conn().WaitForNotification(ctx2)
		cancel()
		if err != nil {
			if ctx2.Err() != nil && ctx.Err() == nil {
				return nil // 期界到/兜底醒=正常一拍
			}
			return err
		}
		_ = n
		if err := fn(ctx); err != nil {
			log.Printf("事件拍 fn 败: %v", err)
		}
	}
}
