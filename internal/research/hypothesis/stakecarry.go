// stakecarry.go —— stake_carry 双头 carry 枚举臂（T-0320 标定案）：能力面
// 二维积枚举——product_family='staking'（可质押资产，左元）×'swap-carry'
// （空单付费 venue，右元）按 ccy join→双头净年化→三态（阈亮=卡+评估+提案
// 止 fin_proposed；阈内=评估 rejected 如实；数据缺席=该对不评如实）。
// 业主 TRX 自营盘结构同构（现货质押收息+空单收 swap——cmc-trx-vs-company
// 案 canon）：本臂=发现层对该族结构的自动复生径。
package hypothesis

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"o/internal/platform/execfam"
	"o/internal/research/funding"
)

// stakeCarryScan 双头臂一拍（ScanSignals 尾调——与 xvenueCarryScan 同槽）。
func (r *Runner) stakeCarryScan(ctx context.Context) {
	th := 300.0 // 净年化阈 bps（research.signal.stake_carry_net_bps 热启）
	if v, err := r.cfg.GetFloat("research.signal.stake_carry_net_bps"); err == nil && v > 0 {
		th = v
	}
	buf := 50.0 // 双腿摩擦缓冲 bps（research.signal.stake_carry_buffer_bps 热启）
	if v, err := r.cfg.GetFloat("research.signal.stake_carry_buffer_bps"); err == nil && v > 0 {
		buf = v
	}
	staleH := 0.0 // 质押率行龄门（research.stake.stale_hours；<=0=不闸）
	if v, err := r.cfg.GetFloat("research.stake.stale_hours"); err == nil && v > 0 {
		staleH = v
	}
	// 左元：可质押资产集（staking 维注册行——stakable 旗+率源字段）。
	rows, err := r.pool.Query(ctx, `SELECT cost_attrs->>'ccy' FROM fin_capability_surface
		WHERE product_family='staking' AND enabled
		  AND (cost_attrs->>'stakable')::boolean`)
	if err != nil {
		r.rej.Inc("sig_write_err")
		return
	}
	var ccys []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err == nil && c != "" {
			ccys = append(ccys, c)
		}
	}
	rows.Close()
	for _, ccy := range ccys {
		sy, ok, err := funding.SpotYield(ctx, r.pool, ccy, staleH)
		if err != nil {
			r.rej.Inc("sig_write_err")
			continue
		}
		if !ok || sy.APR <= 0 {
			continue // 质押率源缺席——该资产不枚举如实
		}
		// 右元：该资产空单付费 venue 面（swap-carry 维注册行）。
		crows, err := r.pool.Query(ctx, `SELECT venue, surface, cost_attrs FROM fin_capability_surface
			WHERE product_family='swap-carry' AND enabled AND cost_attrs->>'ccy'=$1`, ccy)
		if err != nil {
			r.rej.Inc("sig_write_err")
			continue
		}
		type cv struct {
			venue, surface string
			attrs          map[string]any
		}
		var cs []cv
		for crows.Next() {
			var x cv
			var ab []byte
			if err := crows.Scan(&x.venue, &x.surface, &ab); err != nil {
				continue
			}
			x.attrs = map[string]any{}
			_ = json.Unmarshal(ab, &x.attrs)
			cs = append(cs, x)
		}
		crows.Close()
		for _, c := range cs {
			shortBps, meta, ok := r.shortCarryAnn(ctx, c.venue, c.attrs)
			if !ok || shortBps <= 0 {
				continue // 空单付费面数据缺席/非正——该对不枚举如实
			}
			net := sy.APR*1e4 + shortBps - buf
			r.stakeCarryEval(ctx, c.venue, ccy, sy, shortBps, meta, buf, net, th)
		}
	}
}

