// run.go —— dcd 臂 Runner（T-0102）：注册表读枚举面→feed 扫产品→快照落
// md_dcd_product→评估闸→fin_opp_eval/fin_proposal 同链。触发=慢周期拍
// （research.dcd.scan_interval_min 热启）——产品在架面无推送事件（REST 保留面，
// okx-asset-funding 档界桩）。枚举面=探员岗第二面（岗心跳由 funding Runner
// 持有——同岗多面不重复注册）。
package dcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/rejects"
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
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg,
		rej: rejects.New("research", "dcd_missing_params", "dcd_feed_failed",
			"dcd_registry_read_failed", "dcd_write_err", "dcd_coins_absent")}
}

// AttachFeed 挂枚举读件（cmd 装配层 REST 签名闭包）。
func (r *Runner) AttachFeed(fn Feed) *Runner { r.feed = fn; return r }

// AttachCoins 挂逐币可动余量读件（notional 闸判据源）。
func (r *Runner) AttachCoins(fn Coins) *Runner { r.coins = fn; return r }

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
	return nil
}

// scanSurface 单面：快照留痕+评估落卡+阈闸提案。
func (r *Runner) scanSurface(ctx context.Context, s Surface, ps []Product, p Params) error {
	for _, prod := range ps {
		spot := r.spot(ctx, prod.Uly)
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
	return nil
}

// spot uly 现价（md_ticks 最新——与卡面/对拍同源估值面）。
func (r *Runner) spot(ctx context.Context, uly string) float64 {
	var last *float64
	_ = r.pool.QueryRow(ctx, `SELECT last FROM md_ticks WHERE inst_id=$1
		ORDER BY ts DESC LIMIT 1`, uly).Scan(&last)
	if last == nil {
		_ = r.pool.QueryRow(ctx, `SELECT last FROM md_ticks WHERE inst_id=$1
			ORDER BY ts DESC LIMIT 1`, uly+"-SWAP").Scan(&last)
	}
	if last == nil {
		return 0
	}
	return *last
}

// availOf 锁仓侧可动余量（C=基础币、P=USDT）；读件缺=ok=false（unevaluable）。
func (r *Runner) availOf(ctx context.Context, p Product) (float64, bool) {
	if r.coins == nil {
		return 0, false
	}
	coins, err := r.coins(ctx)
	if err != nil {
		r.rej.Inc("dcd_coins_absent")
		return 0, false
	}
	if p.Direction == "P" {
		return coins["USDT"], true
	}
	return coins[firstSeg(p.Uly)], true
}

// directionStacked 单 uly 方向不叠闸（在途提案/策略任一在案=叠）。
func (r *Runner) directionStacked(ctx context.Context, uly, tmpl string) (bool, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM fin_proposal
		WHERE direction=$2 AND (inst_id=$1 OR inst_id LIKE $1||'-%')
		AND state IN ('fin_proposed','fin_armed'))
		+ (SELECT count(*) FROM fin_strategy
		WHERE direction=$2 AND (inst_id=$1 OR inst_id LIKE $1||'-%')
		AND state<>'retired')`, uly, tmpl).Scan(&n)
	return n > 0, err
}

// persistProduct 在架产品快照（每扫一帧留痕——评估可溯源）。
func (r *Runner) persistProduct(ctx context.Context, s Surface, p Product, spot float64) error {
	var raw json.RawMessage
	if len(p.Raw) > 0 {
		raw = p.Raw
	} else {
		raw = json.RawMessage("{}")
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO md_dcd_product(ts,venue,surface,uly,product_id,
		direction,strike,expiry,annualized_yield,min_size,max_size,spot,raw)
		VALUES(now(),$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		s.Venue, s.Surface, p.Uly, p.ProductID, p.Direction,
		nullF(p.Strike), nullT(p.Expiry), nullF(p.AnnualizedYield),
		nullF(p.MinSize), nullF(p.MaxSize), nullF(spot), raw)
	return err
}

// persistEval 评估卡落表；proposed 且无 open 同品提案→fin_proposal 行（闸后唯一提案径）。
func (r *Runner) persistEval(ctx context.Context, e Eval) error {
	legs, _ := json.Marshal(e.Legs())
	ev, _ := json.Marshal(e.Evidence())
	var evalID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,
		gross_apr,net_apr,verdict,legs,evidence)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		e.Product.ProductID, e.Template, e.Template, nullF(e.APR), nullF(e.APR),
		e.Verdict, legs, ev).Scan(&evalID); err != nil {
		return err
	}
	if e.Verdict != "proposed" {
		if e.Reason != "" {
			log.Printf("[research/dcd] %s 闸拒 %s（%s）", e.Product.ProductID, e.Verdict, e.Reason)
		}
		return nil
	}
	var open int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE inst_id=$1 AND state='fin_proposed'",
		e.Product.ProductID).Scan(&open); err != nil {
		return err
	}
	if open > 0 {
		return nil // 已挂提案不重复提案
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,$2,$3,$4)`, evalID, e.Product.ProductID, e.Template, nullF(e.APR))
	if err == nil {
		log.Printf("[research/dcd] 提案落卡 %s %s apr=%.4f（验证官闸过）",
			e.Product.ProductID, e.Template, e.APR)
	}
	return err
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
