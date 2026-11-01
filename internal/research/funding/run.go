package funding

import (
	"context"
	"errors"
	"fmt"
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
	"research.funding.taker_fee_bps", "research.funding.maker_fee_bps", "research.funding.slippage_bps",
	"research.funding.fail_cost_bps", "research.funding.window_loss_bps",
	"research.funding.min_net_apr", "research.funding.settles_per_day",
}

// Runner 枚举+评估+闸岗（双员工：探员=枚举面，验证官=评估+提案闸面）。
type Runner struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	watch *armwatch.Watch // 臂活性看门（T-0117——四臂共用件）
	// T-0150 现货腿存在性预核读件（instruments/SPOT 目录——装配层挂
	// PublicGet 闭包复用现读面勿新造轮；未挂=预核让位放行 fail-open？否：
	// 读件未挂=预核缺席如实跳过核（存量行为）——挂线后 fail-closed 生效。
	spotFeed func(ctx context.Context) ([]byte, error)
	spotAt   time.Time
	spots    map[string]bool
}

// AttachSpotFeed 挂 SPOT 目录读件（T-0150 起草闸——market/tickers 全量
// 响应 instId 集复用，零新端点）。非价格源：只取 instId 存在性集
// （目录注册面），价格消费不落此径（T-0171 价格源律）。
func (r *Runner) AttachSpotFeed(fn func(ctx context.Context) ([]byte, error)) *Runner {
	r.spotFeed = fn
	return r
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New("funding 枚举"),
		rej: rejects.New("research", "missing_params", "stale_funding", "neg_funding_infeasible",
			"write_err", "earn_rate_degraded", "no_spot_leg")}
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
	if v, e := get(paramKeys[1]); e == nil {
		p.MakerFeeBps, p.MakerFeeSet = v, true // T-0247 分层：在册含 0/负按真值
	}
	if p.SlippageBps, err = get(paramKeys[2]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[1], err)
	}
	if p.FailCostBps, err = get(paramKeys[3]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[3], err)
	}
	if p.WindowLossBps, err = get(paramKeys[4]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[4], err)
	}
	if p.MinNetAPR, err = get(paramKeys[5]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[6], err)
	}
	if p.SettlesPerDay, err = get(paramKeys[6]); err != nil {
		return p, fmt.Errorf("%s: %w", paramKeys[5], err)
	}
	return p, nil
}

// scan 一拍：Earn 现率→md_funding 最新行/inst_id → 动态底五拆评估 → 落卡 → 阈闸提案。
// 读评分离+写拍批内合流（T-0162）：先排干 md_funding 快照完成评估（读面零写），
// 写面整体单事务落卡——同事务空 payload NOTIFY 同值天然去重=全拍至多一醒
// （旧逐行 autocommit 每行一醒=ev_eval 洪泛根，配合 0064 实质变触发器双治）。
func (r *Runner) scan(ctx context.Context) error {
	// T-0244 期界包络（dcd 同型——无界长扫挂死径封；funding.scan_timeout_s 热启缺=480s）
	to := 8 * time.Minute
	if v, err := r.cfg.GetFloat("funding.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx

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
	macro := r.nearestMacro(ctx) // T-0029：产径注解源——一拍一查全拍复用
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ON (inst_id) inst_id, funding_rate, ts
		FROM md_funding ORDER BY inst_id, ts DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var batch []Eval
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
		applyMacro(&e, macro) // 近窗事件→evidence.macro（纯注解层）
		if e.Verdict == "rejected" && rate < 0 {
			r.rej.Inc("neg_funding_infeasible")
		}
		batch = append(batch, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		for _, e := range batch {
			if err := r.persist(ctx, tx, e, p); err != nil {
				r.rej.Inc("write_err")
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	r.watch.OK() // 评估拍完成=产出（恢复清标）
	return nil
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

// ScanOnce 单拍壳（工具/实证件消费——scan 本体私有语义不变）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

// macroEvt 近窗高冲击事件（评估卡 evidence.macro 注解源——T-0029）。
type macroEvt struct {
	kind, title string
	date        time.Time
}

// nearestMacro 一拍一查（scan 单查全拍复用——逐卡零重查）：缺键/horizon=0/
// 无近窗事件=nil（fail-open 不注解——此面非闸，0=关闸语义）。
func (r *Runner) nearestMacro(ctx context.Context) *macroEvt {
	h, err := r.cfg.GetFloat("research.macro.horizon_days")
	if err != nil || h <= 0 {
		return nil
	}
	var kind, title string
	var d time.Time
	err = r.pool.QueryRow(ctx, `SELECT kind,title,event_date FROM md_macro_event
		WHERE event_date BETWEEN current_date AND current_date+$1::int
		ORDER BY event_date LIMIT 1`, int(h)).Scan(&kind, &title, &d)
	if err != nil {
		return nil // 无近窗事件=不注解（缺省正常）
	}
	return &macroEvt{kind: kind, title: title, date: d}
}

// applyMacro 评估卡注解（evidence.macro=kind+title+date+days_out）——纯
// evidence 层，net_apr 判据零触（T-0029 验收：同输入下 verdict 不变）。
func applyMacro(e *Eval, m *macroEvt) {
	if m == nil {
		return
	}
	if e.Evidence == nil {
		e.Evidence = map[string]any{}
	}
	e.Evidence["macro"] = map[string]any{
		"kind": m.kind, "title": m.title, "date": m.date.Format("2006-01-02"),
		"days_out": int(time.Until(m.date).Hours() / 24),
	}
}
