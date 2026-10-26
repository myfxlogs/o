package lifecycle

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
	"o/internal/platform/roster"
)

var critKeys = []string{
	"invest.lifecycle.paper_min_hours", "invest.lifecycle.paper_min_net_usd",
	"invest.lifecycle.paper_max_dd_usd", "invest.lifecycle.live_switch",
	"invest.paper.notional_usd",
}

// Runner 命官岗：提案→paper 开户→虚拟结算→判据晋升→live 开仓消费（T-0050）。
type Runner struct {
	pool   *pgxpool.Pool
	denyAt map[int64]time.Time // T-0136 ③拒放冷窗（策略→最近拒放戳）

	cfg *config.Client
	op  Opener
	rej *rejects.Door
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, denyAt: map[int64]time.Time{}, cfg: cfg,
		rej: rejects.New("lifecycle", "illegal_transition", "missing_criteria", "write_err", "live_switch_off", "open_failed", "template_blocked", "xvenue_open_forbidden", "blacklist_retire",
			"facts_missing", "missing_param", "oscillation_fuse", "dcd_quote_missing")}
}

func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "命官", "portfolio", "生命周期官", nil); err != nil {
		errc <- fmt.Errorf("命官 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "命官", 30*time.Second, errc)
	// 事件化（T-0048：周期环拆——策略/评估卡行变 ev_strategy/ev_eval 驱动）。
	events.ListenLoop(ctx, r.pool, "lifecycle",
		[]string{"ev_strategy", "ev_eval"}, r.scan)
}

func (r *Runner) getF(key string) (float64, error) { return r.cfg.GetFloat(key) }

// criteria 全键读（缺任一键=fail-closed 整拍停——判据不虚构）。
func (r *Runner) criteria() (minH, minNet, maxDD, notional float64, liveSw bool, err error) {
	if minH, err = r.getF(critKeys[0]); err != nil {
		return
	}
	if minNet, err = r.getF(critKeys[1]); err != nil {
		return
	}
	if maxDD, err = r.getF(critKeys[2]); err != nil {
		return
	}
	if notional, err = r.getF(critKeys[4]); err != nil {
		return
	}
	liveSw, err = r.cfg.GetBool(critKeys[3])
	return
}

// scan 一拍：①fin_proposed 提案→paper 开户 ②paper 吃 funding 新行虚拟记账
// ③paper 满判据→probe 迁移留痕（live 仍闸下——本票止于 probe）。
func (r *Runner) scan(ctx context.Context) error {
	minH, minNet, maxDD, notional, liveSw, err := r.criteria()
	if err != nil {
		r.rej.Inc("missing_criteria")
		return fmt.Errorf("lifecycle 判据缺位 fail-closed: %w", err)
	}
	if err := r.retireStaleDegraded(ctx); err != nil {
		return err
	}
	if err := r.admitProposals(ctx, notional, minH, minNet, maxDD); err != nil {
		return err
	}
	if err := r.dcdPaperAccounting(ctx); err != nil {
		log.Printf("[lifecycle] dcd paper 记账败: %v", err)
	}
	if err := r.paperAccounting(ctx); err != nil {
		return err
	}
	if err := r.promote(ctx, minH, minNet, maxDD, liveSw); err != nil {
		return err
	}
	// ④ live 开仓消费（T-0050 首仓序④——无未平仓行的 live 行过 Opener 缝）。
	r.openLive(ctx)
	return nil
}

// paperTemplates 有 paper 径的模板族白名单（T-0124 模板门闸——无 executor 的
// 模板提案不晋升，留 fin_proposed 待 executor 票；扩张新族须在此登记）。
var paperTemplates = map[string]bool{"funding-hedge": true, "dcd_call": true} // T-0160：dcd_call 接 paper 径

// admitProposals 新提案开户（state=paper；criteria 快照入行——判据代次留痕；
// 模板门闸：join fin_opp_eval 取模板，无 paper 径族拦下不建策略）。
func (r *Runner) admitProposals(ctx context.Context, notional, minH, minNet, maxDD float64) error {
	rows, err := r.pool.Query(ctx, `SELECT p.id,p.inst_id,p.direction,e.template FROM fin_proposal p
		JOIN fin_opp_eval e ON e.id=p.eval_id
		WHERE p.state='fin_proposed' AND NOT EXISTS(
			SELECT 1 FROM fin_strategy s WHERE s.proposal_ref=p.id)`)
	if err != nil {
		return err
	}
	type prop struct {
		id   int64
		inst string
		dir  string
		tmpl string
	}
	var ps []prop
	for rows.Next() {
		var x prop
		if err := rows.Scan(&x.id, &x.inst, &x.dir, &x.tmpl); err != nil {
			rows.Close()
			return err
		}
		ps = append(ps, x)
	}
	rows.Close()
	blocked := map[string]int{}
	for _, x := range ps {
		if !paperTemplates[x.tmpl] {
			blocked[x.tmpl]++
			continue // executor 缺席=看见了做不了——提案留 proposed（模板门闸，T-0124）
		}
		if x.tmpl == "xvenue-carry" {
			if err := r.admitXvenuePair(ctx, x.id, x.inst, x.dir, notional, minH, minNet, maxDD); err != nil {
				r.rej.Inc("write_err")
				return err
			}
			continue // 跨所双腿径（T-0144——主从策略对，proposal_ref 同引）
		}
		crit, _ := json.Marshal(map[string]float64{"min_hours": minH, "min_net": minNet, "max_dd": maxDD})
		var sid int64
		// 同键去重（T-0022：同 venue+inst_id+direction+template 同时最多一个非
		// retired 策略行——T-0124 键绑模板防跨族串仓；已挂 active 行=复用不新建，
		// 提案照样 armed 被吸收）。
		err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,criteria,direction)
			VALUES($1,$2,$3,'paper',$4,$5,$6)
			ON CONFLICT (venue,inst_id,direction,template_ref) WHERE state<>'retired' DO NOTHING
			RETURNING id`, x.inst, x.tmpl, x.id, notional, crit, x.dir).Scan(&sid)
		if err != nil {
			if err.Error() == "no rows in result set" {
				if _, err := r.pool.Exec(ctx,
					"UPDATE fin_proposal SET state='fin_armed' WHERE id=$1", x.id); err != nil {
					return err
				}
				log.Printf("[lifecycle] %s %s 同键 active 行在挂——提案 armed 复用不叠仓", x.inst, x.dir)
				continue
			}
			r.rej.Inc("write_err")
			return err
		}
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
			VALUES($1,'proposed','paper','验证官提案过阈→paper 沙箱开户')`, sid); err != nil {
			return err
		}
		if _, err := r.pool.Exec(ctx,
			"UPDATE fin_proposal SET state='fin_armed' WHERE id=$1", x.id); err != nil {
			return err
		}
		log.Printf("[lifecycle] #%d %s paper 开户 notional=%.0f", sid, x.inst, notional)
	}
	if len(blocked) > 0 {
		r.rej.Inc("template_blocked")
		log.Printf("[lifecycle] 模板门闸拦下 %+v（无 paper 径——提案留 fin_proposed 待 executor 票）", blocked)
	}
	return r.retireStaleArmed(ctx)
}

