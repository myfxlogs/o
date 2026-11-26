// matrix_cfdleg.go —— O-41368 CFD 方向族单腿格执行径（ShapeCFDSingleLeg）：
// mt5_dir_mom——ExecuteDir 产线同码径，差异=腿 venue='exness' 经
// mtexec.Executor 路由（mt5sim 仿真 MT5 venue——OrderSend/OrderClose/
// OrderModify 产线同签名+Cli/Trading/Dialect 三缝）；手数=lot 网格 floor
// 合成（ContractSize/MinLot/LotStep 仿真册）；哨戒平走 CloseExec 单腿
// ExecuteDir；对拍两锚=仓行已平+CFD 在仓手数归零（dirRecon 同构）。
package simdrill

import (
	"context"
	"fmt"
	"math"

	"o/internal/guard"
	"o/internal/ledger"
	"o/internal/platform/excell"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/execwiring"
	"o/internal/trading/mtexec"
)

// execCFDSingleLeg CFD 方向族单腿径（ShapeCFDSingleLeg——matrix_dirleg.
// execSingleLeg 同构产线同码：DirSide(direction)→side、lot 网格手数合成、
// 哨戒单腿平走 CloseExec spec.Legs[1] 空→ExecuteDir、对拍两锚=仓行已平+
// CFD 在仓手数归零；腿 venue='exness'=mt5sim 仿真户路由键）。
func (m *Matrix) execCFDSingleLeg(ctx context.Context, res *CellResult, c excell.Cell,
	t excell.Tier, stratID int64, inst string) {
	if inst == "" {
		inst = "EURUSD" // 空代表格兜底 canonical（cell_key 不含 inst——代表性乐器任意合法键）
	}
	side, err := execution.DirSide(c.Direction)
	if err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" CFD 单腿 direction 非闭集: "+c.Direction
		return
	}
	mt := newCFDVenue(inst)
	cs, minLot, lotStep, _, ok := mt.Meta(inst)
	if !ok {
		res.Verdict, res.Detail = "fail", res.Detail+" CFD 乐器不在册: "+inst
		return
	}
	lots := math.Floor(t.Notional/pxOf(inst)/cs/lotStep+1e-9) * lotStep
	if lots < minLot {
		res.Verdict, res.Detail = "fail", res.Detail+fmt.Sprintf(
			"CFD 手数低于 min_lot（notional=%.2f px=%.8g ctVal=%.8g→lots=%.4f<minLot=%.4f）",
			t.Notional, pxOf(inst), cs, lots, minLot)
		return
	}
	// 行情真相锚（execSingleLeg 同律——venue 成交价与 md_ticks 同价防时差型假 drift）。
	if _, err := m.Pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'sim',$1,$2)
		ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`,
		inst, pxOf(inst)); err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 行情锚写败: "+err.Error()
		return
	}
	defer retireSimTicks(ctx, m.Pool, [2]execution.Leg{{InstID: inst}})
	router := execution.RouterExecutor{"exness": mtexec.Executor{T: mt, C: mt.Cli(), X: mt.Dialect()}}
	runner := execution.NewRunner(m.Pool, m.Cfg, router).WithBookVenue("sim") // 工单#22576：演习腿台账归 sim 域
	legLabel := "cfd_long"
	if side == "sell" {
		legLabel = "cfd_short"
	}
	dec := m.Gate.Check(ctx, risk.Intent{StrategyRef: stratID, Venue: "exness", InstID: inst,
		Legs: []string{legLabel}, NotionalUSD: t.Notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		res.Verdict, res.Detail = "fail", res.Detail+" 演习开仓被闸拒（"+dec.Reason+"）"
		return
	}
	execID, err := runner.ExecuteDir(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: stratID, InstID: inst, NotionalUSD: t.Notional,
		Legs: [2]execution.Leg{{Venue: "exness", InstID: inst, Side: side,
			Sz: trimFloat(lots), Notional: t.Notional, CtVal: cs}}})
	if err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 开仓执行败: "+err.Error()
		return
	}
	res.OpenExecID = execID
	var posID int64
	if err := m.Pool.QueryRow(ctx,
		`SELECT id, state<>'closed' FROM fin_position WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`,
		execID).Scan(&posID, new(bool)); err != nil || posID == 0 {
		res.Verdict, res.Detail = "fail", res.Detail+" 演习仓行缺（开仓事实不可读）"
		return
	}
	var stillOpen bool
	_ = m.Pool.QueryRow(ctx, `SELECT state<>'closed' FROM fin_position WHERE id=$1`, posID).Scan(&stillOpen)
	if !stillOpen {
		res.Verdict, res.Detail = "fail", res.Detail+" hold 段仓已闭（开仓未留持窗）"
		return
	}
	// 哨戒强平（产线径同 execSingleLeg——单腿仓 CloseSpec.Legs[1] 空→ExecuteDir；
	// sizer venue 路由现读=mt5sim 仿真册直读）。
	w := guard.NewWarden(m.Pool, m.Cfg, m.Gate, execwiring.CloseExec{R: runner})
	w.AttachVenueSizer("exness", mt)
	w.AttachSizer(mt)
	w.AttachCtVal(func(_ context.Context, venue, instID string) (float64, bool) {
		if venue == "exness" {
			cs, _, _, _, ok := mt.Meta(instID)
			return cs, ok
		}
		return 0, false
	})
	if err := w.ScanOnce(ctx); err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 哨戒拍败: "+err.Error()
		return
	}
	res.CloseExecID = closeExecIDOf(ctx, m.Pool, stratID)
	// 结算对账（产线径同 execSingleLeg）。
	if err := ledger.NewRunner(m.Pool, m.Cfg).ScanOnce(ctx); err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 结算拍败: "+err.Error()
		return
	}
	// 对拍两锚：①仓行已平 ②CFD 在仓手数归零（venue 真相残留判）。
	var closed bool
	if err := m.Pool.QueryRow(ctx, `SELECT state='closed' FROM fin_position WHERE id=$1`,
		posID).Scan(&closed); err != nil || !closed {
		res.Verdict, res.Detail = "fail", res.Detail+" CFD 单腿平未落账（仓行未闭）"
		return
	}
	if math.Abs(mt.OpenedLots(inst)) > lotStep/2+1e-9 {
		res.Verdict, res.Detail = "fail", res.Detail+fmt.Sprintf(
			"CFD 残仓 %.8g 手——单腿 %s 平未净敞口", mt.OpenedLots(inst), inst)
		return
	}
	res.Detail += fmt.Sprintf(" CFD 单腿全径绿(open=#%d close=#%d)", execID, res.CloseExecID)
}
