package simdrill

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// fin_0225 simdrill 域闸：venue='sim' 行必须挂演习仓
// （position→simdrill 策略），实仓/无仓挂 sim 物理拒。
func TestSimDomainGate(t *testing.T) {
	pool := testutil.PoolIsolated(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}

	mkpos := func(t *testing.T, tmpl string) int64 {
		var sid, eid, pid int64
		if err := pool.QueryRow(ctx,
			`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
			VALUES('GT-USDT',$1,'paper',50,'okx') RETURNING id`, tmpl).Scan(&sid); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx,
			`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
			VALUES($1,'GT-USDT',50,'open') RETURNING id`, sid).Scan(&eid); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx,
			`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
			VALUES($1,$2,'GT-USDT',50) RETURNING id`, eid, sid).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		return pid
	}
	ins := func(posID int64) error {
		_, err := pool.Exec(ctx, `INSERT INTO fin_cashflow
			(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
			VALUES('fee','sim','USDT',-0.01,$1,0,'test',now())`, posID)
		return err
	}
	mkleg := func(t *testing.T, tmpl string) int64 {
		var sid, eid, lid int64
		if err := pool.QueryRow(ctx,
			`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
			VALUES('LG-USDT',$1,'paper',50,'okx') RETURNING id`, tmpl).Scan(&sid); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx,
			`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
			VALUES($1,'LG-USDT',50,'open') RETURNING id`, sid).Scan(&eid); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx,
			`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state,px)
			VALUES($1,1,'gt-leg','okx','buy','LG-USDT',1,'filled',1) RETURNING id`, eid).Scan(&lid); err != nil {
			t.Fatal(err)
		}
		return lid
	}
	insLeg := func(legID int64) error {
		_, err := pool.Exec(ctx, `INSERT INTO fin_cashflow
			(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
			VALUES('spot_buy','sim','USDT',-1,$1,$2,'fin_order_leg',now())`, nil, legID)
		return err
	}

	simPos := mkpos(t, "simdrill:cell:dir_perp_mom@okx/short_perp#usd=50")
	if err := ins(simPos); err != nil {
		t.Fatalf("演习仓 sim 行被拒（闸误伤合法域）: %v", err)
	}
	realPos := mkpos(t, "dir_perp_mom")
	if err := ins(realPos); err == nil {
		t.Fatal("实仓 sim 行未被拒——域闸失效")
	}
	if err := ins(99999999); err == nil {
		t.Fatal("无仓 sim 行未被拒")
	}
	// 径②：未挂仓腿行——溯 leg→execution→策略。演习链放行,实链拒。
	if err := insLeg(mkleg(t, "simdrill:cell:dir_perp_mom@okx/long_spot#usd=50")); err != nil {
		t.Fatalf("演习腿链未挂仓 sim 行被拒（径②误伤——开仓径腿先于仓生）: %v", err)
	}
	if err := insLeg(mkleg(t, "grid_spot")); err == nil {
		t.Fatal("实链腿未挂仓 sim 行未被拒")
	}
	// UPDATE 补挂段：sim 腿行挂回演习仓放行,挂进实仓拒。
	var cashID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM fin_cashflow WHERE venue='sim' AND source_table='fin_order_leg' LIMIT 1`).Scan(&cashID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_cashflow SET position_ref=$1 WHERE id=$2`, simPos, cashID); err != nil {
		t.Fatalf("sim 行补挂演习仓被拒: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_cashflow SET position_ref=$1 WHERE id=$2`, realPos, cashID); err == nil {
		t.Fatal("sim 行补挂实仓未被拒——UPDATE 段闸失效")
	}
}
