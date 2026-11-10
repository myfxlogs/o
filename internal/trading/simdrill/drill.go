// Package simdrill —— 执行径全径演习 harness（T-0426——纸面跑 venue 语义）。
//
// 跑真码不跑影抄：risk.Gate→execution.Runner→guard.Warden→ledger 结算
// 全产线同码，只替 venue IO（simvenue.SimVenue 注入 Executor+Sizer 双缝）+
// bills 费流水钩（演习机代行采集钩——费行落 fin_cashflow 同 ingest 口径）。
// 全径：合成腿开仓（Gate→Runner）→哨戒平仓（Warden→execwiring 装配缝→Runner）
// →结算对账（ledger.ScanOnce）→账实对拍（venue 真相×台账四锚）。
//
// 异常升件径（禁静默红）：任一环账实不平/drift 假报/残袋非零/合约残仓
// →ops_event 升件行（seat='演习'，esc_class='执行径演习'）+verdict=fail
// 落 fin_exec_drill——毕业闸 fail-closed 消费。
package simdrill

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/guard"
	"o/internal/ledger"
	"o/internal/platform/config"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/execwiring"
	"o/internal/trading/simvenue"
)

// 事件簿口径（升件行 seat/ticket/esc_class——DM 消费锚）。
const (
	Seat     = "演习"
	Ticket   = "T-0426"
	EscClass = "执行径演习"
	// StratTemplate 演习 ephemeral 策略打标（审计可辨）。
	StratTemplate = "simdrill"
)

// retireEphemeral 演习 ephemeral 策略行即退+Transition 留痕——
// fin_strategy_active_dedupe (venue,inst,direction,template_ref)
// 活跃唯一键下 state='degraded' 行占键：degraded_ttl 24h 兜底
// 不赶日拍间隔（23.5h<24h——2026-11-10 产线实证矩阵二跑 6 格
// 全撞唯一键）。演习毕=语义终结态即退；产线律=Transition 留痕
// （retire.go 钦定——裸 UPDATE 零留痕禁）。degraded 外态不动
// （他径迁过态不抢改）。
func retireEphemeral(ctx context.Context, pool *pgxpool.Pool, stratID int64) {
	if tag, _ := pool.Exec(ctx,
		`UPDATE fin_strategy SET state='retired' WHERE id=$1 AND state='degraded'`,
		stratID); tag.RowsAffected() > 0 {
		_, _ = pool.Exec(ctx,
			`INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
			 VALUES($1,'degraded','retired','drill_ephemeral_done')`, stratID)
	}
}

// Result 演习结论（verdict 入 fin_exec_drill——毕业闸凭据）。
type Result struct {
	Verdict     string // pass|fail
	StrategyRef int64  // 被闸策略（凭据归属——非演习 ephemeral 行）
	InstID      string
	OpenExecID  int64
	CloseExecID int64
	Trigger     string // 平仓触发 kind（drift 假报检测锚）
	Detail      string
}

// Drill 演习机（纸面钱域——venue=simvenue 合成件；消费件全产线同码）。
type Drill struct {
	Pool  *pgxpool.Pool
	Cfg   *config.Client
	Gate  *risk.Gate
	Venue *simvenue.SimVenue
}

