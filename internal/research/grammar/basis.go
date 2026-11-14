// basis.go —— 季度基差族评估拆分件（行数闸拆分——evalwrite.go/volsell.go
// 同款先例；T-0298 往返费用摊销+阈值入 config+符号面判据在本件）。
package grammar

import (
	"context"
	"log"
	"regexp"
	"time"

	"o/internal/platform/execfam"
	"o/internal/research/funding" // T-0298：basis 费用摊销+Earn 底同源（白名单边——禁第二份算式）
)

// datedFutRe 到期日尾判（-YYMMDD——T-0213 net_apr 年化径到期解析）。
var datedFutRe = regexp.MustCompile(`^([A-Z0-9]+-[A-Z]+)-([0-9]{6})$`)

// basisParams 基差族判据（T-0298：费键复用 research.funding.*——同 venue 同费表
// 不另起一套；阈=research.basis.min_net_apr；到期门=research.basis.min_days_to_expiry。
// 任一键缺=missing_param fail-closed——禁代码字面量判据）。
func (r *Runner) basisParams() (funding.Params, float64, float64, error) {
	var p funding.Params
	var err error
	get := r.cfg.GetFloat
	if p.TakerFeeBps, err = get("research.funding.taker_fee_bps"); err != nil {
		return p, 0, 0, err
	}
	if v, e := get("research.funding.maker_fee_bps"); e == nil { // 软键——缺=双 taker 回退（T-0247 分层同 funding 族）
		p.MakerFeeBps, p.MakerFeeSet = v, true
	}
	if p.SlippageBps, err = get("research.funding.slippage_bps"); err != nil {
		return p, 0, 0, err
	}
	if p.FailCostBps, err = get("research.funding.fail_cost_bps"); err != nil {
		return p, 0, 0, err
	}
	if p.WindowLossBps, err = get("research.funding.window_loss_bps"); err != nil {
		return p, 0, 0, err
	}
	minNet, err := get("research.basis.min_net_apr")
	if err != nil {
		return p, 0, 0, err
	}
	minDays, err := get("research.basis.min_days_to_expiry")
	if err != nil {
		return p, 0, 0, err
	}
	return p, minNet, minDays, nil
}

