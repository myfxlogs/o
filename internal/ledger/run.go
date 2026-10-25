package ledger

import (
	"context"
	"fmt"
	"log"
	"o/internal/platform/events"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
)

// Runner 司账+核账双岗（结算闭环+对账环——账本只消费事实行不触发钱动作）。
type Runner struct {
	al *notify.Alerter
	// mtm 对账净值偏移读件（T-0076 合约盯市增量+T-0087 现货全值并轨——venue
	// 权益含现货资产市值随价动，ledger 侧现货腿加全值、合约腿加增量使 NAV 同
	// 口径；缺件=0 不判 fail-closed 让位）。
	mtm func(ctx context.Context) (map[[2]string]float64, error)
	// bal venue 侧余额现读件（T-0048：对账=检查时点 balances+savings 直连——
	// (venue,ccy)→equity 实读聚合；nil=对账件缺席跳过不炸）。
	bal func(ctx context.Context) (map[[2]string]float64, error)
	// earnings savings 累计收益读件（T-0055 利息入账源）。
	earnings func(ctx context.Context) (map[string]float64, error)
	pool     *pgxpool.Pool
	cfg      *config.Client
	rej      *rejects.Door
}

// WithAlerter 挂告警器（nil=静默降级）。
func (r *Runner) WithAlerter(a *notify.Alerter) *Runner { r.al = a; return r }

// WithEarnEarnings 挂 savings 累计收益读件（T-0055——cmd 注入 ccy→earnings 累计值；
// Earn 利息 bills 无此面实证，唯 savings/balance.earnings 字段为真源）。
func (r *Runner) WithEarnEarnings(fn func(ctx context.Context) (map[string]float64, error)) *Runner {
	r.earnings = fn
	return r
}

// WithBalances 挂 venue 侧余额现读件（cmd 装配层接线——对账检查时点直读）。
func (r *Runner) WithBalances(fn func(ctx context.Context) (map[[2]string]float64, error)) *Runner {
	r.bal = fn
	return r
}

// WithMTM 挂持仓盯市偏移读件（T-0076——venue×ccy 偏移额；无持仓=空 map）。
func (r *Runner) WithMTM(fn func(ctx context.Context) (map[[2]string]float64, error)) *Runner {
	r.mtm = fn
	return r
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg,
		rej: rejects.New("ledger", "diff_alert", "missing_param", "bal_missing", "write_err",
			"recon_probe_failed")}
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
			"SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE position_ref=$1 AND kind NOT IN ('earn_in','earn_out')", id).Scan(&net); err != nil {
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
		r.rej.Inc("bal_missing")
		log.Printf("WARN [ledger] 对账 venue 现读件未挂接——对账缺岗可见（fail-closed 让位）")
		return nil
	}
	// T-0062 根修③+T-0087 扩面：在途换形/划转窗 debounce——30s 内有执行活动
	// （leg/execution）或划转活动=本轮跳过不判，容差不降。划转触发面两件：
	// fin_cashflow earn_in/earn_out 行（票面钦定）+fin_risk_decision transfer 行
	// （T-0060 停写 earn 行后划转的唯一实况留痕，afterMove 每动必落）。
	// 列名勘误：fin_order_leg 时间列=ts、fin_execution=created_at（0009 schema）
	// ——旧式 created_at/opened_at 恒报错被吞，debounce 自立起未生效过（T-0087 实证）。
	var inflight bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM fin_order_leg WHERE ts > now()-interval '30 seconds'
		UNION SELECT 1 FROM fin_execution WHERE created_at > now()-interval '30 seconds'
		UNION SELECT 1 FROM fin_cashflow WHERE kind IN ('earn_in','earn_out')
			AND ts > now()-interval '30 seconds'
		UNION SELECT 1 FROM fin_risk_decision WHERE reason='transfer'
			AND ts > now()-interval '30 seconds')`).Scan(&inflight); err != nil {
		r.rej.Inc("recon_probe_failed")
		log.Printf("WARN [ledger] 在途窗探测败——本轮对账让位不判: %v", err)
		return nil
	}
	if inflight {
		log.Printf("[ledger] 对账跳过——执行/划转在途窗（30s 内活动，不判不警）")
		return nil
	}
	// T-0055：Earn 利息先入账再对拍（savings amt 已含应计利息——不入账=账本
	// 系统性偏低且逐日走阔；事件化高水位幂等）。
	r.postInterest(ctx)
	ven, err := r.bal(ctx)
	if err != nil {
		r.rej.Inc("write_err")
		return fmt.Errorf("venue 现读: %w", err)
	}
	// T-0100 ①③：对拍白名单=基金的钱口径 venue 域折叠（FundCcy 账户域作用
	// 集折叠成 venue×ccy——venue 侧 fundSaveBalances 同源只发域内键）；非
	// 本位键 ledger 侧按币量建键——net=cashflow净额（非本位恒 0）+ParkedAmt
	// 停泊台账币数，venue=币数同径比对，单位错配爆雷根除（USD 归一另一径
	// 弃用：台账币数对 venue 币数零价漂）。
	wl := map[[2]string]bool{}
	set, err := FundCcy(ctx, r.pool)
	if err != nil {
		r.rej.Inc("write_err")
		return err
	}
	for k := range set {
		v := k[0]
		if i := strings.IndexByte(v, ':'); i >= 0 {
			v = v[:i] // 账户域作用键折叠回 venue 域
		}
		wl[[2]string{v, k[1]}] = true
	}
	parked, err := ParkedAmt(ctx, r.pool)
	if err != nil {
		r.rej.Inc("write_err")
		return err
	}
	var rs []struct {
		venue, ccy string
		eq, net    float64
	}
	for k, eq := range ven {
		if !wl[k] {
			continue
		}
		var net float64
		_ = r.pool.QueryRow(ctx,
			"SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE venue=$1 AND ccy=$2 AND kind NOT IN ('earn_in','earn_out')",
			k[0], k[1]).Scan(&net)
		net += parked[k] // 非本位停泊币台账币数（venue=币量同径）
		// T-0076：net+盯市偏移=同口径 NAV（持仓浮动与真差分离——偏移读败=0 不加不减）。
		if r.mtm != nil {
			if mv, merr := r.mtm(ctx); merr == nil {
				net += mv[k]
			}
		}
		rs = append(rs, struct {
			venue, ccy string
			eq, net    float64
		}{k[0], k[1], eq, net})
	}
	for _, x := range rs {
		// T-0081 已知残差豁免：未过期豁免额补进容差——只压已定性部分，
		// unexplained 超额仍即报（豁免≠白名单静默一切）。
		var exempt float64
		_ = r.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount_usd),0) FROM fin_known_residual
			WHERE venue=$1 AND ccy=$2 AND expires_at>now()`, x.venue, x.ccy).Scan(&exempt)
		if Diff(x.net, x.eq) > tol+exempt {
			r.rej.Inc("diff_alert")
			r.al.Alert(ctx, "ledger_diff_alert",
				fmt.Sprintf("账实差 $%.2f（容差 $%.2f +已定性豁免 $%.2f）",
					Diff(x.net, x.eq), tol, exempt))
			log.Printf("ALERT [ledger] 对账差 %s/%s venue=%.4f ledger=%.4f（容差 %.2f）", x.venue, x.ccy, x.eq, x.net, tol)
		}
	}
	return nil
}

