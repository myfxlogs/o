// Package gapscan —— 差扫在编岗（T-0102 v3/v4，D-0012 检测层永不离线）：
//   - 周期枚举 venue 可枚举目录（okx /finance/* 产品族先落；mt5 目录枚举留
//     注册表 placeholder 不施工）→ 与能力面注册表对拍 → 未记面写 fin_surface_gap
//     行（幂等：last_seen 刷新不重复建行）。
//   - v4 面分类器：gap 行产品元数据→模板族映射（收益锁仓型→carry；永续含
//     funding→对冲；CFD swap 正值→carry/负值→对冲腿或趋势；期权型→波动〔后期〕）。
//     分类命中→自动生候选 fin_opp_eval/fin_proposal（provenance=surface_gap）
//     走既有毕业线——generated 策略只许 paper 段（lifecycle 既有链，无特殊通道），
//     并发生成上限 config 键（默认 3）防模板洪水灌爆沙箱。
//   - 闸下死面（读面不通/零产品/无收益件）→ autorejected+拒因；分不出族进
//     unclassified 等会话（罕见件才用人）。
package gapscan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
	"o/internal/research/armwatch"
)

// ProbeFace 目录探针面（目录种子——只收技能档/文档锚实过验的面，禁猜端点）。
type ProbeFace struct {
	Venue    string
	Surface  string
	Endpoint string
	Kind     string // 收益形态提示（locked-yield/…——分类器元数据辅助位）
}

// OKXCatalog okx 可枚举目录种子（面名以 okx-asset-funding 档实测端点为准；
// 目录扩面=补行+证据，非猜端点）。
var OKXCatalog = []ProbeFace{
	{Venue: "okx", Surface: "dcd", Endpoint: "/api/v5/finance/sfp/dcd/currency-pair", Kind: "locked-yield"},
	{Venue: "okx", Surface: "earn-flex", Endpoint: "/api/v5/finance/savings/lending-rate-summary?ccy=USDT", Kind: "locked-yield"},
}

// ProbeResult 面探针结论（glue 由响应形提取——products 计数+收益带位）。
type ProbeResult struct {
	Products     int
	YieldBearing bool
}

// FaceAbsentError 面缺席（路由 404/405——面名错或面下架，区别于瞬态读败）。
type FaceAbsentError struct{ Status int }

func (e *FaceAbsentError) Error() string { return fmt.Sprintf("face absent http %d", e.Status) }

// FaceBizError 面在但业务拒（200+code≠0——面存在信号，非死面非瞬态）。
type FaceBizError struct{ Code, Msg string }

func (e *FaceBizError) Error() string {
	return fmt.Sprintf("face biz reject code=%s msg=%s", e.Code, e.Msg)
}

// Probe 面探针读件（注入缝——research 域零 venue 依赖，红线 research↛trading；
// 生产闭包=cmd 装配 okcli 签名 GET，测试=夹具）。
type Probe func(ctx context.Context, f ProbeFace) (ProbeResult, error)

// Runner 差扫在编岗（心跳+卡面可见；gap 队列写+分类器裁态+候选生成）。
type Runner struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	probe Probe
	watch *armwatch.Watch // 臂活性看门（T-0117——四臂共用件）
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New("差扫"),
		rej: rejects.New("research", "gap_probe_failed", "gap_write_err",
			"gap_gen_cap", "gap_registry_read_failed")}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.watch.AttachAlerter(a); return r }

// AttachProbe 挂面探针读件（cmd 装配层签名 GET 闭包）。
func (r *Runner) AttachProbe(fn Probe) *Runner { r.probe = fn; return r }

