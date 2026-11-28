package guard

// legs.go —— 腿集读面件（O-41392 从 facts.go 拆立：行棘轮顶线——
// legSet/leg 型+labels/reverse/perpSign/of 方法+legs() 查询面）。

import "context"

// legSet 一仓双腿（fin_order_leg 按 execution 取；a/b=leg 序）。
type legSet struct {
	venue  string
	a, b   leg
	one    bool // 缺腿/>2 腿异常态（棒5c:恰一腿归 single 不再混畸形——方向族单腿是契约非病灶）
	single bool // 恰一腿（方向族契约位——tend 走单腿平仓径非反向原子径）
	zero   bool // O-41392：查询成功且 0 行=零腿产品形（DCD/结构品正常形——
	// venue 到期结算径收殓，TTL 等时龄判词不适用）；>2 腿/读败不带本位=真畸形照报
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
	switch {
	case ls.single:
		out = append(out, ls.a.side+"@"+ls.a.inst) // 方向族单腿=一签
	case !ls.one:
		out = append(out, ls.a.side+"@"+ls.a.inst, ls.b.side+"@"+ls.b.inst)
	}
	return out
}

// reverse 反向腿（平仓语义：side 取反、inst 沿原仓、量按 closeWant 消歧）。
// 单腿仓=[1]出单腿+[1]空槽（消费侧按 InstID=="" 跳过——棒5c）。
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
	if ls.single {
		return [2]CloseLeg{mk(ls.a), {}} // 单腿=一腿一空槽（sizedCloseLegs/execwiring 按空槽跳）
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
		`SELECT inst_id, side, sz, venue, sz_unit, qty_base, venue_ticket FROM fin_order_leg WHERE execution_id=$1 ORDER BY leg_no`, execID)
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
	if err := rows.Err(); err != nil {
		w.rej.Inc("facts_missing")
		return legSet{one: true} // 迭代败=读败畸形非零腿产品形（O-41392——零腿豁免不禁它）
	}
	if ls.a.inst == "" {
		return legSet{one: true, zero: true} // 零腿=产品形（DCD/结构品——venue 到期收殓径管，O-41392）
	}
	if ls.b.inst == "" {
		ls.single = true // 恰一腿=方向族契约（棒5c——单腿平仓径消费位）
	}
	return ls
}
