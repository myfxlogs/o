// signals.go —— 信号驱动生成臂（T-0140）：md_signal 新鲜信号背离/突变→
// 假想卡（ev_funding 事件驱动 D-0008-B；阈 app_config 热启；Fresh 闸内才认，
// 停桥=零卡不误报；信号只进 evidence/触发零提案径）。
package hypothesis

import (
	"context"
	"encoding/json"
	"log"
	"math"
)

// ScanSignals 信号臂一拍（T-0140 生成语义）：新鲜信号背离/突变→假想卡——
// funding_view 现值 vs avg_7d 背离超阈（regime 翻转形，family=carry）；
// depth5 spread_bps 超阈（流动性形，family=hedge）。阈全 app_config 热启
// （缺/坏=臂让位不产卡 fail-closed）；Fresh 闸（120s）内才认——停桥=零卡
// 不误报；UNIQUE(venue,surface) 幂等（同面待审卡在不重生）；信号只进
// evidence/触发，零提案径（起草席非执行席不变）。
func (r *Runner) ScanSignals(ctx context.Context) error {
	div, err := r.cfg.GetFloat("research.signal.funding_div_ratio")
	if err != nil || div <= 0 {
		r.rej.Inc("sig_missing_params")
		log.Printf("[research/hypo] 信号臂阈缺位——让位不产卡（research.signal.funding_div_ratio）")
		return nil
	}
	spreadTh, err := r.cfg.GetFloat("research.signal.spread_bps")
	if err != nil || spreadTh <= 0 {
		r.rej.Inc("sig_missing_params")
		log.Printf("[research/hypo] 信号臂阈缺位——让位不产卡（research.signal.spread_bps）")
		return nil
	}
	rows, err := r.pool.Query(ctx, `SELECT kind, key, val, meta FROM md_signal
		WHERE kind IN ('funding_view','depth5') AND ts > now()-interval '120 seconds'`)
	if err != nil {
		r.rej.Inc("sig_write_err")
		return err
	}
	type sig struct {
		kind, key string
		val       float64
		meta      map[string]any
	}
	var ss []sig
	for rows.Next() {
		var x sig
		var metaB []byte
		if err := rows.Scan(&x.kind, &x.key, &x.val, &metaB); err != nil {
			rows.Close()
			return err
		}
		x.meta = map[string]any{}
		_ = json.Unmarshal(metaB, &x.meta)
		ss = append(ss, x)
	}
	rows.Close()
	for _, x := range ss {
		switch {
		case x.kind == "funding_view":
			avg, _ := x.meta["avg_7d"].(float64)
			if avg == 0 {
				continue // 均值缺席——背离不可判如实
			}
			div_now := (x.val - avg) / math.Abs(avg)
			if math.Abs(div_now) < div {
				continue
			}
			r.draftSignal(ctx, "okx", "market-funding-rate", "carry",
				"funding_divergence", div_now, x)
		case x.kind == "depth5":
			if x.val < spreadTh {
				continue
			}
			r.draftSignal(ctx, "okx", "books5", "hedge",
				"spread_spike", x.val, x)
		}
	}
	return nil
}

// draftSignal 信号驱动卡（draft 同径——gapID=0 溯源 trigger+信号入 evidence；
// 幂等=同面待审卡在则跳过如实；零提案径）。
func (r *Runner) draftSignal(ctx context.Context, venue, surface, family, trigger string, strength float64, x struct {
	kind string
	key  string
	val  float64
	meta map[string]any
}) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_hypothesis
		WHERE venue=$1 AND surface=$2`, venue, surface).Scan(&n); err != nil || n > 0 {
		return // 既有卡（幂等）/读败让位
	}
	meta := map[string]any{
		"trigger": trigger, "trigger_strength": strength,
		"signals":       []Sig{{Kind: x.kind, Key: x.key, Val: x.val, Meta: x.meta}},
		"surface_basis": "md_signal 新鲜信号（T-0140 生成语义——非面差枚举）",
	}
	id, err := r.draft(ctx, venue, surface, family, 0, meta)
	if err != nil {
		r.rej.Inc("sig_write_err")
		return
	}
	log.Printf("[research/hypo] 信号驱动卡 #%d %s/%s（%s 强度=%.4f——待决策席审）",
		id, venue, surface, trigger, strength)
}
