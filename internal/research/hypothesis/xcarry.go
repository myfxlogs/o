// xcarry.go —— xvenue_carry 跨所 carry 臂（T-0144/D-0022 终态件；返修归正
// 双腿经济学）：OKX perp × MT5 CFD 对冲 carry 首个真跨所模板。
//
//	负 diff 径（正主）：长 OKX perp 收 funding+空 MT5 CFD（swap_short=0 零成本）
//	  →net=−diff−spread 摊销−缓冲；
//	正 diff 径：空 OKX perp 收 funding+长 MT5 CFD——swap_long 成本端入账
//	  （meta.swap_long_ann_bps，返修①双腿年化源）
//	  →net=+fundingAnn−|swap_long_ann|−成本；swap 成本吃光 funding=net<0
//	  →评估 rejected/economics 如实不撞哑（卡不起草——返修④方向感知阈）。
//
// 卡条件=方向感知净阈（|net|≥th 才起草）。链环止于提案 fin_proposed；
// paper 双腿 lifecycle 径；真执行=T-0145。
package hypothesis

import (
	"context"
	"encoding/json"
	"log"
	"math"
)

// xvenueCarryScan 信号臂 xvenue_carry 支路（ScanSignals 尾调）：carry_diff
// 鲜信号→方向感知净年化→三态（立卡+评估/仅评估拒/零动作）。
func (r *Runner) xvenueCarryScan(ctx context.Context) {
	th := 300.0 // 净年化阈 bps（research.signal.xvenue_carry_diff_bps 热启）
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
		dir, net, feasible := r.xvenueCarryNet(ctx, x.key, x.val, x.meta)
		switch {
		case !feasible:
			// 正 diff 径经济学不通（swap_long 成本≥funding）——评估 rejected
			// 如实不撞哑，卡不起草（返修④）。
			r.xvenueCarryEval(ctx, x.key, dir, net, x.val, x.meta, false)
		case math.Abs(net) >= th:
			r.xvenueCarryCard(ctx, x.key, x.val, x.meta, dir, net)
		default: // 阈内零动作
		}
	}
}

// xvenueCarryNet 方向感知净年化（返修③分径）：返 (direction, net, feasible)。
//
//	diff<0（正主）：net=−diff−spreadAmort−buffer（恒可行——负 diff 即正收入端）；
//	diff>0：net=+fundingAnn−|swap_long_ann|−spreadAmort−buffer——swap_long
//	  年化缺（meta 无双腿年化源=桥旧版）=不可判 false；<0=经济学不通 false。
func (r *Runner) xvenueCarryNet(ctx context.Context, okxInst string, diff float64, meta map[string]any) (string, float64, bool) {
	spreadAmort := r.spreadAmortBPS(ctx, okxInst)
	buffer := r.slippageBufferBPS()
	if diff < 0 {
		return "long_perp_short_cfd", -diff - spreadAmort - buffer, true
	}
	dir := "short_perp_long_cfd"
	fa := numMeta(meta, "funding_ann_bps")
	lraw, has := meta["swap_long_ann_bps"]
	if !has {
		return dir, 0, false // 双腿年化源缺——不可判如实
	}
	longAnn, _ := lraw.(float64)
	net := fa - math.Abs(longAnn) - spreadAmort - buffer
	if net < 0 {
		return dir, net, false
	}
	return dir, net, true
}

// spreadAmortBPS 进出双边点差按持有窗年化占比摊销（max_hold 热启）；
// 点差缺席=摊销零（evidence spread_bps 缺席如实——非成本低估面）。
func (r *Runner) spreadAmortBPS(ctx context.Context, okxInst string) float64 {
	holdH := 72.0
	if v, err := r.cfg.GetFloat("guard.max_hold_hours"); err == nil && v > 0 {
		holdH = v
	}
	var spread float64
	if err := r.pool.QueryRow(ctx, `SELECT val FROM md_signal
		WHERE kind='xvenue_spread' AND key=$1 AND ts > now()-interval '120 seconds'`, okxInst).Scan(&spread); err != nil {
		return 0
	}
	return math.Abs(spread) * 2 * (holdH / (365 * 24))
}

// slippageBufferBPS 双腿滑点缓冲（taker×2 腿热启）。
func (r *Runner) slippageBufferBPS() float64 {
	if v, err := r.cfg.GetFloat("research.funding.taker_fee_bps"); err == nil && v > 0 {
		return v * 2
	}
	return 8.0
}

// xvenueCarryCard 立卡+评估（UNIQUE(venue,surface) 幂等——同面待审卡在则只
// 刷评估行不重立卡）。
func (r *Runner) xvenueCarryCard(ctx context.Context, okxInst string, diff float64, meta map[string]any, dir string, net float64) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_hypothesis
		WHERE venue='okx' AND surface='xvenue-carry' AND family='xvenue_carry'
		AND evidence->>'inst'=$1`, okxInst).Scan(&n); err != nil {
		r.rej.Inc("hypo_write_err")
		return
	}
	if n == 0 {
		prims, _ := json.Marshal(legVocab["xvenue_carry"])
		reqs, _ := json.Marshal(reqTemplates["xvenue_carry"])
		ev, _ := json.Marshal(map[string]any{
			"provenance": "hypothesis_officer", "inst": okxInst,
			"trigger": "carry_diff_signal", "trigger_strength": diff,
			"signals":       []Sig{{Kind: "carry_diff", Key: okxInst, Val: diff, Meta: meta}},
			"direction":     dir,
			"net_ann_bps":   net,
			"surface_basis": "md_signal 新鲜信号（T-0144 信号驱动——非面差枚举）"})
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_hypothesis
			(venue,surface,family,leg_primitives,requirements,evidence)
			VALUES('okx','xvenue-carry','xvenue_carry',$1::jsonb,$2::jsonb,$3::jsonb)
			ON CONFLICT (venue,surface) DO NOTHING`, prims, reqs, ev); err != nil {
			r.rej.Inc("hypo_write_err")
			log.Printf("[research/hypo] xvenue_carry 卡写败: %v", err)
			return
		}
		log.Printf("[research/hypo] xvenue_carry 卡立 %s（diff=%.0fbps 净=%.0fbps——待决策席审）", okxInst, diff, net)
	}
	r.xvenueCarryEval(ctx, okxInst, dir, net, diff, meta, true)
}

