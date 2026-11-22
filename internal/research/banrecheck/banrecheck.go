// Package banrecheck —— 模板禁前提复核拍（工单O-33393③——误杀探测器）：
// fin_opp_template enabled=f ∧ ban_premise 非空行的禁用前提当可证伪命题
// 周期重核——前提证伪→升件复审（非自动解禁，解禁=判词件）；前提仍真→
// ops_event 复核行续命（「活着的禁令」可审计）；owner_order 类=业主线
// 不机核仅龄报。复核拍独立于提案面——眼死了探照灯也得亮（refire 吃提案
// 面的依赖病根修：hunger 复活闸证据源=fin_opp_eval，本臂证据源=venue
// 实测/产品面直读）。
package banrecheck

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/research/armwatch"
)

// Row 复核对象行（fin_opp_template 禁因三件套+capability 面端点族）。
type Row struct {
	ID             int64
	Template       string
	Venue          string
	Surface        string
	BanPremise     string
	BanPremiseKind string
	BannedAt       *time.Time
	EndpointFamily string // fin_capability_surface.endpoint_family（venue_capability 核径锚）
	ProductFamily  string
}

// Result 前提重核结论（Verdict 闭集：falsified=前提证伪/upheld=前提仍立/
// unknown=核径不可用或面外——unknown 不产行只计数）。
type Result struct {
	Verdict string
	Detail  string // 实证细节（判词锚——所查族/端点/读数）
}

// CheckFunc 前提重核缝（cmd 装配注入——research 域零 venue 依赖红线：
// venue API 调用闭包在 cmd/research 落地，本臂只管判/去重/落账）。
type CheckFunc func(ctx context.Context, r Row) (Result, error)

// Runner 禁前提复核岗（md 池=模板册/能力面/产品面正本；fin 池=ops_event
// 写径——esc 未挂=守位不核不炸，装配缺件如实缺升件）。
type Runner struct {
	pool    *pgxpool.Pool // PG-md——fin_opp_template/fin_capability_surface 正本
	cfg     *config.Client
	esc     *pgxpool.Pool // SG fin——ops_event 写径
	check   CheckFunc
	alerter *notify.Alerter
	watch   *armwatch.Watch
	rej     *rejects.Door
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg, watch: armwatch.New("模板禁前提复核"),
		rej: rejects.New("research", "banrecheck_rows_err", "banrecheck_check_err",
			"banrecheck_esc_err", "banrecheck_report_err", "banrecheck_stamp_err")}
}

// AttachEscPool ops_event 写径挂接（fin 域——md 池无 ops_event 面；
// nil=守位不核）。
func (r *Runner) AttachEscPool(p *pgxpool.Pool) *Runner { r.esc = p; return r }

// AttachCheck 前提重核缝挂接（venue_capability/apr_regime 派核径——
// nil=机核缺位只走 owner_order 龄报径）。
func (r *Runner) AttachCheck(fn CheckFunc) *Runner { r.check = fn; return r }

// AttachAlerter 业主告警通道挂接。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner {
	r.alerter = a
	r.watch.AttachAlerter(a)
	return r
}

// Run 复核拍在岗（周期拍——产品面无推送事件同 dcd 径；ev 帧驱动会把
// 复核拍绑进枚举风暴，不取）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	iv := 6 * time.Hour
	if v, err := r.cfg.GetFloat("research.banrecheck.scan_interval_min"); err == nil && v > 0 {
		iv = time.Duration(v) * time.Minute
	}
	tk := time.NewTicker(iv) // sched-exempt-ok 禁前提复核拍——慢周期判据重核
	// （6h 级，sigdist/silent 同例）：独立于提案面存活是本臂存在理由，
	// ev 帧驱动会把复核绑进行情流，账户静默窗=复核死窗不可接受。
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			if err := r.scan(ctx); err != nil {
				log.Printf("WARN [research/banrecheck] 拍败: %v", err)
			}
		}
	}
}

// ScanOnce 单拍壳（实证件消费）。
func (r *Runner) ScanOnce(ctx context.Context) error { return r.scan(ctx) }

