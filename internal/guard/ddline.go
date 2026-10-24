// ddline.go —— 回撤硬线执行件（T-0025 段一；并入「仓巡」产出窗——业主令 2026-10-24
// 军令机械化：-3%/30d 净权益回撤触线=全平+kill_switch 落闸+告警，无自动复位）。
// 数据源（会签补正①）：venue 现读 equity（balances 实读含 uPnL，对账环
// 同基准）→每拍采点 md_equity_curve 事实行（append-only 统计面；快照行本身 WS 事件源，
// 本件零 venue 轮询）。判撤语义（会签补正②）：史满窗口→30d 滚动 peak-to-now；
// 史不足→交易起点至今 HWM（开业首月无保护洞）。
// 触线动作序：kill_switch 先落（封新开——close 径 T-0021 免检不回退）→逐仓 dd_30d
// 全平（复用仓巡 tend 径：过闸/反向原子平/节拍重试）→告警计数+log 升级。
// 复位语义：本件永不写 kill_switch=false——翻闸只认显式热改（重启不复位，测试锚）。
// -3% 定值只认业主令（skills/策略评估与仓位：硬线调整越权即停）——本件只执行不裁量。
package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const (
	ddTrigger = "dd_30d" // 硬线触发名（fin_guard_close.trigger 留痕）
)

// ddScan 一拍：采点→判撤→触线动作链。缺键/陈源=fail-closed 本拍不判+计数
// （键由迁移种子——缺席=告警面可见，禁虚构判据）。
func (w *Warden) ddScan(ctx context.Context) {
	thr, err := w.getF("risk.max_dd_30d")
	if err != nil {
		w.rej.Inc("dd_missing_param")
		log.Printf("ALERT [guard] risk.max_dd_30d 缺键——回撤硬线不监视（fail-closed）: %v", err)
		return
	}
	days, err := w.getF("risk.dd_window_days")
	if err != nil || days <= 0 {
		w.rej.Inc("dd_missing_param")
		log.Printf("ALERT [guard] risk.dd_window_days 缺键/非正——回撤硬线不监视: %v", err)
		return
	}
	cur, ok := w.captureEquity(ctx)
	if !ok {
		return // 计数门已挂（equity_stale/missing）——陈源/缺源不判
	}
	if tripped, err := w.killSwitchOn(ctx); err != nil {
		return
	} else if tripped {
		w.ddResume(ctx) // 触线态：仍有在仓行则每拍续平（节拍/上限走热启键），全平后归静默
		return
	}
	peak, full, err := w.windowPeak(ctx, days)
	if err != nil {
		w.rej.Inc("write_err")
		return
	}
	if peak <= 0 {
		w.rej.Inc("dd_no_base")
		return // 无正权益基线（净权益≤0 已是超限态——但基线无据不虚构判撤）
	}
	dd := (peak - cur) / peak
	if dd < thr {
		return // 未触线零动作（测试锚：阈下不误触）
	}
	// 触线：kill_switch 先落闸（封新开）→全平→升级留痕。
	if _, err := w.pool.Exec(ctx,
		"UPDATE app_config SET value='true' WHERE key='risk.kill_switch'"); err != nil {
		w.rej.Inc("write_err")
		log.Printf("ALERT [guard] 回撤触线但 kill_switch 落闸写败（%v）——人工落闸", err)
		return
	}
	w.rej.Inc("dd_hardline_tripped")
	w.al.Alert(ctx, "dd_hardline_tripped",
		fmt.Sprintf("dd=%.4f≥%.4f peak=%.2f cur=%.2f", dd, thr, peak, cur))
	seg := "30d 滚动窗"
	if !full {
		seg = "起点至今 HWM（史不足窗口）"
	}
	log.Printf("ALERT [guard] ERROR 回撤硬线触发：dd=%.4f ≥ %.4f（%s，peak=%.2f cur=%.2f）——kill_switch 落闸+逐仓全平，复盘后显式翻闸", dd, thr, seg, peak, cur)
	w.ddResume(ctx) // 触发拍即首轮续平（首败次拍起按节拍重入）
}

