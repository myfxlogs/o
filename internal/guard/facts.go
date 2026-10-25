package guard

// 事实读面与状态写面（guard 拆件：核心循环见 guard.go）。

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
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
	szUnit            string  // T-0107 量纲语义位：base/quote/contract；空=历史未消歧
	qtyBase           float64 // venue 实填币量（accFillSz）——spot 盯市唯一真源
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
		`SELECT inst_id, side, sz, venue, sz_unit, qty_base FROM fin_order_leg WHERE execution_id=$1 ORDER BY leg_no`, execID)
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
		if err := rows.Scan(&l.inst, &l.side, &l.sz, &l.venue, &su, &qb); err != nil {
			w.rej.Inc("facts_missing")
			return legSet{one: true}
		}
		if su != nil {
			l.szUnit = *su
		}
		if qb != nil {
			l.qtyBase = *qb
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
func (w *Warden) fundingRate(ctx context.Context, inst, col string) (float64, bool) {
	var rate *float64
	err := w.pool.QueryRow(ctx, fmt.Sprintf(
		"SELECT %s FROM md_funding WHERE inst_id=$1 AND %s IS NOT NULL ORDER BY ts DESC LIMIT 1",
		col, col), inst).Scan(&rate)
	if err != nil || rate == nil {
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
		"SELECT open_imbalance_bps FROM fin_position WHERE id=$1", posID).Scan(&open); err != nil || open == nil {
		// 基线播种：首拍实算落列，本拍零杀（出生残差入基线）。
		if _, err := w.pool.Exec(ctx,
			"UPDATE fin_position SET open_imbalance_bps=$2 WHERE id=$1 AND open_imbalance_bps IS NULL", posID, imbNow); err != nil {
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

// VenueSizer 平仓量 venue 现读面（cmd 装配注入——guard 不识 venue 方言）。
// D-0008 钦定径（canon cex-close-sz-fee-in-base）：再操作量从 venue 实况推
// ——现货平仓量=availBal（taker 费基准币内扣致实持<委托）、合约=posCcy 张数； 禁抄腿行委托量。读败=fail-closed 不构平仓腿。
type VenueSizer interface {
	SpotAvailBal(ctx context.Context, instID string) (float64, error)
	ContractPos(ctx context.Context, instID string) (float64, error) // 净持仓张数 abs
	LotSz(ctx context.Context, instID string) (float64, error)       // 数量步长
}

// alignCloseSz 平仓量对齐：min(腿记录, venue 实读) 按 lotSz 网格向下取整； 对齐后≤0=false（fail-closed 不发残量单）。
func alignCloseSz(want, avail, lotSz float64) (float64, bool) {
	sz := math.Min(want, avail)
	if lotSz > 0 {
		sz = math.Floor(sz/lotSz+1e-9) * lotSz
	}
	if sz <= 0 {
		return 0, false
	}
	return sz, true
}

// sizedCloseLegs 反向腿 venue 实读定量（T-0046 ②根修）——任一腿读败/对齐零= false（fail-closed：不构 intent 不发半仓平仓）。
func (w *Warden) sizedCloseLegs(ctx context.Context, cls [2]CloseLeg) ([2]CloseLeg, bool) {
	if w.sizer == nil {
		return cls, false // 未接线=禁发（现读面缺席——腿行推导禁线）
	}
	for i, cl := range cls {
		var avail float64
		var err error
		if strings.HasSuffix(cl.InstID, "-SWAP") {
			avail, err = w.sizer.ContractPos(ctx, cl.InstID)
		} else {
			avail, err = w.sizer.SpotAvailBal(ctx, cl.InstID)
		}
		if err != nil {
			return cls, false
		}
		lot, err := w.sizer.LotSz(ctx, cl.InstID)
		if err != nil {
			return cls, false
		}
		sz, ok := alignCloseSz(cl.Sz, avail, lot)
		if !ok {
			return cls, false
		}
		cls[i].Sz = sz
	}
	return cls, true
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

// markClosed 平仓成交态迁移（同 tx：原仓+反向腿衍生仓行一并 closed——衍生行不关 =哨戒自触发循环；guard 子面记录收官）。
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
	// T-0046 ⑥：exec 态闭环——原仓 exec 行（open 挂账）+平仓 exec 行一并 closed
	//（不关=earn sweepIdle 的 inFlight 闸永堵——首仓事故实证副作用）。
	if _, err := tx.Exec(ctx, `UPDATE fin_execution SET state='closed'
		WHERE state IN ('open','legged')
		  AND (id=$2 OR id IN (SELECT execution_id FROM fin_position WHERE id=$1))`, posID, closeExecID); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		w.rej.Inc("write_err")
		return err
	}
	return nil
}
