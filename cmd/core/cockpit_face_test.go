package main

import (
	"context"
	"testing"

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
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES(0,0,'STUB',100)`); err != nil {
		t.Fatal("仓桩:", err)
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,class)
		VALUES('deposit','okx','USDT',1000,'equity')`); err != nil {
		t.Fatal("金桩:", err)
	}
	r2, err := c.GetCockpit(ctx, nil)
	if err != nil || r2.Money == nil {
		t.Fatal(err)
	}
	if r2.Money.Exposure != 100 || r2.Money.ExposureLimit != 100 {
		t.Fatalf("敞口/上限对拍错: expo=%.1f lim=%.1f", r2.Money.Exposure, r2.Money.ExposureLimit)
	}
	if r2.Money.ExposureState != 2 {
		t.Fatalf("越限红判态错: %d", r2.Money.ExposureState)
	}
	// missing 段：拆 ops_event 视图语义拔径——表在所以不置 missing（断言面=零冒缺表）
	for _, m := range r2.Missing {
		if m == "seats" || m == "armed" {
			t.Fatalf("不该 missing: %v", r2.Missing)
		}
	}
}