// captureEquity 净权益采点（venue 现读件→md_equity_curve 事实行——T-0048 快照
// 出局：硬线判定时点 equity 实读，读件缺/读败=计数不采不判）。
func (w *Warden) captureEquity(ctx context.Context) (float64, bool) {
	if w.equity == nil {
		w.rej.Inc("equity_missing")
		return 0, false
	}
	eq, err := w.equity(ctx)
	if err != nil {
		w.rej.Inc("equity_missing")
		log.Printf("ALERT [guard] 净权益现读败——回撤判撤让位 fail-closed: %v", err)
		return 0, false
	}
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_equity_curve(ts,venue,ccy,equity,src) VALUES(now(),'okx','USDT',$1,'guard_dd')`, eq); err != nil {
		w.rej.Inc("write_err")
		return 0, false
	}
	return eq, true
}

// windowPeak 窗口峰值（venue/ccy 维）。返回 full=史满窗口（peak 取滚动窗）；
// 史不足=起点至今 HWM（会签补正②——开业首月照判不豁免）。
func (w *Warden) windowPeak(ctx context.Context, days float64) (peak float64, full bool, err error) {
	var earliest time.Time
	err = w.pool.QueryRow(ctx, `SELECT min(ts) FROM md_equity_curve
		WHERE venue='okx' AND ccy='USDT'`).Scan(&earliest)
	if err != nil {
		return 0, false, err
	}
	full = time.Since(earliest) >= time.Duration(days*float64(24*time.Hour))
	q := `SELECT max(equity) FROM md_equity_curve WHERE venue='okx' AND ccy='USDT'`
	if full {
		q += fmt.Sprintf(" AND ts >= now()-interval '%d days'", int(days))
	}
	err = w.pool.QueryRow(ctx, q).Scan(&peak)
	return peak, full, err
}

// killSwitchOn 闸态读（缺键=error 上抛——闸面判据缺=本拍不动钱）。
func (w *Warden) killSwitchOn(ctx context.Context) (bool, error) {
	raw, err := w.cfg.Get("risk.kill_switch")
	if err != nil {
		w.rej.Inc("dd_missing_param")
		return false, err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		w.rej.Inc("dd_missing_param")
		return false, err
	}
	return s == "true", nil
}

// ddResume 触线态每拍续平循环：仍可平的仓（节拍/上限走热启键）继续 dd_30d 强制径；
// 上限耗尽的仓持续告警不静默（敞口在最大回撤时刻禁无限期开放）；全平后归静默
// （不重复落闸——dd_hardline_tripped 计数只加一次，触发拍已计）。
func (w *Warden) ddResume(ctx context.Context) {
	backoff, maxAtt, err := w.retryKeys()
	if err != nil {
		w.rej.Inc("dd_missing_param")
		log.Printf("ALERT [guard] 触线态重试键缺（%v）——续平停，人工介入", err)
		return
	}
	// 耗尽仓持续告警（在仓行 attempts≥上限=平不掉——每拍声张，禁静默漂移）。
	var stuck int
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM fin_position p
		LEFT JOIN fin_guard_close g ON g.position_id = p.id
		WHERE p.closed_at IS NULL AND COALESCE(g.attempts,0) >= $1`, maxAtt).Scan(&stuck); err != nil {
		w.rej.Inc("write_err")
		return
	}
	if stuck > 0 {
		w.rej.Inc("dd_close_exhausted")
		w.al.Alert(ctx, "dd_close_exhausted", fmt.Sprintf("%d 仓耗尽敞口仍开——人工介入", stuck))
		log.Printf("ALERT [guard] ERROR 回撤触线平仓重试耗尽——%d 仓敞口仍开，人工介入", stuck)
	}
	// 仍可平仓（节拍窗内）续平。
	rows, err := w.pool.Query(ctx, `SELECT p.id, p.execution_id, p.strategy_ref, p.inst_id,
		p.notional_usd, p.opened_at, COALESCE(g.attempts,0), COALESCE(s.state,'')
		FROM fin_position p
		LEFT JOIN fin_guard_close g ON g.position_id = p.id
		LEFT JOIN fin_strategy s ON s.id = p.strategy_ref
		WHERE p.closed_at IS NULL AND COALESCE(g.state,'open') <> 'closed'
		  AND COALESCE(g.attempts,0) < $1
		  AND (g.last_try_at IS NULL OR now() - g.last_try_at >= make_interval(secs => $2))`,
		maxAtt, backoff)
	if err != nil {
		w.rej.Inc("write_err")
		log.Printf("ALERT [guard] 触线续平扫仓败（%v）——人工升级", err)
		return
	}
	var batch []posRow
	for rows.Next() {
		var p posRow
		if err := rows.Scan(&p.id, &p.execID, &p.stratRef, &p.inst, &p.notional, &p.opened,
			&p.attempts, &p.stratState); err != nil {
			rows.Close()
			w.rej.Inc("write_err")
			return
		}
		batch = append(batch, p)
	}
	rows.Close()
	for _, p := range batch {
		w.tend(ctx, p, maxAtt, ddTrigger)
	}
}
