// ddline.go —— 回撤硬线执行件 v2（工单#20799；T-0025 段一军令机械化不变：
// -3%/30d 净权益回撤触线=全平+kill_switch 落闸+告警，无自动复位）。
// v2 判撤形态（O-DD1）：采样只进内存→5 分钟柱（risk.dd_bar_seconds）中位数
// 落 md_equity_curve 柱行（src='guard_dd_bar'）→收柱才判撤→峰值只读柱行。
// 根修点：kill_switch 走 GetBool 宽容读（产线值是 fin_0008 种的 jsonb 布尔——
// 旧 killSwitchOn 单型 string 解每拍解析败=硬线失明三周，canon
// config-json-type-fragility 复发实证）；判据/开关/权益读不出=dd_line_blind
// 告警不静默。权益三分量非原子读的单采点毛刺（划转/演习瞬间低 11%~81% 实证）
// 由柱中位数+稀柱门槛（ddBarMinSamples）吸收。
// 触线动作序：kill_switch 先落（封新开——close 径 T-0021 免检不回退）→逐仓 dd_30d
// 全平（复用仓巡 tend 径：过闸/反向原子平/节拍重试）→告警计数+log 升级。
// 复位语义：本件永不写 kill_switch=false——翻闸只认显式热改（重启不复位，测试锚）。
// -3% 定值只认业主令（skills/策略评估与仓位：硬线调整越权即停）——本件只执行不裁量。
package guard

import (
	"context"
	"fmt"
	"log"
	"time"
)

const (
	ddTrigger = "dd_30d" // 硬线触发名（fin_guard_close.trigger 留痕）
)

// clock 测试时钟缝（w.now nil=time.Now）。
func (w *Warden) clock() time.Time {
	if w.now != nil {
		return w.now()
	}
	return time.Now()
}

