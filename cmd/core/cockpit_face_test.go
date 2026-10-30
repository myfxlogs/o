package main

import (
	"context"
	"fmt"
	"testing"

	"o/internal/gen/dashboardv1"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0239 验收锚：五块各块渲染数据非空（mock 行注入断言）。
func TestGetCockpitBlocks(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(context.Background(), pool, "md"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := coreService{pool: pool}
	// 桩数据：ops_event 一行 + armed 提案一行（fin_opp_eval+fin_proposal 夹具同族）
	if _, err := c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('席桩','领单','T-9999','桩')`); err != nil {
		t.Fatal("ops_event 桩:", err)
	}
	var eid int64
	if err := c.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('STUB','funding-hedge','short_perp_long_spot','proposed','[]','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal("eval 桩:", err)
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,state,net_apr)
		VALUES($1,'STUB','short_perp_long_spot','fin_armed',0)`, eid); err != nil {
		t.Fatal("提案桩:", err)
	}
	r, err := c.GetCockpit(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Seats) == 0 || r.Seats[0].Stage != "施工" {
		t.Fatalf("存活/阶段面空或派生错: %+v", r.Seats)
	}
	if r.Money == nil {
		t.Fatal("钱面空")
	}
	if len(r.Tickets) == 0 {
		t.Fatal("工单板空")
	}
	if len(r.Armed) == 0 {
		t.Fatal("待开仓空")
	}
	// 敞口段：未平仓 100usd + deposit 1000 → 上限 100 顶格黄（ratio=1.0→红）
	var xid int64
	if err := c.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES(0,0,gen_random_uuid(),'STUB',120,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatal("执行桩:", err)
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES($1,0,'STUB',120)`, xid); err != nil {
		t.Fatal("仓桩:", err)
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at,class)
		VALUES('deposit','okx','USDT',1000,NULL,0,'manual',now(),'equity')`); err != nil {
		t.Fatal("金桩:", err)
	}
	r2, err := c.GetCockpit(ctx, nil)
	if err != nil || r2.Money == nil {
		t.Fatal(err)
	}
	if r2.Money.Exposure != 120 || r2.Money.ExposureLimit != 100 {
		t.Fatalf("敞口/上限对拍错: expo=%.1f lim=%.1f", r2.Money.Exposure, r2.Money.ExposureLimit)
	}
	if r2.Money.ExposureState != 2 {
		t.Fatalf("越限红判态错: %d", r2.Money.ExposureState)
	}
	// 平台分列：crypto+MT5+total 三行在+合计=crypto（mt5Live 未注=0 不冒）
	if len(r2.Money.Platforms) != 3 {
		t.Fatalf("平台行数错: %d", len(r2.Money.Platforms))
	}
	var cr, tot *dashboardv1.CockpitPlatformRow
	for _, pr := range r2.Money.Platforms {
		if pr.Platform == "crypto" {
			cr = pr
		}
		if pr.Platform == "total" {
			tot = pr
		}
	}
	if cr == nil || tot == nil || cr.Exposure != 120 || tot.Exposure != 120 || tot.Deposited != 1000 {
		t.Fatalf("平台行数对拍错: %+v %+v", cr, tot)
	}
	if r2.AsOf == nil {
		t.Fatal("as_of 戳缺")
	}
	// missing 段：拆 ops_event 视图语义拔径——表在所以不置 missing（断言面=零冒缺表）
	for _, m := range r2.Missing {
		if m == "seats" || m == "armed" {
			t.Fatalf("不该 missing: %v", r2.Missing)
		}
	}
	// T-0267 件A 成本列：ops_ticket_usage 两行聚合（含一行 NULL=不可得账）
	// →工单板 total_tokens=实测行合计+usage_rows=账行数（NULL 行照计行数）。
	if _, err := c.pool.Exec(ctx, `INSERT INTO ops_ticket_usage(ticket,kind,model,prompt_tokens,completion_tokens,cached_tokens)
		VALUES('T-9999','run','m',100,20,5),('T-9999','audit','m',NULL,NULL,NULL)`); err != nil {
		t.Fatal("用量桩:", err)
	}
	r3, err := c.GetCockpit(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tk := range r3.Tickets {
		if tk.Ticket != "T-9999" {
			continue
		}
		found = true
		if tk.TotalTokens != 120 || tk.UsageRows != 2 {
			t.Fatalf("用量聚合错: toks=%d rows=%d", tk.TotalTokens, tk.UsageRows)
		}
	}
	if !found {
		t.Fatal("T-9999 工单行缺——聚合 JOIN 断径")
	}
}

// T-0262 验收锚①②：工单板每页≤10+跨页不重不漏+稳定序（ts DESC,ticket DESC）对拍。
func TestGetCockpitTicketPaging(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := coreService{pool: pool}
	const n = 25
	for i := 0; i < n; i++ {
		if _, err := pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES('席桩','领单',$1,'桩',now()-$2*interval '1 second')`,
			fmt.Sprintf("T-%04d", 9000+i), n-i); err != nil {
			t.Fatal("工单桩:", err)
		}
	}
	// 同戳对：决胜键 ticket DESC 实证（同 ts 序仍确定）
	for _, tk := range []string{"T-8ZZZ", "T-8AAA"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES('席桩','领单',$1,'桩','2026-10-30 00:00:00+00')`, tk); err != nil {
			t.Fatal("同戳桩:", err)
		}
	}
	get := func(off int32) *dashboardv1.GetCockpitResponse {
		r, err := c.GetCockpit(ctx, &dashboardv1.GetCockpitRequest{TicketLimit: 10, TicketOffset: off})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	// ① 每页≤10+total=27（25 递戳+2 同戳），跨页收集判重
	var all []*dashboardv1.CockpitTicket
	seen := map[string]bool{}
	for off := int32(0); ; off += 10 {
		r := get(off)
		if r.TicketsTotal != 27 {
			t.Fatalf("tickets_total 错: %d", r.TicketsTotal)
		}
		if len(r.Tickets) > 10 {
			t.Fatalf("页条目超10: %d", len(r.Tickets))
		}
		for _, tk := range r.Tickets {
			if seen[tk.Ticket] {
				t.Fatalf("跨页重条: %s", tk.Ticket)
			}
			seen[tk.Ticket] = true
		}
		all = append(all, r.Tickets...)
		if len(r.Tickets) < 10 {
			break
		}
	}
	if len(all) != 27 {
		t.Fatalf("跨页漏条: %d/27", len(all))
	}
	// ② 稳定序：相邻对满足 (ts DESC, ticket DESC)
	for i := 1; i < len(all); i++ {
		a, b := all[i-1], all[i]
		at, bt := a.Ts.AsTime(), b.Ts.AsTime()
		if at.Before(bt) || (at.Equal(bt) && a.Ticket < b.Ticket) {
			t.Fatalf("序漂移: %s@%v 后随 %s@%v", a.Ticket, at, b.Ticket, bt)
		}
	}
	// 序锚+决胜键实证：最新=T-9024，同戳尾二=T-8ZZZ,T-8AAA（ticket DESC）
	if all[0].Ticket != "T-9024" || all[25].Ticket != "T-8ZZZ" || all[26].Ticket != "T-8AAA" {
		t.Fatalf("序锚错: 首=%s 尾二=%s,%s", all[0].Ticket, all[25].Ticket, all[26].Ticket)
	}
}

// T-0241 三律③拔径桩：视图缺席→missing 名段+其余块不受染。
func TestGetCockpitMissingDegrade(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := coreService{pool: pool}
	if _, err := pool.Exec(ctx, "DROP VIEW IF EXISTS ops_seat_latest CASCADE"); err != nil {
		t.Fatal("拆视图:", err)
	}
	r, err := c.GetCockpit(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range r.Missing {
		if m == "seats" {
			found = true
		}
	}
	if !found {
		t.Fatalf("拔 seats 径未显缺: %v", r.Missing)
	}
}
