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
	// StratTemplate 演习 ephemeral 策略打标（审计可辨；24h TTL 自退役不复用）。
	StratTemplate = "simdrill"
)

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
	// ① 策略型事实（inst/notional——演习对型跑）。
	var inst string
	var notional float64
	if err := d.Pool.QueryRow(ctx,
		`SELECT inst_id,notional_usd FROM fin_strategy WHERE id=$1`, strategyRef).
		Scan(&inst, &notional); err != nil {
		return res, fmt.Errorf("策略型事实读败: %w", err)
	}
	res.InstID = inst
	if !strings.HasSuffix(inst, "-SWAP") {
		return res, fmt.Errorf("演习面限 SWAP 键策略（funding-hedge 毕业域）: %s", inst)
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
	// ③ 演习 ephemeral 策略行（同型同名义，state='degraded'——哨戒降级联动
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
	}()
	// ④ 合成腿（quote 买现货腿+SWAP 反向腿——量落 lotSz 网格，名义对半）。
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
	// ⑤ 开仓（产线径：Gate 闸→Runner 双腿原子——live_switch 闸自验）。
	runner := execution.NewRunner(d.Pool, d.Cfg, d.Venue)
	dec := d.Gate.Check(ctx, risk.Intent{StrategyRef: drillStrat, Venue: "okx", InstID: inst,
		Legs: []string{"spot_long", "perp_short"}, NotionalUSD: notional, Leverage: 1, Kind: "open"})
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
	// ⑥ 开仓后 venue 真相快照（drift 假报检测锚——平仓前时点均衡判）。
	openBalanced := d.balanced(spotInst, swapInst)
	// ⑦ 哨戒平仓（产线径：Warden→execwiring 装配缝→Runner——sizer=venue 现读）。
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
	// ⑧ 费流水入账（bills 钩同径——开+平全fills 一次归仓；USDT 域同 ingest）。
	d.postSimFees(ctx, posID)
	// ⑨ 结算对账（产线径：settle 落位+postmortem 判）。
	if err := ledger.NewRunner(d.Pool, d.Cfg).ScanOnce(ctx); err != nil {
		return res, fmt.Errorf("演习结算拍败: %w", err)
	}
	// ⑩ 账实对拍四锚（venue 真相×台账——任一锚破=fail+升件）。
	res.Detail, res.Verdict = d.recon(ctx, res, spotInst, swapInst, openBalanced)
	// ⑪ 凭据落账（毕业闸消费面——fail 亦落：闸可读败因）。
	if _, err := d.Pool.Exec(ctx, `INSERT INTO fin_exec_drill(strategy_ref,inst_id,verdict,rules_version,detail)
		VALUES($1,$2,$3,$4,$5)`, strategyRef, inst, res.Verdict, simvenue.RulesVersion, res.Detail); err != nil {
		return res, fmt.Errorf("演习凭据落账败: %w", err)
	}
	return res, nil
}

// recon 账实对拍四锚（detail=人读结论；verdict=fail=病面已升件）。
func (d *Drill) recon(ctx context.Context, res Result, spot, swap simvenue.Instrument, openBalanced bool) (string, string) {
	var closed bool
	var posID int64
	if err := d.Pool.QueryRow(ctx, `SELECT id,closed_at IS NOT NULL FROM fin_position
		WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`, res.OpenExecID).Scan(&posID, &closed); err != nil {
		return d.fail(ctx, res, "演习仓行缺（开仓事实不可读）"), "fail"
	}
	// 锚⓪全径闭环：仓未平=平仓径败（拒单/执行败）——演习未走完即 fail 禁假绿。
	if !closed {
		return d.fail(ctx, res,
			fmt.Sprintf("演习未平（触发=%s close_exec=#%d）——平仓径拒单/执行败", orDash(res.Trigger), res.CloseExecID)), "fail"
	}
	// 锚①drift 假报：开仓后 venue 真相均衡却报 delta_drift=假报（升件）。
	if res.Trigger == "delta_drift" && openBalanced {
		return d.fail(ctx, res,
			"delta_drift 假报——venue 真相开仓双腿均衡（出生粒度域）"), "fail"
	}
	// 锚②残袋非零：仓关而 venue 货在且未挂账（挂账=账实同步显形非盲）。
	var bagQty float64
	_ = d.Pool.QueryRow(ctx, `SELECT COALESCE(bare_bag_qty,0) FROM fin_position
		WHERE id=$1`, posID).Scan(&bagQty)
	resid := d.Venue.Balance(baseOf(spot))
	dust := math.Max(spot.LotSz, 1e-9)
	if closed && resid > math.Max(dust, bagQty)+1e-9 {
		return d.fail(ctx, res,
			fmt.Sprintf("残袋非零未挂账——venue 实余 %.8g %s>挂账 %.8g（closed 了但货还在）", resid, baseOf(spot), bagQty)), "fail"
	}
	// 锚③合约净持归零：SWAP 平仓后净持≠0=敞口残（升件——非资产挂账域）。
	if closed && math.Abs(d.Venue.Position(swap.InstID)) > swap.LotSz/2+1e-9 {
		return d.fail(ctx, res,
			fmt.Sprintf("合约残仓 %.8g 张——平未净敞口", d.Venue.Position(swap.InstID))), "fail"
	}
	// 锚④现金对平：venue USDT 流水 Δ==Σfin_cashflow（费行入账后——口径差
	// >分位=账实背离升件）。
	var sum float64
	if err := d.Pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE position_ref=$1 AND kind NOT IN ('earn_in','earn_out')`, posID).Scan(&sum); err != nil {
		return d.fail(ctx, res, fmt.Sprintf("fin_cashflow 归仓和读败: %v", err)), "fail"
	}
	var venueSum float64
	for _, f := range d.Venue.Fills() {
		venueSum += f.QuoteAmt // 计价币流水（fixture 域 USDT 计价——venue 真相）
	}
	if math.Abs(venueSum-sum) > 0.01 {
		return d.fail(ctx, res,
			fmt.Sprintf("现金账实不平——venue 流水 %.6f vs 台账 %.6f（差 %.6f>分位）", venueSum, sum, venueSum-sum)), "fail"
	}
	return fmt.Sprintf("全径绿（open=#%d close=#%d 触发=%s 现金对平 %.6f 残余 %s=%.8g）",
		res.OpenExecID, res.CloseExecID, orDash(res.Trigger), sum, baseOf(spot), resid), "pass"
}

// balanced venue 真相均衡判（开仓后双腿名义对齐——lotSz 粒度+费内残差容）。
func (d *Drill) balanced(spot, swap simvenue.Instrument) bool {
	spotNotional := d.Venue.Balance(baseOf(spot)) * spot.Px
	perpNotional := math.Abs(d.Venue.Position(swap.InstID)) * swap.CtVal * swap.Px
	avg := (spotNotional + perpNotional) / 2
	if avg <= 0 {
		return false
	}
	return math.Abs(spotNotional-perpNotional)/avg*1e4 < 50 // 50bps 出生粒度容（基线播种域）
}

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