// daySnapshot 日切（venue×ccy 净额行——NAV=venue 总权益口径 T-0060 钉死；
// earn_in/earn_out 内部划转显式排除——历史行在禁改=过滤非删）。
func (r *Runner) daySnapshot(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_ledger_snapshot(venue,ccy,net_usd,snap_date)
		SELECT venue,ccy,sum(amount),CURRENT_DATE FROM fin_cashflow
		WHERE kind NOT IN ('earn_in','earn_out') GROUP BY venue,ccy
		ON CONFLICT(venue,ccy,snap_date) DO UPDATE SET net_usd=EXCLUDED.net_usd,ts=now()`)
	return err
}

// postInterest Earn 利息事件化入账：earnings 累计值 vs 最近 earn_interest 事件行
// 高水位，delta≥0.005 → 落事件行（append-only 真源）+interest cashflow 行
// （source=事件行——禁推断记账）。首拍=播种事件行零入账（历史累计归基线校准面）。
func (r *Runner) postInterest(ctx context.Context) {
	if r.earnings == nil {
		return
	}
	ens, err := r.earnings(ctx)
	if err != nil {
		r.rej.Inc("bal_missing")
		return
	}
	for ccy, cur := range ens {
		if ccy != "USDT" && ccy != "USD" {
			continue
		}
		var last *float64
		_ = r.pool.QueryRow(ctx, `SELECT (data->>'earnings')::numeric FROM md_account_events
			WHERE venue='okx' AND channel='earn_interest' AND ccy=$1 ORDER BY ts DESC LIMIT 1`, ccy).Scan(&last)
		if last != nil && cur-*last < 0.005 {
			continue // 应计不足半分不入账（逐拍对拍不放大 REST/写面）
		}
		// md_account_events 无代理 PK（hypertable）——source_ref=事件行 ts 毫秒
		//（append-only 单调，真源定位口径：source_table='md_account_events'+ts 毫秒）。
		var evID int64
		if err := r.pool.QueryRow(ctx, `INSERT INTO md_account_events(ts,venue,channel,ccy,data)
			VALUES(now(),'okx','earn_interest',$1,jsonb_build_object('earnings',$2::numeric))
			RETURNING (extract(epoch from ts)*1000)::bigint`,
			ccy, cur).Scan(&evID); err != nil {
			r.rej.Inc("write_err")
			continue
		}
		if last == nil {
			continue // bootstrap 播种：高水位立起，零入账
		}
		if delta := cur - *last; delta > 0 {
			if _, err := Post(ctx, r.pool, Entry{Kind: "interest", Venue: "okx", Ccy: ccy,
				Amount: delta, SourceRef: evID, SourceTable: "md_account_events"}); err != nil {
				r.rej.Inc("write_err")
				log.Printf("[ledger] interest 入账败: %v", err)
			}
		}
	}
}
