package guard

// 事实读面与状态写面（guard 拆件：核心循环见 guard.go）。

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// fundingRate funding 行读件（latest=已结算/next=预测分口径；无行/空值=false）。
// T-0128 同病灶过闸：funding 行亦断言新鲜度——结算窗 8h 故阈=9h
// （guard.funding_max_age_min 缺省 540min；陈行=缺 fail-closed）。
func (w *Warden) fundingRate(ctx context.Context, inst, col string) (float64, bool) {
	var rate *float64
	var ts time.Time
	err := w.pool.QueryRow(ctx, fmt.Sprintf(
		"SELECT %s,ts FROM md_funding WHERE inst_id=$1 AND %s IS NOT NULL ORDER BY ts DESC LIMIT 1",
		col, col), inst).Scan(&rate, &ts)
	if err != nil || rate == nil {
		return 0, false
	}
	maxMin := 540.0
	if v, err := w.getF("guard.funding_max_age_min"); err == nil && v > 0 {
		maxMin = v
	}
	if time.Since(ts) > time.Duration(maxMin*float64(time.Minute)) {
		return 0, false
	}
	return *rate, true
}
func (w *Warden) latestFunding(ctx context.Context, inst string) (float64, bool) {
	return w.fundingRate(ctx, inst, "funding_rate")
}
func (w *Warden) nextFunding(ctx context.Context, inst string) (float64, bool) {
	return w.fundingRate(ctx, inst, "next_funding_rate")
}

// driftTriggered 漂移双闸（T-0064）：①绝对兜底 |imb|>500bps 无条件杀——
// 仅施于「基线域内」仓（|open|≤cap：该仓应近衡,裸值失守=真杀）；②语义闸
// |imb_now−imb_open|>band——基线 NULL=首拍实算落列零杀（基线播种拍）。
// 播种态豁免裸值兜底（升件#14117 修）：|open|>cap 的仓=粒度底残差域
// （KB lotsz-granularity 实证 $24 腿×0.01 张网格≈9800bps），abs-cap 前置
// 会把相对闸死码化+首拍残差不播种即杀——基线超阈仓只走相对闸域。
const driftAbsCapBps = 500

func (w *Warden) driftTriggered(ctx context.Context, posID int64, imbNow, bandBps float64) bool {
	var open *float64
	if err := w.pool.QueryRow(ctx,
		"SELECT open_imbalance_bps FROM fin_position WHERE id=$1", posID).Scan(&open); err != nil || open == nil {
		// 基线播种：首拍实算落列，本拍零杀（出生残差入基线——粒度底
		// 残差先入列,次拍起相对闸接管；abs-cap 不前置抢杀播种拍）。
		if _, err := w.pool.Exec(ctx,
			"UPDATE fin_position SET open_imbalance_bps=$2 WHERE id=$1 AND open_imbalance_bps IS NULL", posID, imbNow); err != nil {
			w.rej.Inc("write_err")
		}
		return false
	}
	if math.Abs(*open) <= driftAbsCapBps &&
		(imbNow > driftAbsCapBps || imbNow < -driftAbsCapBps) {
		return true // 绝对兜底闸：基线域内仓裸腿级失守无条件杀
	}
	return math.Abs(imbNow-*open) > bandBps
}

func (w *Warden) getF(k string) (float64, error) {
	// T-0046 ④根修：单型 string 解读废裸数字键（canon config-json-type-fragility） ——改走平台双型读器（number|string 两形皆收）。
	return w.cfg.GetFloat(k)
}

// note skip 态限频结构化日志（T-0046 ④c——无触发也是决策必须可观测； 计数器有可见出口，10min 键级节流防噪音——alert 级噪音禁线）。
func (w *Warden) note(key, msg string) {
	if w.notes == nil {
		w.notes = map[string]time.Time{}
	}
	if t, ok := w.notes[key]; ok && time.Since(t) < 10*time.Minute {
		return
	}
	w.notes[key] = time.Now()
	log.Printf("[guard] skip 态可见化: %s", msg)
}

// retryKeys 重试节拍键（缺任一=整轮 fail-closed 跳过）。
func (w *Warden) retryKeys() (backoff, maxAtt int64, err error) {
	bf, err1 := w.getF("guard.retry_backoff_seconds")
	ma, err2 := w.getF("guard.retry_max_attempts")
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("guard.retry_* 键缺件 fail-closed: %v/%v", err1, err2)
	}
	return int64(bf), int64(ma), nil
}