// shortCarryAnn 空单付费年化派发（rate_source 闭集——venue 方言各归其源）：
//
//	cmc_realized = CMC 实收隔夜结算（fin_cmc_history OVERNIGHT_SETTLEMENT 30d
//	               日均 /fin_cmc_positions 空腿名义×365——venue 实付非报价）；
//	mt5_swap     = md_signal mt5_swap meta.swap_short_ann_bps（桥采 SymbolParams）；
//	funding_view = md_signal funding_view val>0（okx 空 perp 收 funding 年化）。
//
// 表域缺席（42P01）/行缺/非正=ok=false 如实。
func (r *Runner) shortCarryAnn(ctx context.Context, venue string, attrs map[string]any) (float64, map[string]any, bool) {
	src, _ := attrs["rate_source"].(string)
	switch src {
	case "cmc_realized":
		inst, _ := attrs["instrument"].(string)
		return r.cmcRealizedSwapAnn(ctx, inst)
	case "mt5_swap":
		sym, _ := attrs["symbol"].(string)
		var bps float64
		var metaB []byte
		if err := r.pool.QueryRow(ctx, `SELECT meta->>'swap_short_ann_bps', meta FROM md_signal
			WHERE kind='mt5_swap' AND key=$1 AND ts>now()-interval '8 hours'`, sym).Scan(&bps, &metaB); err != nil {
			return 0, nil, false
		}
		meta := map[string]any{}
		_ = json.Unmarshal(metaB, &meta)
		return bps, meta, bps > 0
	case "funding_view":
		inst, _ := attrs["inst"].(string)
		var rate float64
		if err := r.pool.QueryRow(ctx, `SELECT val FROM md_signal
			WHERE kind='funding_view' AND key=$1
			  AND (meta->>'next_funding_ts')::timestamptz > now()`, inst).Scan(&rate); err != nil {
			return 0, nil, false
		}
		ann := rate * 3 * 365 * 1e4 // 8h 期×3/日×365→bps（sigdist 同口径）
		return ann, map[string]any{"funding_rate": rate}, ann > 0
	default:
		return 0, nil, false // 率源未注册方言——缺席如实
	}
}