// xvenueCarryEval 净年化四件分解评估（fin_opp_eval template='xvenue-carry'）：
// verdict 闸序 = 经济学可行→容量三件（okx depth5 顶档×0.5 / mt5 contract_size
// ×max_volume 名义）→okx 腿黑名单→MT5 腿 trade-order 面在册→净负；
// feasible=false（经济学不通）=rejected/net_ann_economics 如实。
func (r *Runner) xvenueCarryEval(ctx context.Context, okxInst, dir string, net, diff float64, meta map[string]any, feasible bool) {
	fa := numMeta(meta, "funding_ann_bps")
	sa := numMeta(meta, "mt5_swap_ann_bps")
	mt5Sym, _ := meta["mt5_symbol"].(string)
	spread, sErr := r.spreadVal(ctx, okxInst)
	spreadAmort := r.spreadAmortBPS(ctx, okxInst)
	buffer := r.slippageBufferBPS()
	// 容量三件。
	capacity := 0.0
	var depthTop float64
	if err := r.pool.QueryRow(ctx, `SELECT meta->'bid_usd'->>0 FROM md_signal
		WHERE kind='depth5' AND key=$1 AND ts > now()-interval '120 seconds'`, okxInst).Scan(&depthTop); err == nil && depthTop > 0 {
		capacity = depthTop * 0.5 // 顶档可吸保守折半
	}
	cs := numMeta(meta, "contract_size")
	mv := numMeta(meta, "max_volume")
	if cs > 0 && mv > 0 {
		var mid float64
		if e := r.pool.QueryRow(ctx, `SELECT (bid+ask)/2 FROM md_ticks
			WHERE venue='mt5' AND inst_id=$1 AND bid>0 AND ask>0`, mt5Sym).Scan(&mid); e == nil && mid > 0 {
			capacity = minCapacity(capacity, mid*cs*mv) // MT5 名义上限
		}
	}
	// verdict 闸序。
	verdict, reason := "proposed", ""
	switch {
	case !feasible:
		verdict, reason = "rejected", "net_ann_economics"
	case capacity <= 0:
		verdict, reason = "rejected", "capacity_missing"
	default:
		var banned int
		if e := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_untradeable
			WHERE inst_id=$1`, okxInst).Scan(&banned); e == nil && banned > 0 {
			verdict, reason = "rejected", "okx_leg_untradeable"
		}
	}
	if verdict == "proposed" {
		var mt5Exec int
		if e := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_capability_surface
			WHERE venue='mt5' AND surface='trade-order' AND enabled`).Scan(&mt5Exec); e != nil || mt5Exec == 0 {
			verdict, reason = "rejected", "mt5_leg_capability_missing"
		} else if net < 0 {
			verdict, reason = "rejected", "net_ann_negative"
		}
	}
	if sErr != nil {
		spread = 0 // 点差缺席——evidence 零值如实
	}
	ev, _ := json.Marshal(map[string]any{
		"funding_ann_bps": fa, "swap_ann_bps": sa,
		"swap_long_ann_bps": numMeta(meta, "swap_long_ann_bps"),
		"spread_bps":        spread, "spread_amort_bps": spreadAmort,
		"slippage_buffer_bps": buffer, "net_ann_bps": net,
		"capacity_usd": capacity, "direction": dir,
		"mt5_symbol": mt5Sym, "reason": reason})
	var evalID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
		VALUES($1,'xvenue-carry',$2,NULL,$3,$4,'[]',$5)
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		net_apr=EXCLUDED.net_apr, verdict=EXCLUDED.verdict, evidence=EXCLUDED.evidence
		RETURNING id`, okxInst, dir, net/10000, verdict, ev).Scan(&evalID); err != nil {
		r.rej.Inc("hypo_write_err")
		log.Printf("[research/hypo] xvenue_carry 评估写败: %v", err)
		return
	}
	if verdict == "proposed" {
		var n int
		if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
			WHERE eval_id=$1`).Scan(&n); err == nil && n == 0 {
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
				VALUES($1,$2,$3,$4)`, evalID, okxInst, dir, net/10000); err != nil {
				r.rej.Inc("hypo_write_err")
				return
			}
			log.Printf("[research/hypo] xvenue_carry 提案立 %s（净年化 %.0fbps 容量≈%.0fUSD——fin_proposed 止，真执行 T-0145）",
				okxInst, net, capacity)
		}
	} else {
		log.Printf("[research/hypo] xvenue_carry 评估 %s=%s/%s（净 %.0fbps 容量≈%.0fUSD——无机会/做不了如实）",
			okxInst, verdict, reason, net, capacity)
	}
}

// spreadVal 点差现值鲜读（评估 evidence 逐环实值）。
func (r *Runner) spreadVal(ctx context.Context, okxInst string) (float64, error) {
	var v float64
	err := r.pool.QueryRow(ctx, `SELECT val FROM md_signal
		WHERE kind='xvenue_spread' AND key=$1 AND ts > now()-interval '120 seconds'`, okxInst).Scan(&v)
	return v, err
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
