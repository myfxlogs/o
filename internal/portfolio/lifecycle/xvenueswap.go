// xvenueswap.go —— xvenue-carry mt5 腿 swap 持续记账（T-0377 断链补焊③）：
// 开户时 swap_snapshot 只锁快照（amount=0）不产账——mt5 腿 swap 摊提此前
// 零径（paperAccounting 只吃 md_funding=okx perp funding 端，mt5 CFD 无行）。
// 驱动=md_signal kind='mt5_swap' 新行（ts>该腿最近 swap fill）：按方向腿取
// meta swap_long_ann_bps（long_cfd）/swap_short_ann_bps（short_cfd）年化按
// Δh 摊提，负值=成本（负 fill）、正值=收入——与 okx 腿 funding 对冲同源对拍
// （T-0377 锚②：快照值与记账值同键同源）。ann 缺/零（SOLUSDm ann_ok=false
// 单位方言不可年化）=跳过如实不造零账；行缺=跳过（fail-closed）。
package lifecycle

import (
	"context"
	"encoding/json"
	"time"
)

// xvenueSwapAccounting mt5 腿 swap 摊提一拍（scan 内随 paperAccounting 同序）。
func (r *Runner) xvenueSwapAccounting(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.direction, s.notional_usd,
		COALESCE((SELECT max(ts) FROM fin_paper_fill f WHERE f.strategy_id=s.id AND f.kind='swap'), s.entered_at)
		FROM fin_strategy s
		WHERE s.state='paper' AND s.template_ref='xvenue-carry' AND s.venue='mt5'`)
	if err != nil {
		return err
	}
	type leg struct {
		id       int64
		inst     string
		dir      string
		notional float64
		last     time.Time
	}
	var legs []leg
	for rows.Next() {
		var x leg
		if err := rows.Scan(&x.id, &x.inst, &x.dir, &x.notional, &x.last); err != nil {
			rows.Close()
			return err
		}
		legs = append(legs, x)
	}
	rows.Close()
	for _, x := range legs {
		// 方向腿年化取键（副行 direction=long_cfd/short_cfd——admitXvenuePair 向取反钉）。
		key := "swap_short_ann_bps"
		if x.dir == "long_cfd" {
			key = "swap_long_ann_bps"
		} else if x.dir != "short_cfd" {
			continue // 非副腿向=不认（fail-closed 不臆测）
		}
		var metaB []byte
		var sigTs time.Time
		if err := r.pool.QueryRow(ctx, `SELECT meta, ts FROM md_signal
			WHERE kind='mt5_swap' AND key=$1 ORDER BY ts DESC LIMIT 1`, x.inst).Scan(&metaB, &sigTs); err != nil {
			continue // 信号行缺=摊提缺件如实（不造零账）
		}
		var meta map[string]any
		if err := json.Unmarshal(metaB, &meta); err != nil {
			continue
		}
		ann, _ := meta[key].(float64)
		if ann == 0 {
			continue // 该腿 swap 零/不可年化=无账可摊（如实缺席）
		}
		dh := sigTs.Sub(x.last).Hours()
		if dh <= 0 {
			continue // 无新行/行先于记账基准=零摊提（ts 门防重摊）
		}
		amt := ann / 10000 * x.notional * dh / (365 * 24)
		detail, _ := json.Marshal(map[string]any{
			"ann_bps": ann, "hours": dh, "signal_ts": sigTs.Format(time.RFC3339)})
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
			VALUES($1,'swap',$2,$3,$4,$5)`, x.id, x.inst, amt, detail, sigTs); err != nil {
			return err
		}
	}
	return nil
}
