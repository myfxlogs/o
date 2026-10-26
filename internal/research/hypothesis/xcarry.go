// xcarry.go —— xvenue_carry 跨所 carry 臂（T-0144/D-0022 终态件）：OKX perp ×
// MT5 CFD 对冲 carry 首个真跨所模板。链环：carry_diff 信号亮牌（大负值=负
// funding 对冲机会——票面钦定语义钉）→假想卡（legVocab xvenue_carry 闭集）→
// 评估（净年化四件分解+容量三件+verdict 双腿能力账双闸）→提案至
// fin_proposed 止（paper 双腿开户 lifecycle 径；真执行=T-0145 另票——本票
// armed→open 显式禁，抢跑=裸腿风险）。
package hypothesis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
)

// xvenueCarryScan 信号臂 xvenue_carry 支路（ScanSignals 尾调）：carry_diff
// |val|≥阈→立卡+评估→提案（fin_proposed 步进，armed 由 lifecycle 径）。
func (r *Runner) xvenueCarryScan(ctx context.Context) {
	th := 300.0 // 亮牌阈 bps（research.signal.xvenue_carry_diff_bps 热启）
	if v, err := r.cfg.GetFloat("research.signal.xvenue_carry_diff_bps"); err == nil && v > 0 {
		th = v
	}
	rows, err := r.pool.Query(ctx, `SELECT key, val, meta FROM md_signal
		WHERE kind='carry_diff' AND ts > now()-interval '120 seconds'`)
	if err != nil {
		r.rej.Inc("hypo_write_err")
		return
	}
	type sig struct {
		key  string
		val  float64
		meta map[string]any
	}
	var ss []sig
	for rows.Next() {
		var x sig
		var metaB []byte
		if err := rows.Scan(&x.key, &x.val, &metaB); err != nil {
			rows.Close()
			return
		}
		x.meta = map[string]any{}
		_ = json.Unmarshal(metaB, &x.meta)
		ss = append(ss, x)
	}
	rows.Close()
	for _, x := range ss {
		if math.Abs(x.val) < th {
			continue
		}
		r.xvenueCarryCard(ctx, x.key, x.val, x.meta)
	}
}

// xvenueCarryCard 立卡+评估（UNIQUE(venue,surface) 幂等——同面待审卡在则只
// 刷评估行不重立卡）。
func (r *Runner) xvenueCarryCard(ctx context.Context, okxInst string, diff float64, meta map[string]any) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_hypothesis
		WHERE venue='okx' AND surface='xvenue-carry' AND family='xvenue_carry'
		AND evidence->>'inst'=$1`, okxInst).Scan(&n); err != nil {
		r.rej.Inc("hypo_write_err")
		return
	}
	if n == 0 {
		prims, _ := json.Marshal([]string{"okx_perp_long", "okx_perp_short", "mt5_cfd_short", "mt5_cfd_long"})
		reqs, _ := json.Marshal(map[string]any{
			"evaluator":  "跨所净年化四件分解（funding_ann/swap_ann/spread/滑点）",
			"risk":       []string{"min_net_ann_bps", "spread_amort_hold_h"},
			"executors":  []string{"okx_perp", "mt5_cfd"}, // 双腿 executor——真执行 T-0145
			"data_needs": []string{"carry_diff", "xvenue_spread", "mt5_swap", "depth5", "md_ticks mt5"}})
		ev, _ := json.Marshal(map[string]any{
			"provenance": "hypothesis_officer", "inst": okxInst,
			"trigger": "carry_diff_signal", "trigger_strength": diff,
			"signals":       []Sig{{Kind: "carry_diff", Key: okxInst, Val: diff, Meta: meta}},
			"direction":     directionOf(diff), // 负 diff=长 perp 收 funding+空 CFD（正主）
			"surface_basis": "md_signal 新鲜信号（T-0144 信号驱动——非面差枚举）"})
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_hypothesis
			(venue,surface,family,leg_primitives,requirements,evidence)
			VALUES('okx','xvenue-carry','xvenue_carry',$1::jsonb,$2::jsonb,$3::jsonb)
			ON CONFLICT (venue,surface) DO NOTHING`, prims, reqs, ev); err != nil {
			r.rej.Inc("hypo_write_err")
			log.Printf("[research/hypo] xvenue_carry 卡写败: %v", err)
			return
		}
		log.Printf("[research/hypo] xvenue_carry 卡立 %s（diff=%.0fbps——待决策席审）", okxInst, diff)
	}
	r.xvenueCarryEval(ctx, okxInst, diff, meta)
}

// directionOf 腿向语义（票面钦定钉）：大负 carry_diff=负 funding→长 OKX perp
// 收 funding+空 MT5 CFD 对冲（正主）；大正=反向（空 perp+长 CFD——swap_long
// 成本入账由评估如实算）。
func directionOf(diff float64) string {
	if diff < 0 {
		return "long_perp_short_cfd"
	}
	return "short_perp_long_cfd"
}

