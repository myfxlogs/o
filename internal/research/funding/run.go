package funding

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
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
	// T-0320 质押分量率行新鲜度门（research.stake.stale_hours 热启——
	// params 拍读一次；<=0=不闸率行龄）。
	stakeStaleH float64
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
			"write_err", "earn_rate_degraded", "gate_kill", "no_spot_leg", "stake_yield_err")}
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

	// T-0314 gate.kill 消费腿：kill_trigger 裁定词落键=true→本族评估全拦
	// （拒因 gate_kill 计数——评估读面非写入径）。
	if k, err := r.cfg.GetBool("gate.kill.funding_carry"); err == nil && k {
		r.rej.Inc("gate_kill")
		return nil
	}
	r.watch.SetBeat(freshnessMax)
	r.watch.Check(ctx) // 臂静默判定（T-0117——beat=funding 新鲜度窗 25h）
	p, err := r.params()
	if err != nil {
		r.rej.Inc("missing_params")
		return fmt.Errorf("判据缺位=不可评估（fail-closed）: %w", err)
	}
	// 工单#19797①费面实测合流（T-0521 opener 侧同式——eval 成本侧与
	// lifecycle.friction 实测同族口径，禁双 taker 虚高）：fee/slip 分量
	// 实测值新鲜（≤30d 对齐测量窗）则替声明参数；换算同 breakevenOK——
	// RoundTripBps fee 项=4×taker、slip 项=2×slip，故 taker=fee/4、
	// slip=slip/2 走 CostAPR 单源函数（自检 I 不破）；fail/window 成本项
	// 实测面覆盖不到=声明值续守。venue=okx（md_funding 域即 okx perp 面）。
	if fee, e := r.cfg.GetFloat("lifecycle.friction_fee_bps.okx"); e == nil && fee > 0 {
		if slip, e := r.cfg.GetFloat("lifecycle.friction_slip_bps.okx"); e == nil && slip >= 0 {
			if at, e := r.cfg.GetFloat("lifecycle.friction_at.okx"); e == nil &&
				time.Since(time.Unix(int64(at), 0)) < 30*24*time.Hour {
				p.TakerFeeBps, p.MakerFeeSet = fee/4, false
				p.SlippageBps = slip / 2
			}
		}
	}
	earnAPR, fresh, err := r.earnRateAPR(ctx)
	if err != nil {
		return err
	}
	if !fresh {
		r.rej.Inc("earn_rate_degraded") // 降级不阻断：退回静态底+告警计数（D-0006）
	}
	p.EarnRateAPR = earnAPR
	r.stakeStaleH = r.stakeStale() // T-0320：质押分量率行龄门（拍级一次非逐 inst）
	macro := r.nearestMacro(ctx)   // T-0029：产径注解源——一拍一查全拍复用
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
		// T-0320 质押腿分量：基币质押率源解析（缺席=Yield 零值入评不虚构）。
		sy, yok, yerr := SpotYield(ctx, r.pool, strings.TrimSuffix(instID, "-USDT-SWAP"), r.stakeStaleH)
		if yerr != nil {
			r.rej.Inc("stake_yield_err")
			sy = Yield{}
		} else if !yok {
			sy = Yield{}
		}
		// 工单#19797②摊销窗实测合流（T-0521 同式——eval HoldDays 分母与
		// opener breakevenOK 同判据一源）：正 funding 族（rate>0→
		// short_perp_long_spot 收租向）分母=min(声明上限 max_hold/24,
		// 该 inst 收租号连续段 30d 中位)；无该号段=摊销窗不虚构 fail-closed
		// 拒（与开闸判据同结论——eval 不虚构放行）。rate<0 反向族本即拒，
		// 不耗实测查询。measured 源入 evidence 留痕。
		pi := p
		if rate > 0 {
			measH, merr := MeasuredHoldDays(ctx, r.pool, instID, 1.0)
			if merr != nil {
				r.rej.Inc("write_err")
				return merr
			}
			if measH <= 0 {
				e := Eval{InstID: instID, Direction: "short_perp_long_spot",
					Legs:     map[string]any{"perp": "short", "spot": "long"},
					Verdict:  "rejected",
					Reason:   "hold_window_unmeasured:30d 无该号收租段——摊销窗不虚构 fail-closed",
					Evidence: map[string]any{"funding_rate": rate, "hold_days_src": "unmeasured"}}
				batch = append(batch, e)
				continue
			}
			pi.HoldDays = math.Min(p.HoldDays, measH)
		}
		e := Evaluate(instID, rate, pi, sy)
		if rate > 0 {
			if e.Evidence == nil {
				e.Evidence = map[string]any{}
			}
			e.Evidence["hold_days_src"] = "measured"
		}
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
	return EarnRate(ctx, r.pool, hours)
}

// EarnRate Earn 现率共用读件（D-0006 动态比较底——T-0298 grammar basis 族/
// T-0300 scout 族同底挂同一读件零复制）。staleHours=新鲜度窗；行缺/陈旧=
// (nil,false) 降级，调用方各自计数门留痕。
func EarnRate(ctx context.Context, pool *pgxpool.Pool, staleHours float64) (*float64, bool, error) {
	var rate float64
	var ts time.Time
	err := pool.QueryRow(ctx, `SELECT rate,ts FROM md_earn_rate
		WHERE venue='okx' AND ccy='USDT' ORDER BY ts DESC LIMIT 1`).Scan(&rate, &ts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if time.Since(ts) > time.Duration(staleHours*float64(time.Hour)) {
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
