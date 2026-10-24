// Package facts —— 账户事实面（T-0041 业主令：全面了解不猜）。两 venue 户层
// （posMode/acctLv/equity/leverage/ccy 余额）+乐器层（ctVal/ctValCcy/minSz/
// TradeMode 逐只）→md_venue_fact 单源层（UPSERT 幂等）。员工「集使」（编制册
// 数据域批拉采集调度岗——facts 采集行产出窗）：boot 全谱采集+低频周期刷
// （慢项——业主令明示 account 层低频刷合法）。消费律：动态项（TradeMode/posMode）
// 下单时点现读——本表=决策输入层非放行证（NOX posID=468 裸露教训锚）。
package facts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/rejects"
	"phase/internal/platform/roster"
)

const (
	beatEvery    = 30 * time.Second
	refreshEvery = time.Hour // 慢项低频刷（户层/乐器属性——非行情面）
)

// Fact 一行事实（kind=account|instrument；Value=JSON 原生形）。
type Fact struct {
	Kind  string
	Key   string
	Value any
}

// Source venue 采集源（注入式——okx/mt5 装配在 cmd，facts 不识 venue 方言）。
type Source interface {
	Venue() string
	Collect(ctx context.Context) ([]Fact, error)
}

// Collector 集使岗体。
type Collector struct {
	pool    *pgxpool.Pool
	sources []Source
	rej     *rejects.Door
}

func NewCollector(pool *pgxpool.Pool, sources ...Source) *Collector {
	return &Collector{pool: pool, sources: sources,
		rej: rejects.New("facts", "collect_failed", "write_err")}
}

// Run 上岗：Register+Beat+boot 全谱采集→低频周期刷。
func (c *Collector) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, c.pool, "集使", "data", "批拉采集调度岗——venue 账户/乐器 facts 单源层", nil); err != nil {
		errc <- fmt.Errorf("集使 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, c.pool, "集使", beatEvery, errc)
	c.collectAll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(refreshEvery): // sched-exempt-ok 观测层低频刷——venue 无户配/乐器推送面，决策径不读本表
			c.collectAll(ctx)
		}
	}
}

// collectAll 逐源采集落库（单源败=计数不连坐他源）。
func (c *Collector) collectAll(ctx context.Context) {
	for _, s := range c.sources {
		fs, err := s.Collect(ctx)
		if err != nil {
			c.rej.Inc("collect_failed")
			log.Printf("[facts] %s 采集败: %v", s.Venue(), err)
			continue
		}
		n := 0
		for _, f := range fs {
			if err := Upsert(ctx, c.pool, s.Venue(), f); err != nil {
				c.rej.Inc("write_err")
				continue
			}
			n++
		}
		log.Printf("[facts] %s 落库 %d 行（account+instrument）", s.Venue(), n)
	}
}

// Upsert 幂等落行（value jsonb 原生形；ts 刷写时点）。
func Upsert(ctx context.Context, pool *pgxpool.Pool, venue string, f Fact) error {
	v, err := json.Marshal(f.Value)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `INSERT INTO md_venue_fact(venue,kind,key,value) VALUES($1,$2,$3,$4)
		ON CONFLICT(venue,kind,key) DO UPDATE SET value=EXCLUDED.value,ts=now()`,
		venue, f.Kind, f.Key, v)
	return err
}

// CollectOnce 单次全谱采集（探针/装配外取证面——Run 的 boot 拍同径）。
func (c *Collector) CollectOnce(ctx context.Context) { c.collectAll(ctx) }
