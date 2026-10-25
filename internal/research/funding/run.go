package funding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
	"o/internal/research/armwatch"
)

const (
	freshnessMax = 25 * time.Hour // funding 事实行新鲜度门（8h 结算×3+余量）
)

var paramKeys = []string{
	"research.funding.taker_fee_bps", "research.funding.slippage_bps",
	"research.funding.fail_cost_bps", "research.funding.window_loss_bps",
	"research.funding.min_net_apr", "research.funding.settles_per_day",
}

// Runner 枚举+评估+闸岗（双员工：探员=枚举面，验证官=评估+提案闸面）。
type Runner struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	watch *armwatch.Watch // 臂活性看门（T-0117——四臂共用件）
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New("funding 枚举"),
		rej: rejects.New("research", "missing_params", "stale_funding", "neg_funding_infeasible",
			"write_err", "earn_rate_degraded")}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.watch.AttachAlerter(a); return r }

// Run 双员工在岗+扫描循环（阻塞至 ctx 取消）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	for _, e := range [][3]string{{"探员", "research", "机会枚举岗"}, {"验证官", "research", "评估卡+提案闸岗"}} {
		if err := roster.Register(ctx, r.pool, e[0], e[1], e[2], nil); err != nil {
			errc <- fmt.Errorf("%s register: %w", e[0], err)
			return
		}
		go roster.BeatLoop(ctx, r.pool, e[0], 30*time.Second, errc)
	}
	// 事件化（T-0048：周期环拆——funding 帧 ev_funding 驱动评估一拍）。
	events.ListenLoop(ctx, r.pool, "research", []string{"ev_funding"}, r.scan)
}

// params 判据全自 config 快照读；任一键缺=不可评估面（fail-closed 不虚构判据）。
func (r *Runner) params() (Params, error) {
	var p Params
	get := func(k string) (float64, error) { return r.cfg.GetFloat(k) }
	var err error
	if p.TakerFeeBps, err = get(paramKeys[0]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[0], err)
	}
	if p.SlippageBps, err = get(paramKeys[1]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[1], err)
	}
	if p.FailCostBps, err = get(paramKeys[2]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[2], err)
	}
	if p.WindowLossBps, err = get(paramKeys[3]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[3], err)
	}
	if p.MinNetAPR, err = get(paramKeys[4]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[4], err)
	}
	if p.SettlesPerDay, err = get(paramKeys[5]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[5], err)
	}
	return p, nil
}

// scan 一拍：Earn 现率→md_funding 最新行/inst_id → 动态底五拆评估 → 落卡 → 阈闸提案。
func (r *Runner) scan(ctx context.Context) error {
	r.watch.SetBeat(freshnessMax)
	r.watch.Check(ctx) // 臂静默判定（T-0117——beat=funding 新鲜度窗 25h）
	p, err := r.params()
	if err != nil {
		r.rej.Inc("missing_params")
		return fmt.Errorf("判据缺位=不可评估（fail-closed）: %w", err)
	}
	earnAPR, fresh, err := r.earnRateAPR(ctx)
	if err != nil {
		return err
	}
	if !fresh {
		r.rej.Inc("earn_rate_degraded") // 降级不阻断：退回静态底+告警计数（D-0006）
	}
	p.EarnRateAPR = earnAPR
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ON (inst_id) inst_id, funding_rate, ts
		FROM md_funding ORDER BY inst_id, ts DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var instID string
		var rate float64
		var ts time.Time
		if err := rows.Scan(&instID, &rate, &ts); err != nil {
			return err
		}
		if time.Since(ts) > freshnessMax {
			r.rej.Inc("stale_funding")
			continue // 陈数据不评估（快照禁令——陈行非现值）
		}
		e := Evaluate(instID, rate, p)
		if e.Verdict == "rejected" && rate < 0 {
			r.rej.Inc("neg_funding_infeasible")
		}
		if err := r.persist(ctx, e, p); err != nil {
			r.rej.Inc("write_err")
			return err
		}
	}
	r.watch.OK() // 评估拍完成=产出（恢复清标）
	return rows.Err()
}

// earnRateAPR 最新 Earn 现率（USDT，动态比较底取值源）。缺失/陈旧/新鲜度键缺=
// (nil,false)=降级静态底——降级不阻断，计数门留痕。
func (r *Runner) earnRateAPR(ctx context.Context) (*float64, bool, error) {
	hours, err := r.cfg.GetFloat("research.earn.stale_hours")
	if err != nil || hours <= 0 {
		return nil, false, nil // 新鲜度门键缺=不敢当鲜（fail-closed 降级）
	}
	var rate float64
	var ts time.Time
	err = r.pool.QueryRow(ctx, `SELECT rate,ts FROM md_earn_rate
		WHERE venue='okx' AND ccy='USDT' ORDER BY ts DESC LIMIT 1`).Scan(&rate, &ts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if time.Since(ts) > time.Duration(hours*float64(time.Hour)) {
		return nil, false, nil
	}
	return &rate, true, nil
}

// persist 评估卡落表；proposed 且无 open 提案→fin_proposal 行（闸后唯一提案径）。
func (r *Runner) persist(ctx context.Context, e Eval, p Params) error {
	legs, _ := json.Marshal(e.Legs)
	ev, _ := json.Marshal(e.Evidence)
	var evalID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
		VALUES($1,'funding-hedge',$2,$3,$4,$5,$6,$7)
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		gross_apr=EXCLUDED.gross_apr,net_apr=EXCLUDED.net_apr,verdict=EXCLUDED.verdict,
		legs=EXCLUDED.legs,evidence=EXCLUDED.evidence RETURNING id`,
		e.InstID, e.Direction, e.GrossAPR, e.NetAPR, e.Verdict, legs, ev).Scan(&evalID); err != nil {
		return err
	}
	if e.Verdict != "proposed" {
		return nil
	}
	var open int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE inst_id=$1 AND state='fin_proposed'", e.InstID).Scan(&open); err != nil {
		return err
	}
	if open > 0 {
		return nil // 已挂提案不重复提案（滞留禁超 SLA 由生命周期票管）
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,$2,$3,$4)`, evalID, e.InstID, e.Direction, e.NetAPR)
	if err == nil {
		log.Printf("[research] 提案落卡 %s %s net_apr=%.4f（验证官闸过）", e.InstID, e.Direction, e.NetAPR)
	}
	return err
}

// ScanOnce 单拍壳（工具/实证件消费——scan 本体私有语义不变）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

// annotateMacro 高冲击日历注解（T-0029）：查 event_date−today ≤ horizon_days
// 最近事件→evidence.macro（kind+date+days_out）；缺键/无事件=fail-open 不注解。
func (r *Runner) annotateMacro(ctx context.Context, e *Eval) {
	h, err := r.cfg.GetFloat("research.macro.horizon_days")
	if err != nil {
		return // 缺键 fail-open——此面非闸不注解
	}
	var kind, title string
	var d time.Time
	err = r.pool.QueryRow(ctx, `SELECT kind,title,event_date FROM md_macro_event
		WHERE event_date BETWEEN current_date AND current_date+$1::int
		ORDER BY event_date LIMIT 1`, int(h)).Scan(&kind, &title, &d)
	if err != nil {
		return // 无近窗事件=不注解（缺省正常）
	}
	e.Evidence["macro"] = map[string]any{
		"kind": kind, "title": title, "date": d.Format("2006-01-02"),
		"days_out": int(time.Until(d).Hours() / 24),
	}
}
