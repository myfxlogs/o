// cockpit_face.go —— T-0239 业主驾驶舱 GetCockpit：五块只读聚合
// （存活+钱面+工单+待开仓+在做什么）。服务端判态派生，前端零判据。
package main

import (
	"context"
	"time"

	"o/internal/gen/dashboardv1"
)

// stageOf 事件→阶段派生（钦定映射：领单=施工/收口·移交=等审计/挂起=挂起/
// 裁决=闭环；开工=调研中——票面语「各席最新一条自报映射」）。
func stageOf(ev string) string {
	switch ev {
	case "领单", "开工":
		return "施工"
	case "收口", "移交":
		return "等审计"
	case "挂起":
		return "挂起"
	case "裁决":
		return "闭环"
	default:
		return ev
	}
}

func (c coreService) GetCockpit(ctx context.Context,
	_ *dashboardv1.GetCockpitRequest) (*dashboardv1.GetCockpitResponse, error) {
	r := &dashboardv1.GetCockpitResponse{Money: &dashboardv1.CockpitMoney{}}

	// ①⑤ 存活面+在做什么：每席最新自报→阶段派生
	rows, err := c.pool.Query(ctx,
		`SELECT seat,ticket,event,ts,note FROM ops_seat_latest`)
	if err != nil {
		r.Missing = append(r.Missing, "seats")
	}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s dashboardv1.CockpitSeat
			var ts *time.Time
			if err := rows.Scan(&s.Seat, &s.Ticket, &s.Event, &ts, &s.Note); err == nil {
				s.Stage = stageOf(s.Event)
				if ts != nil {
					s.Ts = tsProto(*ts)
				}
				r.Seats = append(r.Seats, &s)
			}
		}
		if rows.Err() != nil {
			r.Missing = append(r.Missing, "seats")
		}
	}

	// ② 钱面：D-0037 口径月净利+本金水位+近30d 日净序列
	moneyErr := false
	if err := c.pool.QueryRow(ctx,
		`SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		 WHERE ts>date_trunc('month',now()) AND kind IN ('interest','funding_settle','fee','spot_buy','spot_sell')`).Scan(&r.Money.NetMonth); err != nil {
		moneyErr = true
	}
	if err := c.pool.QueryRow(ctx,
		`SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='deposit'`).Scan(&r.Money.Capital); err != nil {
		moneyErr = true
	}
	dr, err := c.pool.Query(ctx,
		`SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		 WHERE ts>now()-interval '30 days' AND kind IN ('interest','funding_settle','fee','spot_buy','spot_sell')
		 GROUP BY ts::date ORDER BY ts::date`)
	if err == nil {
		defer dr.Close()
		for dr.Next() {
			var v float64
			if dr.Scan(&v) == nil {
				r.Money.Daily = append(r.Money.Daily, v)
			}
		}
	} else {
		moneyErr = true
	}
	// 敞口段（T-0241 D-0038 标尺）：未平名义合计 vs 本金×10% 硬上限——
	// 三态服务端判（<90%绿 顶格黄 越限红）；查询败=missing 显缺非冒零。
	if err := c.pool.QueryRow(ctx,
		`SELECT COALESCE(sum(notional_usd),0) FROM fin_position WHERE closed_at IS NULL`).Scan(&r.Money.Exposure); err != nil {
		moneyErr = true
	}
	if moneyErr {
		r.Missing = append(r.Missing, "money")
	}
	// 平台分列（T-0241 D-0038⑤升维——承诺/已入/停泊/敞口×平台+合计；
	// committed=配置键 cockpit.committed_<plat>（归属业主候答——键面留位）；
	// crypto=fin_cashflow deposit/earn 净+未平名义；MT5=mt5Live equity 现读）。
	// ②committed 键缺席显缺非吞零（驳回修②——三律③同口径）：键缺
	// =missing 名段+该行 committed 字段置 -1 哨兵（前端显「未设」非 0）。
	var commC, commM float64
	if err := c.pool.QueryRow(ctx, `SELECT value::float FROM app_config WHERE key='cockpit.committed_crypto'`).Scan(&commC); err != nil {
		commC, r.Missing = -1, append(r.Missing, "committed")
	}
	if err := c.pool.QueryRow(ctx, `SELECT value::float FROM app_config WHERE key='cockpit.committed_mt5'`).Scan(&commM); err != nil {
		commM = -1
		if !containsStr(r.Missing, "committed") {
			r.Missing = append(r.Missing, "committed")
		}
	}
	var parked float64
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE kind IN ('earn_in','earn_out')`).Scan(&parked)
	crow := &dashboardv1.CockpitPlatformRow{Platform: "crypto", Committed: commC,
		Deposited: r.Money.Capital, Parked: parked, Exposure: r.Money.Exposure}
	r.Money.Platforms = append(r.Money.Platforms, crow)
	mrow := &dashboardv1.CockpitPlatformRow{Platform: "MT5", Committed: commM}
	if c.mt5Live != nil {
		if accts, e := c.mt5Live(ctx); e == nil {
			for _, a := range accts {
				mrow.Deposited += a.Equity
			}
		} else {
			r.Missing = append(r.Missing, "mt5")
		}
	}
	r.Money.Platforms = append(r.Money.Platforms, mrow)
	tot := &dashboardv1.CockpitPlatformRow{Platform: "total",
		Committed: commC + commM, Deposited: crow.Deposited + mrow.Deposited,
		Parked: crow.Parked + mrow.Parked, Exposure: crow.Exposure + mrow.Exposure}
	r.Money.Platforms = append(r.Money.Platforms, tot)
	// ①敞口分母=跨平台已入合计（D-0038 帽按全平台本金非 crypto 单口径——
	// 驳回修①：MT5 equity 3.2K 入分母防假红）。
	r.Money.ExposureLimit = tot.Deposited * 0.10
	if r.Money.ExposureLimit > 0 {
		switch ratio := tot.Exposure / r.Money.ExposureLimit; {
		case ratio > 1.0:
			r.Money.ExposureState = 2
		case ratio >= 0.9:
			r.Money.ExposureState = 1
		default:
			r.Money.ExposureState = 0
		}
	}
	now := time.Now()
	r.AsOf = tsProto(now)

	// ③ 工单板：ops_ticket_state 派生态（闭环只看近 7d）
	tr, err := c.pool.Query(ctx,
		`SELECT ticket,event,ts,seat FROM ops_ticket_state ORDER BY ts DESC`)
	if err != nil {
		r.Missing = append(r.Missing, "tickets")
	}
	if err == nil {
		defer tr.Close()
		for tr.Next() {
			var tk dashboardv1.CockpitTicket
			var ts *time.Time
			if tr.Scan(&tk.Ticket, &tk.State, &ts, &tk.Seat) != nil {
				continue
			}
			switch tk.State {
			case "领单", "开工", "收口", "移交":
				tk.State = "在飞"
			case "挂起":
				tk.State = "挂起"
			default:
				if ts != nil && time.Since(*ts) > 7*24*time.Hour {
					continue // 闭环超 7d 出局（近7d闭环计数面）
				}
				tk.State = "闭环"
			}
			if ts != nil {
				tk.Ts = tsProto(*ts)
			}
			r.Tickets = append(r.Tickets, &tk)
		}
	}

	// ④ 待开仓：armed 提案+TTL 倒计时（24h 默认同 lifecycle 口径）
	ar, err := c.pool.Query(ctx,
		`SELECT id,inst_id,direction,created_at,
			COALESCE((SELECT value::float FROM app_config WHERE key='lifecycle.armed_ttl_h'),24)-EXTRACT(EPOCH FROM now()-created_at)/3600.0
		 FROM fin_proposal WHERE state='fin_armed' ORDER BY created_at`)
	if err != nil {
		r.Missing = append(r.Missing, "armed")
	}
	if err == nil {
		defer ar.Close()
		for ar.Next() {
			var a dashboardv1.CockpitArmed
			var t time.Time
			if ar.Scan(&a.Id, &a.InstId, &a.Direction, &t, &a.TtlHoursLeft) == nil {
				a.Created = tsProto(t)
				r.Armed = append(r.Armed, &a)
			}
		}
	}
	return r, nil
}

// containsStr 名段判存（missing 幂等件）。
func containsStr(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
