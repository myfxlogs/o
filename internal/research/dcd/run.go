// run.go —— dcd 臂 Runner（T-0102）：注册表读枚举面→feed 扫产品→快照落
// md_dcd_product→评估闸→fin_opp_eval/fin_proposal 同链。触发=慢周期拍
// （research.dcd.scan_interval_min 热启）——产品在架面无推送事件（REST 保留面，
// okx-asset-funding 档界桩）。枚举面=探员岗第二面（岗心跳由 funding Runner
// 持有——同岗多面不重复注册）。
package dcd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/research/armwatch"
)

// Coins 逐币可动余量读件（notional 闸判据源——cmd 装配注入；nil/缺币=闸缺位
// unevaluable fail-closed 禁虚构判据）。
type Coins func(ctx context.Context) (map[string]float64, error)

// Runner dcd 枚举臂（feed 缺席=枚举让位 WARN——凭据落位重启即上岗）。
type Runner struct {
	pool   *pgxpool.Pool
	cfg    *config.Client
	rej    *rejects.Door
	feed   Feed
	coins  Coins
	lastAt time.Time // 最小间隔节流（T-0102 修补7——ev_account 每 ~5s 触发曾
	// 致产品面 429 风暴；research.dcd.scan_interval_min 热启，缺=30min 底）。
	watch *armwatch.Watch // 臂活性看门（T-0117 返修——四臂共用件非 dcd 独份）
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New("dcd 枚举"),
		rej: rejects.New("research", "dcd_missing_params", "dcd_feed_failed",
			"dcd_registry_read_failed", "dcd_write_err", "dcd_coins_absent")}
}

// AttachFeed 挂枚举读件（cmd 装配层 REST 签名闭包）。
func (r *Runner) AttachFeed(fn Feed) *Runner { r.feed = fn; return r }

// AttachCoins 挂逐币可动余量读件（notional 闸判据源）。
func (r *Runner) AttachCoins(fn Coins) *Runner { r.coins = fn; return r }

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修：走 Alert() 非 log 前缀）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.watch.AttachAlerter(a); return r }

// Run 枚举臂在岗（funding 同构事件化——T-0048 周期环拆：ev_account/ev_funding
// 帧驱动评估一拍；产品在架面无推送事件，账户/结算帧=节拍源，D-0008-B 轮询禁）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	events.ListenLoop(ctx, r.pool, "research-dcd", []string{"ev_account", "ev_funding"}, r.scan)
}

