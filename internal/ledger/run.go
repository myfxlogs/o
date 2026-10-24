package ledger

import (
	"context"
	"fmt"
	"log"
	"phase/internal/platform/events"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/notify"
	"phase/internal/platform/rejects"
	"phase/internal/platform/roster"
)

// Runner 司账+核账双岗（结算闭环+对账环——账本只消费事实行不触发钱动作）。
type Runner struct {
	al *notify.Alerter
	// bal venue 侧余额现读件（T-0048：对账=检查时点 balances+savings 直连——
	// (venue,ccy)→equity 实读聚合；nil=对账件缺席跳过不炸）。
	bal  func(ctx context.Context) (map[[2]string]float64, error)
	pool *pgxpool.Pool
	cfg  *config.Client
	rej  *rejects.Door
}

// WithAlerter 挂告警器（nil=静默降级）。
func (r *Runner) WithAlerter(a *notify.Alerter) *Runner { r.al = a; return r }

// WithBalances 挂 venue 侧余额现读件（cmd 装配层接线——对账检查时点直读）。
func (r *Runner) WithBalances(fn func(ctx context.Context) (map[[2]string]float64, error)) *Runner {
	r.bal = fn
	return r
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg,
		rej: rejects.New("ledger", "diff_alert", "missing_param", "write_err")}
}

func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	for _, e := range [][3]string{{"司账", "ledger", "入账+结算岗"}, {"核账", "ledger", "对账环岗"}} {
		if err := roster.Register(ctx, r.pool, e[0], e[1], e[2], nil); err != nil {
			errc <- fmt.Errorf("%s register: %w", e[0], err)
			return
		}
		go roster.BeatLoop(ctx, r.pool, e[0], 30*time.Second, errc)
	}
	// 事件化（T-0048：周期环拆——cashflow 写+venue 余额帧+仓行变驱动）。
	events.ListenLoop(ctx, r.pool, "ledger",
		[]string{"ev_cashflow", "ev_account", "ev_position"}, r.scan)
}

func (r *Runner) tol(ctx context.Context) (float64, error) {
	return r.cfg.GetFloat("ledger.recon_tol_usd")
}

// scan 一拍：①closed 仓结算落位 ②对账环（快照 vs 净额，差即计门）③日切快照。
func (r *Runner) scan(ctx context.Context) error {
	if err := r.settle(ctx); err != nil {
		return err
	}
	if err := r.reconcile(ctx); err != nil {
		return err
	}
	return r.daySnapshot(ctx)
}

// settle closed 未结算仓：净额=Σcashflow（position_ref 过滤）落 settled_*。
func (r *Runner) settle(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT id FROM fin_position
		WHERE closed_at IS NOT NULL AND settled_at IS NULL`)
	if err != nil {
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
		var net float64
		if err := r.pool.QueryRow(ctx,
			"SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE position_ref=$1", id).Scan(&net); err != nil {
			return err
		}
		if _, err := r.pool.Exec(ctx,
			"UPDATE fin_position SET settled_amount=$2,settled_ccy='USDT',settled_at=now() WHERE id=$1", id, net); err != nil {
			r.rej.Inc("write_err")
			return err
		}
		log.Printf("[ledger] 仓 #%d 结算落位 net=%.4f USDT", id, net)
	}
	return nil
}

// reconcile 快照 equity vs 账本净额（venue×ccy 配对；容差热启，差即 diff_alert 计门——
// 快照陈>5min 不判〔okx-private-ws 契约：静户零事件≠WS 死，陈=fail-closed 不单边判〕）。
func (r *Runner) reconcile(ctx context.Context) error {
	tol, err := r.tol(ctx)
	if err != nil {
		r.rej.Inc("missing_param")
		return err
	}
	// T-0048 根修：venue 侧=检查时点 balances+savings 现读聚合（快照行出局——
	// 判定面只读现读；读件缺席=对账面让位不炸——司账/结算径仍事件驱动）。
	if r.bal == nil {
		return nil
	}
	ven, err := r.bal(ctx)
	if err != nil {
		r.rej.Inc("write_err")
		return fmt.Errorf("venue 现读: %w", err)
	}
	var rs []struct {
		venue, ccy string
		eq, net    float64
	}
	for k, eq := range ven {
		if k[1] != "USDT" && k[1] != "USD" {
			continue
		}
		var net float64
		_ = r.pool.QueryRow(ctx,
			"SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE venue=$1 AND ccy=$2",
			k[0], k[1]).Scan(&net)
		rs = append(rs, struct {
			venue, ccy string
			eq, net    float64
		}{k[0], k[1], eq, net})
	}
	for _, x := range rs {
		if Diff(x.net, x.eq) > tol {
			r.rej.Inc("diff_alert")
			r.al.Alert(ctx, "ledger_diff_alert", "账实差超限——人工对拍")
			log.Printf("ALERT [ledger] 对账差 %s/%s venue=%.4f ledger=%.4f（容差 %.2f）", x.venue, x.ccy, x.eq, x.net, tol)
		}
	}
	return nil
}

// daySnapshot 日切（venue×ccy 净额行——NAV 底数；USDT/USD 各记各的行零折算）。
func (r *Runner) daySnapshot(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_ledger_snapshot(venue,ccy,net_usd,snap_date)
		SELECT venue,ccy,sum(amount),CURRENT_DATE FROM fin_cashflow GROUP BY venue,ccy
		ON CONFLICT(venue,ccy,snap_date) DO UPDATE SET net_usd=EXCLUDED.net_usd,ts=now()`)
	return err
}
