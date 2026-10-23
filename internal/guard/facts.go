package guard

// 事实读面与状态写面（guard 拆件：核心循环见 guard.go）。

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// legSet 一仓双腿（fin_order_leg 按 execution 取；a/b=leg 序）。
type legSet struct {
	venue string
	a, b  leg
	one   bool // 单腿/无腿/>2 腿异常态
}

type leg struct {
	inst, side string
	sz         float64
}

func (ls legSet) labels() []string {
	out := []string{}
	if !ls.one {
		out = append(out, ls.a.side+"@"+ls.a.inst, ls.b.side+"@"+ls.b.inst)
	}
	return out
}

// reverse 双腿反向（平仓语义：side 取反、inst/sz 沿原仓）。
func (ls legSet) reverse() [2]CloseLeg {
	flip := func(side string) string {
		if side == "buy" || side == "long" {
			return "sell"
		}
		return "buy"
	}
	return [2]CloseLeg{
		{Venue: ls.venue, InstID: ls.a.inst, Side: flip(ls.a.side), Sz: ls.a.sz},
		{Venue: ls.venue, InstID: ls.b.inst, Side: flip(ls.b.side), Sz: ls.b.sz},
	}
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
		`SELECT inst_id, side, sz, venue FROM fin_order_leg WHERE execution_id=$1 ORDER BY leg_no`, execID)
	if err != nil {
		w.rej.Inc("facts_missing")
		return legSet{one: true}
	}
	defer rows.Close()
	var ls legSet
	for rows.Next() {
		var l leg
		if err := rows.Scan(&l.inst, &l.side, &l.sz, &ls.venue); err != nil {
			w.rej.Inc("facts_missing")
			return legSet{one: true}
		}
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

// latestFunding 最新 funding 事实行（md_funding 按 ts 最新；无行/空值=false）。
func (w *Warden) latestFunding(ctx context.Context, inst string) (float64, bool) {
	var rate *float64
	if err := w.pool.QueryRow(ctx,
		"SELECT funding_rate FROM md_funding WHERE inst_id=$1 ORDER BY ts DESC LIMIT 1", inst).Scan(&rate); err != nil || rate == nil {
		return 0, false
	}
	return *rate, true
}

// notionalNow 腿现价名义额（sz×最新 md_ticks.last；无 tick/空值=false）。
func (w *Warden) notionalNow(ctx context.Context, l leg) (float64, bool) {
	var last *float64
	if err := w.pool.QueryRow(ctx,
		"SELECT last FROM md_ticks WHERE inst_id=$1 ORDER BY ts DESC LIMIT 1", l.inst).Scan(&last); err != nil || last == nil {
		return 0, false
	}
	return l.sz * *last, true
}

func (w *Warden) getF(k string) (float64, error) {
	raw, err := w.cfg.Get(k)
	if err != nil {
		return 0, err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
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
	_, err := w.pool.Exec(ctx, `INSERT INTO fin_guard_close(position_id,state,trigger_kind,attempts,last_try_at,updated_at)
		VALUES($1,'closing',$2,1,now(),now())
		ON CONFLICT(position_id) DO UPDATE SET state='closing',trigger_kind=$2,
		  attempts=fin_guard_close.attempts+1,last_try_at=now(),updated_at=now()`, posID, trig)
	if err != nil {
		w.rej.Inc("write_err")
	}
	return err
}

func (w *Warden) markFailed(ctx context.Context, posID int64) {
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_guard_close SET state='close_failed',updated_at=now() WHERE position_id=$1`, posID); err != nil {
		w.rej.Inc("write_err")
	}
}

// markClosed 平仓成交态迁移（同 tx：原仓+反向腿衍生仓行一并 closed——衍生行不关
// =哨戒自触发循环；guard 子面记录收官）。
func (w *Warden) markClosed(ctx context.Context, posID, closeExecID int64) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		w.rej.Inc("write_err")
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`UPDATE fin_position SET closed_at=now() WHERE (id=$1 OR execution_id=$2) AND closed_at IS NULL`,
		posID, closeExecID); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE fin_guard_close SET state='closed',closed_at=now(),updated_at=now() WHERE position_id=$1`, posID); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	return nil
}
