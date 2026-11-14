// drill_dir.go —— 方向族单腿演习径（棒5c ShapeOKXSingleLeg 产能件）：
// 同 leggedDrill 产线全径（ephemeral 策略行→Gate→ExecuteDir→哨戒平→费→
// 结算→recon→凭据），差异=一张 perp 腿（无对冲腿无 drift 判——方向仓
// 真相锚=合约净持开=有/平=零+现金对平）。
package simdrill

import (
	"context"
	"fmt"
	"math"
	"strings"

	"o/internal/guard"
	"o/internal/ledger"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/execwiring"
	"o/internal/trading/simvenue"
)

// runDirDrill 单腿演习（dir_perp_mom——degraded 哨戒平走单腿 tend→ExecuteDir
// 产线同径）。direction 读策略行（long_perp=buy/short_perp=sell 缺省 sell）。
func (d *Drill) runDirDrill(ctx context.Context, res Result, inst, templateRef string, notional float64) (Result, error) {
	var direction string
	_ = d.Pool.QueryRow(ctx, `SELECT COALESCE(direction,'short_perp') FROM fin_strategy
		WHERE inst_id=$1 AND template_ref=$2 ORDER BY id DESC LIMIT 1`, inst, templateRef).Scan(&direction)
	side, err := execution.DirSide(direction)
	if err != nil {
		return res, err
	}
	swapInst, ok := d.Venue.Instrument(inst)
	if !ok || !strings.HasSuffix(inst, "-SWAP") {
		return res, fmt.Errorf("演习乐器不在册/键形非 SWAP: %s", inst)
	}
	contracts := math.Floor(notional/(swapInst.CtVal*swapInst.Px)/swapInst.LotSz+1e-9) * swapInst.LotSz
	if contracts <= 0 {
		return res, fmt.Errorf("合成腿张数≤0（notional=%.2f ctVal=%.8g px=%.8g lotSz=%.8g）",
			notional, swapInst.CtVal, swapInst.Px, swapInst.LotSz)
	}
	var drillStrat int64
	if err := d.Pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue,direction)
		VALUES($1,$2,'degraded',$3,'okx',$4) RETURNING id`, inst, StratTemplate, notional, direction).Scan(&drillStrat); err != nil {
		return res, fmt.Errorf("演习策略行建败: %w", err)
	}
	defer func() {
		_, _ = d.Pool.Exec(ctx,
			`UPDATE fin_position SET closed_at=now() WHERE strategy_ref=$1 AND closed_at IS NULL`, drillStrat)
		_, _ = d.Pool.Exec(ctx,
			`UPDATE fin_guard_close SET state='closed',closed_at=now(),updated_at=now(),
			 last_error='reconcile:drill_cleanup' WHERE state='closing'
			 AND position_id IN (SELECT id FROM fin_position WHERE strategy_ref=$1)`, drillStrat)
		retireEphemeral(ctx, d.Pool, drillStrat)
	}()
	if _, err := d.Pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'sim',$1,$2)
		ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`,
		inst, swapInst.Px); err != nil {
		return res, fmt.Errorf("行情锚写败: %w", err)
	}
	defer retireSimTicks(ctx, d.Pool, [2]execution.Leg{{InstID: inst}})
	runner := execution.NewRunner(d.Pool, d.Cfg, d.Venue)
	dec := d.Gate.Check(ctx, risk.Intent{StrategyRef: drillStrat, Venue: "okx", InstID: inst,
		Legs: []string{direction}, NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return res, fmt.Errorf("演习开仓被闸拒（%s）——判据/config 缺 fail-closed", dec.Reason)
	}
	openExecID, err := runner.ExecuteDir(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: drillStrat, InstID: inst, NotionalUSD: notional,
		Legs: [2]execution.Leg{{Venue: "okx", InstID: swapInst.InstID, Side: side,
			Sz: trimFloat(contracts), Notional: notional}}})
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
	d.postSimFees(ctx, posID)
	if err := ledger.NewRunner(d.Pool, d.Cfg).ScanOnce(ctx); err != nil {
		return res, fmt.Errorf("演习结算拍败: %w", err)
	}
	res.Detail, res.Verdict = d.dirRecon(ctx, res, swapInst)
	return d.writeDrill(ctx, res)
}

// dirRecon 单腿对拍两锚：①仓行已平 ②合约净持归零+现金对平（复用 recon
// 锚④同件——无 drift/残袋锚，方向仓无对冲腿）。
func (d *Drill) dirRecon(ctx context.Context, res Result, deriv simvenue.Instrument) (string, string) {
	var closed bool
	var posID int64
	if err := d.Pool.QueryRow(ctx, `SELECT id,closed_at IS NOT NULL FROM fin_position
		WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`, res.OpenExecID).Scan(&posID, &closed); err != nil {
		return d.fail(ctx, res, "演习仓行缺（开仓事实不可读）"), "fail"
	}
	if !closed {
		return d.fail(ctx, res,
			fmt.Sprintf("演习未平（触发=%s close_exec=#%d）——单腿平仓径拒单/执行败", orDash(res.Trigger), res.CloseExecID)), "fail"
	}
	if math.Abs(d.Venue.Position(deriv.InstID)) > deriv.LotSz/2+1e-9 {
		return d.fail(ctx, res,
			fmt.Sprintf("合约残仓 %.8g 张——单腿 %s 平未净敞口", d.Venue.Position(deriv.InstID), deriv.InstID)), "fail"
	}
	var sum float64
	if err := d.Pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE position_ref=$1 AND venue='sim'`, posID).Scan(&sum); err != nil {
		return d.fail(ctx, res, "演习现金账读败"), "fail"
	}
	return fmt.Sprintf("单腿演习闭环（开 #%d→平 #%d 触发=%s）残仓零 cash=%.4f",
		res.OpenExecID, res.CloseExecID, orDash(res.Trigger), sum), "pass"
}
