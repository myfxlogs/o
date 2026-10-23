package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/rejects"
	"phase/internal/platform/roster"
)

const scanEvery = 60 * time.Second

var critKeys = []string{
	"invest.lifecycle.paper_min_hours", "invest.lifecycle.paper_min_net_usd",
	"invest.lifecycle.paper_max_dd_usd", "invest.lifecycle.live_switch",
	"invest.paper.notional_usd",
}

// Runner 命官岗：提案→paper 开户→虚拟结算→判据晋升。
type Runner struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	rej  *rejects.Door
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg,
		rej: rejects.New("lifecycle", "illegal_transition", "missing_criteria", "write_err", "live_switch_off")}
}

func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "命官", "portfolio", "生命周期官", nil); err != nil {
		errc <- fmt.Errorf("命官 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "命官", 30*time.Second, errc)
	for {
		if err := r.scan(ctx); err != nil {
			log.Printf("[lifecycle] scan: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(scanEvery):
		}
	}
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
	if err := r.admitProposals(ctx, notional, minH, minNet, maxDD); err != nil {
		return err
	}
	if err := r.paperAccounting(ctx); err != nil {
		return err
	}
	return r.promote(ctx, minH, minNet, maxDD, liveSw)
}

// admitProposals 新提案开户（state=paper；criteria 快照入行——判据代次留痕）。
func (r *Runner) admitProposals(ctx context.Context, notional, minH, minNet, maxDD float64) error {
	rows, err := r.pool.Query(ctx, `SELECT p.id,p.inst_id,p.direction FROM fin_proposal p
		WHERE p.state='fin_proposed' AND NOT EXISTS(
			SELECT 1 FROM fin_strategy s WHERE s.proposal_ref=p.id)`)
	if err != nil {
		return err
	}
	type prop struct {
		id   int64
		inst string
		dir  string
	}
	var ps []prop
	for rows.Next() {
		var x prop
		if err := rows.Scan(&x.id, &x.inst, &x.dir); err != nil {
			rows.Close()
			return err
		}
		ps = append(ps, x)
	}
	rows.Close()
	for _, x := range ps {
		crit, _ := json.Marshal(map[string]float64{"min_hours": minH, "min_net": minNet, "max_dd": maxDD})
		var sid int64
		err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,criteria)
			VALUES($1,'funding-hedge',$2,'paper',$3,$4) RETURNING id`, x.inst, x.id, notional, crit).Scan(&sid)
		if err != nil {
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
	return nil
}
