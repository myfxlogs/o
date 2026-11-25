// emergency.go —— 末路只减仓出口（工单#22247：守卫正径 plan/sizer 失效
// 后敞口无人能归零——attempts 耗尽仓永出 scanOnce 批的第二出口）。
//
// 两触发同径：
//
//	auto  = 正径失败达阈（attempts>=maxAtt）且仓非对冲态（裸敞口或
//	        venue 全清待收殓——对冲态双腿俱在不自动动，受令径才碰）；
//	order = ops_event「止血/仓#N」令行（自我席令票/处置席令行同通道，
//	        令径形态=票面自裁钦定本形）。
//
// 与正径分界（票面钦定）：不共用 plan（gate.Check/令牌）与 sizer
// （sizedCloseLegs/alignCloseSz）——腿量=venue 实读残量对齐 lot 网格；
// reduce-only 不可增敞口（合约腿 Close=true→reduceOnly；现货腿卖
// availBal=只减向）；幂等（跨次重跑键=venue 实读残量≈0→不发单，
// 单次内传输败 Resolve 同 clOrdId 定真态）；落账（fin_execution
// 无令牌位+fin_order_leg 腿行+bareBags 残袋对拍+markClosed 决策行）。
// venue 实读败=整仓让位 fail-closed 升件不发单（锚②）。
// 执行体见 emergency_exit.go（行数棘轮拆件）。
package guard

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"o/internal/mt5"
)

// EmergencyExecutor 末路平仓执行缝（execwiring 适配——guard 不 import
// execution 方向闸零边；未接线=末路缺席，attempts 耗尽照旧留痕不假救）。
type EmergencyExecutor interface {
	ExecuteEmergency(ctx context.Context, spec EmergencySpec) (execID int64, err error)
}

// EmergencySpec 末路出口规格——腿量已 venue 实读定量（Sz=venue 方言量
// 对齐后值；Ticket=mt5 平仓凭据透传）。
type EmergencySpec struct {
	PositionID  int64
	StrategyRef int64
	InstID      string
	NotionalUSD float64
	Trigger     string // auto|order——落账触发词
	Legs        []EmergencyLeg
}

// EmergencyLeg 末路平仓腿（Side=平仓侧已定——spot 恒 sell 只减向；
// 合约腿按 venue 符号位/腿账侧定，reduceOnly 兜底防反向新开）。
type EmergencyLeg struct {
	Venue, InstID, Side string
	Sz                  float64
	Ticket              int64
}

// emergencyState 末路出口态（exec 缝+受令去重+单仓节流——scanOnce 单
// goroutine 驱动不加锁同前提）。
type emergencyState struct {
	exec EmergencyExecutor
	seen map[int64]bool
	last map[int64]time.Time
}

// AttachEmergencyExecutor 挂末路执行缝（cmd 装配注入）。
func (w *Warden) AttachEmergencyExecutor(e EmergencyExecutor) { w.emg.exec = e }

// signedPosSizer 可选面——合约腿符号位现读（net pos 符号；long_short
// 户 posSide 定向）。无此面/读败=腿账侧兜底（reduceOnly 兜底防新开）。
type signedPosSizer interface {
	ContractPosDir(ctx context.Context, instID string) (float64, error)
}

var emgOrderRe = regexp.MustCompile(`止血/仓#(\d+)`)

