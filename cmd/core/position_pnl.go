// position_pnl.go —— 在持仓盈亏卡组装件（O-41385——ListOpenPositions
// 盈亏字段单源：mark=legset.MarkPx、鲜度界=dirTrig 同键、upl_frac=
// dirTrig frac 同式 sign×(mark/entry−1)、fees/funding/trade_pnl=
// fin_order_leg.fee+fin_cashflow 仓级归因直加——禁第二套口径）。
package main

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/platform/legset"
)

// posEntryLeg 仓开腿代表行（leg_no 序首个 filled+px>0——dirTrig entryPx
// 同选法；mark/entry 须同 inst 自洽）。
type posEntryLeg struct {
	px    float64
	inst  string
	venue string
	side  string
	ok    bool
}

// posPnlAgg 腿扫面聚合（entry 代表腿+fees 累计——legs 查询同拍产出）。
type posPnlAgg struct {
	entry posEntryLeg
	fees  float64
}

// aggLeg 腿行并入仓聚合（legs 按 execution_id,leg_no 序到——首个
// filled+px>0 腿=entry 代表；fee 全腿累计负=付原样加）。
func (a *posPnlAgg) aggLeg(state, side, inst, venue string, px, fee *float64) {
	if fee != nil {
		a.fees += *fee
	}
	if a.entry.ok || state != "filled" || px == nil || *px <= 0 {
		return
	}
	a.entry = posEntryLeg{px: *px, inst: inst, venue: venue, side: side, ok: true}
}

// pnlFreshMaxAge mark 鲜度界（dirTrig lastPx 同键——okx=exec.tick_max_age_s，
// cfd 名册判→lifecycle.dir.cfd_tick_max_age_s；cfg 缺席=legset 缺省）。
func pnlFreshMaxAge(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client, venue string) float64 {
	maxAge := legset.TickMaxAgeDefS
	if cfg != nil {
		if v, err := cfg.GetFloat("exec.tick_max_age_s"); err == nil && v > 0 {
			maxAge = v
		}
	}
	if mt5.IsCfdProtoVenue(ctx, pool, venue) {
		maxAge = legset.SparseTickBoundDefS
		if cfg != nil {
			if v, err := cfg.GetFloat("lifecycle.dir.cfd_tick_max_age_s"); err == nil && v > 0 {
				maxAge = v
			}
		}
	}
	return maxAge
}

// posDirSign 判向（dirTrig frac 同判——腿 side 卖/空=−1 优先，腿侧缺省
// 归 direction 前缀 short 判；腿侧明证不被 direction 复合族翻——
// mark/entry 盯的是 entry 腿自身 inst，符号须随被盯腿）。
func posDirSign(direction, side string) float64 {
	switch strings.ToLower(side) {
	case "sell", "short":
		return -1
	case "buy", "long":
		return 1
	}
	if strings.HasPrefix(direction, "short_") {
		return -1
	}
	return 1
}

// posMarkPnl mark 盯市三字段（mark 陈/缺/entry 缺=全 0 如实空不虚构——
// 前端显「价陈」）。返回 markPx/uplFrac/uplUSD。
func (c coreService) posMarkPnl(ctx context.Context, venue, inst string,
	entryPx, sign, notional float64) (markPx, uplFrac, uplUSD float64) {
	if entryPx <= 0 || inst == "" {
		return 0, 0, 0
	}
	px, ts, _, ok := legset.MarkPx(ctx, c.pool, venue, inst)
	if !ok || px <= 0 {
		return 0, 0, 0
	}
	if time.Since(ts) > time.Duration(pnlFreshMaxAge(ctx, c.pool, c.cfg, venue)*float64(time.Second)) {
		return 0, 0, 0 // 陈价=价缺同径——不产陈 mark 数
	}
	frac := sign * (px/entryPx - 1) // dirTrig frac 同式（名义线性近似）
	return px, frac, notional * frac
}
