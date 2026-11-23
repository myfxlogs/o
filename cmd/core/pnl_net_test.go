// pnl_net_test.go —— 工单#34619 回归夹具：fin_pnl_net 单源视图判别面。
// 「无取得腿的清仓卖出」→净产出不变；「带仓现货买卖」→照计净差；
// 「误记仓腿」（decision 源 NULL 腿↔同币种在册反向腿 ±120s）→配对
// 计入保两腿同口径；native 币域腿（ccy∉USDT/USD）→域外不混算。
package main

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestFinPnlNetCaliber(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	// 桩腿（source_ref 任意——NOT NULL 即可；position_ref 无 FK 约束面）
	legs := []struct {
		kind   string
		ccy    string
		amount float64
		pos    any
	}{
		{"trade_pnl", "USDT", 5, nil},         // 真损益——计
		{"spot_sell", "USDT", 9999, nil},      // 无取得腿清仓卖——排除（幻影病灶形）
		{"spot_buy", "USDT", -100, 777},       // 带仓买腿——计
		{"spot_sell", "USDT", 90, 777},        // 带仓卖腿——计（净差 -10）
		{"spot_sell", "BTC", -0.5, nil},       // native 域腿——ccy 闸外
		{"spot_sell", "BTC", -0.5, 777},       // native 域带仓腿——ccy 闸外
		{"fee", "USDT", -1, nil},              // 费腿——计
		{"recon_adjust", "USDT", -10000, nil}, // 非损益 kind——天然不入
	}
	for i, l := range legs {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow
			(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
			VALUES($1,'okx',$2,$3,$4,$5,'test',$6)`,
			l.kind, l.ccy, l.amount, l.pos, 9000+i, time.Now()); err != nil {
			t.Fatalf("腿%d %s: %v", i, l.kind, err)
		}
	}
	// VVV 形配对夹具：residual_clear decision 清扫新开仓腿币——
	// 仓86 VVV-USDT 买腿（position_ref 在册）+ decision 源 NULL 卖腿。
	var decID, execID, legID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_risk_decision
		(intent,decision,reason,config_gen,kind)
		VALUES('{"op":"residual_clear","ccy":"VVV","venue":"okx"}'::jsonb,
		       'allow','allow',1,'open') RETURNING id`).Scan(&decID); err != nil {
		t.Fatal("decision 桩:", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution
		(inst_id,notional_usd,state) VALUES('VVV-USDT',50,'legged')
		RETURNING id`).Scan(&execID); err != nil {
		t.Fatal("execution 桩:", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_order_leg
		(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,'cl-test-vvv','okx','buy','VVV-USDT',1.9,'filled')
		RETURNING id`, execID).Scan(&legID); err != nil {
		t.Fatal("order_leg 桩:", err)
	}
	// 在册买腿 -50（position_ref=888）+ 误记 NULL 卖腿 +48（decision 源）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow
		(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('spot_buy','okx','USDT',-50,888,$1,'fin_order_leg',now())`, legID); err != nil {
		t.Fatal("配对买腿:", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow
		(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('spot_sell','okx','USDT',48,NULL,$1,'fin_risk_decision',now())`, decID); err != nil {
		t.Fatal("配对卖腿:", err)
	}
	// 反例：decision 源 NULL 腿但无同币种仓腿——仍排除（ETC/TRX 形）
	var decID2 int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_risk_decision
		(intent,decision,reason,config_gen,kind)
		VALUES('{"op":"residual_clear","ccy":"ETC","venue":"okx"}'::jsonb,
		       'allow','allow',1,'open') RETURNING id`).Scan(&decID2); err != nil {
		t.Fatal("decision 桩2:", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow
		(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('spot_sell','okx','USDT',777,NULL,$1,'fin_risk_decision',now())`, decID2); err != nil {
		t.Fatal("无配对清仓卖:", err)
	}
	var net float64
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(sum(amount),0) FROM fin_pnl_net
		 WHERE occurred_at>now()-interval '1 hour'`).Scan(&net); err != nil {
		t.Fatal("视图求和:", err)
	}
	// 预期=5(trade_pnl) + (-100+90)(带仓现货净差) + (-1)(fee)
	//      + (-50+48)(VVV 配对两腿同计) = -8
	// 幻影 +9999/+777 无配对清仓卖、BTC 域腿全排除。
	if net != -8 {
		t.Fatalf("fin_pnl_net 净=%v want -8（幻影腿/native腿混入或配对腿误拆）", net)
	}
	// ops_state 重建面在场（commercial_stall 臂改吃视图——迁移链自证）
	var hasView bool
	if err := pool.QueryRow(ctx, `SELECT count(*)>0 FROM pg_views
		WHERE viewname='fin_pnl_net'`).Scan(&hasView); err != nil || !hasView {
		t.Fatalf("fin_pnl_net 视图缺席 err=%v", err)
	}
}