func (w *Warden) markClosing(ctx context.Context, posID int64, trig string) error {
	w.markPosClosing(ctx, posID) // FSM open→closing 尽力写（#25806——漂移由 reconcile 兜）
	_, err := w.pool.Exec(ctx, `INSERT INTO fin_guard_close(position_id,state,trigger_kind,attempts,last_try_at,last_attempt_at,updated_at)
		VALUES($1,'closing',$2,1,now(),now(),now())
		ON CONFLICT(position_id) DO UPDATE SET state='closing',trigger_kind=$2,
		  attempts=fin_guard_close.attempts+1,last_try_at=now(),last_attempt_at=now(),updated_at=now()`, posID, trig)
	if err != nil {
		w.rej.Inc("write_err")
	}
	return err
}

func (w *Warden) markFailed(ctx context.Context, posID int64, reason string) {
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_guard_close SET state='close_failed',last_error=$2,updated_at=now() WHERE position_id=$1`, posID, reason); err != nil {
		w.rej.Inc("write_err")
	}
}

// noteAttemptFail 平仓尝试败因落表（T-0428）——state 不迁（sizer 败径按节拍
// 原样重试，状态机语义不动）；last_error 逐次覆盖为最新败因，平成不清（
// attempts>1 的仓靠它记「为什么重试过」——清了记忆即断）。
func (w *Warden) noteAttemptFail(ctx context.Context, posID int64, reason string) {
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_guard_close SET last_error=$2,updated_at=now() WHERE position_id=$1`, posID, reason); err != nil {
		w.rej.Inc("write_err")
	}
}

// markClosed 平仓成交态迁移（同 tx：原仓+反向腿衍生仓行一并 closed——衍生行不关 =哨戒自触发循环；guard 子面记录收官）。
// T-0415 红桩①：态迁提交后续写 ops_decision close 行——触发判据+实况（reason）
// +实现的vs期望的（payload expected_net_apr/realized_usd/expected_carry_usd），
// 「为什么平」从此可问；记忆写不阻钱径（写败升件自报非回滚已平事实）。
func (w *Warden) markClosed(ctx context.Context, p posRow, closeExecID int64, trig closeTrig) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		w.rej.Inc("write_err")
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`UPDATE fin_position SET closed_at=now() WHERE (id=$1 OR execution_id=$2) AND state<>'closed'`,
		p.id, closeExecID); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	// upsert 非 UPDATE（工单#22247：受令末路可及无 guard 行仓——closed 终态
	// 行必立，「attempts 耗尽仓未平」无行滞留面归零；trigger_kind 同步真值）。
	if _, err := tx.Exec(ctx,
		`INSERT INTO fin_guard_close(position_id,state,trigger_kind,closed_at,updated_at)
		 VALUES($1,'closed',$2,now(),now())
		 ON CONFLICT(position_id) DO UPDATE SET state='closed',closed_at=now(),
		   trigger_kind=$2,updated_at=now()`, p.id, trig.kind); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	// T-0046 ⑥：exec 态闭环——原仓 exec 行（open 挂账）+平仓 exec 行一并 closed
	//（不关=earn sweepIdle 的 inFlight 闸永堵——首仓事故实证副作用）。
	if _, err := tx.Exec(ctx, `UPDATE fin_execution SET state='closed'
		WHERE state IN ('open','legged')
		  AND (id=$2 OR id IN (SELECT execution_id FROM fin_position WHERE id=$1))`, p.id, closeExecID); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	w.writeCloseDecision(ctx, p, closeExecID, trig) // T-0415 红桩①平仓决策行
	return nil
}

// trailingIncomePositive 收益失效判实读（T-0111）：fin_cashflow 归仓行
// kind=funding 近 24h 实收合计>0=仓仍在赚——免超龄处决。读败/无行=false
// 保守侧（证据缺不豁免——fail-closed 回年龄判据）。
func (w *Warden) trailingIncomePositive(ctx context.Context, posID int64) bool {
	var sum float64
	if err := w.pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0) FROM fin_cashflow
		WHERE position_ref=$1 AND kind='funding_settle' AND occurred_at>now()-interval '24 hours' AND ccy IN ('USDT','USD') `,
		posID).Scan(&sum); err != nil {
		log.Printf("WARN [guard] 仓 %d trailing 实收读败（豁免判据回退年龄闸）: %v", posID, err)
		return false
	}
	return sum > 0
}
