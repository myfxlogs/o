package main

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mtmReader 对账净值偏移读件（venue×ccy——T-0076 合约增量+T-0087 现货全值并轨）：
// 合约腿只有保证金+浮盈=加增量 sign×sz×ctv×(现价−入价)；现货腿持真币=加全值
// sign×sz×现价（买入现金流已计流出，全值项补资产本金，增量已含在现价里）。
// 现价=md_ticks 最新 last；价缺=该腿按 0 加不减（不产假偏移）。
func mtmReader(pool *pgxpool.Pool,
	ctv func(ctx context.Context, venue, inst string) (float64, bool),
) func(ctx context.Context) (map[[2]string]float64, error) {
	return func(ctx context.Context) (map[[2]string]float64, error) {
		out := map[[2]string]float64{}
		rows, err := pool.Query(ctx, `SELECT l.venue,l.inst_id,l.side,l.sz,l.px,l.sz_unit,l.qty_base
			FROM fin_order_leg l JOIN fin_position p ON p.execution_id=l.execution_id
			WHERE p.state<>'closed' AND l.state='filled' AND l.px IS NOT NULL`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var venue, inst, side string
			var sz, entry float64
			var su *string
			var qb *float64
			if err := rows.Scan(&venue, &inst, &side, &sz, &entry, &su, &qb); err != nil {
				continue
			}
			qtyBase, szUnit := 0.0, ""
			if qb != nil {
				qtyBase = *qb
			}
			if su != nil {
				szUnit = *su
			}
			sign := 1.0
			if side == "sell" {
				sign = -1.0
			}
			if !strings.HasSuffix(inst, "-SWAP") {
				// O-34287 验收③：非合约腿偏移=基币原量入基币键（qty_base
				// 本币量不折 USDT——venue 基币余额/ledger 原币腿同径币量
				// 对拍；币量径无价依赖，USD 估值只进估值层不喂对账）。
				// inst BASE-QUOTE 双段解不出=跳过不产假偏移。
				parts := strings.SplitN(inst, "-", 2)
				if len(parts) != 2 {
					continue
				}
				switch {
				case qtyBase > 0:
					out[[2]string{venue, parts[0]}] += sign * qtyBase
				case szUnit == "base":
					out[[2]string{venue, parts[0]}] += sign * sz
				}
				continue
			}
			var now float64
			if err := pool.QueryRow(ctx,
				`SELECT v FROM (SELECT last v,ts FROM md_ticks WHERE inst_id=$1
					UNION ALL SELECT px,ts FROM md_mark_index WHERE inst_id=$1) q
					ORDER BY ts DESC LIMIT 1`,
				inst).Scan(&now); err != nil || now <= 0 {
				continue
			}
			mult := 1.0
			if cv, ok := ctv(ctx, venue, inst); ok {
				mult = cv
			}
			out[[2]string{venue, "USDT"}] += sign * sz * mult * (now - entry)
		}
		// O-33207 手动仓浮盈并入——fin_venue_position 在仓行（O 单已在
		// legs 径不重复入册=双簿防线）：unreal=sign×lots×contract_size×
		// (mark−open_px)，mark=md_ticks venue×inst 最新 (bid+ask)/2
		// （CFD 符号 venue 键同域；价缺按 0 加不减同腿径）。
		vrows, verr := pool.Query(ctx, `SELECT venue,symbol,side,lots,contract_size,open_px,ccy,mtm_off
			FROM fin_venue_position WHERE state='open' AND (open_px IS NOT NULL OR mtm_off IS NOT NULL)`)
		if verr != nil {
			return nil, verr
		}
		defer vrows.Close()
		for vrows.Next() {
			var venue, symbol, side, ccy string
			var lots, cs, entry float64
			var off *float64
			if err := vrows.Scan(&venue, &symbol, &side, &lots, &cs, &entry, &ccy, &off); err != nil {
				continue
			}
			if off != nil {
				out[[2]string{venue, ccy}] += *off // OKX upl 直给径——venue 现成 uPnL 带号
				continue
			}
			sign := 1.0
			if side == "sell" {
				sign = -1.0
			}
			var mark float64
			if err := pool.QueryRow(ctx,
				`SELECT (bid+ask)/2 FROM md_ticks WHERE venue=$1 AND inst_id=$2
				 ORDER BY ts DESC LIMIT 1`,
				venue, symbol).Scan(&mark); err != nil || mark <= 0 {
				continue
			}
			out[[2]string{venue, ccy}] += sign * lots * cs * (mark - entry)
		}
		return out, nil
	}
}
