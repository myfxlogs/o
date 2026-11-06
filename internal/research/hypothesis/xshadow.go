// xshadow.go —— 影子评估层消费臂（T-0408④）：xvenue_shadow_carry 鲜行→同
// 数学净年化（carryNetK——真/影共享件）→fin_opp_eval 三态判词
// （template='xvenue-carry-shadow' 可辨——禁混真评估/真开闸径，真消费者
// template 精确键天然免疫）。
//
//	判词真（proposed）→condreq「map 闸建议」携评估证据呈业主
//	  （fin_condition_gap kind=permission surface=bridge.xvenue.map.shadow
//	  ——bridge.xvenue.map owner 域不破：只呈不写；批→落 map→既有
//	  onboard map 闸→paper 径；onboard 草案 pending 异 surface 零撞）；
//	判词假（rejected/unevaluable）=零打扰留账可查——零闸动零提案。
//
// 提名≠判词≠落闸：本臂只出判词+呈建议——判词真也不触 map/提案/策略。
package hypothesis

import (
	"context"
	"encoding/json"
	"log"

	"o/internal/platform/condreq"
	"o/internal/platform/execfam"
)

// shadowTemplate 影域评估模板族码（fin_opp_eval.template——真/影可辨键）。
const shadowTemplate = "xvenue-carry-shadow"

// xvenueShadowScan 影子评估臂一拍（ScanSignals 尾调）：xvenue_shadow_carry
// 鲜行（8h 源节拍界——最慢源 mt5_swap 4h 拍同根）→逐对判词。零阈拦——
// 影子域职责=全候选过筛（判词真才呈，判词假留账——筛是评估本职非阈）。
func (r *Runner) xvenueShadowScan(ctx context.Context) {
	rows, err := r.pool.Query(ctx, `SELECT key, val, meta FROM md_signal
		WHERE kind='xvenue_shadow_carry' AND ts > now()-interval '8 hours'`)
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
		dir, net, feasible := r.carryNetK(ctx, x.key, x.val, x.meta, "xvenue_shadow_spread")
		r.xvenueShadowEval(ctx, x.key, dir, net, x.val, x.meta, feasible)
	}
}

