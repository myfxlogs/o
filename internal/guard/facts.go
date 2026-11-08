package guard

// 事实读面与状态写面（guard 拆件：核心循环见 guard.go）。

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// legSet 一仓双腿（fin_order_leg 按 execution 取；a/b=leg 序）。
type legSet struct {
	venue string
	a, b  leg
	one   bool // 单腿/无腿/>2 腿异常态
}

type leg struct {
	inst, side, venue string
	sz                float64
	szUnit            string  // T-0107 量纲语义位：base/quote/contract/lot（T-0145 mt5 手数）；空=历史未消歧
	qtyBase           float64 // venue 实填币量（accFillSz）——spot 盯市唯一真源
	ticket            int64   // T-0145：venue 仓凭据（MT5 ticket——hedge 户 OrderClose 必携；无票 venue=0）
}

func (ls legSet) labels() []string {
	out := []string{}
	if !ls.one {
		out = append(out, ls.a.side+"@"+ls.a.inst, ls.b.side+"@"+ls.b.inst)
	}
	return out
}

// reverse 双腿反向（平仓语义：side 取反、inst 沿原仓、量按 closeWant 消歧）。
func (ls legSet) reverse() [2]CloseLeg {
	flip := func(side string) string {
		if side == "buy" || side == "long" {
			return "sell"
		}
		return "buy"
	}
	mk := func(l leg) CloseLeg {
		want, ok := l.closeWant()
		// T-0145 根修：venue 按腿取非父腿抄送（跨 venue 双腿——mt5 腿抄 okx
		// venue=平仓路由错 venue 病灶）；ticket 透传 venue 仓凭据（mt5 平
		// 仓 OrderClose 必携——无票=failure 禁反向单）。
		return CloseLeg{Venue: l.venue, InstID: l.inst, Side: flip(l.side), Sz: want, SzKnown: ok, Ticket: l.ticket}
	}
	return [2]CloseLeg{mk(ls.a), mk(ls.b)}
}

// perpSign funding 收付向：position inst 的腿 short/sell=+1（正 funding 收方）、
// long/buy=−1；腿缺/侧不可辨=false（调用方计数不误判）。
func perpSign(ls legSet, inst string) (float64, bool) {
	l, ok := ls.of(inst)
	if !ok {
		return 0, false
	}
	switch l.side {
	case "sell", "short":
		return 1, true
	case "buy", "long":
		return -1, true
	}
	return 0, false
}

func (ls legSet) of(inst string) (leg, bool) {
	if ls.a.inst == inst {
		return ls.a, true
	}
	if ls.b.inst == inst {
		return ls.b, true
	}
	return leg{}, false
}

func (w *Warden) legs(ctx context.Context, execID int64) legSet {
	rows, err := w.pool.Query(ctx,
		fmt.Sprintf(`SELECT inst_id, side, sz, venue, sz_unit, qty_base, venue_ticket FROM %s WHERE execution_id=$1 ORDER BY leg_no`, w.dom.OrderLeg()), execID)
	if err != nil {
		w.rej.Inc("facts_missing")
		return legSet{one: true}
	}
	defer rows.Close()
	var ls legSet
	for rows.Next() {
		var l leg
		var su *string
		var qb *float64
		var tk *int64
		if err := rows.Scan(&l.inst, &l.side, &l.sz, &l.venue, &su, &qb, &tk); err != nil {
			w.rej.Inc("facts_missing")
			return legSet{one: true}
		}
		if su != nil {
			l.szUnit = *su
		}
		if qb != nil {
			l.qtyBase = *qb
		}
		if tk != nil {
			l.ticket = *tk
		}
		ls.venue = l.venue
		if ls.a.inst == "" {
			ls.a = l
		} else if ls.b.inst == "" {
			ls.b = l
		} else {
			ls.one = true // >2 腿=契约外形态，计数防误判
		}
	}
	if ls.a.inst == "" || ls.b.inst == "" {
		return legSet{one: true}
	}
	return ls
}

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

// driftTriggered 漂移双闸（T-0064）：①绝对兜底 |imb|>500bps 无条件杀；②语义闸
// |imb_now−imb_open|>band——基线 NULL=首拍实算落列零杀（基线播种拍）。
const driftAbsCapBps = 500

func (w *Warden) driftTriggered(ctx context.Context, posID int64, imbNow, bandBps float64) bool {
	if imbNow > driftAbsCapBps || imbNow < -driftAbsCapBps {
		return true // 绝对兜底闸：裸腿级失衡无条件杀（基线也拦不住的失守）
	}
	var open *float64
	if err := w.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT open_imbalance_bps FROM %s WHERE id=$1", w.dom.Position()), posID).Scan(&open); err != nil || open == nil {
		// 基线播种：首拍实算落列，本拍零杀（出生残差入基线）。
		if _, err := w.pool.Exec(ctx,
			fmt.Sprintf("UPDATE %s SET open_imbalance_bps=$2 WHERE id=$1 AND open_imbalance_bps IS NULL", w.dom.Position()), posID, imbNow); err != nil {
			w.rej.Inc("write_err")
		}
		return false
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
	_, err := w.pool.Exec(ctx, fmt.Sprintf(`INSERT INTO %s(position_id,state,trigger_kind,attempts,last_try_at,last_attempt_at,updated_at)
		VALUES($1,'closing',$2,1,now(),now(),now())
		ON CONFLICT(position_id) DO UPDATE SET state='closing',trigger_kind=$2,
		  attempts=%[1]s.attempts+1,last_try_at=now(),last_attempt_at=now(),updated_at=now()`, w.dom.GuardClose()), posID, trig)
	if err != nil {
		w.rej.Inc("write_err")
	}
	return err
}

func (w *Warden) markFailed(ctx context.Context, posID int64, reason string) {
	if _, err := w.pool.Exec(ctx,
		fmt.Sprintf(`UPDATE %s SET state='close_failed',last_error=$2,updated_at=now() WHERE position_id=$1`, w.dom.GuardClose()), posID, reason); err != nil {
		w.rej.Inc("write_err")
	}
}

// noteAttemptFail 平仓尝试败因落表（T-0428）——state 不迁（sizer 败径按节拍
// 原样重试，状态机语义不动）；last_error 逐次覆盖为最新败因，平成不清（
// attempts>1 的仓靠它记「为什么重试过」——清了记忆即断）。
func (w *Warden) noteAttemptFail(ctx context.Context, posID int64, reason string) {
	if _, err := w.pool.Exec(ctx,
		fmt.Sprintf(`UPDATE %s SET last_error=$2,updated_at=now() WHERE position_id=$1`, w.dom.GuardClose()), posID, reason); err != nil {
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
		fmt.Sprintf(`UPDATE %s SET closed_at=now() WHERE (id=$1 OR execution_id=$2) AND closed_at IS NULL`, w.dom.Position()),
		p.id, closeExecID); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	if _, err := tx.Exec(ctx,
		fmt.Sprintf(`UPDATE %s SET state='closed',closed_at=now(),updated_at=now() WHERE position_id=$1`, w.dom.GuardClose()), p.id); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	// T-0046 ⑥：exec 态闭环——原仓 exec 行（open 挂账）+平仓 exec 行一并 closed
	//（不关=earn sweepIdle 的 inFlight 闸永堵——首仓事故实证副作用）。
	if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET state='closed'
		WHERE state IN ('open','legged')
		  AND (id=$2 OR id IN (SELECT execution_id FROM %[2]s WHERE id=$1))`, w.dom.Execution(), w.dom.Position()), p.id, closeExecID); err != nil {
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
