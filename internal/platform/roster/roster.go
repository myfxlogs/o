// Package roster —— 花名册读写（fin_employee_registry）。运行时本体=1 号员工：
// boot Register（upsert 幂等）+ 周期 Beat 刷 last_beat（活性台账，非业务数据刷新）。
package roster

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// registryPool 席位台账池（fin_employee_registry 归 fin——D-0030 §五「直 beat」：
// 分域后 dev 侧进程 boot 钉 SG 隧道径池，此后 Register/Beat 全走该池，业务池
// 不掺席位面；nil=未钉（同库单机期/测试语义——pool 自身即册）。
var registryPool *pgxpool.Pool

// SetRegistryPool 钉席位台账池（节点 boot 一次性——并发前调用）。
func SetRegistryPool(p *pgxpool.Pool) { registryPool = p }

func target(pool *pgxpool.Pool) *pgxpool.Pool {
	if registryPool != nil {
		return registryPool
	}
	return pool
}

// Register 自注册上岗：name 冲突行更新 dept/title/meta 并刷 last_beat（重启幂等）。
func Register(ctx context.Context, pool *pgxpool.Pool, name, dept, title string, meta []byte) error {
	if len(meta) == 0 {
		meta = []byte("{}")
	}
	_, err := target(pool).Exec(ctx, `INSERT INTO fin_employee_registry(name,dept,title,meta,last_beat)
		VALUES($1,$2,$3,$4,now())
		ON CONFLICT(name) DO UPDATE SET dept=EXCLUDED.dept, title=EXCLUDED.title,
			meta=EXCLUDED.meta, status='active', last_beat=now()`,
		name, dept, title, meta)
	if err != nil {
		return fmt.Errorf("roster register %s: %w", name, err)
	}
	return nil
}

// Beat 刷心跳（行不在=未注册→fail-closed 报错，调用方须先 Register）。
func Beat(ctx context.Context, pool *pgxpool.Pool, name string) error {
	tag, err := target(pool).Exec(ctx,
		"UPDATE fin_employee_registry SET last_beat=now() WHERE name=$1", name)
	if err != nil {
		return fmt.Errorf("roster beat %s: %w", name, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("roster beat %s: 未注册行（须先 Register）", name)
	}
	return nil
}

// BeatLoop 周期心跳 goroutine；errc 上报失败（beat 断=活性台账陈——声张不静默）。
func BeatLoop(ctx context.Context, pool *pgxpool.Pool, name string, every time.Duration, errc chan<- error) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := Beat(ctx, pool, name); err != nil {
				select {
				case errc <- err:
				default:
				}
			}
		}
	}
}
