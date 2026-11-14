// dirtrig.go —— 方向族单腿仓 live 出场判据件（棒5c——dir_perp_mom 真户径）：
// 停/盈/TTL/翻面四判同 dirpaper 纸面语义，真仓口径差异=盈亏直算非 fill 摊提
// （entry=fin_order_leg 实腿价，mark=md_ticks 鲜价，sign=腿侧向）。
// 判据键与 paper 径同源（lifecycle.dir.*——一套判据两径消费防双源漂移）。
package guard

import (
	"context"
	"fmt"
	"time"
)

// dirTrig 单腿方向位出场判（legs.single 专属——evaluate 内挂 ③前位）：
// stop=浮亏破 stop_frac×notional；take=浮盈达 take_frac×notional；ttl=超龄；
// flip=自身方向最新评估已拒（信念消亡——与 TTL 机械闸互补）。
// 返回零值 trig=无触发（evaluate 续 ④降级/退役判）。
func (w *Warden) dirTrig(ctx context.Context, p posRow, legs legSet) closeTrig {
	var dir, tmpl string
	if err := w.pool.QueryRow(ctx, `SELECT COALESCE(direction,''),COALESCE(template_ref,'')
		FROM fin_strategy WHERE id=$1`, p.stratRef).Scan(&dir, &tmpl); err != nil {
		w.rej.Inc("facts_missing")
		w.note("facts_missing:dir", fmt.Sprintf("仓 %d 策略方向/模板读败——出场判据跳过", p.id))
		return closeTrig{}
	}
	sign := 1.0
	if legs.a.side == "sell" || dir == "short_perp" {
		sign = -1.0 // 腿侧与 direction 双源同判（sell=空——不一致时 sell 优先如实）
	}
	var entryPx float64
	if err := w.pool.QueryRow(ctx, `SELECT px FROM fin_order_leg
		WHERE execution_id=$1 AND state='filled' ORDER BY leg_no LIMIT 1`,
		p.execID).Scan(&entryPx); err != nil || entryPx <= 0 {
		w.rej.Inc("facts_missing")
		w.note("facts_missing:entry_px", fmt.Sprintf("仓 %d 入场实腿价读败——出场判据跳过", p.id))
		return closeTrig{}
	}
	px, ok := w.lastPx(ctx, legs.a.inst)
	if !ok {
		w.rej.Inc("facts_missing")
		w.note("facts_missing:dir_mark", fmt.Sprintf("仓 %d mark 缺/陈——出场判据跳过（不虚构不残守）", p.id))
		return closeTrig{}
	}
	stop, take, ttl := 0.15, 0.30, 240.0 // 缺省同 dirpaper 内置值（键读败=同值非另设）
	if v, err := w.getF("lifecycle.dir.stop_frac"); err == nil && v > 0 {
		stop = v
	}
	if v, err := w.getF("lifecycle.dir.take_frac"); err == nil && v > 0 {
		take = v
	}
	if v, err := w.getF("lifecycle.dir.ttl_h"); err == nil && v > 0 {
		ttl = v
	}
	frac := sign * (px/entryPx - 1) // 名义 frac=盈亏/仓名义（线性 perp 近似同 paper 径）
	facts := map[string]any{"entry_px": entryPx, "mark_px": px, "frac": frac,
		"direction": dir, "stop_frac": stop, "take_frac": take, "ttl_h": ttl}
	switch {
	case frac <= -stop:
		return closeTrig{kind: "dir_stop_loss",
			reason: fmt.Sprintf("dir_stop_loss：frac=%.4f<=-stop=%.2f（entry=%.6g→mark=%.6g dir=%s）", frac, stop, entryPx, px, dir),
			facts:  facts}
	case frac >= take:
		return closeTrig{kind: "dir_take_profit",
			reason: fmt.Sprintf("dir_take_profit：frac=%.4f>=take=%.2f（entry=%.6g→mark=%.6g dir=%s）", frac, take, entryPx, px, dir),
			facts:  facts}
	case time.Since(p.opened).Hours() > ttl:
		return closeTrig{kind: "dir_ttl",
			reason: fmt.Sprintf("dir_ttl：age=%.1fh>ttl=%.0fh", time.Since(p.opened).Hours(), ttl),
			facts:  facts}
	}
	// 信号翻面：自身方向最新评估已拒=信念消亡（dirmom 拍距内陈判据=同 paper 径语义）。
	var v string
	if err := w.pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval
		WHERE inst_id=$1 AND template=$2 AND direction=$3 ORDER BY ts DESC LIMIT 1`,
		legs.a.inst, tmpl, dir).Scan(&v); err == nil && v == "rejected" {
		facts["latest_verdict"] = v
		return closeTrig{kind: "dir_signal_flip",
			reason: fmt.Sprintf("dir_signal_flip：%s %s %s 最新评估 rejected——持仓信念消亡", legs.a.inst, tmpl, dir),
			facts:  facts}
	}
	return closeTrig{}
}

// lastPx md_ticks 鲜价读（venue<>'sim'——exec.tick_max_age_s 闸同 notionalNow 径）。
func (w *Warden) lastPx(ctx context.Context, inst string) (float64, bool) {
	var last *float64
	var ts time.Time
	if err := w.pool.QueryRow(ctx, `SELECT v,ts FROM (
			SELECT last v,ts FROM md_ticks WHERE inst_id=$1 AND venue<>'sim'
			UNION ALL SELECT px,ts FROM md_mark_index WHERE inst_id=$1 AND venue<>'sim') q
			ORDER BY ts DESC LIMIT 1`, inst).Scan(&last, &ts); err != nil || last == nil {
		return 0, false
	}
	maxAge := 120.0
	if v, err := w.getF("exec.tick_max_age_s"); err == nil && v > 0 {
		maxAge = v
	}
	if time.Since(ts) > time.Duration(maxAge*float64(time.Second)) {
		return 0, false
	}
	return *last, true
}
