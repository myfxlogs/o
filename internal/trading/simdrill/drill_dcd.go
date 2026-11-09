// drill_dcd.go —— DCD 产品域演习径（drill.go 行棘轮拆件；
// 语义归位=单策略径 dcd_put/dcd_call 毕业闸凭据产能面）。
package simdrill

import (
	"context"
	"fmt"
	"strings"

	"o/internal/trading/dcdexec"
	"o/internal/trading/simvenue"
)

// runDCDDrill DCD 产品全径演习（dcd_put/dcd_call 毕业闸凭据——probe→live
// 闸全策略同待遇）。产线同径=矩阵 runDCD 同码序：产品夹具播种→ephemeral
// 策略行→dcdexec.Open(dcdSim 缝)→hold 态检→Settle 到期结算（DCD 无平仓API,
// SETTLED 即平仓语义）→exec/position 态迁 tx 同写→OpenExposure 零残留对账。
func (d *Drill) runDCDDrill(ctx context.Context, res Result, inst, templateRef string, notional float64) (Result, error) {
	// 产品夹具（notionalCcy=USDT——dcdexec.Open 锁仓币读源,缺件 fail-closed）。
	dir := "C"
	if strings.Contains(templateRef, "put") {
		dir = "P"
	}
	if _, err := d.Pool.Exec(ctx, `INSERT INTO md_dcd_product
		(ts,venue,surface,uly,product_id,direction,raw)
		VALUES(now(),'sim','drill',$1,$2,$3,'{"notionalCcy":"USDT"}')
		ON CONFLICT (venue,product_id) DO UPDATE SET raw=EXCLUDED.raw,ts=now()`,
		"SIM-"+inst, inst, dir); err != nil {
		return res, fmt.Errorf("dcd 产品夹具播种败: %w", err)
	}
	// ephemeral 策略行（degraded——同 SWAP 径标审计/免竞态）。
	var drillStrat int64
	if err := d.Pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		VALUES($1,$2,'degraded',$3,'okx') RETURNING id`, inst, StratTemplate, notional).Scan(&drillStrat); err != nil {
		return res, fmt.Errorf("演习策略行建败: %w", err)
	}
	defer func() { // 同 SWAP 径:sim 仓即时关账防仓巡拾起对真 venue 发单。
		_, _ = d.Pool.Exec(ctx,
			`UPDATE fin_position SET closed_at=now() WHERE strategy_ref=$1 AND closed_at IS NULL`, drillStrat)
		_, _ = d.Pool.Exec(ctx,
			`UPDATE fin_guard_close SET state='closed',closed_at=now(),updated_at=now(),
			 last_error='reconcile:drill_cleanup' WHERE state='closing'
			 AND position_id IN (SELECT id FROM fin_position WHERE strategy_ref=$1)`, drillStrat)
	}()

	sim := newDcdSim(inst, notional)
	execID, err := dcdexec.Open(ctx, dcdexec.Deps{Pool: d.Pool, Gate: d.Gate}, sim,
		dcdexec.Spec{StrategyRef: drillStrat, Venue: "okx", InstID: inst,
			Direction: templateRef, Template: templateRef, NotionalUSD: notional})
	if err != nil {
		res.Verdict, res.Detail = "fail", d.fail(ctx, res, "dcd 开仓败: "+err.Error())
		return d.writeDrill(ctx, res)
	}
	res.OpenExecID = execID
	if st := sim.OrderState(); st != "LIVE" {
		res.Verdict, res.Detail = "fail", d.fail(ctx, res, "dcd hold 态异常="+st)
		return d.writeDrill(ctx, res)
	}
	var posID int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM fin_position WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`, execID).
		Scan(&posID); err != nil || posID == 0 {
		res.Verdict, res.Detail = "fail", d.fail(ctx, res, "dcd 仓行缺")
		return d.writeDrill(ctx, res)
	}
	// 到期结算（venue 态迁 SETTLED→exec/position 态迁产线同律,tx 同写防半成）。
	sim.Settle()
	estate := dcdexec.ExecState(sim.OrderState())
	if estate == "" {
		res.Verdict, res.Detail = "fail", d.fail(ctx, res, "dcd 结算态未载映射表（禁默认 open）")
		return d.writeDrill(ctx, res)
	}
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		res.Verdict, res.Detail = "fail", d.fail(ctx, res, "dcd 结算 tx 起败")
		return d.writeDrill(ctx, res)
	}
	if _, err := tx.Exec(ctx, `UPDATE fin_execution SET state=$1 WHERE id=$2`, estate, execID); err != nil {
		_ = tx.Rollback(ctx)
		res.Verdict, res.Detail = "fail", d.fail(ctx, res, "dcd exec 态迁败")
		return d.writeDrill(ctx, res)
	}
	if _, err := tx.Exec(ctx, `UPDATE fin_position SET closed_at=now() WHERE id=$1`, posID); err != nil {
		_ = tx.Rollback(ctx)
		res.Verdict, res.Detail = "fail", d.fail(ctx, res, "dcd 仓平写败")
		return d.writeDrill(ctx, res)
	}
	if err := tx.Commit(ctx); err != nil {
		res.Verdict, res.Detail = "fail", d.fail(ctx, res, "dcd 结算提交败")
		return d.writeDrill(ctx, res)
	}
	res.CloseExecID = execID
	// 对账：结算后 venue 侧无敞口（残留判锚）。
	if sim.OpenExposure() != 0 {
		res.Verdict, res.Detail = "fail", d.fail(ctx, res, "dcd 结算后残留非零")
		return d.writeDrill(ctx, res)
	}
	res.Detail = fmt.Sprintf(" 全径绿(open=#%d settle=#%d)", execID, execID)
	return d.writeDrill(ctx, res)
}

// writeDrill 凭据落账公共段（SWAP/dcd 两径同位——毕业闸消费面,
// fail 亦落：闸可读败因）。
func (d *Drill) writeDrill(ctx context.Context, res Result) (Result, error) {
	if _, err := d.Pool.Exec(ctx, `INSERT INTO fin_exec_drill(strategy_ref,inst_id,verdict,rules_version,detail)
		VALUES($1,$2,$3,$4,$5)`, res.StrategyRef, res.InstID, res.Verdict, simvenue.RulesVersion, res.Detail); err != nil {
		return res, fmt.Errorf("演习凭据落账败: %w", err)
	}
	return res, nil
}