// rows 复核对象集（enabled=f ∧ ban_premise 非空——candidate/测试面/
// 失考行天然不入；capability 面端点族随行供 venue_capability 核径）。
func (r *Runner) rows(ctx context.Context) ([]Row, error) {
	rows, err := r.pool.Query(ctx, `SELECT t.id,t.template,t.venue,t.surface,
		coalesce(t.ban_premise,''),coalesce(t.ban_premise_kind,''),t.banned_at,
		coalesce(cs.endpoint_family,''),coalesce(cs.product_family,'')
		FROM fin_opp_template t
		LEFT JOIN fin_capability_surface cs
			ON cs.venue=t.venue AND cs.surface=t.surface
		WHERE NOT t.enabled AND t.ban_premise IS NOT NULL
		  AND t.ban_premise_kind IS NOT NULL AND t.ban_premise_kind<>'none'
		ORDER BY t.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var rw Row
		if err := rows.Scan(&rw.ID, &rw.Template, &rw.Venue, &rw.Surface,
			&rw.BanPremise, &rw.BanPremiseKind, &rw.BannedAt,
			&rw.EndpointFamily, &rw.ProductFamily); err != nil {
			return nil, err
		}
		out = append(out, rw)
	}
	return out, rows.Err()
}

// scan 一拍：逐行按前提类派核→证伪升件/仍立复核行/不可核计数。
func (r *Runner) scan(ctx context.Context) error {
	if r.esc == nil {
		return nil // 守位——写径缺不核（装配缺件如实缺升件）
	}
	rs, err := r.rows(ctx)
	if err != nil {
		r.rej.Inc("banrecheck_rows_err")
		return fmt.Errorf("禁前提行读败: %w", err)
	}
	for _, rw := range rs {
		res, err := r.checkRow(ctx, rw)
		if err != nil {
			r.rej.Inc("banrecheck_check_err")
			log.Printf("WARN [research/banrecheck] %s@%s/%s 核径败: %v",
				rw.Template, rw.Venue, rw.Surface, err)
			continue // 单行核败不拖全集——下拍重核
		}
		r.stamp(ctx, rw, res.Verdict)
		switch res.Verdict {
		case "falsified":
			if err := r.escalate(ctx, rw, res); err != nil {
				r.rej.Inc("banrecheck_esc_err")
				return err
			}
		case "upheld", "owner_order":
			if err := r.reportUphold(ctx, rw, res); err != nil {
				r.rej.Inc("banrecheck_report_err")
				return err
			}
		}
	}
	return nil
}

// checkRow 前提类派核：owner_order=业主线不机核（仅龄报——越线防呆）；
// 余类走注入核径（nil=unknown 不产行）。
func (r *Runner) checkRow(ctx context.Context, rw Row) (Result, error) {
	if rw.BanPremiseKind == "owner_order" {
		return Result{Verdict: "owner_order",
			Detail: "业主令类前提——机核跳过仅龄报（业主线不归机器翻案）"}, nil
	}
	if r.check == nil {
		return Result{Verdict: "unknown", Detail: "核径未装配"}, nil
	}
	return r.check(ctx, rw)
}

// stamp 复核结论回写模板行（ban_last_verdict/ban_last_check_at——名册视图
// 「前提成不成立」同答键；写败计数续拍不炸——结论行/升件照常）。
func (r *Runner) stamp(ctx context.Context, rw Row, verdict string) {
	if _, err := r.pool.Exec(ctx,
		`UPDATE fin_opp_template SET ban_last_verdict=$2,ban_last_check_at=now() WHERE id=$1`,
		rw.ID, verdict); err != nil {
		r.rej.Inc("banrecheck_stamp_err")
		log.Printf("WARN [research/banrecheck] %s 结论回写败: %v", tag(rw), err)
	}
}

// escScope 升件去重键（模板×面——同模板异面各自立案不互抑）。
func escScope(rw Row) string {
	return fmt.Sprintf("TMPLBAN-%s-%s/%s", rw.Template, rw.Venue, rw.Surface)
}

// tag 行锚（复核行/升件 note 公共前缀——去重查询与人读同串）。
func tag(rw Row) string {
	return fmt.Sprintf("banrecheck %s@%s/%s", rw.Template, rw.Venue, rw.Surface)
}

// escalate 前提证伪→升件复审（非自动解禁——判词件归处置径）。
// 去重：ops_esc_pending 同 scope 在案=不重复报；24h 内同 scope 升件
// 已落=窗口内不再报（簇折叠面照旧）。
func (r *Runner) escalate(ctx context.Context, rw Row, res Result) error {
	var dup int
	if err := r.esc.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM ops_esc_pending WHERE esc_scope=$1)
		+(SELECT count(*) FROM ops_event WHERE event='升件' AND esc_scope=$1
			AND ts>now()-interval '24 hours')`,
		escScope(rw)).Scan(&dup); err != nil {
		return fmt.Errorf("升件去重读败: %w", err)
	}
	if dup > 0 {
		return nil
	}
	age := ""
	if rw.BannedAt != nil {
		age = fmt.Sprintf(" 禁龄=%.1fd", time.Since(*rw.BannedAt).Hours()/24)
	}
	note := fmt.Sprintf("%s 前提证伪——kind=%s 实证=%s。%s——复审非自动解禁（解禁=判词件，引证据落 ops_event 判词行）",
		tag(rw), rw.BanPremiseKind, res.Detail, rw.BanPremise)
	if _, err := r.esc.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('research/banrecheck','升件','O-33393',$1,'判词复审',$2)`,
		note+age, escScope(rw)); err != nil {
		return fmt.Errorf("证伪升件行: %w", err)
	}
	log.Printf("[research/banrecheck] %s 前提证伪升件（%s）", tag(rw), res.Detail)
	return nil
}

// reportUphold 前提仍立→复核行续命（report_hours 窗内同模板不重报——
// 「活着的禁令」可审计节奏，默认 72h）。
func (r *Runner) reportUphold(ctx context.Context, rw Row, res Result) error {
	hours := 72.0
	if v, err := r.cfg.GetFloat("research.banrecheck.report_hours"); err == nil && v > 0 {
		hours = v
	}
	var last *time.Time
	if err := r.esc.QueryRow(ctx, `SELECT max(ts) FROM ops_event
		WHERE seat='research/banrecheck' AND event='记事' AND note LIKE '%'||$1||'%'`,
		tag(rw)).Scan(&last); err != nil {
		return fmt.Errorf("复核行去重读败: %w", err)
	}
	if last != nil && time.Since(*last) < time.Duration(hours)*time.Hour {
		return nil
	}
	age := "禁龄=未知"
	if rw.BannedAt != nil {
		age = fmt.Sprintf("禁龄=%.1fd", time.Since(*rw.BannedAt).Hours()/24)
	}
	note := fmt.Sprintf("%s 前提仍立——kind=%s %s（%s）。%s",
		tag(rw), rw.BanPremiseKind, res.Detail, age, rw.BanPremise)
	if _, err := r.esc.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('research/banrecheck','记事','O-33393',$1)`, note); err != nil {
		return fmt.Errorf("复核行: %w", err)
	}
	return nil
}
