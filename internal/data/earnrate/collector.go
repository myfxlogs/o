// Package earnrate —— Earn 现率采集（钦定 REST 面 /finance/savings/lending-rate-summary；
// 拉取件装配层注入=数据域不引 trading 包）。采败=告警计数不造假；拍长热启。
package earnrate

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/rejects"
)

// Fetcher 拉取件（cmd 装配 okxexec——钦定保留面）。
type Fetcher func(ctx context.Context) (json.RawMessage, error)

// Collector 现率采集岗。
type Collector struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	fetch Fetcher
	rej   *rejects.Door
}

func NewCollector(pool *pgxpool.Pool, cfg *config.Client, f Fetcher) *Collector {
	return &Collector{pool: pool, cfg: cfg, fetch: f,
		rej: rejects.New("earnrate", "fetch_err", "parse_err", "missing_param", "write_err")}
}

// Run 采拍循环（拍长 data.earn.poll_sec 热启；缺键 fail-closed 每拍重试读）。
func (c *Collector) Run(ctx context.Context, errc chan<- error) {
	for {
		every := 300 * time.Second
		if v, err := c.cfg.GetFloat("data.earn.poll_sec"); err == nil && v > 0 {
			every = time.Duration(v) * time.Second
		} else if err != nil {
			c.rej.Inc("missing_param")
		}
		if err := c.collect(ctx); err != nil {
			log.Printf("[earnrate] collect: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}
	}
}

// collect 一拍：lending-rate-summary→USDT 行落事实表。
func (c *Collector) collect(ctx context.Context) error {
	if c.fetch == nil {
		return nil
	}
	raw, err := c.fetch(ctx)
	if err != nil {
		c.rej.Inc("fetch_err")
		return fmt.Errorf("fetch: %w", err)
	}
	// lending-rate-summary 字段族：estRate（预估）/preRate/avgRate——取 estRate 主、avgRate 备。
	var rows []struct {
		Ccy     string `json:"ccy"`
		EstRate string `json:"estRate"`
		AvgRate string `json:"avgRate"`
		PreRate string `json:"preRate"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		c.rej.Inc("parse_err")
		return err
	}
	for _, r := range rows {
		rs := r.EstRate
		if rs == "" {
			rs = r.AvgRate
		}
		if rs == "" {
			rs = r.PreRate
		}
		f, err := strconv.ParseFloat(rs, 64)
		if err != nil || r.Ccy == "" {
			continue
		}
		if _, err := c.pool.Exec(ctx,
			"INSERT INTO md_earn_rate(venue,ccy,rate) VALUES('okx',$1,$2)", r.Ccy, f); err != nil {
			c.rej.Inc("write_err")
		}
	}
	return nil
}

// CollectOnce 单拍壳（工具/实证件消费）。
func (c *Collector) CollectOnce(ctx context.Context) error { return c.collect(ctx) }
