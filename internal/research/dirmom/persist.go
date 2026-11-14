// persist.go —— 评估卡+提案落表面（funding/persist.go 同款写拍批内合流件）。
package dirmom

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"o/internal/platform/execfam"
)

// closes 末 n 根确认棒收盘序（旧→新；不足 n 返实得数）。
func (r *Runner) closes(ctx context.Context, inst string, n int) ([]float64, time.Time, int, error) {
	rows, err := r.pool.Query(ctx, `SELECT c,ts FROM md_klines
		WHERE venue='okx' AND inst_id=$1 AND bar=$2 AND confirm
		ORDER BY ts DESC LIMIT $3`, inst, bar, n)
	if err != nil {
		return nil, time.Time{}, 0, err
	}
	defer rows.Close()
	var cl []float64
	var last time.Time
	first := true
	for rows.Next() {
		var c float64
		var ts time.Time
		if err := rows.Scan(&c, &ts); err != nil {
			return nil, time.Time{}, 0, err
		}
		if first {
			last, first = ts, false
		}
		cl = append([]float64{c}, cl...) // 倒序→正序（PREPEND——n 小无常数忧）
	}
	return cl, last, len(cl), rows.Err()
}

// writeEval 评估卡落表（upsert 自然键）+proposed 且无 open 提案/策略→提案行
// （闸后唯一提案径——fin_strategy md 侧只读副本 dedupe 同 funding persist）。
func (r *Runner) writeEval(ctx context.Context, tx pgx.Tx, inst, dir, verdict, reason string,
	c card, p Params) error {
	if verdict == "proposed" && !execfam.Has(Template) {
		verdict, reason = "rejected", "exec_family_missing" // T-0330 前置拒同闸
	}
	ev, _ := json.Marshal(map[string]any{
		"z": c.z, "sigma_d": c.sigma, "mom": c.mom, "n_bars": c.bars,
		"lookback": p.Lookback, "hold_days": p.HoldDays, "persist": p.Persist,
		"friction_rt": p.FrictionRT, "net_apr": c.netAPR,
		"last_bar_ts": c.lastTs, "reason": reason,
	})
	legs, _ := json.Marshal([]string{map[string]string{
		"long_perp": "perp_long", "short_perp": "perp_short"}[dir]})
	var evalID int64
	if err := tx.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		gross_apr=EXCLUDED.gross_apr,net_apr=EXCLUDED.net_apr,verdict=EXCLUDED.verdict,
		legs=EXCLUDED.legs,evidence=EXCLUDED.evidence RETURNING id`,
		inst, Template, dir, c.netAPR+p.FrictionRT*365/p.HoldDays, c.netAPR,
		verdict, legs, ev).Scan(&evalID); err != nil {
		return err
	}
	if verdict != "proposed" {
		return nil
	}
	var open int
	if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM fin_proposal
		WHERE inst_id=$1 AND direction=$2 AND state IN ('fin_proposed','fin_armed'))
		+ (SELECT count(*) FROM fin_strategy
		WHERE inst_id=$1 AND direction=$2 AND state<>'retired')`,
		inst, dir).Scan(&open); err != nil {
		return err
	}
	if open > 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,$2,$3,$4)`, evalID, inst, dir, c.netAPR)
	if err == nil {
		log.Printf("[research/dirmom] 提案落卡 %s %s z=%.2f net_apr=%.4f（paper-only 族）",
			inst, dir, c.z, c.netAPR)
	}
	return err
}