// surfaces 枚举面读表（注册表×模板适用声明 join——「注册+声明=探员即枚举」
// 扩张机制唯一真源；关态面随全集返出——扩张自证断言面，scan 只取 enabled）。
func (r *Runner) surfaces(ctx context.Context) ([]Surface, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT cs.venue,cs.surface,cs.endpoint_family,
		cs.product_family,cs.enabled
		FROM fin_capability_surface cs
		JOIN fin_opp_template t ON t.venue=cs.venue AND t.surface=cs.surface
		WHERE cs.rw_class='read' AND t.enabled AND t.template IN ('dcd_call','dcd_put')
		ORDER BY 1,2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Surface
	for rows.Next() {
		var s Surface
		if err := rows.Scan(&s.Venue, &s.Surface, &s.EndpointFamily, &s.ProductFamily, &s.Enabled); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// scan 一拍：判据→枚举面→逐面扫产品→快照+评估+闸。
func (r *Runner) scan(ctx context.Context) error {
	iv := 30 * time.Minute
	if v, err := r.cfg.GetFloat("research.dcd.scan_interval_min"); err == nil && v > 0 {
		iv = time.Duration(v) * time.Minute
	}
	if time.Since(r.lastAt) < iv {
		return nil // 节流窗内让位（事件风暴≠扫描风暴）
	}
	r.lastAt = time.Now()
	r.watch.SetBeat(iv)
	r.watch.Check(ctx) // T-0117 臂静默判定（业主通道 Alert 一次抑刷屏）
	// T-0244 期界包络：N 对×（调用+300ms 退避）无界长扫——总 deadline 钉死
	// （research.dcd.scan_timeout_s 热启，缺=480s）；ctx 透 feed→okcli 全链
	// 有界，挂死径封（22:44→01:43 型疑面收口）。
	to := 8 * time.Minute
	if v, err := r.cfg.GetFloat("research.dcd.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx
	p, err := r.params()
	if err != nil {
		r.rej.Inc("dcd_missing_params")
		return fmt.Errorf("判据缺位=不可评估（fail-closed）: %w", err)
	}
	ss, err := r.surfaces(ctx)
	if err != nil {
		r.rej.Inc("dcd_registry_read_failed")
		return fmt.Errorf("注册表读败: %w", err)
	}
	if r.feed == nil {
		log.Printf("WARN [research/dcd] 枚举读件缺席（凭据落位后重启即上岗）")
		return nil
	}
	for _, s := range ss {
		if !s.Enabled {
			continue // 关态面不真探（扩张自证只到「注册即被枚举」可见层）
		}
		ps, err := r.feed(ctx, s)
		if err != nil {
			r.rej.Inc("dcd_feed_failed")
			return fmt.Errorf("面 %s 读败 fail-closed（不造零快照）: %w", s.Surface, err)
		}
		if err := r.scanSurface(ctx, s, ps, p); err != nil {
			return err
		}
	}
	if len(ss) > 0 {
		r.watch.OK() // 成功产出拍（恢复清标——再静默可再报）
	}
	return nil
}

// scanSurface 单面：快照留痕+评估落卡+阈闸提案。
func (r *Runner) scanSurface(ctx context.Context, s Surface, ps []Product, p Params) error {
	skipCcy := 0 // T-0117 修4：USDT 计价硬闸（业主令 10-25——非 USDT 产品
	// 枚举期挡不入 eval：异币评出 insufficient_balance=误导性条件缺口；产品
	// 行照留宇宙可见性保留）。
	for _, prod := range ps {
		spot := r.spot(ctx, prod.Uly)
		if prod.QuoteCcy != "" && prod.QuoteCcy != "USDT" {
			skipCcy++
			if err := r.persistProduct(ctx, s, prod, spot); err != nil {
				r.rej.Inc("dcd_write_err")
				return err
			}
			continue // 行留痕不评估
		}
		avail, ok := r.availOf(ctx, prod)
		e := Evaluate(prod, spot, avail, p)
		if !ok {
			e.Verdict, e.Reason = "unevaluable", "coins_absent"
		}
		if err := r.persistProduct(ctx, s, prod, spot); err != nil {
			r.rej.Inc("dcd_write_err")
			return err
		}
		dup, err := r.directionStacked(ctx, prod.Uly, e.Template)
		if err != nil {
			r.rej.Inc("dcd_write_err")
			return err
		}
		if dup && e.Verdict == "proposed" {
			e.Verdict, e.Reason = "rejected", "duplicate_direction"
		}
		if err := r.persistEval(ctx, e); err != nil {
			r.rej.Inc("dcd_write_err")
			return err
		}
	}
	if skipCcy > 0 {
		log.Printf("[research/dcd] 面 %s skipped_scope_ccy=%d（非 USDT 计价产品枚举期挡——范围硬闸）", s.Surface, skipCcy)
	}
	return nil
}

// params 判据全 config 热启读（任一缺=fail-closed 不可评估）。
func (r *Runner) params() (Params, error) {
	var p Params
	for _, kv := range []struct {
		k string
		v *float64
	}{
		{"research.dcd.min_dist_pct", &p.MinDistPct},
		{"research.dcd.max_term_days", &p.MaxTermDays},
		{"research.dcd.min_apr", &p.MinAPR},
		{"research.dcd.notional_pct", &p.NotionalPct},
	} {
		v, err := r.cfg.GetFloat(kv.k)
		if err != nil {
			return p, fmt.Errorf("%s: %w", kv.k, err)
		}
		*kv.v = v
	}
	return p, nil
}

// ScanOnce 单拍壳（工具/实证件消费）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

func nullF(f float64) any {
	if f == 0 {
		return nil
	}
	return f
}

func nullT(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