// Run 在编岗（注册+心跳 30s+慢周期差扫——首拍即扫）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "差扫", "research",
		"面差哨——目录差扫+面分类（检测层永不离线）", nil); err != nil {
		errc <- fmt.Errorf("差扫 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "差扫", 30*time.Second, errc)
	for {
		interval := 360 * time.Minute
		if v, err := r.cfg.GetFloat("research.gap.scan_interval_min"); err == nil && v > 0 {
			interval = time.Duration(v * float64(time.Minute))
		}
		r.watch.SetBeat(interval)
		r.watch.Check(ctx) // 臂静默判定（T-0117——周期差扫 2×节拍窗）
		if err := r.scan(ctx); err != nil {
			errc <- fmt.Errorf("gapscan: %w", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval): // sched-exempt-ok T-0102 v3 钦定周期枚举（目录面无推送事件，D-0012 检测层永不离线——慢拍热启）
		}
	}
}

// scan 一拍：目录探针→注册表对拍→gap 队列→分类器裁态+候选生成。
func (r *Runner) scan(ctx context.Context) error {
	// T-0244 期界包络（dcd 同型——无界长扫挂死径封；gapscan.scan_timeout_s 热启缺=480s）
	to := 8 * time.Minute
	if v, err := r.cfg.GetFloat("gapscan.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx

	if r.probe == nil {
		log.Printf("WARN [research/gap] 面探针缺席（凭据落位后重启即上岗）")
		return nil
	}
	registered, err := r.registryFaces(ctx)
	if err != nil {
		r.rej.Inc("gap_registry_read_failed")
		return fmt.Errorf("注册表读败: %w", err)
	}
	for _, f := range OKXCatalog {
		if f.Venue != "okx" {
			continue // 本票范围：okx 目录枚举先落（mt5 留 placeholder 不施工）
		}
		res, err := r.probe(ctx, f)
		switch {
		case err == nil:
			if err := r.faceAlive(ctx, f, res, registered); err != nil {
				return err
			}
		default:
			var absent *FaceAbsentError
			var biz *FaceBizError
			switch {
			case asErr(err, &absent):
				if err := r.faceDead(ctx, f, fmt.Sprintf("http %d", absent.Status)); err != nil {
					return err
				}
			case asErr(err, &biz):
				// 面在但业务拒——非死面：gap 行维持/立 pending（拒码留 metadata）
				if err := r.faceAlive(ctx, f, ProbeResult{Products: -1}, registered); err != nil {
					return err
				}
				r.rej.Inc("gap_probe_failed")
				log.Printf("WARN [research/gap] 面 %s 业务拒 code=%s（面在——pending 不裁死）", f.Surface, biz.Code)
			default:
				r.rej.Inc("gap_probe_failed")
				log.Printf("WARN [research/gap] 面 %s 瞬态读败——本拍让位不裁（fail-closed）: %v", f.Surface, err)
			}
		}
	}
	r.watch.OK() // 差扫拍完成（目录探针轮毕=产出——死面裁定亦产出）
	return r.classifyPending(ctx)
}

// faceAlive 面通：未注册→gap 行立/刷（幂等）；零产品/无收益=闸下死面。
func (r *Runner) faceAlive(ctx context.Context, f ProbeFace, res ProbeResult, registered map[string]bool) error {
	if registered[f.Venue+"/"+f.Surface] {
		return nil // 已注册面——无差（对拍通过）
	}
	if res.Products == 0 {
		return r.markGap(ctx, f, "autorejected", map[string]any{"products": 0}, "zero_products")
	}
	if res.Products < 0 { // 业务拒形——面在，元数据缺
		return r.markGap(ctx, f, "pending", map[string]any{"biz_reject": true}, "")
	}
	if !res.YieldBearing && f.Kind == "locked-yield" {
		return r.markGap(ctx, f, "autorejected", map[string]any{"products": res.Products}, "no_yield")
	}
	return r.markGap(ctx, f, "pending", map[string]any{
		"products": res.Products, "yield_bearing": res.YieldBearing, "kind": f.Kind}, "")
}

// faceDead 面缺席（目录内实名单面走暗——autorejected 留痕+拒因）。
func (r *Runner) faceDead(ctx context.Context, f ProbeFace, reason string) error {
	return r.markGap(ctx, f, "autorejected", map[string]any{"kind": f.Kind}, reason)
}

// markGap gap 行立/刷（幂等 UNIQUE(venue,surface)——last_seen 刷新不重复建行；
// 已终态行（registered/rejected）不被机器翻案，autorejected 复活面回 pending）。
func (r *Runner) markGap(ctx context.Context, f ProbeFace, state string, meta map[string]any, reason string) error {
	metaB, _ := json.Marshal(meta)
	var cur string
	err := r.pool.QueryRow(ctx, `SELECT state FROM fin_surface_gap
		WHERE venue=$1 AND surface=$2`, f.Venue, f.Surface).Scan(&cur)
	switch {
	case err == nil:
		next := state
		if cur == "registered" || cur == "rejected" || cur == "hypothesized" {
			next = cur // 终态/已起草不机器翻案（T-0106：hypothesized=卡已立）
		}
		if _, err := r.pool.Exec(ctx, `UPDATE fin_surface_gap SET state=$3, metadata=$4,
			reason=$5, last_seen=now() WHERE venue=$1 AND surface=$2`,
			f.Venue, f.Surface, next, metaB, reason); err != nil {
			r.rej.Inc("gap_write_err")
			return err
		}
	case isNoRows(err):
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_surface_gap(venue,surface,state,metadata,reason)
			VALUES($1,$2,$3,$4,$5)`, f.Venue, f.Surface, state, metaB, reason); err != nil {
			r.rej.Inc("gap_write_err")
			return err
		}
	default:
		return err
	}
	return nil
}

// registryFaces 已注册面键集（venue/surface——对拍 baseline）。
func (r *Runner) registryFaces(ctx context.Context) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx, `SELECT venue||'/'||surface FROM fin_capability_surface`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out[k] = true
	}
	return out, rows.Err()
}
