// emergency_exit.go —— 末路执行体（工单#22247——自 emergency.go 拆件，
// 行数棘轮）：逐腿 venue 实读定量→外结腿挂账→执行缝→残袋对拍→
// markClosed。读败/缺 sizer/lot 败=整仓让位 fail-closed 升件。
package guard

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// emergencyExit 单仓末路执行：venue 实读逐腿定量→外结腿挂账→执行缝→
// 残袋对拍→markClosed。读败/缺 sizer/lot 败=整仓让位 fail-closed 升件。
func (w *Warden) emergencyExit(ctx context.Context, p posRow, trig string) {
	if t, ok := w.emg.last[p.id]; ok && time.Since(t) < w.emgCooldown() {
		return // 末路重试节流——venue 持续败不刷告警不空转
	}
	w.emg.last[p.id] = time.Now()
	legs := w.legs(ctx, p.execID)
	if legs.one {
		w.al.Alert(ctx, "emergency_leg_anomaly",
			fmt.Sprintf("仓 %d 末路触发(%s)但腿形态异常——不可定量，人工介入", p.id, trig))
		return
	}
	var cand []leg
	if legs.single {
		cand = []leg{legs.a}
	} else {
		cand = []leg{legs.a, legs.b}
	}
	spec := EmergencySpec{PositionID: p.id, StrategyRef: p.stratRef, InstID: p.inst,
		NotionalUSD: p.notional, Trigger: trig}
	var settled []settledLeg
	for _, l := range cand {
		szr, resid, ok := w.venueSizerResidual(ctx, l)
		if !ok {
			w.rej.Inc("facts_missing")
			w.al.Alert(ctx, "emergency_read_fail",
				fmt.Sprintf("仓 %d %s venue 实读/sizer 缺——末路 fail-closed 不发单（锚②）", p.id, l.inst))
			return
		}
		lot, lerr := szr.LotSz(ctx, l.inst)
		if lerr != nil {
			w.rej.Inc("facts_missing")
			w.al.Alert(ctx, "emergency_lot_fail",
				fmt.Sprintf("仓 %d %s lotSz 读败——末路 fail-closed 不发单", p.id, l.inst))
			return
		}
		q := resid
		if lot > 0 {
			q = math.Floor(resid/lot+1e-9) * lot
		}
		if q <= 0 {
			settled = append(settled, settledLeg{venue: l.venue, instID: l.inst, avail: resid, want: resid})
			continue
		}
		spec.Legs = append(spec.Legs, EmergencyLeg{Venue: l.venue, InstID: l.inst,
			Side: w.emergencySide(ctx, l), Sz: q, Ticket: l.ticket})
	}
	for _, s := range settled {
		w.noteSettledLeg(ctx, p, s)
	}
	trigObj := closeTrig{kind: "emergency_exit",
		reason: fmt.Sprintf("emergency_exit(%s)：正径失效末路只减仓——venue 实读定量 %d 腿", trig, len(spec.Legs)),
		facts:  map[string]any{"emergency_trigger": trig, "emergency_legs": len(spec.Legs)}}
	if len(spec.Legs) == 0 {
		// 全腿已清=簿记收殓（无单可发——externally_settled 事实已挂账）。
		if err := w.markClosed(ctx, p, 0, trigObj); err != nil {
			log.Printf("[guard] WARN 仓 %d 末路收殓态迁写败：%v", p.id, err)
			return
		}
		w.emgEvent(ctx, p, trig, "全腿 venue 已清——簿记收殓 closed（无单可发）")
		log.Printf("[guard] 仓 %d 末路收殓（trigger=%s）——closed", p.id, trig)
		return
	}
	execID, err := w.emg.exec.ExecuteEmergency(ctx, spec)
	if err != nil {
		w.rej.Inc("close_failed")
		w.al.Alert(ctx, "emergency_failed",
			fmt.Sprintf("仓 %d 末路执行败（%s）：%v——下拍按 venue 实读残量续平", p.id, trig, err))
		w.emgEvent(ctx, p, trig, fmt.Sprintf("末路执行败 exec#%d：%v", execID, err))
		return
	}
	// 残袋对拍复用（bareBags 消费 [2]CloseLeg——末路腿同构填充）。
	var cls [2]CloseLeg
	for i, l := range spec.Legs {
		if i >= 2 {
			break
		}
		cls[i] = CloseLeg{Venue: l.Venue, InstID: l.InstID, Side: l.Side,
			Sz: l.Sz, SzKnown: true, Ticket: l.Ticket}
	}
	w.bareBags(ctx, p, legs, cls, execID)
	if err := w.markClosed(ctx, p, execID, trigObj); err != nil {
		log.Printf("[guard] WARN 仓 %d 末路成交但态迁写败：%v", p.id, err)
		return
	}
	w.emgEvent(ctx, p, trig, fmt.Sprintf("末路只减仓成交 exec#%d %d 腿", execID, len(spec.Legs)))
	log.Printf("[guard] 仓 %d 末路平仓（trigger=%s exec=%d 腿=%d）", p.id, trig, execID, len(spec.Legs))
}

func (w *Warden) emgCooldown() time.Duration {
	sec := 300.0
	if v, err := w.getF("guard.emergency_retry_seconds"); err == nil && v > 0 {
		sec = v
	}
	return time.Duration(sec * float64(time.Second))
}

// emgEvent 末路事实行（ops_event append-only——每次末路动作一行，
// 节流键=emg.last 冷却窗内天然限频）。
func (w *Warden) emgEvent(ctx context.Context, p posRow, trig, detail string) {
	if _, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('仓巡','末路平仓','22247',$1)`,
		fmt.Sprintf("仓#%d trigger=%s %s", p.id, trig, detail)); err != nil {
		w.rej.Inc("write_err")
	}
}
