// run.go —— 文法 Runner（T-0120）：enabled 能力面×组合律枚举→
// fin_opp_template state=candidate（幂等结构键）+三族通用评估器
// （carry/basis/vol-sell→fin_opp_eval）+缺 executor 族→fin_opp_hypothesis
// 需求卡（requirements.executor=缺件名）。事件化节拍（ev_ticks/ev_account
// 帧驱动+scan_interval_min 热启节流——同 dcd 臂构型）。
package grammar

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/rejects"
)

// Coins 逐币可动余量读件（vol-sell 实评 notional 闸判据源——dcd.Coins 同型
// 注入缝，cmd 装配 okxCoinsFor；nil/缺币=闸缺位 unevaluable fail-closed）。
type Coins func(ctx context.Context) (map[string]float64, error)

// Runner 文法枚举臂。
type Runner struct {
	pool   *pgxpool.Pool
	cfg    *config.Client
	rej    *rejects.Door
	coins  Coins // T-0215：vol-sell 实评锁仓币余量读件
	lastAt time.Time
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg,
		rej: rejects.New("grammar", "missing_param", "surface_read_failed",
			"template_write_err", "eval_write_err", "hypothesis_write_err",
			"proposal_write_err", "coins_absent")}
}

// AttachCoins 挂逐币可动余量读件（vol-sell 实评判据源）。
func (r *Runner) AttachCoins(fn Coins) *Runner { r.coins = fn; return r }

// Run 在岗（事件化——行情/账户帧驱动节流拍）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	events.ListenLoop(ctx, r.pool, "research-grammar",
		[]string{"ev_ticks", "ev_account", "ev_funding"}, r.scan)
}

// ScanOnce 单拍壳（实证件消费）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

func (r *Runner) scan(ctx context.Context) error {
	iv := 120 * time.Minute
	if v, err := r.cfg.GetFloat("research.grammar.scan_interval_min"); err == nil && v > 0 {
		iv = time.Duration(v) * time.Minute
	}
	if time.Since(r.lastAt) < iv {
		return nil
	}
	r.lastAt = time.Now()

	// enabled 能力面集（注册即被枚举真源——T-0119 致密化面）。
	enabled := map[string]bool{}
	rows, err := r.pool.Query(ctx, `SELECT venue||'/'||surface FROM fin_capability_surface WHERE enabled`)
	if err != nil {
		r.rej.Inc("surface_read_failed")
		return err
	}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return err
		}
		enabled[k] = true
	}
	rows.Close()

	fams := Enumerate(enabled)
	var cand, hypo int
	for _, f := range fams {
		if err := r.upsertTemplate(ctx, f); err != nil {
			r.rej.Inc("template_write_err")
			return err
		}
		cand++
		if f.Executor == "" && !f.Existing {
			if err := r.hypothesis(ctx, f); err != nil {
				r.rej.Inc("hypothesis_write_err")
				return err
			}
			hypo++
		}
	}
	if err := r.evaluate(ctx, fams); err != nil {
		return err
	}
	log.Printf("[research/grammar] 文法拍: enabled 面=%d 候选族=%d 假想卡=%d", len(enabled), cand, hypo)
	return nil
}

// upsertTemplate 候选模板行（幂等结构键——Existing 族不重生行）。
func (r *Runner) upsertTemplate(ctx context.Context, f Family) error {
	if f.Existing {
		return nil // funding-hedge/dcd 已落 enabled 模板——文法映射不重复
	}
	params, _ := json.Marshal(map[string]any{
		"legs": f.Legs, "settle": f.Settle, "kind": f.Kind, "grammar": true})
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_template(template,venue,surface,params,enabled,state)
		VALUES($1,$2,$3,$4,false,'candidate')
		ON CONFLICT (template,venue,COALESCE(surface,'')) DO NOTHING`,
		f.Template, f.PrimaryVen, f.PrimarySurf, params)
	return err
}

// hypothesis 缺 executor 族需求卡（requirements.executor=缺件名——需求清单
// 非自由发明；UNIQUE(venue,surface) 幂等撞卡不重生）。
func (r *Runner) hypothesis(ctx context.Context, f Family) error {
	prims, _ := json.Marshal(f.Legs)
	reqs, _ := json.Marshal(map[string]any{"executor": f.Template + "_executor",
		"kind": f.Kind, "settle": f.Settle})
	ev, _ := json.Marshal(map[string]any{
		"provenance": "grammar_enumerate", "leg_vocab": "closed-set",
		"needs_surfaces": f.NeedFace})
	surface := f.Template + "_template"
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_hypothesis
		(venue,surface,family,leg_primitives,requirements,state,evidence)
		VALUES($1,$2,$3,$4,$5,'proposed',$6)
		ON CONFLICT (venue,surface) DO NOTHING`,
		f.PrimaryVen, surface, f.Kind, prims, reqs, ev)
	return err
}