// emergencyScan 末路扫一拍（scanOnce 尾随——events 驱动同拍）。
// 候选=受令行 ∪ attempts 耗尽仓；非耗尽非受令仓零 venue 读零动作。
func (w *Warden) emergencyScan(ctx context.Context, maxAtt int64) {
	if w.emg.exec == nil {
		return
	}
	ordered := w.emergencyOrders(ctx)
	rows, err := w.pool.Query(ctx, `
		SELECT p.id, p.execution_id, p.strategy_ref, p.inst_id, p.notional_usd, p.opened_at,
		       COALESCE(g.attempts,0), COALESCE(s.state,'')
		FROM fin_position p
		LEFT JOIN fin_guard_close g ON g.position_id = p.id
		LEFT JOIN fin_strategy s ON s.id = p.strategy_ref
		WHERE p.state<>'closed'
		  AND COALESCE(g.state,'open') <> 'closed'`)
	if err != nil {
		w.rej.Inc("write_err")
		log.Printf("[guard] ERROR 末路扫 query 败：%v", err)
		return
	}
	var cands []posRow
	for rows.Next() {
		var p posRow
		if err := rows.Scan(&p.id, &p.execID, &p.stratRef, &p.inst, &p.notional, &p.opened,
			&p.attempts, &p.stratState); err != nil {
			rows.Close()
			return
		}
		cands = append(cands, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return
	}
	for _, p := range cands {
		trig := ""
		if ordered[p.id] {
			trig = "order"
		} else if p.attempts >= maxAtt {
			if naked, ok := w.nakedExposure(ctx, p); ok && naked {
				trig = "auto"
			}
		}
		if trig == "" {
			continue
		}
		w.emergencyExit(ctx, p, trig)
	}
}

// emergencyOrders 受令行拾取（ops_event「止血/仓#N」——窗内新行，
// emg.seen 内存去重防同拍双发；重启窗内重拾=幂等无害）。
func (w *Warden) emergencyOrders(ctx context.Context) map[int64]bool {
	out := map[int64]bool{}
	winMin := 15.0
	if v, err := w.getF("guard.emergency_order_window_min"); err == nil && v > 0 {
		winMin = v
	}
	rows, err := w.pool.Query(ctx, `
		SELECT id, note FROM ops_event
		WHERE ts > now() - make_interval(secs => $1)
		  AND note LIKE '%止血/仓#%' ORDER BY id DESC LIMIT 100`, winMin*60)
	if err != nil {
		w.rej.Inc("write_err")
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var note string
		if err := rows.Scan(&id, &note); err != nil {
			return out
		}
		if w.emg.seen[id] {
			continue
		}
		w.emg.seen[id] = true
		for _, m := range emgOrderRe.FindAllStringSubmatch(note, -1) {
			if pid, err := strconv.ParseInt(m[1], 10, 64); err == nil {
				out[pid] = true
			}
		}
	}
	return out
}

// nakedExposure 非对冲态判（auto 触发闸）：venue 实读残量腿计数——
// 单腿仓=方向仓裸敞口与否按残量；双腿仓恰一腿残量=裸敞口（对侧已清）；
// 双腿俱在=对冲态不自动动；全清=待收殓簿记（naked=true 走 markClosed
// 无单径）。腿形态异常/任一腿读败=ok=false 整仓让位（读不定不动）。
func (w *Warden) nakedExposure(ctx context.Context, p posRow) (bool, bool) {
	legs := w.legs(ctx, p.execID)
	if legs.one {
		return false, false
	}
	resid := 0
	var cand []leg
	if legs.single {
		cand = []leg{legs.a}
	} else {
		cand = []leg{legs.a, legs.b}
	}
	for _, l := range cand {
		_, v, ok := w.venueSizerResidual(ctx, l)
		if !ok {
			w.rej.Inc("facts_missing")
			w.note("emergency_read_fail", fmt.Sprintf("仓 %d %s venue 实读败——末路让位", p.id, l.inst))
			return false, false
		}
		if v > 0 {
			resid++
		}
	}
	switch {
	case legs.single, resid <= 1:
		return true, true // 单腿恒裸域；双腿恰一腿残/全清=非对冲态
	default:
		return false, true // 双腿俱在=对冲态——auto 不动（受令径可达）
	}
}

// venueSizerResidual 腿 venue 实读残量（末路专用读面——不共用
// sizedCloseLegs：读=VenueSizer 原语，定量=本件网格对齐）。
func (w *Warden) venueSizerResidual(ctx context.Context, l leg) (VenueSizer, float64, bool) {
	szr := w.sizerFor(l.venue)
	if szr == nil {
		return nil, 0, false
	}
	var v float64
	var err error
	if mt5.IsCfdProtoVenue(ctx, w.pool, l.venue) || strings.HasSuffix(l.inst, "-SWAP") { // O-37449：cfd 类全族 CFD 腿 ContractPos 径
		v, err = szr.ContractPos(ctx, l.inst)
	} else {
		v, err = szr.SpotAvailBal(ctx, l.inst)
	}
	if err != nil {
		return nil, 0, false
	}
	return szr, v, true
}

// emergencySide 末路腿平仓侧：现货恒 sell（现货只多——卖=唯一只减向，
// 腿账侧不可当方向源）；合约腿 venue 符号位优先（signedPosSizer 在册
// 实读），缺席/读败=腿账侧兜底（reduceOnly 兜底防反向新开）。
func (w *Warden) emergencySide(ctx context.Context, l leg) string {
	if !mt5.IsCfdProtoVenue(ctx, w.pool, l.venue) && !strings.HasSuffix(l.inst, "-SWAP") { // O-37449：cfd 类全族 CFD 腿非现货径
		return "sell"
	}
	if szr, ok := w.sizerFor(l.venue).(signedPosSizer); ok {
		if pos, err := szr.ContractPosDir(ctx, l.inst); err == nil && pos != 0 {
			if pos > 0 {
				return "sell"
			}
			return "buy"
		}
	}
	if l.side == "buy" || l.side == "long" {
		return "sell"
	}
	return "buy"
}
