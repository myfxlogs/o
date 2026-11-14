// matrix_exec.go —— T-0461 矩阵腿形执行径（行数棘轮拆件自 matrix.go）：
// runLegged/execTwoLeg=双腿原子径（Runner.Execute 产线同码——venue 三面
// 注入；哨戒强平走 Warden venue 路由现读面；残留=venue 真相归零判）；
// runDCD=DCD 产品全周期（dcdexec.Open 产线同径）。
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
	"o/internal/trading/dcdexec"
	"o/internal/trading/execution"
	"o/internal/trading/execwiring"
	"o/internal/trading/mt5sim"
	"o/internal/trading/mtexec"
	"o/internal/trading/simvenue"
)

// ---- 腿形径 ----

// runLegged 双腿原子径（Runner.Execute 产线同码——venue 三面注入；
// 哨戒强平走 Warden venue 路由现读面；残留=venue 真相归零判）。
func (m *Matrix) runLegged(ctx context.Context, res *CellResult, c excell.Cell,
	t excell.Tier, stratID int64, inst string, shape excell.LegShape) {
	vs := newOKXVenue(inst)
	var mt *mt5sim.Sim
	var legs [2]execution.Leg
	var flat func() bool

	switch {
	case shape == excell.ShapeXvenue:
		mt = newMT5Venue(inst)
		l, err := xvenueLegs(c, t, inst, mt)
		if err != nil {
			res.Verdict, res.Detail = "fail", res.Detail+" xvenue 腿构: "+err.Error()
			return
		}
		legs = l
		flat = func() bool {
			return mt.OpenedLots(cfdCanonical(inst)) <= 1e-9 &&
				math.Abs(vs.Position(inst)) <= 0.001
		}
	case isBasisTemplate(c.Template):
		l, f, err := basisLegs(c, t, inst, vs)
		if err != nil {
			res.Verdict, res.Detail = "fail", res.Detail+" basis 腿构: "+err.Error()
			return
		}
		legs, flat = l, f
	default: // funding-hedge 族：现货买+SWAP 卖（Drill.Run 同型同名义径）。
		l, f, err := fundingLegs(t, inst, vs)
		if err != nil {
			res.Verdict, res.Detail = "fail", res.Detail+" 腿构: "+err.Error()
			return
		}
		legs, flat = l, f
	}
	m.execTwoLeg(ctx, res, vs, mt, legs, stratID, inst, t.Notional, flat)
}

