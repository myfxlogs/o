// xvenue.go —— 跨所信号蒸馏（T-0141 P2b/D-0022）：xvenue_spread（OKX mark vs
// MT5 mid 价差 bps）+ carry_diff（OKX funding 年化 vs MT5 swap 年化差 bps——
// 负 funding 对冲径的先行亮牌，经济动机钉 2026-10-26：主流币负 funding=正主）。
// 单边陈/缺=不产如实（Fresh 120s 同款窗）；换算缺件（mt5_swap val=NULL 单位
// 方言不可年化）=carry_diff 该对缺席如实。
package sigdist

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
)

// AttachConfig 挂 config 读件（bridge.xvenue.map 热启——nil=跨所族让位）。
func (r *Runner) AttachConfig(cfg *config.Client) { r.cfg = cfg }

// xvenueMap 跨所对映射（{"ETH-USDT-SWAP":"ETHUSDm",…}——okx inst→mt5 symbol）。
func (r *Runner) xvenueMap(ctx context.Context) (map[string]string, error) {
	if r.cfg == nil {
		return nil, fmt.Errorf("config 未挂——bridge.xvenue.map 不可读")
	}
	raw, err := r.cfg.Get("bridge.xvenue.map")
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Xvenue 跨所族一拍（Distill 第三族组——映射缺/坏=让位 fail-closed 不虚构）。
func (r *Runner) Xvenue(ctx context.Context) error {
	xm, err := r.xvenueMap(ctx)
	if err != nil {
		r.rej.Inc("xvenue_map_missing")
		return fmt.Errorf("bridge.xvenue.map: %w", err)
	}
	// 单查 fresh 读件：okx mark（mark_price）/mt5 mid（md_ticks venue='mt5'）/
	// funding_view / mt5_swap——任一单边陈/缺=该对跳过如实。
	for okxInst, mt5Sym := range xm {
		var okxMark, mt5Mid float64
		err := r.pool.QueryRow(ctx, `SELECT px FROM md_mark_index
			WHERE channel='mark_price' AND inst_id=$1 AND ts > now()-interval '120 seconds'`,
			okxInst).Scan(&okxMark)
		if err != nil {
			continue // okx 单边陈/缺
		}
		err = r.pool.QueryRow(ctx, `SELECT (bid+ask)/2 FROM md_ticks
			WHERE venue='mt5' AND inst_id=$1 AND ask>0 AND bid>0 AND ts > now()-interval '120 seconds'`,
			mt5Sym).Scan(&mt5Mid)
		if err != nil || mt5Mid == 0 {
			continue // mt5 单边陈/缺/坏价
		}
		bps := (okxMark - mt5Mid) / mt5Mid * 10000
		meta, _ := json.Marshal(map[string]any{
			"mt5_symbol": mt5Sym, "okx_mark": okxMark, "mt5_mid": mt5Mid})
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
			VALUES('xvenue_spread',$1,$2,$3::jsonb,now())
			ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val,
			meta=EXCLUDED.meta, ts=EXCLUDED.ts`, okxInst, bps, string(meta)); err != nil {
			return err
		}
		// carry_diff：funding 年化（8h 期×3×365）− MT5 空腿 swap 年化（bps）。
		var funding, swapAnn float64
		fErr := r.pool.QueryRow(ctx, `SELECT val FROM md_signal
			WHERE kind='funding_view' AND key=$1 AND ts > now()-interval '120 seconds'`,
			okxInst).Scan(&funding)
		sErr := r.pool.QueryRow(ctx, `SELECT val FROM md_signal
			WHERE kind='mt5_swap' AND key=$1 AND val IS NOT NULL AND ts > now()-interval '8 hours'`,
			mt5Sym).Scan(&swapAnn)
		if fErr != nil || sErr != nil {
			continue // 换算缺件/单边陈——缺席如实（mt5_swap val=NULL 单位方言不可年化）
		}
		fundingAnn := funding * 3 * 365 * 10000 // 8h 期→年化 bps
		diff := fundingAnn - swapAnn
		meta2, _ := json.Marshal(map[string]any{
			"mt5_symbol": mt5Sym, "funding_ann_bps": fundingAnn, "mt5_swap_ann_bps": swapAnn})
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
			VALUES('carry_diff',$1,$2,$3::jsonb,now())
			ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val,
			meta=EXCLUDED.meta, ts=EXCLUDED.ts`, okxInst, diff, string(meta2)); err != nil {
			return err
		}
	}
	log.Printf("[sigdist] xvenue 拍：%d 对（xvenue_spread/carry_diff）", len(xm))
	return nil
}

// SurfaceAdapter 装配侧信号读闸出口（T-0140 接线件——core 假想官
// AttachSignals 传此闭包；读面零岗，不注册不节拍）。
func SurfaceAdapter(pool *pgxpool.Pool) func(ctx context.Context, venue, surface string) ([]Sig, bool, error) {
	return New(pool).SurfaceSignals
}
