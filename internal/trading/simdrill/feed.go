// feed.go —— 产线凭据产线缝件（工单#45468：lifecycle=portfolio 层禁引
// internal/risk·internal/trading/simdrill（D-514 分层白名单），装配收编
// 本包内单点——调用侧（cmd/core→lifecycle.AttachDrillFeed）只见 func
// 签名，Gate equity 桩+VenueForInst+Drill.Run 组装不外泄）。
package simdrill

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/risk"
)

// FeedDrill 演习缝实现（probe→live 毕业凭据产线——equity 桩 1e9 同
// cmd/simdrill 口径：演习域不占真账户额度；Gate 每缝一次建复用）。
// 返回 verdict 串（pass|fail）供调用侧落账观测；机械败走 err。
func FeedDrill(pool *pgxpool.Pool, cfg *config.Client) func(ctx context.Context, ref int64, inst string) (string, error) {
	gate := risk.NewGate(pool, cfg).
		WithEquity(func(context.Context, string) (float64, error) { return 1e9, nil })
	return func(ctx context.Context, ref int64, inst string) (string, error) {
		d := &Drill{Pool: pool, Cfg: cfg, Gate: gate, Venue: VenueForInst(inst)}
		res, err := d.Run(ctx, ref)
		if err != nil {
			return "", err
		}
		return res.Verdict, nil
	}
}
