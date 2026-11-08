// cellgate.go —— T-0461 armed→open 矩阵格凭据闸（行数棘轮拆件自
// opener.go）：格集=excell.Enumerate 唯一真源一次装载；候选格归一坐标
// →CellGreen 全档 latest=pass 齐绿才放行——格缺/任一档缺/败=hold 不开仓
// +exec_cell_hold 计数（fail-closed 同构毕业闸消费径；simdrill -matrix
// 跑绿落 cell_key 凭据后自解）。判位=openLive 环内 r.op.Open 前——
// lifetime/breakeven/振荡等前置诊断闸序不变（诊断先行凭据殿后）。
package lifecycle

import (
	"context"
	"log"

	"o/internal/platform/excell"
)

// cellGate 格凭据快照（openLive 拍首一次装载——全候选共享判读面；
// nil=凭据面读败——allow 全 hold fail-closed）。
type cellGate struct {
	idx   map[string]*excell.Cell
	green map[string]bool
}

// loadCellGate 格集枚举+latest 判词装载（读败=nil——write_err 已计；
// 调用侧不早退——候选逐个走 allow=exec_cell_hold 计数面可观测）。
func (r *Runner) loadCellGate(ctx context.Context) *cellGate {
	cells, err := excell.Enumerate(ctx, r.pool)
	if err != nil {
		r.rej.Inc("write_err")
		log.Printf("[lifecycle] 矩阵格集枚举败（armed→open 闸 fail-closed）: %v", err)
		return nil
	}
	green, err := excell.GreenKeys(ctx, r.pool, cells)
	if err != nil {
		r.rej.Inc("write_err")
		log.Printf("[lifecycle] 矩阵格凭据读败（armed→open 闸 fail-closed）: %v", err)
		return nil
	}
	return &cellGate{idx: excell.IndexByCoord(cells), green: green}
}

// allow 格绿放行判（归一坐标→全档齐绿；xvenue 副腿行 venue≠okx 豁免
// ——openLive 环内 xvenue_sub_leg 跳过判同位先行）。
func (g *cellGate) allow(r *Runner, spec OpenSpec) bool {
	if spec.Template == "xvenue-carry" && spec.Venue != "okx" {
		return true
	}
	vp, dir := excell.Normalize(spec.Template, spec.Venue, spec.Direction)
	var cell *excell.Cell
	if g != nil {
		cell = g.idx[excell.CoordKey(spec.Template, vp, dir)]
	}
	if cell == nil || !excell.CellGreen(*cell, g.green) {
		r.rej.Inc("exec_cell_hold")
		log.Printf("[lifecycle] 策略 %d %s 格凭据未绿 hold——%s@%s/%s（simdrill -matrix 跑绿后自解）",
			spec.StrategyRef, spec.InstID, spec.Template, vp, dir)
		return false
	}
	return true
}