// ddScan 一拍：收样→收柱落行→收柱才判撤。缺键/闸读不出=失明告警 fail-closed
// （键由迁移种子——缺席=告警面可见，禁虚构判据）。
func (w *Warden) ddScan(ctx context.Context) {
	if w.equity == nil {
		w.rej.Inc("equity_missing")
		return // 未挂权益件=非硬线岗（simdrill 等跨包 Warden）——零 dd 动作
	}
	thr, err := w.getF("risk.max_dd_30d")
	if err != nil || thr <= 0 || thr >= 1 {
		w.ddBlind(ctx, "判撤参数缺/非法")
		return
	}
	days, err := w.getF("risk.dd_window_days")
	if err != nil || days < 1 {
		w.ddBlind(ctx, "判撤参数缺/非法")
		return
	}
	barSec, err := w.getF("risk.dd_bar_seconds")
	if err != nil || barSec < ddBarSecMin || barSec > ddBarSecMax {
		w.ddBlind(ctx, "判撤参数缺/非法")
		return
	}
	tripped, err := w.cfg.GetBool("risk.kill_switch")
	if err != nil {
		w.ddBlind(ctx, "risk.kill_switch 读不出")
		return
	}
	if tripped {
		w.ddResume(ctx) // 触线态续平不依赖权益读面（读败照平——熔断已落，仓不能不平）
	}
	now := w.clock()
	eq, err := w.equity(ctx)
	if err != nil {
		w.rej.Inc("equity_missing")
		w.ddEqFail(ctx, now, barSec, err)
		return
	}
	w.eqFailSince = time.Time{}
	med, end, n, closed := w.bar.push(now, barSec, eq)
	if !closed {
		return
	}
	if n < ddBarMinSamples {
		w.rej.Inc("dd_bar_thin")
		return // 稀柱不落行不判（中位数结构下限——样本不足不虚构柱值）
	}
	// src 必须以字面量出现——偏索引谓词匹配（md_equity_curve_ddbar_idx）。
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_equity_curve(ts,venue,ccy,equity,src) VALUES($1,'okx','USDT',$2,'guard_dd_bar')`, end, med); err != nil {
		w.rej.Inc("write_err")
		return
	}
	if tripped {
		return // 已触线只续平不重判
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
	dd := (peak - med) / peak
	if dd < thr {
		return // 未触线零动作（测试锚：阈下不误触）
	}
	// T-0415：触线实况载荷留仓行决策素材（ddResume→tend→平仓行 payload 携实况）。
	w.ddFacts = map[string]any{"dd": dd, "dd_threshold": thr, "dd_peak": peak, "dd_current": med}
	// 触线：kill_switch 先落闸（封新开）→全平→升级留痕。
	if _, err := w.pool.Exec(ctx,
		"UPDATE app_config SET value='true' WHERE key='risk.kill_switch'"); err != nil {
		w.rej.Inc("write_err")
		log.Printf("ALERT [guard] 回撤触线但 kill_switch 落闸写败（%v）——人工落闸", err)
		return
	}
	w.rej.Inc("dd_hardline_tripped")
	w.al.Alert(ctx, "dd_hardline_tripped",
		fmt.Sprintf("dd=%.4f≥%.4f peak=%.2f cur=%.2f", dd, thr, peak, med))
	seg := "30d 滚动窗"
	if !full {
		seg = "起点至今 HWM（史不足窗口）"
	}
	log.Printf("ALERT [guard] ERROR 回撤硬线触发：dd=%.4f ≥ %.4f（%s，peak=%.2f cur=%.2f）——kill_switch 落闸+逐仓全平，复盘后显式翻闸", dd, thr, seg, peak, med)
	w.ddResume(ctx) // 触发拍即首轮续平（首败次拍起按节拍重入）
}

// windowPeak 窗口峰值（venue/ccy 维，只读柱行——历史 guard_dd/test 等非柱行
// 一律不入峰值：硬线自首根柱起算）。返回 full=柱行 min(ts) 距今≥days（史满
// 窗口→30d 滚动 peak）；史不足=首根柱至今 HWM（会签补正②语义沿柱行面）。
func (w *Warden) windowPeak(ctx context.Context, days float64) (peak float64, full bool, err error) {
	var earliest *time.Time
	if err = w.pool.QueryRow(ctx, `SELECT min(ts) FROM md_equity_curve
		WHERE venue='okx' AND ccy='USDT' AND src='guard_dd_bar'`).Scan(&earliest); err != nil {
		return 0, false, err
	}
	full = earliest != nil && time.Since(*earliest) >= time.Duration(days*float64(24*time.Hour))
	if full {
		err = w.pool.QueryRow(ctx, `SELECT max(equity) FROM md_equity_curve
			WHERE venue='okx' AND ccy='USDT' AND src='guard_dd_bar'
			  AND ts >= now()-make_interval(days => $1::int)`, int(days)).Scan(&peak)
	} else {
		err = w.pool.QueryRow(ctx, `SELECT max(equity) FROM md_equity_curve
			WHERE venue='okx' AND ccy='USDT' AND src='guard_dd_bar'`).Scan(&peak)
	}
	return peak, full, err
}

// ddBlind 硬线失明（判据/开关读不出）：计数+限频日志+告警。why 用稳定短语不带变量
// （Alerter 按指纹退避——带错误串=每次新指纹=刷屏）；原始 err 只进日志。
func (w *Warden) ddBlind(ctx context.Context, why string) {
	w.rej.Inc("dd_missing_param")
	w.note("dd_blind", "ALERT [guard] 回撤硬线失明——"+why)
	w.al.Alert(ctx, "dd_line_blind", why)
}

// ddEqFail 权益读败：限频日志；连续读败满一柱长=失明告警（瞬时读败不报）。
func (w *Warden) ddEqFail(ctx context.Context, now time.Time, barSec float64, err error) {
	w.note("dd_eqfail", fmt.Sprintf("ALERT [guard] 净权益现读败——回撤判撤让位 fail-closed: %v", err))
	if w.eqFailSince.IsZero() {
		w.eqFailSince = now
		return
	}
	if now.Sub(w.eqFailSince) >= time.Duration(barSec*float64(time.Second)) {
		w.note("dd_blind", "ALERT [guard] 回撤硬线失明——净权益连续读败")
		w.al.Alert(ctx, "dd_line_blind", "净权益连续读败")
	}
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
		if err := rows.Scan(&p.id, &p.execID, &p.stratRef, &p.inst, &p.notional,
			&p.opened, &p.attempts, &p.stratState); err != nil {
			rows.Close()
			w.rej.Inc("write_err")
			return
		}
		batch = append(batch, p)
	}
	rows.Close()
	for _, p := range batch {
		w.tend(ctx, p, maxAtt, closeTrig{kind: ddTrigger,
			reason: "dd_30d：回撤硬线触线——kill_switch 落闸逐仓全平",
			facts:  w.ddFacts})
	}
}