// execTwoLeg 双腿执行+哨戒强平+残留对拍（产线径：Gate→Runner.Execute→
// Warden venue 路由→ledger 结算；flatFn=venue 真相残留判）。
func (m *Matrix) execTwoLeg(ctx context.Context, res *CellResult, vs *simvenue.SimVenue,
	mt *mt5sim.Sim, legs [2]execution.Leg, stratID int64, inst string,
	notional float64, flatFn func() bool) {
	router := execution.RouterExecutor{"okx": vs}
	if mt != nil {
		router["mt5"] = mtexec.Executor{T: mt, C: mt.Cli(), X: mt.Dialect()}
	}
	// 行情真相锚（drill_test 同律——venue 成交价与 md_ticks 同价防时差型
	// 假 drift；全腿统一格价=双腿名义天然对齐带内）。
	for _, l := range legs {
		if _, err := m.Pool.Exec(ctx,
			`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'sim',$1,$2)
			ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`,
			l.InstID, pxOf(inst)); err != nil {
			res.Verdict, res.Detail = "fail", res.Detail+" 行情锚写败: "+err.Error()
			return
		}
	}
	defer retireSimTicks(ctx, m.Pool, legs)
	runner := execution.NewRunner(m.Pool, m.Cfg, router)
	legLabels := make([]string, 0, 2)
	for _, l := range legs {
		tag, dir := "perp", "long"
		switch {
		case l.Venue == "mt5":
			tag = "cfd"
		case strings.HasSuffix(l.InstID, "-SWAP"):
			tag = "perp"
		case datedFutRe.MatchString(l.InstID):
			tag = "fut"
		default:
			tag = "spot"
		}
		if l.Side == "sell" {
			dir = "short"
		}
		legLabels = append(legLabels, tag+"_"+dir)
	}
	dec := m.Gate.Check(ctx, risk.Intent{StrategyRef: stratID, Venue: "okx", InstID: inst,
		Legs: legLabels, NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		res.Verdict, res.Detail = "fail", res.Detail+" 演习开仓被闸拒（"+dec.Reason+"）"
		return
	}
	execID, err := runner.Execute(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: stratID, InstID: inst, NotionalUSD: notional, Legs: legs})
	if err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 开仓执行败: "+err.Error()
		return
	}
	res.OpenExecID = execID
	var posID int64
	if err := m.Pool.QueryRow(ctx,
		`SELECT id, closed_at IS NULL FROM fin_position WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`,
		execID).Scan(&posID, new(bool)); err != nil || posID == 0 {
		res.Verdict, res.Detail = "fail", res.Detail+" 演习仓行缺（开仓事实不可读）"
		return
	}
	// hold 段——仓在期间敞口真（closed_at 未落=在持；强平前快照锚）。
	var stillOpen bool
	_ = m.Pool.QueryRow(ctx, `SELECT closed_at IS NULL FROM fin_position WHERE id=$1`, posID).Scan(&stillOpen)
	if !stillOpen {
		res.Verdict, res.Detail = "fail", res.Detail+" hold 段仓已闭（开仓未留持窗）"
		return
	}
	// 哨戒强平（产线径：Warden→execwiring 装配缝→Runner——degraded 策略
	// 联动必平；sizer venue 路由现读 AttachVenueSizer）。
	w := guard.NewWarden(m.Pool, m.Cfg, m.Gate, execwiring.CloseExec{R: runner})
	w.AttachVenueSizer("okx", vs)
	w.AttachSizer(vs)
	if mt != nil {
		w.AttachVenueSizer("mt5", mt)
	}
	w.AttachCtVal(func(_ context.Context, venue, instID string) (float64, bool) {
		if venue == "mt5" && mt != nil {
			cs, _, _, _, ok := mt.Meta(instID)
			return cs, ok
		}
		return vs.CtValOf(instID)
	})
	if err := w.ScanOnce(ctx); err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 哨戒拍败: "+err.Error()
		return
	}
	res.CloseExecID = closeExecIDOf(ctx, m.Pool, stratID)
	// 结算对账（产线径：settle 落位+postmortem 判）。
	if err := ledger.NewRunner(m.Pool, m.Cfg).ScanOnce(ctx); err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 结算拍败: "+err.Error()
		return
	}
	// 残留对拍（venue 真相归零——合约净持/现货裸袋/CFD 在仓 ticket 三锚）。
	if !flatFn() {
		res.Verdict, res.Detail = "fail", res.Detail+" 残留非零（venue 真相未归零）"
		return
	}
	res.Detail += fmt.Sprintf(" 全径绿(open=#%d close=#%d)", execID, res.CloseExecID)
}

// ---- DCD 径 ----

