// fasttrack.go —— O-41014 快窗信号 probe 直通径（方向族 edge 测量装置）：
// eval 极值判词∧dir 族白名单→mint state='probe' 小仓跳 paper 证明窗。
// 只越 paper 证明窗——confGate/entryGate/armed_ttl/stop-loss/dd/并发闸
// 全继承原样（armed≠open：入场条件到点 openProbe 才开仓）。
// 测量装置非快线：probe P&L 归举一反三/评估域裁方向族生死，本件不主张 edge。
package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"o/internal/platform/decision"
)

// admitProp 提案行快照（admitProposals 循环共用——自 run.go 局部型提升
// 包级：commitMint/fasttrack 判定同吃一形）。
type admitProp struct {
	id, evalID int64
	inst       string
	dir        string
	tmpl       string
	netApr     float64
	venue      string
}

// fasttrackTmpl 直通白名单=dir 族∩单腿 perp∩direction∈{long_perp,
// short_perp}∩dirOpen 真单径在册（dirLegLabel/BuildDirLeg/ExecuteDir
// 同吃）。mt5_dir_mom 例外不收——CFD 腿无方向真单径（venue=mt5/exness
// 官名 meta 不入 BuildDirLeg perp 径）；xsec-perp-mom-basket 复合
// direction 词表外 DirSide 闭集拒；mark-last-disloc-fade direction=
// perp_fade 同拒——白名单外照走 paper 慢证明不株连。
var fasttrackTmpl = map[string]bool{
	"dir_perp_mom": true, "perp-trend-daily": true,
}

// fasttrackParams 直通阈件包（probe.fasttrack_*——fin_0265 种键
// fail-closed：enabled 缺/假=全走 paper；enabled 真但 score/notional/
// maxOpen 缺坏=本拍直通关断走 paper 不炸主径）。
type fasttrackParams struct {
	enabled bool
	score   float64 // 强度阈=提案 net_apr（fin_opp_eval 实有机读字段——
	// S0 定标：dir_perp_mom 30d proposed eval 分布 p90≈8.19/p95≈10.43
	// （生产实读 154 行）——缺省 8.0=极值档 top~10%，非拍脑袋数）
	notional float64
	maxOpen  int64
	open     int // 本拍已直通数（allow 消费槽——后置闸拦=槽漏向保守向）
}

// fasttrackLoad 阈件装载（enabled 读败=关；余键缺坏=关+判词——
// 直通是岔路不是主径，件坏不拖 paper 开户拍）。
func fasttrackLoad(getF func(string) (float64, error), getB func(string) (bool, error), getI func(string) (int64, error)) fasttrackParams {
	p := fasttrackParams{}
	en, err := getB("probe.fasttrack_enabled")
	if err != nil || !en {
		return p
	}
	sc, err1 := getF("probe.fasttrack_score")
	nt, err2 := getF("probe.fasttrack_notional_usd")
	mo, err3 := getI("probe.fasttrack_max_open")
	if err1 != nil || err2 != nil || err3 != nil || sc <= 0 || nt <= 0 || mo <= 0 {
		log.Printf("[lifecycle] fasttrack 阈件缺坏（score/notional/maxOpen——本拍直通关断走 paper）")
		return p
	}
	p.enabled, p.score, p.notional, p.maxOpen = true, sc, nt, mo
	return p
}

// allow 直通资格判（消耗并发槽——返回 (生效 notional, 直通?)；
// 后置闸（shapeGate/deadSig/xvenue）拦下=槽漏向保守向不虚报容量）。
func (p *fasttrackParams) allow(x admitProp) (float64, bool) {
	if !p.enabled || int64(p.open) >= p.maxOpen || !fasttrackTmpl[x.tmpl] {
		return 0, false
	}
	if x.dir != "long_perp" && x.dir != "short_perp" {
		return 0, false
	}
	if x.netApr < p.score {
		return 0, false
	}
	p.open++
	return p.notional, true
}