// Run 全径演习一跑（对被闸策略的型跑：同 inst 同名义——开→平→对账全径）。
// verdict 缺省 pass；任一环病=翻 fail+升件（禁静默）——返回 err 仅限机械败
// （判据/事实读不到——病面走 verdict 不走 error，升件行为准）。
func (d *Drill) Run(ctx context.Context, strategyRef int64) (Result, error) {
	res := Result{StrategyRef: strategyRef, Verdict: "pass"}
	// ① 策略型事实（inst/notional/template——演习对型跑）。
	var inst, templateRef string
	var notional float64
	if err := d.Pool.QueryRow(ctx,
		`SELECT inst_id,notional_usd,template_ref FROM fin_strategy WHERE id=$1`, strategyRef).
		Scan(&inst, &notional, &templateRef); err != nil {
		return res, fmt.Errorf("策略型事实读败: %w", err)
	}
	res.InstID = inst
	if !strings.HasSuffix(inst, "-SWAP") {
		switch {
		// DCD 产品域（dcd_put/dcd_call/vol_sell 毕业闸同凭据——probe→live
		// 闸全策略同待遇,T-0426 判据不偏爱 funding-hedge；strangle 复合键
		// runDCDDrill 内拆腿,T-0522）。
		case templateRef == "dcd_put" || templateRef == "dcd_call" ||
			templateRef == "vol_sell_single_call" || templateRef == "vol_sell_strangle":
			return d.runDCDDrill(ctx, res, inst, templateRef, notional)
		// basis 族=-YYMMDD 期货键（T-0522 假想卡骨架批——毕业域模板未载=
		// 凭据不可产fail-forever,同坑戒 KB simdrill-gate-domain-coverage-mismatch）。
		case isBasisTemplate(templateRef):
			return d.runBasisDrill(ctx, res, inst, templateRef, notional)
		}
		return res, fmt.Errorf("演习面限 SWAP/dcd/basis 键策略（毕业域模板未载——新执行族注册须同 commit 补演习格）: %s template=%s", inst, templateRef)
	}
	// ② 乐器面（venue 册直取——T-0427 品种主档前同径，不落表；SWAP 键剥尾
	// 即现货腿对键——CORE-USDT-SWAP→CORE-USDT）。
	spotInst, ok := d.Venue.Instrument(strings.TrimSuffix(inst, "-SWAP"))
	if !ok {
		return res, fmt.Errorf("演习乐器不在册: spot 腿缺")
	}
	swapInst, ok := d.Venue.Instrument(inst)
	if !ok {
		return res, fmt.Errorf("演习乐器不在册: %s", inst)
	}
	// ③ 合成腿（quote 买现货腿+SWAP 反向腿——量落 lotSz 网格，名义对半）。
	half := notional / 2
	contracts := math.Floor(half/(swapInst.CtVal*swapInst.Px)/swapInst.LotSz+1e-9) * swapInst.LotSz
	if contracts <= 0 {
		return res, fmt.Errorf("合成腿张数≤0（notional=%.2f ctVal=%.8g px=%.8g lotSz=%.8g）",
			half, swapInst.CtVal, swapInst.Px, swapInst.LotSz)
	}
	legs := [2]execution.Leg{
		{Venue: "okx", InstID: spotInst.InstID, Side: "buy", TgtQuote: true,
			Sz: trimFloat(half), Notional: half},
		{Venue: "okx", InstID: swapInst.InstID, Side: "sell",
			Sz: trimFloat(contracts), Notional: half},
	}
	return d.leggedDrill(ctx, res, inst, notional, legs,
		[]string{"spot_long", "perp_short"}, swapInst, spotInst)
}

