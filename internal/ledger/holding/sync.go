// Package holding —— 持有资产台账（原币登记层：存在性入账、不折算、
// 不入 fin_cashflow/净额）。采集=钦定 REST 保留面（okx-rest-boundary：
// /asset/balances 资金户+/finance/savings/balance earn 面——venue 拉取件
// 装配层注入，数据/账本域不反向依赖交易域）。采不到=陈旧标记不抹值。
package holding

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/events"
	"o/internal/platform/rejects"
)

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
	// 事件化（T-0048：周期环拆——venue 余额帧 ev_account 驱动采拍）。
	events.ListenLoop(ctx, s.pool, "holding", []string{"ev_account"}, s.sync)
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
	// 缺席即删（T-0478 钦定）：成功拍收尾——fetch 覆盖域（venue='okx'×本拍
	// 返回 (kind,venue_ref) 对）内 source='api' 存量行不在返回集→DELETE；
	// 镜像=venue 实存真源非审计留痕（审计痕在 venue bills/ops_event 另存）。
	// 护线：bag.go 显形行 kind='spot_dust' 但 venue_ref='' 出覆盖对域、
	// 手工行 source='owner_declared'、structured_order 非 fetch 域天然不收；
	// 本拍未覆盖的 (kind,venue_ref) 对不动（半拍不核销——earn 面单独缺席
	// 不连坐 spot 域）。fetch 败拍走上方早退全标 stale 不删。
	if len(rows) > 0 {
		seen := map[[3]string]bool{}
		pairSeen := map[[2]string]bool{}
		var pk, pv, kk, kc, kv []string
		for _, r := range rows {
			if k := [3]string{r.Kind, r.Ccy, r.VenueRef}; !seen[k] {
				seen[k] = true
				kk = append(kk, r.Kind)
				kc = append(kc, r.Ccy)
				kv = append(kv, r.VenueRef)
			}
			if p := [2]string{r.Kind, r.VenueRef}; !pairSeen[p] {
				pairSeen[p] = true
				pk = append(pk, r.Kind)
				pv = append(pv, r.VenueRef)
			}
		}
		if _, err := s.pool.Exec(ctx, `DELETE FROM fin_holding h
			WHERE h.venue='okx' AND h.source='api'
			  AND (h.kind,h.venue_ref) IN (SELECT t.k,t.v FROM unnest($1::text[],$2::text[]) AS t(k,v))
			  AND NOT EXISTS (SELECT 1 FROM unnest($3::text[],$4::text[],$5::text[]) AS t(k,c,v)
			                  WHERE t.k=h.kind AND t.c=h.ccy AND t.v=h.venue_ref)`,
			pk, pv, kk, kc, kv); err != nil {
			s.rej.Inc("write_err")
		}
	}
	return nil
}