// xvenueShadowEval 影域判词落行（carryVerdict 同核——mid 快照回落供名义
// 容量；数据缺=unevaluable 非 rejected——depth5 未采≠簿薄）。
// proposed→condreq 呈批（不落闸）；rejected/unevaluable→零闸动。
func (r *Runner) xvenueShadowEval(ctx context.Context, okxInst, dir string, net, diff float64, meta map[string]any, feasible bool) {
	fa := numMeta(meta, "funding_ann_bps")
	sa := numMeta(meta, "mt5_swap_ann_bps")
	mt5Sym, _ := meta["mt5_symbol"].(string)
	verdict, reason, capacity, spread, spreadAmort, buffer := r.carryVerdict(
		ctx, okxInst, dir, net, meta, feasible, "xvenue_shadow_spread",
		numMeta(meta, "mid"), true)
	ev, _ := json.Marshal(map[string]any{
		"funding_ann_bps": fa, "swap_ann_bps": sa,
		"swap_long_ann_bps": numMeta(meta, "swap_long_ann_bps"),
		"spread_bps":        spread, "spread_amort_bps": spreadAmort,
		"slippage_buffer_bps": buffer, "net_ann_bps": net,
		"capacity_usd": capacity, "direction": dir,
		"mt5_symbol": mt5Sym, "reason": reason,
		"shadow": true, "candidate": true})
	var evalID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
		VALUES($1,$5,$2,NULL,$3,$4,'[]',$6)
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		net_apr=EXCLUDED.net_apr, verdict=EXCLUDED.verdict, evidence=EXCLUDED.evidence
		RETURNING id`, okxInst, dir, net/10000, verdict, shadowTemplate, ev).Scan(&evalID); err != nil {
		r.rej.Inc("hypo_write_err")
		log.Printf("[research/hypo] xvenue_shadow 评估写败: %v", err)
		return
	}
	if verdict != "proposed" {
		log.Printf("[research/hypo] xvenue_shadow 评估 %s=%s/%s（净 %.0fbps——留账零闸动）",
			okxInst, verdict, reason, net)
		return
	}
	// 判词真→condreq「map 闸建议」携证据呈业主（owner 域只呈不写）。
	if r.fin == nil {
		r.rej.Inc("hypo_write_err")
		log.Printf("WARN [research/hypo] xvenue_shadow %s proposed 而 fin 池未挂——呈批径缺位（评估行已落账）", okxInst)
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"venue": "mt5", "suggested_pairs": map[string]string{okxInst: mt5Sym},
		"direction": dir, "net_ann_bps": net, "capacity_usd": capacity,
		"eval_id": evalID, "eval_template": shadowTemplate,
		"note": "影子评估层 map 闸建议（T-0408）——批→落 bridge.xvenue.map→既有 map 闸→paper 径；影子层只呈不写（owner 域不破）"})
	if err := condreq.Request(ctx, r.fin, "permission", "mt5",
		"bridge.xvenue.map.shadow", string(payload)); err != nil {
		r.rej.Inc("hypo_write_err")
		log.Printf("WARN [research/hypo] xvenue_shadow %s condreq 写败: %v", okxInst, err)
		return
	}
	log.Printf("[research/hypo] xvenue_shadow %s=%s（净 %.0fbps 容量≈%.0fUSD）→condreq map 闸建议呈业主（不落闸）",
		okxInst, mt5Sym, net, capacity)
}

// carryVerdict 判词计算核（真/影同一评估器——T-0408 防冗余共享；闸序
// feasible→容量三件→黑名单→mt5 能力面→净负→exec 族）。
//
//	missUneq=true=影域数据缺不可判（depth5 未采≠簿薄）→unevaluable；
//	false=真域 rejected 原义不动。
//	midFallback=影域目录快照 mid（meta.mid——tick 缺时名义容量源；真域
//	传 0=行为不动）。
//
// 返 verdict/reason/capacity/spread/spreadAmort/buffer（evidence 四件由
// 调用方组——模板/呈批径域差在写侧非判词核）。
func (r *Runner) carryVerdict(ctx context.Context, okxInst, dir string, net float64,
	meta map[string]any, feasible bool, spreadKind string, midFallback float64,
	missUneq bool) (verdict, reason string, capacity, spread, spreadAmort, buffer float64) {
	mt5Sym, _ := meta["mt5_symbol"].(string)
	spread, _ = r.spreadValK(ctx, okxInst, spreadKind) // 缺席=0 值如实
	spreadAmort = r.spreadAmortBPSK(ctx, okxInst, spreadKind)
	buffer = r.slippageBufferBPS()
	// 容量三件。
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
		} else if midFallback > 0 {
			capacity = minCapacity(capacity, midFallback*cs*mv) // 影域快照 mid
		}
	}
	// verdict 闸序。
	verdict, reason = "proposed", ""
	switch {
	case !feasible:
		verdict, reason = "rejected", "net_ann_economics"
	case capacity <= 0:
		if missUneq {
			verdict, reason = "unevaluable", "capacity_missing"
		} else {
			verdict, reason = "rejected", "capacity_missing"
		}
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
	// T-0330 前置拒：执行径未建族——xvenue-carry armed→open 显式禁（T-0145
	// 未派案），全谱过阈也 direct rejected/exec_family_missing 如实，不进
	// 提案队列淤积等死（族码=本行 template 列；影域同守——判词真≠可执行）。
	if verdict == "proposed" && !execfam.Has("xvenue-carry") {
		verdict, reason = "rejected", "exec_family_missing"
	}
	return verdict, reason, capacity, spread, spreadAmort, buffer
}
