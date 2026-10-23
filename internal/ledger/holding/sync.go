// Package holding —— 持有资产台账（原币登记层：存在性入账、不折算、
// 不入 fin_cashflow/净额）。采集=钦定 REST 保留面（okx-rest-boundary：
// /asset/balances 资金户+/finance/savings/balance earn 面——venue 拉取件
// 装配层注入，数据/账本域不反向依赖交易域）。采不到=陈旧标记不抹值。
package holding

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/rejects"
)

const scanEvery = 5 * time.Minute // 资产端点采拍（账本侧低频同步——非行情流替代）

// AssetRow venue 回报行（ccy/amount/可用/冻结+kind 归位）。
type AssetRow struct {
	Ccy      string
	Amount   float64
	Kind     string // spot_dust|earn_flexible|earn_structured
	Product  string
	VenueRef string
	ValRef   map[string]any
}

// Fetcher 资产拉取件（装配层 cmd 接 okxexec——/asset/balances+earn 面）。
type Fetcher func(ctx context.Context) ([]AssetRow, error)

// Syncer 台账同步岗。
type Syncer struct {
	pool  *pgxpool.Pool
	fetch Fetcher
	rej   *rejects.Door
}

func NewSyncer(pool *pgxpool.Pool, fetch Fetcher) *Syncer {
	return &Syncer{pool: pool, fetch: fetch,
		rej: rejects.New("holding", "fetch_err", "write_err")}
}

// Run 采拍循环。
func (s *Syncer) Run(ctx context.Context, errc chan<- error) {
	for {
		if err := s.sync(ctx); err != nil {
			log.Printf("[holding] sync: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(scanEvery):
		}
	}
}

// sync 一拍：拉资产→UPSERT 行（asof 刷新，stale=false）；拉败=全行标陈旧不抹值。
func (s *Syncer) sync(ctx context.Context) error {
	if s.fetch == nil {
		return nil
	}
	rows, err := s.fetch(ctx)
	if err != nil {
		s.rej.Inc("fetch_err")
		if _, derr := s.pool.Exec(ctx,
			"UPDATE fin_holding SET stale=true WHERE source='api'"); derr != nil {
			return derr
		}
		return fmt.Errorf("采集败→陈旧标记: %w", err)
	}
	for _, r := range rows {
		blob, _ := json.Marshal(r.ValRef)
		if _, err := s.pool.Exec(ctx, `INSERT INTO fin_holding(venue,kind,ccy,product,amount,venue_ref,val_ref,asof_ts,stale)
			VALUES('okx',$1,$2,$3,$4,$5,$6,now(),false)
			ON CONFLICT(venue,kind,ccy,venue_ref) DO UPDATE
			SET amount=EXCLUDED.amount,val_ref=EXCLUDED.val_ref,asof_ts=now(),stale=false`,
			r.Kind, r.Ccy, r.Product, r.Amount, r.VenueRef, blob); err != nil {
			s.rej.Inc("write_err")
		}
	}
	return nil
}
