// Package config —— 配置读写唯一入口（D-0005 严格热启）。
// 契约：LISTEN config_changed + 代次列 + 断线重连=快照重载；消费方按次读或订阅，
// 禁 boot 一次性捕获；读错=fail-closed（未就绪/缺键/解析错一律 error）。
package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotReady = errors.New("config: 快照未就绪（boot 未完成 Load）")

type Client struct {
	pool *pgxpool.Pool

	mu   sync.RWMutex
	snap map[string]json.RawMessage
	gen  int64 // 快照代次=全表 generation 上界（变化即重载判据）
	ok   bool
}

func New(pool *pgxpool.Pool) *Client { return &Client{pool: pool, snap: map[string]json.RawMessage{}} }

// Load 快照重载（boot 首读/断线重连愈合共用径）。
func (c *Client) Load(ctx context.Context) error {
	rows, err := c.pool.Query(ctx, "SELECT key, value, generation FROM app_config")
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}
	snap := map[string]json.RawMessage{}
	var gen int64
	for rows.Next() {
		var k string
		var v json.RawMessage
		var g int64
		if err := rows.Scan(&k, &v, &g); err != nil {
			rows.Close()
			return fmt.Errorf("config scan: %w", err)
		}
		snap[k] = v
		if g > gen {
			gen = g
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("config rows: %w", err)
	}
	c.mu.Lock()
	c.snap, c.gen, c.ok = snap, gen, true
	c.mu.Unlock()
	return nil
}

// Get 按次读当前快照值；未就绪/缺键=fail-closed error。
func (c *Client) Get(key string) (json.RawMessage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ok {
		return nil, ErrNotReady
	}
	v, hit := c.snap[key]
	if !hit {
		return nil, fmt.Errorf("config: 键 %q 不存在（无证据=fail-closed，禁默认值冒充）", key)
	}
	return v, nil
}

// Generation 当前快照代次（观测/调试面）。
func (c *Client) Generation() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gen
}

// Watch LISTEN config_changed：通知到达→快照重载；连接断→重连重听并重载。
// 通知载荷=表名（app_config/app_secrets）；app_secrets 变化同样触发重载保持代次一致。
func (c *Client) Watch(ctx context.Context) error {
	const retry = 2 * time.Second
	for {
		conn, err := c.pool.Acquire(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			time.Sleep(retry)
			continue
		}
		if _, err := conn.Exec(ctx, "LISTEN config_changed"); err != nil {
			conn.Release()
			time.Sleep(retry)
			continue
		}
		// 断线重连=快照重载（听上之前窗内变更可能漏通知——重载兜底）
		if err := c.Load(ctx); err != nil {
			conn.Release()
			time.Sleep(retry)
			continue
		}
		err = c.listenLoop(ctx, conn.Conn())
		conn.Release()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = err // 断线/超时→回环重连
	}
}

func (c *Client) listenLoop(ctx context.Context, conn *pgx.Conn) error {
	for {
		if _, err := conn.WaitForNotification(ctx); err != nil {
			return err
		}
		// 载荷仅表名——重载即正确响应，不解析载荷驱动行为
		if err := c.Load(ctx); err != nil {
			return err
		}
	}
}

// Secret 读取 app_secrets 密文行（按次直读库——密文不进快照内存常驻面）。
func (c *Client) Secret(ctx context.Context, key string) ([]byte, error) {
	var ct []byte
	if err := c.pool.QueryRow(ctx,
		"SELECT ciphertext FROM app_secrets WHERE key=$1", key).Scan(&ct); err != nil {
		return nil, fmt.Errorf("config: secret %q 读败：%w", key, err)
	}
	return ct, nil
}
