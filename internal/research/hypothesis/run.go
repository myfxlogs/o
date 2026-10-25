// Package hypothesis —— 假想官（T-0106，编制补岗——业主令「系统不得圈死」）：
// 面差→模板假想卡→批准→注册的原创链起草席。消费 pending gap（family 已由
// gapscan classify 归类）→起草 fin_opp_hypothesis 待审卡（surface 实据+family
// +腿原语候选〔基元词汇表组合非自由发明〕+evaluator/risk 需求清单）→gap 翻
// hypothesized 卡引用回填。
//
// 假想官=起草席非执行席：产出是结构化待审卡非可执行策略——批准权=决策席；
// adopted→模板注册（fin_opp_template 行+direction 枚举+评估臂）显式走批准径
// 非自动。零写径证：不触 fin_opp_template/fin_proposal/任何 venue 调用——
// 只写假想卡+消费的 gap 行（队列移交机制）。
package hypothesis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
	"o/internal/research/armwatch"
)

// legVocab 腿原语基元词汇表（闭集——假想卡组合器只从此表按 family 取，
// 非自由发明；扩词=改表+票，非运行时）。
var legVocab = map[string][]string{
	"carry": {"locked_ccy_sell_call", "locked_usdt_sell_put", "earn_flexible_park", "funding_collect"},
	"hedge": {"spot_long", "spot_short", "perp_long", "perp_short", "funding_collect"},
}

// reqTemplates 需求清单模板（family→evaluator/risk 阈/数据面需求——假想卡是
// 需求书不是实现码；真新族 evaluator/executor 照走任务单）。
var reqTemplates = map[string]map[string]any{
	"carry": {
		"evaluator":       "face-native yield 评估（枚举臂或适配器）",
		"risk_thresholds": []string{"apr_min", "term_days", "notional_pct"},
		"data_needs":      []string{"md_ticks spot", "coins avail", "face products feed"},
	},
	"hedge": {
		"evaluator":       "净 APR 五项拆解（funding-hedge 语义）",
		"risk_thresholds": []string{"min_net_apr", "notional_pct"},
		"data_needs":      []string{"md_funding rate", "md_ticks spot"},
	},
}

// Runner 假想官在岗体（enroll 注册+心跳；ev_gap 事件驱动——D-0008-B 正解）。
type Runner struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	watch *armwatch.Watch // 臂活性看门（T-0117——四臂共用件）
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New("假想官"),
		rej: rejects.New("research", "hypo_missing_params", "hypo_write_err")}
}

// enroll 花名册落位（T-0078 判据：名单岗必须 runtime 在行——编制册研究域
// 「假想官」卡 runtime 投影）。
func (r *Runner) enroll(ctx context.Context) error {
	return roster.Register(ctx, r.pool, "假想官", "research",
		"面差→模板假想卡起草（起草席非执行席——批准权归决策席）", nil)
}

// Run 在岗（注册+心跳 30s+ev_gap 事件驱动消费）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := r.enroll(ctx); err != nil {
		errc <- fmt.Errorf("假想官 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "假想官", 30*time.Second, errc)
	events.ListenLoop(ctx, r.pool, "research-hypo", []string{"ev_gap"}, r.scan)
}

// params 判据全 app_config 机读（缺/坏=fail-closed 不起草不虚构判据）。
func (r *Runner) params() (minProducts float64, families map[string]bool, err error) {
	if minProducts, err = r.cfg.GetFloat("research.hypothesis.min_products"); err != nil {
		return 0, nil, fmt.Errorf("research.hypothesis.min_products: %w", err)
	}
	var raw string
	if raw, err = r.cfg.GetString("research.hypothesis.families"); err != nil {
		return 0, nil, fmt.Errorf("research.hypothesis.families: %w", err)
	}
	families = map[string]bool{}
	for _, f := range strings.Split(strings.Trim(raw, `"`), ",") {
		if f = strings.TrimSpace(f); f != "" {
			families[f] = true
		}
	}
	return minProducts, families, nil
}

