package earn

// O-40829 意向豁免桩测锚（spec 验收①②③④⑤+反证）：
// ①venue 负债真值>0→双段零动钱；②在途/持仓/借币负债在案→双段让位；
// ③他 venue 持仓不连坐（MT5 仓抵押物不在 okx 户）；④venue 读败=fail-closed
// 不扫；⑤总闸 false=旧行为全复；反证=判据全阴性时闲钱照常被扫（闸不死）。

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// intentKeys 双闲置+阈下常态键（idleFund+idle 双源俱超阈——双段全罩判据）。
func intentKeys() map[string]string {
	return map[string]string{"earn.sweep_min_usd": "100"}
}

// attachBothIdle 双段闲余俱超阈（funding 500+trading 150——任一段动钱即现形）。
func attachBothIdle(w *Worker) {
	w.AttachIdleFunding(func(context.Context) (float64, error) { return 500, nil })
	w.AttachIdle(func(context.Context) (float64, error) { return 150, nil })
}

// noMoves 断言本拍零动钱（purchases+transfers 双面）。
func noMoves(t *testing.T, fake *fakeAPI, tag string) {
	t.Helper()
	if len(fake.purchases) != 0 || len(fake.transfers) != 0 {
		t.Fatalf("%s：意向豁免应零动钱 purchases=%v transfers=%v", tag, fake.purchases, fake.transfers)
	}
}

// TestIntentExemptVenueLiab ①venue 负债真值（liab stub=true=担保钱在仓）
// →sweepIdle 双段全让位零动钱+计数门。
func TestIntentExemptVenueLiab(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, intentKeys())
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	attachBothIdle(w)
	w.AttachLiabCheck(func(context.Context) (bool, error) { return true, nil })
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	noMoves(t, fake, "venue 负债在仓")
	if w.rej.Snapshot()["earn/sweep_intent_exempt"] == 0 {
		t.Fatal("sweep_intent_exempt 计数门未计")
	}
}

// TestIntentExemptLiabReadFail ④venue 负债读败=fail-closed 不扫（不虚构
// 「没有用途」——读败按有用途豁免）。
func TestIntentExemptLiabReadFail(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, intentKeys())
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	attachBothIdle(w)
	w.AttachLiabCheck(func(context.Context) (bool, error) { return false, fmt.Errorf("venue 读败假件") })
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	noMoves(t, fake, "venue 读败")
}

// TestIntentExemptMarginLoan ③本 venue fin_margin_loan open 在册→让位；
// 他 venue 同名负债不连坐（venue 域限定判据）。
func TestIntentExemptMarginLoan(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, intentKeys())
	defer w.pool.Exec(ctx, "DELETE FROM fin_margin_loan WHERE venue=$1", w.venue)
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	attachBothIdle(w)
	w.AttachLiabCheck(func(context.Context) (bool, error) { return false, nil })
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr) VALUES($1,'TST','TST-USDT',100,0.5)`,
		w.venue); err != nil {
		t.Fatal(err)
	}
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	noMoves(t, fake, "margin 借币负债在册")
}

// TestIntentExemptMarginLoanOtherVenue 他 venue 借币负债不连坐本户扫面。
func TestIntentExemptMarginLoanOtherVenue(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, intentKeys())
	defer w.pool.Exec(ctx, "DELETE FROM fin_margin_loan WHERE venue='otherv'")
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	defer w.pool.Exec(ctx, "DELETE FROM fin_order_leg WHERE cl_ord_id LIKE 'INTENT-OTHER-%'")
	defer w.pool.Exec(ctx, "DELETE FROM fin_position WHERE inst_id='INTENT-OTHER-USDT'")
	defer w.pool.Exec(ctx, "DELETE FROM fin_execution WHERE inst_id='INTENT-OTHER-USDT'")
	attachBothIdle(w)
	w.AttachLiabCheck(func(context.Context) (bool, error) { return false, nil })
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr) VALUES('otherv','TST','TST-USDT',100,0.5)`); err != nil {
		t.Fatal(err)
	}
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	// funding 段先动——500 直购一笔即证放行（交易户段下一拍再扫）。
	if len(fake.purchases) != 1 {
		t.Fatalf("他 venue 负债不该连坐：purchases=%v", fake.purchases)
	}
}

// TestIntentExemptPositionOpen ②本 venue 在持仓（腿 venue=tstearn）→让位。
func TestIntentExemptPositionOpen(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, intentKeys())
	defer cleanupIntentPosTag(ctx, w.pool, "INTENT-POS")
	attachBothIdle(w)
	w.AttachLiabCheck(func(context.Context) (bool, error) { return false, nil })
	seedIntentPos(t, ctx, w, "INTENT-POS", w.venue)
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	noMoves(t, fake, "本 venue 在持仓")
}

