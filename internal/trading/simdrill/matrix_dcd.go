// matrix_dcd.go —— DCD 产品全周期演习径（O-41383 棘轮拆件自 matrix_exec.go）：
// runDCD=产线同径 dcdexec.Open（venue 三步缝=dcdSim）；强平=到期结算径：
// DCD 无 OrderClose API——order-status SETTLED 即平仓语义，exec/position
// 态迁同 dcdExecState 映射产线律。
// T-0522：strangle 复合键「Cprod+Pprod」拆双腿——各腿等权名义逐腿
// quote→trade→status→settle（volSellStrangle 执行器同构；腿序 C 先 P 后
// 同 evalVolSell 构键律）；单键=单腿径不变。
package simdrill

import (
	"context"
	"fmt"
	"strings"

	"o/internal/platform/excell"
	"o/internal/trading/dcdexec"
)

// runDCD DCD 产品全周期（产线同径 dcdexec.Open——venue 三步缝=dcdSim；
// 强平=到期结算径：DCD 无 OrderClose API——order-status SETTLED 即平仓
// 语义，exec/position 态迁同 dcdExecState 映射产线律）。
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
