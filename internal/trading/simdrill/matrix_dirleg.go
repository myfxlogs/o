// matrix_dirleg.go —— O-37649 方向族单腿格执行径（行数棘轮拆件自
// matrix_exec.go）：execSingleLeg=ShapeOKXSingleLeg 产线同码径——
// DirSide(direction)→side、合约张数=全额名义合成、ExecuteDir 开仓、
// 哨戒单腿平（CloseExec spec.Legs[1] 空→ExecuteDir）、dirRecon 两锚=
// 仓行已平+合约净持归零；闸/哨戒/结算装配面与 execTwoLeg 同。
package simdrill

import (
	"context"
	"fmt"
	"math"
	"strings"

	"o/internal/guard"
	"o/internal/ledger"
	"o/internal/platform/excell"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/execwiring"
	"o/internal/trading/simvenue"
)

// execSingleLeg 方向族单腿径（ShapeOKXSingleLeg——drill_dir.runDirDrill
// 同径产线同码：DirSide(direction)→side、合约张数=全额名义合成、哨戒
// 单腿平走 CloseExec spec.Legs[1] 空→ExecuteDir、dirRecon 两锚=仓行
// 已平+合约净持归零；闸/哨戒/结算装配面与 execTwoLeg 同）。
func (m *Matrix) execSingleLeg(ctx context.Context, res *CellResult, vs *simvenue.SimVenue,
	c excell.Cell, t excell.Tier, stratID int64, inst string) {
	side, err := execution.DirSide(c.Direction)
	if err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 单腿 direction 非闭集: "+c.Direction
		return
	}
	sw, ok := vs.Instrument(inst)
	if !ok || !strings.HasSuffix(inst, "-SWAP") {
		res.Verdict, res.Detail = "fail", res.Detail+" 单腿乐器不在册/键形非 SWAP: "+inst
		return
	}
	contracts := math.Floor(t.Notional/(sw.CtVal*sw.Px)/sw.LotSz+1e-9) * sw.LotSz
	if contracts <= 0 {
		res.Verdict, res.Detail = "fail", res.Detail+fmt.Sprintf(
			"合成腿张数≤0（notional=%.2f ctVal=%.8g px=%.8g lotSz=%.8g）",
			t.Notional, sw.CtVal, sw.Px, sw.LotSz)
		return
	}
	// 行情真相锚（execTwoLeg 同律——venue 成交价与 md_ticks 同价防时差型假 drift）。
	if _, err := m.Pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'sim',$1,$2)
		ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`,
		inst, pxOf(inst)); err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 行情锚写败: "+err.Error()
		return
	}
	defer retireSimTicks(ctx, m.Pool, [2]execution.Leg{{InstID: inst}})
	router := execution.RouterExecutor{"okx": vs}
	runner := execution.NewRunner(m.Pool, m.Cfg, router).WithBookVenue("sim") // 工单#22576：演习腿台账归 sim 域
	dec := m.Gate.Check(ctx, risk.Intent{StrategyRef: stratID, Venue: "okx", InstID: inst,
		Legs: []string{c.Direction}, NotionalUSD: t.Notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		res.Verdict, res.Detail = "fail", res.Detail+" 演习开仓被闸拒（"+dec.Reason+"）"
		return
	}
	execID, err := runner.ExecuteDir(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: stratID, InstID: inst, NotionalUSD: t.Notional,
		Legs: [2]execution.Leg{{Venue: "okx", InstID: inst, Side: side,
			Sz: trimFloat(contracts), Notional: t.Notional}}})
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
	// 哨戒强平（产线径同 execTwoLeg——单腿仓 CloseSpec.Legs[1] 空→ExecuteDir）。
	w := guard.NewWarden(m.Pool, m.Cfg, m.Gate, execwiring.CloseExec{R: runner})
	w.AttachVenueSizer("okx", vs)
	w.AttachSizer(vs)
	w.AttachCtVal(func(_ context.Context, _, instID string) (float64, bool) {
		return vs.CtValOf(instID)
	})
	if err := w.ScanOnce(ctx); err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 哨戒拍败: "+err.Error()
		return
	}
	res.CloseExecID = closeExecIDOf(ctx, m.Pool, stratID)
	// 结算对账（产线径同 execTwoLeg）。
	if err := ledger.NewRunner(m.Pool, m.Cfg).ScanOnce(ctx); err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 结算拍败: "+err.Error()
		return
	}
	// dirRecon 两锚：①仓行已平 ②合约净持归零（venue 真相残留判）。
	var closed bool
	if err := m.Pool.QueryRow(ctx, `SELECT state='closed' FROM fin_position WHERE id=$1`,
		posID).Scan(&closed); err != nil || !closed {
		res.Verdict, res.Detail = "fail", res.Detail+" 单腿平未落账（仓行未闭）"
		return
	}
	if math.Abs(vs.Position(inst)) > sw.LotSz/2+1e-9 {
		res.Verdict, res.Detail = "fail", res.Detail+fmt.Sprintf(
			"合约残仓 %.8g 张——单腿 %s 平未净敞口", vs.Position(inst), inst)
		return
	}
	res.Detail += fmt.Sprintf(" 单腿全径绿(open=#%d close=#%d)", execID, res.CloseExecID)
}