// TestIntentExemptPositionOtherVenue ③他 venue 持仓不连坐（MT5/DCD 仓抵押物
// 不在 okx 户——腿 venue=otherv 的仓不挡本户扫面）。
func TestIntentExemptPositionOtherVenue(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, intentKeys())
	defer cleanupIntentPosTag(ctx, w.pool, "INTENT-OTHERV")
	attachBothIdle(w)
	w.AttachLiabCheck(func(context.Context) (bool, error) { return false, nil })
	seedIntentPos(t, ctx, w, "INTENT-OTHERV", "otherv")
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.purchases) != 1 {
		t.Fatalf("他 venue 持仓不该连坐：purchases=%v", fake.purchases)
	}
}

// TestIntentExemptRedeemInflight ①earn_redeem 在途窗（decision 行先于仓位
// 行落账——redeem→book 序间隙唯一信号）→让位。
func TestIntentExemptRedeemInflight(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, intentKeys())
	defer w.pool.Exec(ctx,
		`DELETE FROM fin_risk_decision WHERE intent->>'venue'=$1 AND intent->>'op'='earn_redeem'`, w.venue)
	attachBothIdle(w)
	w.AttachLiabCheck(func(context.Context) (bool, error) { return false, nil })
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		 VALUES($1,'allow','transfer',$2)`,
		fmt.Sprintf(`{"kind":"transfer","op":"earn_redeem","venue":"%s","ccy":"USDT","amt":1000}`, w.venue),
		w.cfg.Generation()); err != nil {
		t.Fatal(err)
	}
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	noMoves(t, fake, "赎回在途窗")
}

// TestIntentExemptGateOff ⑤总闸 earn.sweep_intent_exempt=false→旧行为全复
// （回滚径——判据全阳性照扫）。
func TestIntentExemptGateOff(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{
		"earn.sweep_min_usd":       "100",
		"earn.sweep_intent_exempt": "false",
		"earn.sweep_liab_check":    "true",
	})
	defer w.pool.Exec(ctx, "DELETE FROM fin_margin_loan WHERE venue=$1", w.venue)
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	attachBothIdle(w)
	w.AttachLiabCheck(func(context.Context) (bool, error) { return true, nil })
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO fin_margin_loan(venue,ccy,inst_id,amt,apr) VALUES($1,'TST','TST-USDT',100,0.5)`,
		w.venue); err != nil {
		t.Fatal(err)
	}
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	// 旧行为：funding 段 500 直购一笔即证闸关放行。
	if len(fake.purchases) != 1 {
		t.Fatalf("总闸关应旧行为全复：purchases=%v", fake.purchases)
	}
}

// TestIntentExemptCleanSweeps 反证：判据源全接线且全阴性→闲钱照常被扫
// （证明没把停泊环闸死成永停——spec 验收反证锚）。
func TestIntentExemptCleanSweeps(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, intentKeys())
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	attachBothIdle(w)
	w.AttachLiabCheck(func(context.Context) (bool, error) { return false, nil })
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.purchases) != 1 {
		t.Fatalf("判据全阴性应照扫：purchases=%v", fake.purchases)
	}
}

// seedIntentPos 造「exec(legged)+position(open)+leg(venue 参数)」在持仓链——
// exec 用 legged 态不撞 T-0462 在途旧闸（隔离判据②③单变量）。
func seedIntentPos(t *testing.T, ctx context.Context, w *Worker, tag, legVenue string) {
	t.Helper()
	var eid int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		 VALUES(1,1,NULL,$1,100,'legged') RETURNING id`, tag+"-USDT").Scan(&eid); err != nil {
		t.Fatal(err)
	}
	var pid int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,inst_id,notional_usd) VALUES($1,$2,100) RETURNING id`,
		eid, tag+"-USDT").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		 VALUES($1,1,$2,$3,'buy',$4,1,'filled')`,
		eid, "INTENT-"+tag, legVenue, tag+"-USDT"); err != nil {
		t.Fatal(err)
	}
}

// cleanupIntentPosTag 按 tag 前缀级联清 leg→position→execution（inst_id=
// tag+"-USDT"、cl_ord_id="INTENT-"+tag 同锚）。
func cleanupIntentPosTag(ctx context.Context, pool *pgxpool.Pool, tag string) {
	pool.Exec(ctx, `DELETE FROM fin_order_leg WHERE cl_ord_id=$1`, "INTENT-"+tag)
	pool.Exec(ctx, `DELETE FROM fin_position WHERE inst_id=$1`, tag+"-USDT")
	pool.Exec(ctx, `DELETE FROM fin_execution WHERE inst_id=$1`, tag+"-USDT")
}