// retireStaleDegraded degraded 超龄退役（T-0098 方案 A：degraded=暂停待复盘——
// 有界暂停合莫环自净，无限期暂停=墓碑锁同键去重，BTC #1 实证 860 提案被吸）。
// TTL=config strategy.degraded_ttl_hours，键缺/非正=默认 24h（自疗径不配键不瘫）。
func (r *Runner) retireStaleDegraded(ctx context.Context) error {
	ttl := 24.0
	if v, err := r.cfg.GetFloat("strategy.degraded_ttl_hours"); err == nil && v > 0 {
		ttl = v
	}
	rows, err := r.pool.Query(ctx, `SELECT id FROM fin_strategy
		WHERE state='degraded' AND entered_at < now() - make_interval(hours => $1)`, ttl)
	if err != nil {
		r.rej.Inc("write_err")
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := Transition(ctx, r.pool, id, "retired",
			"degraded 超龄自动退役——莫环重生窗", false); err != nil {
			r.rej.Inc("write_err")
			return err
		}
		log.Printf("[lifecycle] #%d degraded 超 %.0fh 自动退役（莫环重生窗——同键解锁复评）", id, ttl)
	}
	return nil
}

// retireStaleArmed armed 提案卫生闸（T-0069：同键只留最新一行 armed——
// 被吸收/被取代的旧武装退役，表不再随周期线性胀；paper/probe/live 语义零触）。
func (r *Runner) retireStaleArmed(ctx context.Context) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fin_proposal SET state='fin_retired'
		WHERE state='fin_armed' AND id NOT IN (
			SELECT max(id) FROM fin_proposal WHERE state='fin_armed'
			GROUP BY inst_id,direction)`)
	if err == nil && tag.RowsAffected() > 0 {
		log.Printf("[lifecycle] armed 退役 %d 行（同键只留最新——淤积卫生闸）", tag.RowsAffected())
	}
	// T-0136 ②armed 超龄退役：提案/策略 armed 超 lifecycle.armed_ttl_h
	//（缺省 24h）未开火→自动退役+留痕（释放额度预算+T-0069 淤积同族治）。
	ttl := 24.0
	if v, err := r.cfg.GetFloat("lifecycle.armed_ttl_h"); err == nil && v > 0 {
		ttl = v
	}
	tag2, err2 := r.pool.Exec(ctx, `UPDATE fin_proposal SET state='fin_retired'
		WHERE state='fin_armed' AND created_at<now()-make_interval(hours=>$1)`, ttl)
	if err2 == nil && tag2.RowsAffected() > 0 {
		log.Printf("[lifecycle] armed 提案超 %.0fh 退役 %d 行（未开火额度释放）", ttl, tag2.RowsAffected())
	}
	// 返修①A：策略退役谓词收紧——live/probe 态+无 open 持仓+最近活动戳
	//（fin_position 末次开仓时间）超 ttl 才退役；有仓/近期开过的不误杀。
	tag3, err3 := r.pool.Exec(ctx, `UPDATE fin_strategy s SET state='retired'
		WHERE s.state IN ('live','probe') AND NOT EXISTS(
			SELECT 1 FROM fin_position p WHERE p.strategy_ref=s.id AND p.closed_at IS NULL)
		AND COALESCE((SELECT max(p2.opened_at) FROM fin_position p2 WHERE p2.strategy_ref=s.id),
			s.entered_at)<now()-make_interval(hours=>$1)`, ttl)
	if err3 == nil && tag3.RowsAffected() > 0 {
		log.Printf("[lifecycle] armed 策略超 %.0fh 退役 %d 行（未开火额度释放）", ttl, tag3.RowsAffected())
	}
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	return err3
}
