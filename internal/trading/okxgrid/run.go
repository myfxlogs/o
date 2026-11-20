// run.go —— okxgrid Runner（venue 原生 bot 管护臂）：拍内三件事——
// ①kill 闸消费（gate.kill.okx_grid=true→全 launched bot 显式 API 停+升件
// ——venue bot 不响应 O 内 kill 语义，拔线须走 stop-order-algo 正径）
// ②权益快照（launched 单 Details→fin_bot_equity_snap 行+fin_bot_order
// 末态同步——bot 持仓≠fin_position 形，记 bot 单实体+周期快照不逐腿，
// 取值理由=票面④钦定）
// ③拉起门（live 态 grid_spot 策略且无 bot→okxgrid.live_switch 业主首闸
// 放行才拉起——手动期首闸票面钉：首批拉起须业主令显式放）。
package okxgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/roster"
)

// Runner venue bot 管护臂（fin 池——bot 实体账与权益快照皆 fin 域）。
type Runner struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	ad   *Adapter
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client, ad *Adapter) *Runner {
	return &Runner{pool: pool, cfg: cfg, ad: ad}
}

// Run 常驻（roster 员工注册+心跳+慢拍——snap_sec 键热调，缺省 300s；
// sched-exempt-ok 监控/拔线兜底拍非事件径）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "网格bot官", "trading", "venue原生bot管护", nil); err != nil {
		errc <- fmt.Errorf("网格bot官 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "网格bot官", 30*time.Second, errc)
	sec := 300
	if v, err := r.cfg.GetFloat("okxgrid.snap_sec"); err == nil && v > 0 {
		sec = int(v)
	}
	t := time.NewTicker(time.Duration(sec) * time.Second) // sched-exempt-ok venue 原生 bot 态重对账——无事件流可订（orders-algo 无 push 面在案），快照同步钦定周期非工作扫描
	defer t.Stop()
	for {
		if err := r.scan(ctx); err != nil {
			log.Printf("[okxgrid] 拍败（本拍让位下拍重试）: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// scan 一拍三件（各段独立败情隔离——kill 段败不掩快照段）。
func (r *Runner) scan(ctx context.Context) error {
	if err := r.killWatch(ctx); err != nil {
		log.Printf("[okxgrid] kill 闸消费败: %v", err)
	}
	if err := r.snapSync(ctx); err != nil {
		return err
	}
	return r.launchGate(ctx)
}

// killWatch gate.kill.okx_grid 消费——置位=全 launched 显式 API 停。
func (r *Runner) killWatch(ctx context.Context) error {
	k, err := r.cfg.GetBool("gate.kill.okx_grid")
	if err != nil || !k {
		return err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id,algo_id,inst_id FROM fin_bot_order WHERE state='launched'`)
	if err != nil {
		return err
	}
	type bot struct {
		id   int64
		algo string
		inst string
	}
	var bs []bot
	for rows.Next() {
		var b bot
		if err := rows.Scan(&b.id, &b.algo, &b.inst); err != nil {
			rows.Close()
			return err
		}
		bs = append(bs, b)
	}
	rows.Close()
	stopType := "1" // 钦定回收形——拔线=卖基币回 quote 不留币敞口
	if v, err := r.cfg.GetFloat("okxgrid.kill_stop_type"); err == nil && v == 2 {
		stopType = "2"
	}
	for _, b := range bs {
		if err := r.ad.Stop(ctx, b.algo, b.inst, stopType); err != nil {
			log.Printf("[okxgrid] bot %s 停败（下拍重试）: %v", b.algo, err)
			continue
		}
		if _, err := r.pool.Exec(ctx,
			`UPDATE fin_bot_order SET state='killed',stopped_at=now() WHERE id=$1`, b.id); err != nil {
			return err
		}
		r.esc(ctx, "okxgrid_kill", "钱径",
			fmt.Sprintf("gate.kill.okx_grid 拔线——bot %s(%s) 已 venue 停（stopType=%s 回收形）", b.algo, b.inst, stopType))
		log.Printf("[okxgrid] bot %s kill 闸拔线停", b.algo)
	}
	return nil
}

// snapSync 在飞 bot 权益快照（Details 全字段原行+解析列双存——快照账
// 全量保载荷非列裁剪，协议面取证「丢载荷=信息损失」教训在案）。
func (r *Runner) snapSync(ctx context.Context) error {
	rows, err := r.pool.Query(ctx,
		`SELECT id,algo_id FROM fin_bot_order WHERE state='launched'`)
	if err != nil {
		return err
	}
	type bot struct {
		id   int64
		algo string
	}
	var bs []bot
	for rows.Next() {
		var b bot
		if err := rows.Scan(&b.id, &b.algo); err != nil {
			rows.Close()
			return err
		}
		bs = append(bs, b)
	}
	rows.Close()
	for _, b := range bs {
		d, err := r.ad.Details(ctx, b.algo)
		if err != nil {
			log.Printf("[okxgrid] bot %s details 拉败（下拍重试）: %v", b.algo, err)
			continue
		}
		raw, _ := json.Marshal(d)
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_bot_equity_snap
			(bot_id,algo_id,inst_id,investment,grid_profit,float_profit,total_pnl,
			 trade_num,arbitrage_num,run_px,fee,fee_ccy,funding_fee,details)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			b.id, d.AlgoID, d.InstID,
			parseF(d.Investment), parseF(d.GridProfit), parseF(d.FloatProfit),
			parseF(d.TotalPnl), int64(parseF(d.TradeNum)), int64(parseF(d.ArbitrageNum)),
			parseF(d.RunPx), parseF(d.Fee), d.FeeCcy, parseF(d.FundingFee), raw); err != nil {
			return err
		}
		st := d.State
		if st == "stopped" || st == "stopping" {
			st = "stopped"
			if _, err := r.pool.Exec(ctx,
				`UPDATE fin_bot_order SET state=$2,stopped_at=COALESCE(stopped_at,now()),
				 last_details=$3,last_synced_at=now() WHERE id=$1 AND state='launched'`,
				b.id, st, raw); err != nil {
				return err
			}
		} else if _, err := r.pool.Exec(ctx,
			`UPDATE fin_bot_order SET last_details=$2,last_synced_at=now() WHERE id=$1`,
			b.id, raw); err != nil {
			return err
		}
	}
	return nil
}

// launchGate 拉起门——live 态 grid_spot 策略未挂 bot 且业主首闸放
// （okxgrid.live_switch=true）才拉起；双闸另棒在案：excell.Drillable
// （grid_spot 未产能=probe→live 结构性拦）+本键（手动期首闸票面钉）。
func (r *Runner) launchGate(ctx context.Context) error {
	sw, err := r.cfg.GetBool("okxgrid.live_switch")
	if err != nil || !sw {
		return err // 手动期首闸——缺键/非 true fail-closed 不拉起
	}
	maxInv := 100.0
	if v, err := r.cfg.GetFloat("okxgrid.max_invest_usd"); err == nil && v > 0 {
		maxInv = v
	}
	rows, err := r.pool.Query(ctx, `SELECT s.id,s.inst_id,s.notional_usd,e.evidence
		FROM fin_strategy s
		JOIN fin_proposal p ON p.id=s.proposal_ref
		JOIN fin_opp_eval e ON e.id=p.eval_id
		WHERE s.template_ref='grid_spot' AND s.state='live'
		  AND NOT EXISTS(SELECT 1 FROM fin_bot_order b WHERE b.strategy_id=s.id)`)
	if err != nil {
		return err
	}
	type cand struct {
		sid      int64
		inst     string
		notional float64
		ev       json.RawMessage
	}
	var cs []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.sid, &c.inst, &c.notional, &c.ev); err != nil {
			rows.Close()
			return err
		}
		cs = append(cs, c)
	}
	rows.Close()
	for _, c := range cs {
		var ev struct {
			BandLo     float64 `json:"band_lo"`
			BandHi     float64 `json:"band_hi"`
			NLines     int     `json:"n_lines"`
			Capital    float64 `json:"capital"`
			LineNotion float64 `json:"line_notional_usd"`
		}
		if err := json.Unmarshal(c.ev, &ev); err != nil ||
			ev.BandLo <= 0 || ev.BandHi <= ev.BandLo || ev.NLines < 2 {
			log.Printf("[okxgrid] #%d 判词证据参数不合（band/档数缺）——fail-closed 不拉起", c.sid)
			continue
		}
		inv := ev.Capital
		if inv <= 0 {
			inv = c.notional
		}
		if inv > maxInv {
			inv = maxInv // 投入帽=业主闸下二次保险（live_switch 已放仍帽）
		}
		algoID, err := r.ad.Launch(ctx, LaunchReq{
			InstID: c.inst, MaxPx: ev.BandHi, MinPx: ev.BandLo,
			GridNum: ev.NLines, QuoteSz: inv,
			AlgoClID: fmt.Sprintf("og%d", c.sid), Tag: "okxgrid",
		})
		if err != nil {
			r.esc(ctx, "okxgrid_launch_fail", "钱径",
				fmt.Sprintf("策略#%d %s 拉起败（band %.4f~%.4f 档%d 投$%.2f）: %v", c.sid, c.inst, ev.BandLo, ev.BandHi, ev.NLines, inv, err))
			continue
		}
		params, _ := json.Marshal(map[string]any{
			"maxPx": ev.BandHi, "minPx": ev.BandLo, "gridNum": ev.NLines,
			"quoteSz": inv, "runType": "1",
		})
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_bot_order
			(strategy_id,venue,inst_id,algo_id,algo_ord_type,params,state)
			VALUES($1,'okx',$2,$3,'grid',$4,'launched')`,
			c.sid, c.inst, algoID, params); err != nil {
			return err
		}
		if _, err := r.pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,note) VALUES('okxgrid','记事',$1)`,
			fmt.Sprintf("策略#%d %s bot 拉起 algoId=%s（band %.4f~%.4f 档%d 投$%.2f——业主首闸放）",
				c.sid, c.inst, algoID, ev.BandLo, ev.BandHi, ev.NLines, inv)); err != nil {
			log.Printf("[okxgrid] 拉起记账败: %v", err)
		}
		log.Printf("[okxgrid] 策略#%d bot 拉起 algoId=%s", c.sid, algoID)
	}
	return nil
}

// esc 升件/记事行（ops_event 账——seat='okxgrid' 机器席）。
func (r *Runner) esc(ctx context.Context, scope, class, note string) {
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,esc_scope,esc_class,note) VALUES('okxgrid','升件',$1,$2,$3)`,
		scope, class, note); err != nil {
		log.Printf("[okxgrid] 升件写败（%s）: %v", scope, err)
	}
}