// evalBasis 季度期货基差评估（future tick−spot tick=年化基差−往返费用
// 年化摊销 vs 动态底；T-0298 根修——旧式零费用项+math.Abs 符号面双病灶：
// cost_apr 复用 funding.CostAPR（HoldDays=到期天数，同一函数禁第二份算式）；
// floor=max(静态, Earn 现率)同 funding 族读件；符号判=有符号 net≥floor。
// 季度期货 tick 缺席=unevaluable 真因行——数据面诚实）。
func (r *Runner) evalBasis(ctx context.Context, f Family) error {
	p, minNet, minDays, perr := r.basisParams()
	if perr != nil {
		r.rej.Inc("missing_param")
		return r.writeEval(ctx, "-", f.Template, f.Direction, "unevaluable", "params_missing",
			map[string]any{"kind": f.Kind})
	}
	// Earn 动态底（D-0006——funding.EarnRate 共用读件零复制；缺/陈=静态降级+计数）。
	floor, floorSrc := minNet, "static"
	staleH, herr := r.cfg.GetFloat("research.earn.stale_hours")
	earn, fresh, eerr := funding.EarnRate(ctx, r.pool, staleH)
	switch {
	case herr != nil || staleH <= 0 || eerr != nil || !fresh:
		r.rej.Inc("earn_rate_degraded") // 降级不阻断（同 funding 族语义）
	case earn != nil && *earn > floor:
		floor, floorSrc = *earn, "earn"
	}
	// T-0132 分级后季度期货只有 mark-price 行——md_ticks∪md_mark_index 同视图读。
	rows, err := r.pool.Query(ctx, `SELECT t1.inst_id,t1.px,t2.px FROM
		(SELECT inst_id,px,ts FROM md_mark_index WHERE channel='mark-price' -- T-0171 方言归一：REST 轮询面撤单键域
		UNION ALL SELECT inst_id,last,ts FROM md_ticks) t1
		JOIN (SELECT inst_id,px,ts FROM md_mark_index
		UNION ALL SELECT inst_id,last,ts FROM md_ticks) t2
		ON t2.inst_id=split_part(t1.inst_id,'-',1)||'-'||split_part(t1.inst_id,'-',2)
		WHERE t1.inst_id ~ '-[0-9]{6}$' AND t1.ts>now()-interval '15 minutes'
		AND t2.ts>now()-interval '15 minutes'`)
	if err != nil {
		r.rej.Inc("eval_write_err")
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var inst string
		var fut, spot float64
		if err := rows.Scan(&inst, &fut, &spot); err != nil {
			return err
		}
		found = true
		// T-0213：net_apr=年化基差补发（basis_pct/距到期天数×365——到期日
		// 解析自 inst_id -YYMMDD 尾；无到期日解析=不写 proposed——
		// T-0205 硬闸对齐 funding-hedge 判据维钦定）。
		m := datedFutRe.FindStringSubmatch(inst)
		days := 0.0
		if len(m) == 3 {
			if exp, perr := time.Parse("060102", m[2]); perr == nil {
				days = time.Until(exp).Hours() / 24
			}
		}
		basis := (fut - spot) / spot * 100 // 基差百分比
		// 方向符号（T-0298 会签：深负基差经 math.Abs 误判 proposed 病灶根修
		// =有符号判）——正基差收族 sign=+1；quarter_basis_short 日后启封时
		// 符号翻转（spot_short 腿缺 margin 面，现不枚举——caveat 钉此）。
		sign := 1.0
		if f.Direction == "basis_quarter_short" {
			sign = -1
		}
		var grossApr float64
		if days > 0 {
			grossApr = sign * basis / days * 365 / 100 // 基差%→年化比率（小数口径同 net_apr 域）
		}
		verdict, reason := "rejected", "basis_below_min"
		netApr := 0.0
		var costApr float64
		switch {
		case days <= 0 || days < minDays:
			reason = "basis_expiry_soon" // 到期<min_days=滚动/交割风险——cost 摊销前早退
		default:
			p.HoldDays = days // basis 持仓期=到期日（与 T-0297 同构摊销分母）
			costApr = funding.CostAPR(p)
			netApr = grossApr - costApr
			switch {
			case netApr >= floor:
				verdict, reason = "proposed", ""
			case netApr >= minNet:
				reason = "below_earn" // 静态底过但 Earn 更高——动态底杀
			}
		}
		costModel := "roundtrip_taker_x4_conservative"
		if f.Template == "perp_quarter_spread" {
			costModel += "+perp_only_legs" // 双腿皆衍生无现货腿——四腿型偏保守
		}
		// T-0330 前置拒：执行径未建族（quarter_basis_short 等）proposed 翻
		// rejected/exec_family_missing 如实，不进提案队列（族码=template 列）。
		if verdict == "proposed" && !execfam.Has(f.Template) {
			verdict, reason = "rejected", "exec_family_missing"
		}
		if err := r.writeEval(ctx, inst, f.Template, f.Direction, verdict, reason,
			map[string]any{"fut": fut, "spot": spot, "basis_pct": basis,
				"gross_apr": grossApr, "net_apr": netApr, "days_to_expiry": int(days),
				"cost_rt_bps": funding.RoundTripBps(p), "cost_apr": costApr,
				"min_net_apr": minNet, "floor_apr": floor, "floor_source": floorSrc,
				"cost_model": costModel}); err != nil {
			return err
		}
		if verdict == "proposed" { // T-0216：proposed→fin_proposal 提案径（executor 在位——同 volsell persistLegEval 幂等形）
			var evalID int64
			if err := r.pool.QueryRow(ctx, `SELECT id FROM fin_opp_eval
				WHERE inst_id=$1 AND template=$2 AND direction=$3`, inst, f.Template, f.Direction).Scan(&evalID); err != nil {
				return err
			}
			// T-0142 黑名单滤：venue 拒过的 inst 不再起草（gapscan 同式——
			// 工单#15188 案：BTC-USD-270326 黑名单在册 2h 后仍被本径武装）。
			var banned int
			if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_untradeable
				WHERE inst_id=$1`, inst).Scan(&banned); err == nil && banned > 0 {
				log.Printf("[research/grammar] %s 黑名单命中（venue_untradeable）——候选不起草", inst)
				continue
			}
			// T-0216 返修：预查键对齐 md_1054 闸 (inst_id,direction)+双在途态
			// （原 inst 单键=跨族误吞——同 inst 双 basis 族互斥；direction 已
			// 去分叉后键语义归一，闸吞前先自静=「落卡」假日志根除）。
			var open int
			if err := r.pool.QueryRow(ctx,
				"SELECT count(*) FROM fin_proposal WHERE inst_id=$1 AND direction=$2 AND state IN ('fin_proposed','fin_armed')",
				inst, f.Direction).Scan(&open); err != nil {
				return err
			}
			if open == 0 {
				if _, err := r.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
					VALUES($1,$2,$3,$4)`, evalID, inst, f.Direction, netApr); err != nil {
					r.rej.Inc("proposal_write_err")
					return err
				}
				log.Printf("[research/grammar] basis 提案落卡 %s %s net_apr=%.4f", inst, f.Template, netApr)
			}
		}
	}
	if !found {
		return r.writeEval(ctx, "-", f.Template, f.Direction, "unevaluable", "quarterly_tick_missing",
			map[string]any{"kind": f.Kind})
	}
	return nil
}
