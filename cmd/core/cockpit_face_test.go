package main

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
)

// T-0239 验收锚：五块各块渲染数据非空（mock 行注入断言）。
func TestGetCockpitBlocks(t *testing.T) {
	ctx := context.Background()
	c := coreService{pool: testutil.Pool(t)}
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
}