// cmcRealizedSwapAnn CMC 空单实付年化（30d 日均隔夜结算 USD / 空腿名义 USD
// ×365→bps；样本<3 日/名义缺=不可判缺席如实）。venue 实收序列=venue 付费
// 意愿的最硬证据（业主实盘同径）。
func (r *Runner) cmcRealizedSwapAnn(ctx context.Context, inst string) (float64, map[string]any, bool) {
	var daily float64
	var days int
	var lastTs time.Time
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(AVG(s),0), count(*), max(d) FROM (
		SELECT date_trunc('day',transaction_time) d, SUM(amount) s
		FROM fin_cmc_history
		WHERE transaction_type='OVERNIGHT_SETTLEMENT'
		  AND transaction_time>now()-interval '30 days'
		GROUP BY 1) t`).Scan(&daily, &days, &lastTs)
	if err != nil || days < 3 || daily <= 0 {
		return 0, nil, false
	}
	var qty, px float64
	if err := r.pool.QueryRow(ctx, `SELECT quantity, entry_price FROM fin_cmc_positions
		WHERE instrument_id=$1 AND direction='Sell'
		ORDER BY fetched_at DESC LIMIT 1`, inst).Scan(&qty, &px); err != nil || qty*px <= 0 {
		return 0, nil, false
	}
	bps := daily / (qty * px) * 365 * 1e4
	return bps, map[string]any{
		"instrument": inst, "daily_settle_usd": daily, "notional_usd": qty * px,
		"sample_days": days, "last_settle_ts": lastTs.UTC().Format(time.RFC3339),
		"source": "cmc_realized_30d"}, true
}

// stakeCarryEval 双头评估+卡+提案（UNIQUE(venue,surface) 幂等——同面对卡
// 不重生；评估行 (inst,template,direction) 同键最新赢）。
func (r *Runner) stakeCarryEval(ctx context.Context, venue, ccy string, sy funding.Yield,
	shortBps float64, carryMeta map[string]any, buf, net, th float64) {
	verdict, reason := "rejected", "net_below_min"
	if net >= th {
		verdict, reason = "proposed", ""
	}
	inst := venue + ":" + ccy
	if verdict == "proposed" {
		var n int
		if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_hypothesis
			WHERE venue=$1 AND surface=$2`, venue, "stake_carry-"+ccy).Scan(&n); err == nil && n == 0 {
			prims, _ := json.Marshal(legVocab["stake_carry"])
			reqs, _ := json.Marshal(reqTemplates["stake_carry"])
			ev, _ := json.Marshal(map[string]any{
				"provenance": "hypothesis_officer", "inst": inst,
				"trigger": "stake_carry_enum", "ccy": ccy, "venue": venue,
				"direction": "stake_carry", "net_ann_bps": net,
				"stake_apr_bps": sy.APR * 1e4, "stake_source": sy.Source,
				"short_ann_bps": shortBps, "carry": carryMeta,
				"surface_basis": "capability_surface staking×swap-carry 二维积（T-0320 标定——业主已验证结构同构）"})
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_hypothesis
				(venue,surface,family,leg_primitives,requirements,evidence)
				VALUES($1,$2,'stake_carry',$3::jsonb,$4::jsonb,$5::jsonb)
				ON CONFLICT (venue,surface) DO NOTHING`,
				venue, "stake_carry-"+ccy, prims, reqs, ev); err != nil {
				r.rej.Inc("sig_write_err")
				log.Printf("[research/hypo] stake_carry 卡写败 %s: %v", inst, err)
			} else {
				log.Printf("[research/hypo] stake_carry 卡立 %s（stake=%.0fbps+short=%.0fbps 净=%.0fbps——待决策席审）",
					inst, sy.APR*1e4, shortBps, net)
			}
		}
	}
	// T-0330 前置拒：执行径未建族——executor 缺（假想卡级族）direct
	// rejected/exec_family_missing 如实，不进提案队列；卡仍立（机会面
	// 呈决策席——未建族=看见了做不了，非无机会）。
	if verdict == "proposed" && !execfam.Has("stake-carry") {
		verdict, reason = "rejected", "exec_family_missing"
	}
	legs, _ := json.Marshal(map[string]any{"spot": "long_staked", "short": venue})
	evj, _ := json.Marshal(map[string]any{
		"ccy": ccy, "venue": venue, "stake_apr_bps": sy.APR * 1e4, "stake_source": sy.Source,
		"short_ann_bps": shortBps, "carry": carryMeta, "buffer_bps": buf,
		"threshold_bps": th, "net_ann_bps": net, "reason": reason})
	var evalID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
		VALUES($1,'stake-carry','stake_carry',$2,$3,$4,$5,$6)
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		gross_apr=EXCLUDED.gross_apr,net_apr=EXCLUDED.net_apr,verdict=EXCLUDED.verdict,
		legs=EXCLUDED.legs,evidence=EXCLUDED.evidence
		RETURNING id`, inst, (sy.APR*1e4+shortBps)/1e4, net/1e4, verdict, legs, evj).Scan(&evalID); err != nil {
		r.rej.Inc("sig_write_err")
		log.Printf("[research/hypo] stake_carry 评估写败 %s: %v", inst, err)
		return
	}
	if verdict != "proposed" {
		return
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE eval_id=$1`, evalID).Scan(&n); err == nil && n == 0 {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
			VALUES($1,$2,'stake_carry',$3)`, evalID, inst, net/1e4); err != nil {
			r.rej.Inc("sig_write_err")
			return
		}
		log.Printf("[research/hypo] stake_carry 提案立 %s（净年化 %.0fbps——fin_proposed 止，走 paper→probe→live 生命周期）",
			inst, net)
	}
}
