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
	// T-0383 消费闸根修：carry_diff 鲜度界=源节拍感知非统一 120s——行 ts=
	// 最旧上游观测（T-0164 血统闸语义不动，改的是消费判据非写侧戳），
	// 最慢源=mt5_swap 4h 拍（internal/mt5/bridge.go swapEvery），界=拍×2
	// 安全边=8h，与读侧闸同根对齐（sigdist/xvenue.go mt5_swap 8h 闸、
	// stakecarry.go 同、ops_face 巡检 warn5h/dead9h 在案）——派生径如此
	// 非魔法数。原 120s 对 4h 源=恒陈视缺席（全史 3 评皆撞运气窗实证）。
	// 残险随行：swap 年化材料最陈≈4h（源拍固有——评估读源快照非决策刻
	// 现值；提速须动源拍非闸值，净阈判据容此陈险）。
	rows, err := r.pool.Query(ctx, `SELECT key, val, meta FROM md_signal
		WHERE kind='carry_diff' AND ts > now()-interval '8 hours'`)
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
			// T-0401 晋级径：过阈候选符 ∉ activeSet→临时订阅请求（流式 tick
			// 拾取供确认级评估/下游记账；判词出闭请退订或提案常驻接管）。
			r.nominateSub(ctx, x.key, "xvenue-carry", "carry_diff_threshold")
			r.xvenueCarryCard(ctx, x.key, x.val, x.meta, dir, net)
		default: // 阈内零动作
		}
	}
}

// xvenueCarryNet 方向感知净年化（真域径——spread kind=真行）。返
// (direction, net, feasible)。T-0408 约定漂移根修：T-0203 产者口径已翻
// 「当值方向净 carry（正=好）+meta.direction 钉」，消费侧旧约「diff<0=
// 正收入端」随漂——funding<0 主径判词整体反置（赢对 reject/输对可
// proposed 活 bug，fixture 手植旧约行自洽掩护）。新约：dir=meta.direction
// （产者钉）；net=diff−spreadAmort−buffer（val 已含本腿 swap 成本——禁
// 双重计）；feasible=false=方向钉缺/diff<0 经济学不通。
func (r *Runner) xvenueCarryNet(ctx context.Context, okxInst string, diff float64, meta map[string]any) (string, float64, bool) {
	return r.carryNetK(ctx, okxInst, diff, meta, "xvenue_spread")
}

// carryNetK 方向感知净年化核（真/影同数学——spread kind 区分域）。
func (r *Runner) carryNetK(ctx context.Context, okxInst string, diff float64, meta map[string]any, spreadKind string) (string, float64, bool) {
	spreadAmort := r.spreadAmortBPSK(ctx, okxInst, spreadKind)
	buffer := r.slippageBufferBPS()
	dir, _ := meta["direction"].(string)
	switch dir {
	case "long_perp_short_cfd", "short_perp_long_cfd":
	default:
		return dir, 0, false // 方向钉缺（产者口径前行/形坏）——不可判如实
	}
	net := diff - spreadAmort - buffer
	if diff < 0 {
		return dir, net, false // 方向净 carry 已含本腿成本——负=经济学不通
	}
	return dir, net, true
}

// spreadAmortBPSK 点差摊销核（max_hold 热启；kind 区分真/影域——点差
// 缺席=摊销零（evidence spread_bps 缺席如实——非成本低估面））。
func (r *Runner) spreadAmortBPSK(ctx context.Context, okxInst, kind string) float64 {
	holdH := 72.0
	if v, err := r.cfg.GetFloat("guard.max_hold_hours"); err == nil && v > 0 {
		holdH = v
	}
	var spread float64
	if err := r.pool.QueryRow(ctx, `SELECT val FROM md_signal
		WHERE kind=$1 AND key=$2 AND ts > now()-CASE $1
			WHEN 'xvenue_spread' THEN interval '120 seconds'
			ELSE interval '8 hours' END`, kind, okxInst).Scan(&spread); err != nil {
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
// 判词核=carryVerdict（xshadow.go——真/影同一评估器防冗余）。
func (r *Runner) xvenueCarryEval(ctx context.Context, okxInst, dir string, net, diff float64, meta map[string]any, feasible bool) {
	fa := numMeta(meta, "funding_ann_bps")
	sa := numMeta(meta, "mt5_swap_ann_bps")
	mt5Sym, _ := meta["mt5_symbol"].(string)
	verdict, reason, capacity, spread, spreadAmort, buffer := r.carryVerdict(
		ctx, okxInst, dir, net, meta, feasible, "xvenue_spread", 0, false)
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
		// T-0377 断链根修⑥：count 查询声明 $1 却未传参（pgx expected 1 got 0）——
		// err≠nil 静默跳过提案 INSERT，xvenue-carry 提案永零（登记前被前置拒
		// 掩护不可见）。补传 evalID 恢复提案径；err 面显式留痕不再吞。
		if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
			WHERE eval_id=$1`, evalID).Scan(&n); err != nil {
			r.rej.Inc("hypo_write_err")
			log.Printf("[research/hypo] xvenue_carry 提案幂等查败: %v", err)
		} else if n == 0 {
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
	if verdict == "rejected" {
		r.closeSub(ctx, okxInst, "xvenue-carry") // T-0401：判词拒=退订
	}
}

// spreadValK 点差现值鲜读核（kind 区分真/影域——影域帧龄界 8h 源节拍同根）。
func (r *Runner) spreadValK(ctx context.Context, okxInst, kind string) (float64, error) {
	var v float64
	err := r.pool.QueryRow(ctx, `SELECT val FROM md_signal
		WHERE kind=$1 AND key=$2 AND ts > now()-CASE $1
			WHEN 'xvenue_spread' THEN interval '120 seconds'
			ELSE interval '8 hours' END`, kind, okxInst).Scan(&v)
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