// scan 一拍：pending gap（family 已归类且在可起草族）→假想卡→gap 翻
// hypothesized 卡引用回填。重复面不重生卡（UNIQUE 幂等+已起草面跳过）。
func (r *Runner) scan(ctx context.Context) error {
	minProducts, families, err := r.params()
	if err != nil {
		r.rej.Inc("hypo_missing_params")
		return fmt.Errorf("判据缺位=不起草（fail-closed）: %w", err)
	}
	r.watch.SetBeat(24 * time.Hour)
	r.watch.Check(ctx) // 臂静默判定（T-0117——ev_gap 事件驱动，beat=日窗兜底）
	rows, err := r.pool.Query(ctx, `SELECT id,venue,surface,metadata FROM fin_surface_gap
		WHERE state='pending' ORDER BY id`)
	if err != nil {
		r.rej.Inc("hypo_write_err")
		return err
	}
	type gap struct {
		id      int64
		venue   string
		surface string
		meta    map[string]any
	}
	var gs []gap
	for rows.Next() {
		var g gap
		var metaB []byte
		if err := rows.Scan(&g.id, &g.venue, &g.surface, &metaB); err != nil {
			rows.Close()
			return err
		}
		g.meta = map[string]any{}
		_ = json.Unmarshal(metaB, &g.meta)
		gs = append(gs, g)
	}
	rows.Close()
	for _, g := range gs {
		family, _ := g.meta["family"].(string)
		if !families[family] {
			continue // 不可起草族（unclassified/vol 等）——gap 原态留队列
		}
		products, _ := g.meta["products"].(float64)
		if products < minProducts {
			log.Printf("[research/hypo] %s/%s 产品数 %v<%v——不起草（判据 fail-closed）",
				g.venue, g.surface, g.meta["products"], minProducts)
			continue
		}
		cardID, err := r.draft(ctx, g.venue, g.surface, family, g.id, g.meta)
		if err != nil {
			r.rej.Inc("hypo_write_err")
			return err
		}
		if _, err := r.pool.Exec(ctx, `UPDATE fin_surface_gap
			SET state='hypothesized', hypothesis_id=$2, last_seen=now()
			WHERE id=$1 AND state='pending'`, g.id, cardID); err != nil {
			r.rej.Inc("hypo_write_err")
			return err
		}
		log.Printf("[research/hypo] 假想卡 #%d 立卡 %s/%s（family=%s——待决策席审，零执行权）",
			cardID, g.venue, g.surface, family)
	}
	r.watch.OK() // 消费拍完成=产出（卡零立=无 pending 静默属正常）
	return nil
}

// draft 假想卡落表（幂等——UNIQUE(venue,surface) 撞卡=返既有卡 id 不重生；
// 零写径：只触本表+消费 gap 行，venue 调用零）。
func (r *Runner) draft(ctx context.Context, venue, surface, family string, gapID int64, meta map[string]any) (int64, error) {
	prims, _ := json.Marshal(legVocab[family])
	reqs, _ := json.Marshal(reqTemplates[family])
	meta["provenance"] = "hypothesis_officer"
	meta["gap_id"] = gapID
	meta["leg_vocab"] = "closed-set" // 词汇表闭集声明——非自由发明
	ev, _ := json.Marshal(meta)
	var id int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_hypothesis
		(venue,surface,family,leg_primitives,requirements,state,evidence)
		VALUES($1,$2,$3,$4,$5,'proposed',$6)
		ON CONFLICT (venue,surface) DO NOTHING RETURNING id`,
		venue, surface, family, prims, reqs, ev).Scan(&id); err == nil {
		return id, nil
	}
	// 撞卡=既有卡幂等返 id（重复面不重生）。
	err := r.pool.QueryRow(ctx, `SELECT id FROM fin_opp_hypothesis
		WHERE venue=$1 AND surface=$2`, venue, surface).Scan(&id)
	return id, err
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.watch.AttachAlerter(a); return r }

// ScanOnce 单拍壳（工具/实证件消费）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }
