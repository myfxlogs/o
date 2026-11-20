// persist.go —— 评估卡+提案落表面（dirmom/dirtrend 同款写拍批内合流件）。
package gridfit

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"o/internal/platform/execfam"
	"o/internal/research/evalgate"
)

// bars 末 n 根确认棒 OHLC 序（旧→新；不足 n 返实得数——仿真消费
// h/l/c，o 不取）。
func (r *Runner) bars(ctx context.Context, inst string, n int) ([]kline, time.Time, int, error) {
	rows, err := r.pool.Query(ctx, `SELECT h,l,c,ts FROM md_klines
		WHERE venue='okx' AND inst_id=$1 AND bar=$2 AND confirm
		ORDER BY ts DESC LIMIT $3`, inst, bar, n)
	if err != nil {
		return nil, time.Time{}, 0, err
	}
	defer rows.Close()
	var bs []kline
	var last time.Time
	first := true
	for rows.Next() {
		var b kline
		var ts time.Time
		if err := rows.Scan(&b.h, &b.l, &b.c, &ts); err != nil {
			return nil, time.Time{}, 0, err
		}
		if first {
			last, first = ts, false
		}
		bs = append([]kline{b}, bs...) // 倒序→正序（PREPEND——n 小无常数忧）
	}
	return bs, last, len(bs), rows.Err()
}

// writeEval 评估卡落表（direction 恒 'grid_spot'——网格双边同册不拆向）
// +proposed 且无 open 提案/策略→提案行（闸后唯一提案径——fin_strategy
// md 侧只读副本 dedupe 同 funding persist）。影子族：execfam 未册
// grid_spot→proposed 翻 rejected/exec_family_missing（T-0330 前置拒
// 同闸，T-0408/mt5_dir_mom 先例——eval 卡全可见执行零径）。
func (r *Runner) writeEval(ctx context.Context, tx pgx.Tx, c card, p Params) error {
	verdict, reason := c.verdict, c.reason
	if verdict == "proposed" && !execfam.Has(Template) {
		verdict, reason = "rejected", "exec_family_missing"
	}
	ev, _ := json.Marshal(map[string]any{
		"sim_verdict": c.verdict, "sim_reason": c.reason, // 闸前仿真原判（影域卡证据面）
		"c0": c.c0, "band_lo": c.bandLo, "band_hi": c.bandHi,
		"last_close": c.lastClose,
		"realized":   c.realized, "unrealized": c.unrealized,
		"fees": c.fees, "capital": c.capital,
		"gross_apr": c.grossAPR, "net_apr": c.netAPR,
		"n_lines": c.nLines, "n_buys": c.nBuys, "n_sells": c.nSells,
		"lookback": p.Lookback, "band_pct": p.BandPct, "step_pct": p.StepPct,
		"line_notional_usd": p.LineNotional, "fee_rt": p.FeeRT,
		"min_net_apr": p.MinNetAPR, "blown_tol": p.BlownTol,
		"n_bars": c.bars, "last_bar_ts": c.lastTs, "reason": reason,
	})
	legs, _ := json.Marshal([]string{"spot_long"})
	// 工单#22904 指纹闸：同(inst,template,direction)最新行 verdict+net_apr
	// 未变=信念未迁→跳 INSERT 复用行 id 续提案径（判词/净额变迁仍产新版史）。
	evalID, same, err := evalgate.SameFingerprint(ctx, tx, c.inst, Template, "grid_spot", verdict, c.netAPR)
	if err != nil {
		return err
	}
	if !same {
		if err := tx.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
			c.inst, Template, "grid_spot", c.grossAPR, c.netAPR,
			verdict, legs, ev).Scan(&evalID); err != nil {
			return err
		}
	}
	if verdict != "proposed" {
		return nil
	}
	var open int
	if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM fin_proposal
		WHERE inst_id=$1 AND direction=$2 AND state IN ('fin_proposed','fin_armed'))
		+ (SELECT count(*) FROM fin_strategy
		WHERE inst_id=$1 AND direction=$2 AND state<>'retired')
		+ (SELECT count(*) FROM fin_venue_untradeable WHERE inst_id=$1)`,
		c.inst, "grid_spot").Scan(&open); err != nil {
		return err
	}
	if open > 0 {
		return nil // 在挂/黑名单在册不起草（fin_venue_untradeable 滤=#15188 死案径全员义务）
	}
	_, err = tx.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,$2,$3,$4)`, evalID, c.inst, "grid_spot", c.netAPR)
	if err == nil {
		log.Printf("[research/gridfit] 提案落卡 %s net_apr=%.4f realized=%.4f（影子族 execfam 未册）",
			c.inst, c.netAPR, c.realized)
	}
	return err
}