// leggedDrill 双腿原子径共用体（funding/basis 两族同构——腿构/闸签/对拍锚
// 参数化注入，流程单源：ephemeral 策略行→Gate→Execute→哨戒→费→结算→
// recon 四锚→凭据）。deriv=卖出衍生腿（合约净持锚）long=多头腿
// （SPOT=裸袋锚/合约=净持锚）。
func (d *Drill) leggedDrill(ctx context.Context, res Result, inst string, notional float64,
	legs [2]execution.Leg, gateLegs []string, deriv, long simvenue.Instrument) (Result, error) {
	// 演习 ephemeral 策略行（同型同名义，state='degraded'——哨戒降级联动
	// 必平径免 funding/tick 判据链依赖）。
	var drillStrat int64
	if err := d.Pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		VALUES($1,$2,'degraded',$3,'okx') RETURNING id`, inst, StratTemplate, notional).Scan(&drillStrat); err != nil {
		return res, fmt.Errorf("演习策略行建败: %w", err)
	}
	// 夭折防护：演习未平仓位=产线仓巡竞动作源（sim 仓位无真钱——即时关账
	// 防仓巡拾起对真 venue 发单；幂等——已平仓行 no-op）；病面细节已落
	// ops_event/fin_exec_drill。
	defer func() {
		_, _ = d.Pool.Exec(ctx,
			`UPDATE fin_position SET closed_at=now() WHERE strategy_ref=$1 AND closed_at IS NULL`, drillStrat)
		// T-0511：position 关账后残 'closing' 行=僵尸（探针假升件）——
		// 清扫径同关 guard_close。
		_, _ = d.Pool.Exec(ctx,
			`UPDATE fin_guard_close SET state='closed',closed_at=now(),updated_at=now(),
			 last_error='reconcile:drill_cleanup' WHERE state='closing'
			 AND position_id IN (SELECT id FROM fin_position WHERE strategy_ref=$1)`, drillStrat)
		retireEphemeral(ctx, d.Pool, drillStrat)
	}()
	// 行情真相锚（matrix execTwoLeg 同律——venue 成交价与 md_ticks 同价
	// 防时差型假 drift；全腿统一 venue 册价,真器键 drill 不再靠夹具预播）。
	for _, l := range legs {
		in, ok := d.Venue.Instrument(l.InstID)
		if !ok {
			return res, fmt.Errorf("演习乐器不在册（tick 锚）: %s", l.InstID)
		}
		if _, err := d.Pool.Exec(ctx,
			`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'sim',$1,$2)
			ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`,
			l.InstID, in.Px); err != nil {
			return res, fmt.Errorf("行情锚写败: %w", err)
		}
	}
	// 开仓（产线径：Gate 闸→Runner 双腿原子——live_switch 闸自验）。
	runner := execution.NewRunner(d.Pool, d.Cfg, d.Venue)
	dec := d.Gate.Check(ctx, risk.Intent{StrategyRef: drillStrat, Venue: "okx", InstID: inst,
		Legs: gateLegs, NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return res, fmt.Errorf("演习开仓被闸拒（%s）——判据/config 缺 fail-closed", dec.Reason)
	}
	openExecID, err := runner.Execute(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: drillStrat, InstID: inst, NotionalUSD: notional, Legs: legs})
	if err != nil {
		return res, fmt.Errorf("演习开仓执行败: %w", err)
	}
	res.OpenExecID = openExecID
	var posID int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM fin_position WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`, openExecID).
		Scan(&posID); err != nil || posID == 0 {
		return res, fmt.Errorf("演习仓行缺（开仓事实不可读）: %v", err)
	}
	// 开仓后 venue 真相快照（drift 假报检测锚——平仓前时点均衡判）。
	openBalanced := d.balanced(long, deriv)
	// 哨戒平仓（产线径：Warden→execwiring 装配缝→Runner——sizer=venue 现读）。
	w := guard.NewWarden(d.Pool, d.Cfg, d.Gate, execwiring.CloseExec{R: runner})
	w.AttachSizer(d.Venue)
	w.AttachCtVal(func(_ context.Context, _, instID string) (float64, bool) {
		return d.Venue.CtValOf(instID)
	})
	if err := w.ScanOnce(ctx); err != nil {
		return res, fmt.Errorf("演习哨戒拍败: %w", err)
	}
	res.CloseExecID = closeExecIDOf(ctx, d.Pool, drillStrat)
	_ = d.Pool.QueryRow(ctx, `SELECT COALESCE(trigger_kind::text,'') FROM fin_guard_close
		WHERE position_id=$1`, posID).Scan(&res.Trigger)
	// 费流水入账（bills 钩同径——开+平全fills 一次归仓；USDT 域同 ingest）。
	d.postSimFees(ctx, posID)
	// 结算对账（产线径：settle 落位+postmortem 判）。
	if err := ledger.NewRunner(d.Pool, d.Cfg).ScanOnce(ctx); err != nil {
		return res, fmt.Errorf("演习结算拍败: %w", err)
	}
	// 账实对拍四锚（venue 真相×台账——任一锚破=fail+升件）。
	res.Detail, res.Verdict = d.recon(ctx, res, long, deriv, openBalanced)
	// 凭据落账（毕业闸消费面——fail 亦落：闸可读败因）。
	return d.writeDrill(ctx, res)
}

// 账实对拍四锚+腿名义/均衡判——drill_recon.go（棘轮拆件）。

// postSimFees 费流水入账（bills 钩同径：kind='fee'/USDT 域/归仓 position_ref；
// ingest 同弃非 USDT 域——spot 买基准币内扣费不入）。
func (d *Drill) postSimFees(ctx context.Context, posID int64) {
	for _, f := range d.Venue.Fills() {
		if f.Fee <= 0 || f.FeeCcy != "USDT" {
			continue
		}
		if _, err := d.Pool.Exec(ctx, `INSERT INTO fin_cashflow
			(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
			VALUES('fee','sim','USDT',$1,$2,0,'simvenue',$3)`, -f.Fee, posID, f.Ts); err != nil {
			log.Printf("WARN [simdrill] 费流水落账败（对平锚④将破——升件面）: %v", err)
		}
	}
}

// closeExecIDOf 平仓 exec 行 id（演习 ephemeral 策略专属——close intent 建
// 的 exec 行=id 最大者，演练证据链）。
func closeExecIDOf(ctx context.Context, pool *pgxpool.Pool, drillStrat int64) int64 {
	var id int64
	_ = pool.QueryRow(ctx, `SELECT id FROM fin_execution WHERE strategy_ref=$1
		ORDER BY id DESC LIMIT 1`, drillStrat).Scan(&id)
	return id
}

// fail 病面收口：ops_event 升件行（禁静默红）+verdict=fail。
func (d *Drill) fail(ctx context.Context, res Result, detail string) string {
	if _, err := d.Pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,'fin')`,
		Seat, Ticket, fmt.Sprintf("演习 #%d %s：%s", res.StrategyRef, res.InstID, detail), EscClass); err != nil {
		log.Printf("ERROR [simdrill] 升件行落账败（静默红风险）: %v", err)
	}
	log.Printf("[simdrill] 演习 #%d fail：%s", res.StrategyRef, detail)
	return detail
}

func baseOf(in simvenue.Instrument) string {
	return strings.SplitN(in.InstID, "-", 2)[0]
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func trimFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
