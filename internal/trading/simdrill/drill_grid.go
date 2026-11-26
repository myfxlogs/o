// drill_grid.go —— grid_spot bot 全径演习（O-41370——无腿可钻族以
// venue 原生 bot 全径代腿钻：bot 创建→挂单校验→行情喂入→kill→对账；
// 禁虚构假腿凑闸——钻的是 okxgrid.Adapter 产线同码 Launch/Details/
// Pending/SubOrders/Stop/WithdrawIncome 五端点面，venue 侧=simgrid
// 仿真引擎）。参数推导=okxgrid.launchGate 同源（判词证据 band/档数/
// 投入额映射钉死同式——演习证的是产线将要发的那发请求）。
package simdrill

import (
	"context"
	"encoding/json"
	"fmt"

	"o/internal/platform/excell"
	"o/internal/trading/okxgrid"
)

// gridLaunchParams 判词证据→LaunchReq 参数（launchGate 同映射单源
// 镜像——band_lo/hi→maxPx/minPx、n_lines→gridNum、capital→quoteSz
// （缺回 notional）、AlgoClID=og<sid> 追溯键同式）。
func gridLaunchParams(inst string, sid int64, notional float64, ev json.RawMessage) (okxgrid.LaunchReq, error) {
	var e struct {
		BandLo  float64 `json:"band_lo"`
		BandHi  float64 `json:"band_hi"`
		NLines  int     `json:"n_lines"`
		Capital float64 `json:"capital"`
	}
	if err := json.Unmarshal(ev, &e); err != nil ||
		e.BandLo <= 0 || e.BandHi <= e.BandLo || e.NLines < 2 {
		return okxgrid.LaunchReq{}, fmt.Errorf("判词证据参数不合（band/档数缺）")
	}
	inv := e.Capital
	if inv <= 0 {
		inv = notional
	}
	return okxgrid.LaunchReq{
		InstID: inst, MaxPx: e.BandHi, MinPx: e.BandLo,
		GridNum: e.NLines, QuoteSz: inv,
		AlgoClID: fmt.Sprintf("og%d", sid), Tag: "okxgrid",
	}, nil
}

// runGridDrill 策略级演习（Drill.Run 派路——strategy_ref 凭据产线；
// 判词证据读链=strategy→proposal→eval 同 launchGate 查询形）。
func (d *Drill) runGridDrill(ctx context.Context, res Result, inst, templateRef string, notional float64) (Result, error) {
	var sid int64
	var ev json.RawMessage
	err := d.Pool.QueryRow(ctx, `SELECT s.id, e.evidence
		FROM fin_strategy s
		JOIN fin_proposal p ON p.id=s.proposal_ref
		JOIN fin_opp_eval e ON e.id=p.eval_id
		WHERE s.id=$1`, res.StrategyRef).Scan(&sid, &ev)
	if err != nil {
		return res, fmt.Errorf("grid 判词证据读败（proposal/eval 链缺——演习无参可推）: %w", err)
	}
	req, err := gridLaunchParams(inst, sid, notional, ev)
	if err != nil {
		res.Verdict, res.Detail = "fail", "grid 参数推导: "+err.Error()
		return d.writeDrill(ctx, res)
	}
	v, det, trg := gridVenueDrill(ctx, req)
	res.Verdict, res.Detail, res.Trigger = v, det, trg
	return d.writeDrill(ctx, res)
}

// gridVenueDrill bot 全径共用体（策略径/矩阵格径同引擎——venue=simgrid
// 仿真、Adapter=产线同码；验收序=拉起→挂单在册→喂价成交→kill→残留
// 对账，任一环破=fail）。返 (verdict, detail, trigger)。
func gridVenueDrill(ctx context.Context, req okxgrid.LaunchReq) (string, string, string) {
	det0 := fmt.Sprintf("inst=%s band=%.4f~%.4f 档%d 投$%.2f", req.InstID, req.MinPx, req.MaxPx, req.GridNum, req.QuoteSz)
	sg := okxgrid.NewSimGrid(req.MinPx + (req.MaxPx-req.MinPx)/2)
	ad := okxgrid.NewAdapter(sg)
	algoID, err := ad.Launch(ctx, req)
	if err != nil || algoID == "" {
		return "fail", det0 + fmt.Sprintf(" 拉起败: %v", err), ""
	}
	det := det0 + " algoId=" + algoID
	// 挂单校验：在飞单在册+live 子单>0（bot 创建≠挂单——引擎须真排单）。
	pend, err := ad.Pending(ctx)
	if err != nil {
		return "fail", det + " pending 读败: " + err.Error(), ""
	}
	found := false
	for _, p := range pend {
		if p.AlgoID == algoID {
			found = true
		}
	}
	if !found {
		return "fail", det + " pending 无本 bot（拉起未登记在飞）", ""
	}
	live, err := ad.SubOrders(ctx, algoID, "live")
	if err != nil || len(live) == 0 {
		return "fail", det + fmt.Sprintf(" live 子单空/读败(%v)——挂单校验不通过", err), ""
	}
	// 行情喂入：带底→带顶两段（买单跌破成交+卖单涨破成交全径走一遍）。
	sg.Feed(req.MinPx)
	sg.Feed(req.MaxPx)
	d, err := ad.Details(ctx, algoID)
	if err != nil {
		return "fail", det + " details 读败: " + err.Error(), ""
	}
	if d.TradeNum == "0" {
		return "fail", det + " 喂价零成交（grid 引擎不咬单——执行径病）", ""
	}
	trg := "grid_feed_" + d.TradeNum + "fills"
	// kill：stop-order-algo 停 bot（产线 stopType=1 卖基回收 quote 同式）。
	if err := ad.Stop(ctx, algoID, req.InstID, "1"); err != nil {
		return "fail", det + " stop 败: " + err.Error(), trg
	}
	// 账实对拍：bot 停+live 子单清零+在飞清零。
	d2, err := ad.Details(ctx, algoID)
	if err != nil {
		return "fail", det + " 停后 details 读败: " + err.Error(), trg
	}
	if d2.State != "stopped" {
		return "fail", det + " 停后态=" + d2.State, trg
	}
	live2, _ := ad.SubOrders(ctx, algoID, "live")
	if len(live2) != 0 || sg.RunningCount() != 0 {
		return "fail", det + " 停后残留（live 子单/inflight 非零）", trg
	}
	if _, err := ad.WithdrawIncome(ctx, algoID); err != nil {
		return "fail", det + " 收割径败: " + err.Error(), trg
	}
	return "pass", det + fmt.Sprintf(" trades=%s arb=%s profit=%s 全径绿", d.TradeNum, d.ArbitrageNum, d.GridProfit), trg
}

// execGridCell 矩阵格径（cell_key 凭据——格无判词证据链，带界/档数
// 由仿真价合成：pxOf±15% 带、10 档、quoteSz=格名义；钻的仍是产线
// Adapter 五端点同码径——凭据证执行径产能非参数最优性）。
func (m *Matrix) execGridCell(ctx context.Context, res *CellResult,
	t excell.Tier, stratID int64, inst string) {
	px := pxOf(inst)
	req := okxgrid.LaunchReq{
		InstID: inst, MinPx: px * 0.85, MaxPx: px * 1.15,
		GridNum: 10, QuoteSz: t.Notional,
		AlgoClID: fmt.Sprintf("og%d", stratID), Tag: "okxgrid",
	}
	v, det, _ := gridVenueDrill(ctx, req)
	res.Verdict, res.Detail = v, det
}
