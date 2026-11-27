// drill_cfd.go —— O-41368 CFD 方向族单腿演习径（ShapeCFDSingleLeg 产能件）：
// 同 runDirDrill 产线全径（ephemeral 策略行→Gate→ExecuteDir→哨戒平→费→
// 结算→recon→凭据），差异=mt5sim 仿真 MT5 户（venue=名册活 CFD 户求解——
// mtexec.Executor 三缝 Cli/Trading/Dialect 产线同签名）+手数=lot 网格
// 合成；方向仓真相锚=CFD 在仓手数开=有/平=零+仓行闭。
package simdrill

import (
	"context"
	"fmt"
	"math"

	"o/internal/guard"
	"o/internal/ledger"
	"o/internal/mt5"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/execwiring"
	"o/internal/trading/mt5sim"
	"o/internal/trading/mtexec"
)

// runCFDDrill CFD 单腿演习（mt5_dir_mom——degraded 哨戒平走单腿 tend→
// ExecuteDir 产线同径）。direction 读策略行（long_perp=buy/short_perp=sell）。
func (d *Drill) runCFDDrill(ctx context.Context, res Result, inst, templateRef string, notional float64) (Result, error) {
	var direction string
	_ = d.Pool.QueryRow(ctx, `SELECT COALESCE(direction,'short_perp') FROM fin_strategy
		WHERE inst_id=$1 AND template_ref=$2 ORDER BY id DESC LIMIT 1`, inst, templateRef).Scan(&direction)
	side, err := execution.DirSide(direction)
	if err != nil {
		return res, err
	}
	mt := newCFDVenue(inst)
	cs, minLot, lotStep, _, ok := mt.Meta(inst)
	if !ok {
		return res, fmt.Errorf("CFD 演习乐器不在册: %s", inst)
	}
	lots := math.Floor(notional/pxOf(inst)/cs/lotStep+1e-9) * lotStep
	if lots < minLot {
		return res, fmt.Errorf("CFD 手数低于 min_lot（notional=%.2f px=%.8g ctVal=%.8g→lots=%.4f<minLot=%.4f）",
			notional, pxOf(inst), cs, lots, minLot)
	}
	var drillStrat int64
	if err := d.Pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue,direction)
		VALUES($1,$2,'degraded',$3,'mt5',$4) RETURNING id`, inst, StratTemplate, notional, direction).Scan(&drillStrat); err != nil {
		return res, fmt.Errorf("演习策略行建败: %w", err)
	}
	defer func() {
		_, _ = d.Pool.Exec(ctx,
			`UPDATE fin_position SET closed_at=now() WHERE strategy_ref=$1 AND state<>'closed'`, drillStrat)
		_, _ = d.Pool.Exec(ctx,
			`UPDATE fin_guard_close SET state='closed',closed_at=now(),updated_at=now(),
			 last_error='reconcile:drill_cleanup' WHERE state='closing'
			 AND position_id IN (SELECT id FROM fin_position WHERE strategy_ref=$1)`, drillStrat)
		retireEphemeral(ctx, d.Pool, drillStrat)
	}()
	if _, err := d.Pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'sim',$1,$2)
		ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`,
		inst, pxOf(inst)); err != nil {
		return res, fmt.Errorf("行情锚写败: %w", err)
	}
	defer retireSimTicks(ctx, d.Pool, [2]execution.Leg{{InstID: inst}})
	// O-41383：演习腿 venue=名册活 CFD 户求解一次传各缝（exness 字面量退役
	// ——死户钉 venue=记账活病灶）；零活户=演习 fail 显形，凭据格拒造。
	cfdVenue, err := mt5.ActiveCfdVenue(ctx, d.Pool)
	if err != nil {
		return res, fmt.Errorf("CFD venue 名册求解败: %w", err)
	}
	router := execution.RouterExecutor{cfdVenue: mtexec.Executor{T: mt, C: mt.Cli(), X: mt.Dialect()}}
	runner := execution.NewRunner(d.Pool, d.Cfg, router).WithBookVenue("sim") // 工单#22576：演习腿台账归 sim 域
	legLabel := "cfd_long"
	if side == "sell" {
		legLabel = "cfd_short"
	}
	dec := d.Gate.Check(ctx, risk.Intent{StrategyRef: drillStrat, Venue: cfdVenue, InstID: inst,
		Legs: []string{legLabel}, NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return res, fmt.Errorf("演习开仓被闸拒（%s）——判据/config 缺 fail-closed", dec.Reason)
	}
	openExecID, err := runner.ExecuteDir(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: drillStrat, InstID: inst, NotionalUSD: notional,
		Legs: [2]execution.Leg{{Venue: cfdVenue, InstID: inst, Side: side,
			Sz: trimFloat(lots), Notional: notional, CtVal: cs}}})
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
	w := guard.NewWarden(d.Pool, d.Cfg, d.Gate, execwiring.CloseExec{R: runner})
	w.AttachVenueSizer(cfdVenue, mt)
	w.AttachSizer(mt)
	w.AttachCtVal(func(_ context.Context, venue, instID string) (float64, bool) {
		if venue == cfdVenue {
			cs, _, _, _, ok := mt.Meta(instID)
			return cs, ok
		}
		return 0, false
	})
	if err := w.ScanOnce(ctx); err != nil {
		return res, fmt.Errorf("演习哨戒拍败: %w", err)
	}
	res.CloseExecID = closeExecIDOf(ctx, d.Pool, drillStrat)
	_ = d.Pool.QueryRow(ctx, `SELECT COALESCE(trigger_kind::text,'') FROM fin_guard_close
		WHERE position_id=$1`, posID).Scan(&res.Trigger)
	if err := ledger.NewRunner(d.Pool, d.Cfg).ScanOnce(ctx); err != nil {
		return res, fmt.Errorf("演习结算拍败: %w", err)
	}
	res.Detail, res.Verdict = d.cfdRecon(ctx, res, mt)
	return d.writeDrill(ctx, res)
}

// cfdRecon CFD 单腿对拍两锚：①仓行已平 ②CFD 在仓手数归零（复用
// dirRecon 同构——无 drift/残袋锚，CFD 方向仓无对冲腿；mt5sim 无费
// 流水面故无现金锚）。mt=开平同件仿真户（真相读面唯一源——重建=空仓假绿）。
func (d *Drill) cfdRecon(ctx context.Context, res Result, mt *mt5sim.Sim) (string, string) {
	var closed bool
	var posID int64
	if err := d.Pool.QueryRow(ctx, `SELECT id,state='closed' FROM fin_position
		WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`, res.OpenExecID).Scan(&posID, &closed); err != nil {
		return d.fail(ctx, res, "演习仓行缺（开仓事实不可读）"), "fail"
	}
	if !closed {
		return d.fail(ctx, res,
			fmt.Sprintf("演习未平（触发=%s close_exec=#%d）——CFD 单腿平仓径拒单/执行败", orDash(res.Trigger), res.CloseExecID)), "fail"
	}
	// 在仓手数全量真相读（演习私有 venue 无他席仓——OpenedLots 不筛
	// comment 前缀=全量；残史命中面由 HistoryCount 承载不入本锚）。
	if resid := mt.OpenedLots(res.InstID); math.Abs(resid) > 1e-9 {
		return d.fail(ctx, res,
			fmt.Sprintf("CFD 残仓 %.8g 手——单腿 %s 平未净敞口", resid, res.InstID)), "fail"
	}
	return fmt.Sprintf("CFD 单腿演习闭环（开 #%d→平 #%d 触发=%s）残仓零",
		res.OpenExecID, res.CloseExecID, orDash(res.Trigger)), "pass"
}