// probeFasttrackCount 在册直通 probe 数（并发闸基线——criteria
// fasttrack 旗标=直通件唯一真源，paper 晋升 probe 不沾此径）。
func (r *Runner) probeFasttrackCount(ctx context.Context) int {
	var n int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_strategy
		 WHERE state='probe' AND criteria->>'fasttrack'='true'`).Scan(&n); err != nil {
		return 0 // 读败=按零计——allow 内 maxOpen 闸仍兜底不放大
	}
	return n
}

// commitMint 提案→策略开户提交件（paper/fasttrack-probe 双态同件——
// 自 run.go admitProposals 尾段提升：同键去重吸收/transition 留痕/
// armed 回写/decision 决策行/日志全同构，差分=state+notional+reason+
// paper 摩擦估计（probe 真钱径不估 paper 成本——零触）。
// 返回 absorbed=true=同键 active 行在挂（提案 armed 复用不叠仓）。
func (r *Runner) commitMint(ctx context.Context, x admitProp, state string,
	notional float64, isFT bool, minH, minNet, maxDD float64) (bool, error) {
	crit := map[string]any{"min_hours": minH, "min_net": minNet, "max_dd": maxDD}
	if isFT {
		crit["fasttrack"] = true
	}
	critJ, _ := json.Marshal(crit)
	var sid int64
	err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,criteria,direction,venue)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (venue,inst_id,direction,template_ref) WHERE state<>'retired' DO NOTHING
		RETURNING id`, x.inst, x.tmpl, x.id, state, notional, critJ, x.dir, x.venue).Scan(&sid)
	if err != nil {
		if err.Error() == "no rows in result set" {
			if _, err := r.pool.Exec(ctx,
				"UPDATE fin_proposal SET state='fin_armed' WHERE id=$1", x.id); err != nil {
				return false, err
			}
			log.Printf("[lifecycle] %s %s 同键 active 行在挂——提案 armed 复用不叠仓", x.inst, x.dir)
			return true, nil
		}
		r.rej.Inc("write_err")
		return false, err
	}
	reason := "验证官提案过阈→paper 沙箱开户"
	if isFT {
		reason = "验证官提案过阈+强度极值→probe 直通小仓（fasttrack——跳 paper 证明窗，执行侧闸原样）"
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
		VALUES($1,'proposed',$2,$3)`, sid, state, reason); err != nil {
		return false, err
	}
	if !isFT {
		r.chargePaperFriction(ctx, sid, x.venue, x.inst, notional) // T-0521：开户即扣估计往返摩擦（paper 径专属——probe 真钱仓不估沙箱成本）
	}
	if _, err := r.pool.Exec(ctx,
		"UPDATE fin_proposal SET state='fin_armed' WHERE id=$1", x.id); err != nil {
		return false, err
	}
	payload := map[string]any{
		"proposal_id": x.id, "eval_id": x.evalID, "net_apr": x.netApr,
		"template": x.tmpl, "direction": x.dir, "inst_id": x.inst,
		"notional_usd": notional, "state": state, "criteria": json.RawMessage(critJ)}
	if isFT {
		payload["fasttrack"] = true
	}
	decision.WriteEsc(ctx, r.pool, decision.Row{
		Domain: "fin", Actor: "lifecycle", RefID: strconv.FormatInt(sid, 10),
		Action: "open",
		Reason: fmt.Sprintf("提案过阈→%s 开户（proposal=#%d eval=#%d 期望 net_apr=%.4f）",
			state, x.id, x.evalID, x.netApr),
		Payload: payload})
	log.Printf("[lifecycle] #%d %s %s 开户 notional=%.0f", sid, x.inst, state, notional)
	return false, nil
}

// openProbe fasttrack probe 开仓消费面（scan 尾槽——openLive 后）：
// state='probe'∧criteria.fasttrack=true∧无未平仓→逐个过执行侧闸。
// armed≠open 脊柱：entryCondition 信号续存判词先行（到点才开——
// 39955 等信号=活动语义同源复用 dirEntryGate 族判词）；cellGate/
// shapeGate/振荡保险丝/冷却窗/r.op.Open（闸+腿构+live_switch+双帽）
// 全继承——本件零越任何执行侧闸。
func (r *Runner) openProbe(ctx context.Context) {
	if r.op == nil {
		return
	}
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.direction, s.notional_usd,
		COALESCE(s.venue,'okx'), s.template_ref,
		COALESCE((SELECT p.net_apr FROM fin_proposal p WHERE p.id=s.proposal_ref),0)
		FROM fin_strategy s
		WHERE s.state='probe' AND s.criteria->>'fasttrack'='true' AND NOT EXISTS (
			SELECT 1 FROM fin_position p WHERE p.strategy_ref=s.id AND p.state<>'closed')`)
	if err != nil {
		r.rej.Inc("write_err")
		return
	}
	type cand struct {
		spec   OpenSpec
		refAPR float64
	}
	var cs []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.spec.StrategyRef, &c.spec.InstID, &c.spec.Direction,
			&c.spec.NotionalUSD, &c.spec.Venue, &c.spec.Template, &c.refAPR); err != nil {
			rows.Close()
			r.rej.Inc("write_err")
			return
		}
		cs = append(cs, c)
	}
	rows.Close()
	gate := r.loadCellGate(ctx)
	nav := &shapeGateNavLazy{}
	cd := 300.0
	if v, err := r.cfg.GetFloat("lifecycle.reject_cooldown_s"); err == nil && v > 0 {
		cd = v
	}
	for i, c := range cs {
		if i > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second): // sched-exempt-ok：venue 限速节流等待（openLive 同径——非轮询）
			}
		}
		// 进入条件判词——armed≠open：信号续存才开仓（族判词同源
		// entryCondition——dir 族=最新 eval 仍 proposed∧72h 内）。
		if ok, why := r.entryCondition(ctx, c.spec.InstID, c.spec.Template,
			c.spec.Direction, c.refAPR); !ok {
			r.rej.Inc("entry_cond_defer")
			log.Printf("[lifecycle] probe #%d %s 开仓延评：%s", c.spec.StrategyRef, c.spec.InstID, why)
			continue
		}
		var rounds int
		if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_position
			WHERE strategy_ref=$1 AND opened_at>now()-interval '1 hour'`,
			c.spec.StrategyRef).Scan(&rounds); err != nil {
			r.rej.Inc("write_err")
			continue
		}
		if r.denyAt != nil && time.Since(r.denyAt[c.spec.StrategyRef]) < time.Duration(cd*float64(time.Second)) {
			continue
		}
		if rounds >= 3 {
			r.rej.Inc("oscillation_fuse")
			log.Printf("ALERT [lifecycle] probe 策略 %d %s 1h 内 %d 轮开平——振荡保险丝熔断降级", c.spec.StrategyRef, c.spec.InstID, rounds)
			if terr := Transition(ctx, r.pool, c.spec.StrategyRef, "degraded",
				"同键开平振荡保险丝——1h 内 ≥3 轮开平降级待复盘", false); terr != nil {
				r.rej.Inc("write_err")
			}
			continue
		}
		if !gate.allow(r, c.spec) {
			continue
		}
		if ok, why := r.shapeGateBlock(ctx, fmt.Sprintf("strategy=#%d", c.spec.StrategyRef),
			c.spec.Template, c.spec.NotionalUSD, nav); !ok {
			log.Printf("[lifecycle] probe 策略 %d %s 开仓闸拦：形态闸——%s", c.spec.StrategyRef, c.spec.InstID, why)
			continue
		}
		eid, err := r.op.Open(ctx, c.spec)
		if err != nil {
			var banned int
			if be := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_untradeable
				WHERE inst_id=$1`, c.spec.InstID).Scan(&banned); be == nil && banned > 0 {
				r.rej.Inc("blacklist_retire")
				if terr := Transition(ctx, r.pool, c.spec.StrategyRef, "retired",
					"venue_untradeable:blacklisted 开仓败命中黑名单——退役不重试", false); terr != nil {
					r.rej.Inc("write_err")
				}
				continue
			}
			r.rej.Inc("open_failed")
			log.Printf("[lifecycle] WARN probe 策略 %d 开仓败（%v exec_id=%d）——事件续试", c.spec.StrategyRef, err, eid)
			continue
		}
		r.noteProbeOpen(ctx, c.spec, eid)
		log.Printf("[lifecycle] probe 策略 %d %s fasttrack 开仓成 exec_id=%d notional=%.2f",
			c.spec.StrategyRef, c.spec.InstID, eid, c.spec.NotionalUSD)
	}
}

// noteProbeOpen probe 直通开仓决策行（noteLiveOpen 同构——reason 标
// probe 态不冒 live 名；MoneyFacts 实现基线件同吃）。
func (r *Runner) noteProbeOpen(ctx context.Context, spec OpenSpec, execID int64) {
	payload := map[string]any{
		"exec_id": execID, "inst_id": spec.InstID, "template": spec.Template,
		"direction": spec.Direction, "notional_usd": spec.NotionalUSD,
		"venue": spec.Venue, "fasttrack": true}
	for k, v := range decision.MoneyFacts(ctx, r.pool, spec.StrategyRef, 0) {
		payload[k] = v
	}
	decision.WriteEsc(ctx, r.pool, decision.Row{
		Domain: "fin", Actor: "lifecycle", RefID: strconv.FormatInt(spec.StrategyRef, 10),
		Action: "open",
		Reason: fmt.Sprintf("probe fasttrack 开仓成 exec_id=%d（入场条件到点+执行侧闸全过——%s notional=%.0f）",
			execID, spec.InstID, spec.NotionalUSD),
		Payload: payload})
}