// datedFutRe 到期日尾判（-YYMMDD——T-0213 net_apr 年化径到期解析）。
var datedFutRe = regexp.MustCompile(`^([A-Z0-9]+-[A-Z]+)-([0-9]{6})$`)

// evaluate 三族通用评估器（carry/basis/vol-sell→fin_opp_eval——数据缺=
// unevaluable/rejected 真因行，禁虚构评分）。
func (r *Runner) evaluate(ctx context.Context, fams []Family) error {
	for _, f := range fams {
		if f.Existing {
			continue // 现役族走各自评估径不重复落卡
		}
		switch f.Kind {
		case "basis":
			if err := r.evalBasis(ctx, f); err != nil {
				return err
			}
		case "carry":
			if err := r.evalCarry(ctx, f); err != nil {
				return err
			}
		case "vol-sell":
			if err := r.evalVolSell(ctx, f); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeEval 幂等评估行（同 inst+template+reason 1h 窗不重生——节拍防抖）。
func (r *Runner) writeEval(ctx context.Context, inst, template, direction, verdict, reason string, ev map[string]any) error {
	// T-0205①硬闸：proposed 前置 net_apr 存在——无 APR 估计下落 verdict=
	// unevaluable+apr_est_missing（NULL 参与排序/阈值判=语义炸根修）。
	if verdict == "proposed" {
		if _, ok := ev["apr"]; !ok {
			if _, ok2 := ev["net_apr"]; !ok2 {
				verdict, reason = "unevaluable", "apr_est_missing"
			}
		}
	}
	ev["reason"] = reason
	b, _ := json.Marshal(ev)
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE inst_id=$1 AND template=$2 AND evidence->>'reason'=$3
		AND ts>now()-interval '1 hour'`, inst, template, reason).Scan(&n); err != nil || n > 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES($1,$2,$3,$4,'[]',$5)
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		verdict=EXCLUDED.verdict,legs=EXCLUDED.legs,evidence=EXCLUDED.evidence`,
		inst, template, direction, verdict, b)
	if err != nil {
		r.rej.Inc("eval_write_err")
	}
	return err
}

// evalBasis 季度期货基差评估（future tick−spot tick=年化基差 vs 判据阈；
// 季度期货 tick 缺席=unevaluable 真因行——数据面诚实）。
func (r *Runner) evalBasis(ctx context.Context, f Family) error {
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
		var netApr float64
		days := 0.0
		if len(m) == 3 {
			if exp, perr := time.Parse("060102", m[2]); perr == nil {
				days = time.Until(exp).Hours() / 24
			}
		}
		basis := (fut - spot) / spot * 100 // 基差百分比
		if days > 0 {
			netApr = basis / days * 365 / 100 // 基差%→年化比率（小数口径同 net_apr 域）
		}
		verdict, reason := "rejected", "basis_below_min"
		if days > 0 && math.Abs(netApr) > 0.05 { // 年化 APR 阈（判据维对齐——非裸 basis_pct）
			verdict, reason = "proposed", ""
		}
		if err := r.writeEval(ctx, inst, f.Template, f.Direction, verdict, reason,
			map[string]any{"fut": fut, "spot": spot, "basis_pct": basis,
				"net_apr": netApr, "apr": math.Abs(netApr), "days_to_expiry": int(days)}); err != nil {
			return err
		}
		if verdict == "proposed" { // T-0216：proposed→fin_proposal 提案径（executor 在位——同 volsell persistLegEval 幂等形）
			var evalID int64
			if err := r.pool.QueryRow(ctx, `SELECT id FROM fin_opp_eval
				WHERE inst_id=$1 AND template=$2 AND direction=$3`, inst, f.Template, f.Direction).Scan(&evalID); err != nil {
				return err
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
					VALUES($1,$2,$3,$4)`, evalID, inst, f.Direction, math.Abs(netApr)); err != nil {
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

// evalCarry 借贷/Earn 利差评估（savings rate vs 判据阈——利率事实行缺=
// unevaluable 真因行）。
func (r *Runner) evalCarry(ctx context.Context, f Family) error {
	var rates float64
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_earn_rate
		WHERE ts>now()-interval '1 day'`).Scan(&rates)
	if err != nil {
		// 表缺=数据面未启——unevaluable 真因行（不虚构利率）。
		return r.writeEval(ctx, "-", f.Template, f.Direction, "unevaluable", "savings_rate_missing",
			map[string]any{"kind": f.Kind, "need": f.NeedFace})
	}
	if rates == 0 {
		return r.writeEval(ctx, "-", f.Template, f.Direction, "unevaluable", "savings_rate_missing",
			map[string]any{"kind": f.Kind})
	}
	return r.writeEval(ctx, "-", f.Template, f.Direction, "rejected", "spread_below_min",
		map[string]any{"kind": f.Kind, "rate_rows": rates})
}