// xvenueCarryEval 净年化四件分解评估（fin_opp_eval template='xvenue-carry'）：
// 净年化 = −carry_diff（funding_ann−swap_ann 合成端）− spread 摊销 − 滑点缓冲；
// 容量三件 = min(okx depth5 顶档可吸, mt5 contract_size×max_volume 名义,
// RequiredMargin 等效=名义×initial_margin 率可承)；verdict 双闸 = okx 腿
// venue_untradeable 黑名单滤 + MT5 腿 trade-order 面在册。
func (r *Runner) xvenueCarryEval(ctx context.Context, okxInst string, diff float64, meta map[string]any) {
	fa := numMeta(meta, "funding_ann_bps")
	sa := numMeta(meta, "mt5_swap_ann_bps")
	mt5Sym, _ := meta["mt5_symbol"].(string)
	holdH := 72.0 // spread 摊销持有窗（guard.max_hold_hours 热启）
	if v, err := r.cfg.GetFloat("guard.max_hold_hours"); err == nil && v > 0 {
		holdH = v
	}
	buffer := 8.0 // 双腿滑点缓冲 bps（research.funding.taker_fee_bps×2 腿近似）
	if v, err := r.cfg.GetFloat("research.funding.taker_fee_bps"); err == nil && v > 0 {
		buffer = v * 2
	}
	// spread 摊销：进出双边点差×2（开+平）按持有窗年化占比。
	var spread float64
	sErr := r.pool.QueryRow(ctx, `SELECT val FROM md_signal
		WHERE kind='xvenue_spread' AND key=$1 AND ts > now()-interval '120 seconds'`, okxInst).Scan(&spread)
	spreadAmort := 0.0
	if sErr == nil {
		spreadAmort = math.Abs(spread) * 2 * (holdH / (365 * 24))
	}
	netAnn := -diff - spreadAmort - buffer
	// 容量三件。
	capacity := 0.0
	var depthTop float64
	dErr := r.pool.QueryRow(ctx, `SELECT meta->'bid_usd'->0 FROM md_signal
		WHERE kind='depth5' AND key=$1 AND ts > now()-interval '120 seconds'`, okxInst).Scan(&depthTop)
	if dErr == nil && depthTop > 0 {
		capacity = depthTop * 0.5 // 顶档可吸保守折半
	}
	if cs, ok := meta["contract_size"].(float64); ok && cs > 0 {
		if mv, ok2 := meta["max_volume"].(float64); ok2 && mv > 0 {
			var mid float64
			if e := r.pool.QueryRow(ctx, `SELECT (bid+ask)/2 FROM md_ticks
				WHERE venue='mt5' AND inst_id=$1 AND bid>0 AND ask>0`, mt5Sym).Scan(&mid); e == nil && mid > 0 {
				capMt5 := mid * cs * mv // MT5 名义上限
				capacity = minCapacity(capacity, capMt5)
			}
		}
	}
	// verdict 双闸。
	verdict, reason := "proposed", ""
	if capacity <= 0 {
		verdict, reason = "rejected", "capacity_missing"
	}
	var banned int
	if e := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_untradeable
		WHERE inst_id=$1`, okxInst).Scan(&banned); e == nil && banned > 0 {
		verdict, reason = "rejected", "okx_leg_untradeable"
	}
	var mt5Exec int
	if e := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_capability_surface
		WHERE venue='mt5' AND surface='trade-order' AND enabled`).Scan(&mt5Exec); e != nil || mt5Exec == 0 {
		verdict, reason = "rejected", "mt5_leg_capability_missing"
	}
	if netAnn < 0 && verdict == "proposed" {
		verdict, reason = "rejected", "net_ann_negative"
	}
	// 提案步进（proposed verdict 才立——评估先于执行能力，看见但做不了如实）。
	ev, _ := json.Marshal(map[string]any{
		"funding_ann_bps": fa, "swap_ann_bps": sa,
		"spread_bps": spread, "spread_amort_bps": spreadAmort,
		"slippage_buffer_bps": buffer, "net_ann_bps": netAnn,
		"capacity_usd": capacity, "direction": directionOf(diff),
		"mt5_symbol": mt5Sym, "reason": reason})
	var evalID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
		VALUES($1,'xvenue-carry',$2,NULL,$3,$4,'[]',$5)
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		net_apr=EXCLUDED.net_apr, verdict=EXCLUDED.verdict, evidence=EXCLUDED.evidence
		RETURNING id`, okxInst, directionOf(diff), netAnn/10000, verdict, ev).Scan(&evalID); err != nil {
		r.rej.Inc("hypo_write_err")
		log.Printf("[research/hypo] xvenue_carry 评估写败: %v", err)
		return
	}
	if verdict == "proposed" {
		var n int
		if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
			WHERE eval_id=$1`).Scan(&n); err == nil && n == 0 {
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
				VALUES($1,$2,$3,$4)`, evalID, okxInst, directionOf(diff), netAnn/10000); err != nil {
				r.rej.Inc("hypo_write_err")
				return
			}
			log.Printf("[research/hypo] xvenue_carry 提案立 %s（净年化 %.0fbps 容量≈%.0fUSD——fin_proposed 止，真执行 T-0145）",
				okxInst, netAnn, capacity)
		}
	} else {
		log.Printf("[research/hypo] xvenue_carry 评估 %s=%s/%s（净 %.0fbps 容量≈%.0fUSD——无机会/做不了如实）",
			okxInst, verdict, reason, netAnn, capacity)
	}
}

// numMeta meta 浮点安全取。
func numMeta(m map[string]any, k string) float64 {
	v, _ := m[k].(float64)
	return v
}

// minCapacity 容量取小。
func minCapacity(a, b float64) float64 {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

// 编译期锚：pool/config/rejects 引用面（xcarry 归 Runner 域——域内件）。
var _ = func(r *Runner) func(context.Context, *pgxpool.Pool) { return nil }(nil)
var _ = fmt.Stringer(nil)
var _ = rejects.New
var _ = config.New