// runDCD DCD 产品全周期（产线同径 dcdexec.Open——venue 三步缝=dcdSim；
// 强平=到期结算径：DCD 无 OrderClose API——order-status SETTLED 即平仓
// 语义，exec/position 态迁同 dcdExecState 映射产线律）。
// T-0522：strangle 复合键「Cprod+Pprod」拆双腿——各腿等权名义逐腿
// quote→trade→status→settle（volSellStrangle 执行器同构；腿序 C 先 P 后
// 同 evalVolSell 构键律）；单键=单腿径不变。
func (m *Matrix) runDCD(ctx context.Context, res *CellResult, c excell.Cell,
	t excell.Tier, stratID int64, inst string) {
	parts := strings.Split(inst, "+")
	if c.Template == "vol_sell_strangle" && len(parts) != 2 {
		res.Verdict, res.Detail = "fail", res.Detail+" strangle 复合键形不符（预期 Cprod+Pprod）: "+inst
		return
	}
	n := len(parts)
	per := t.Notional / float64(n) // 等权名义/腿——strangle 执行器同律
	defer retireSimDcdProducts(ctx, m.Pool, parts)
	legs := make([]dcdLeg, 0, n)
	for i, pid := range parts {
		dir := "C"
		if c.Template == "vol_sell_strangle" {
			if i == 1 { // 复合键=C 先 P 后（evalVolSell call+"+"+put 构键律）
				dir = "P"
			}
		} else if strings.Contains(c.Template, "put") {
			dir = "P"
		}
		if err := m.seedDcdProduct(ctx, pid, dir); err != nil {
			res.Verdict, res.Detail = "fail", res.Detail+" dcd 产品夹具写败: "+err.Error()
			return
		}
		sim := newDcdSim(simProductKey(pid), per)
		execID, err := dcdexec.Open(ctx, dcdexec.Deps{Pool: m.Pool, Gate: m.Gate}, sim,
			dcdexec.Spec{StrategyRef: stratID, Venue: "okx", InstID: simProductKey(pid),
				Direction: c.Direction, Template: c.Template, NotionalUSD: per})
		if err != nil {
			// 半构歧态势——volSellStrangle 同判词:已开腿不可回滚（DCD 无撤销面），
			// fail 凭据载明已成腿执行号供人工对账，禁静默。
			res.Verdict, res.Detail = "fail", res.Detail+fmt.Sprintf(
				" dcd 开仓败 leg%d/%d(%s 已开 execs=%v): %v", i+1, n, pid, legExecs(legs), err)
			return
		}
		res.OpenExecID = execID
		if st := sim.OrderState(); st != "LIVE" {
			res.Verdict, res.Detail = "fail", res.Detail+fmt.Sprintf("dcd hold 态异常=%s(%s)", st, pid)
			return
		}
		var posID int64
		if err := m.Pool.QueryRow(ctx,
			`SELECT id FROM fin_position WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`, execID).
			Scan(&posID); err != nil || posID == 0 {
			res.Verdict, res.Detail = "fail", res.Detail+" dcd 仓行缺("+pid+")"
			return
		}
		legs = append(legs, dcdLeg{pid: pid, sim: sim, execID: execID, posID: posID})
	}
	// 强平=到期结算（逐腿 venue 态迁 SETTLED→exec/position 态迁产线同律——
	// tx 同写防半成）。
	for _, lg := range legs {
		lg.sim.Settle()
		estate := dcdexec.ExecState(lg.sim.OrderState())
		if estate == "" {
			res.Verdict, res.Detail = "fail", res.Detail+" dcd 结算态未载映射表（禁默认 open）"
			return
		}
		tx, err := m.Pool.Begin(ctx)
		if err != nil {
			res.Verdict, res.Detail = "fail", res.Detail+" dcd 结算 tx 起败"
			return
		}
		if _, err := tx.Exec(ctx, `UPDATE fin_execution SET state=$1 WHERE id=$2`, estate, lg.execID); err != nil {
			_ = tx.Rollback(ctx)
			res.Verdict, res.Detail = "fail", res.Detail+" dcd exec 态迁败"
			return
		}
		if _, err := tx.Exec(ctx, `UPDATE fin_position SET closed_at=now() WHERE id=$1`, lg.posID); err != nil {
			_ = tx.Rollback(ctx)
			res.Verdict, res.Detail = "fail", res.Detail+" dcd 仓平写败"
			return
		}
		if err := tx.Commit(ctx); err != nil {
			res.Verdict, res.Detail = "fail", res.Detail+" dcd 结算提交败"
			return
		}
		res.CloseExecID = lg.execID
	}
	// 对账：结算后全腿 venue 侧无敞口（OpenExposure=0——残留判锚）。
	for _, lg := range legs {
		if lg.sim.OpenExposure() != 0 {
			res.Verdict, res.Detail = "fail", res.Detail+" dcd 结算后残留非零("+lg.pid+")"
			return
		}
	}
	res.Detail += fmt.Sprintf(" 全径绿(legs=%d open=#%d settle=#%d)", n, res.OpenExecID, res.CloseExecID)
}

// dcdLeg DCD 腿演习轨迹（strangle 双腿/单品单腿共用载体）。
type dcdLeg struct {
	pid    string
	sim    *dcdSim
	execID int64
	posID  int64
}

// legExecs 已开腿执行号集（半构歧态势对账锚——禁静默吞腿）。
func legExecs(legs []dcdLeg) []int64 {
	out := make([]int64, 0, len(legs))
	for _, l := range legs {
		out = append(out, l.execID)
	}
	return out
}

// seedDcdProduct md_dcd_product 演习夹具行（dcdexec.Open notionalCcy 锁仓币
// 回填源——raw.notionalCcy=USDT；direction C=call/P=put 显式传——strangle
// 复合键腿向各异不能按模板派生）。
func (m *Matrix) seedDcdProduct(ctx context.Context, productID, dir string) error {
	_, err := m.Pool.Exec(ctx, `INSERT INTO md_dcd_product
		(ts,venue,surface,uly,product_id,direction,raw)
		VALUES(now(),'sim','drill',$1,$2,$3,$4)
		ON CONFLICT (venue,product_id) DO UPDATE SET raw=EXCLUDED.raw,ts=now()`,
		"SIM-"+productID, simProductKey(productID), dir, `{"notionalCcy":"USDT"}`)
	return err
}
