package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 升件环：死→恰一升件（24h 同窗去重）；复绿→恢复行携升件#id 自闭。
func TestFeedEscLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='t_dead' OR note LIKE '%t_dead%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	svc.escalate(ctx, "断流", "t_dead", "面断流 t_dead：零行——超阈。t_fix")
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='断流' AND esc_scope='t_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件 n=%d err=%v", n, err)
	}
	svc.escalate(ctx, "断流", "t_dead", "重报") // 同窗重报不暴
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='t_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("去重 n=%d err=%v", n, err)
	}
	svc.recoverNote(ctx, "断流", "t_dead", time.Now().Add(-10*time.Minute))
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE seat='探测席' AND event='恢复' AND note LIKE '%t_dead%'`).Scan(&note); err != nil {
		t.Fatalf("恢复行缺: %v", err)
	}
	var id int64
	_ = pool.QueryRow(ctx, `SELECT id FROM ops_event WHERE esc_scope='t_dead'`).Scan(&id)
	if want := "升件#" + strconv.FormatInt(id, 10); !strings.Contains(note, want) {
		t.Fatalf("恢复行未携升件引: %s（望含 %s）", note, want)
	}
}

// reconEscCheck 歧态腿探：腿在案→升件；未销在案→不暴；销后归零→恢复行；
// 新批腿→未销才抑放行重升（同窗 dedup 不误吞新事证）。
func TestReconEscAmbiguousLegs(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_order_leg WHERE cl_ord_id LIKE 't-rec%';
		DELETE FROM fin_execution WHERE inst_id='T-REC-USDT';
		DELETE FROM ops_event WHERE esc_scope='ambiguous_legs'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES(NULL,'T-REC-USDT',10,'open') RETURNING id`).Scan(&eid); err != nil {
		t.Fatalf("exec: %v", err)
	}
	var legID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,'t-rec-1','okx','buy','T-REC-USDT',1,'ambiguous') RETURNING id`, eid).Scan(&legID); err != nil {
		t.Fatalf("leg: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}

	svc.reconEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='对账' AND esc_scope='ambiguous_legs'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件 n=%d err=%v", n, err)
	}
	var escID int64
	var note string
	if err := pool.QueryRow(ctx, `SELECT id,note FROM ops_event
		WHERE esc_class='对账' AND esc_scope='ambiguous_legs'`).Scan(&escID, &note); err != nil {
		t.Fatalf("升件行: %v", err)
	}
	if !strings.Contains(note, "歧态腿 1 条") {
		t.Fatalf("note 缺实数: %s", note)
	}
	svc.reconEscCheck(ctx, deadSince) // 未销在案不暴
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='ambiguous_legs'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("未销抑 n=%d", n)
	}

	// 腿裁掉→恢复行携升件#id 自闭
	if _, err := pool.Exec(ctx, `UPDATE fin_order_leg SET state='failed' WHERE id=$1`, legID); err != nil {
		t.Fatalf("裁腿: %v", err)
	}
	svc.reconEscCheck(ctx, deadSince)
	var rec string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE seat='探测席' AND event='恢复' AND note LIKE '%ambiguous_legs%'`).Scan(&rec); err != nil {
		t.Fatalf("恢复行缺: %v", err)
	}
	if want := "升件#" + strconv.FormatInt(escID, 10); !strings.Contains(rec, want) {
		t.Fatalf("恢复行未携引: %s", rec)
	}

	// 新批腿→已销不抑,重升
	if _, err := pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,'t-rec-2','okx','sell','T-REC-USDT',1,'ambiguous')`, eid); err != nil {
		t.Fatalf("新腿: %v", err)
	}
	svc.reconEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='ambiguous_legs'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("新批重升 n=%d", n)
	}
}

// orphanEscCheck：pending 积压超 TTL→升件 orphan_*；积压销→恢复自闭。
func TestOrphanEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_eng_lesson WHERE signature='t-orph';
		DELETE FROM ops_event WHERE esc_scope='orphan_eng_lesson'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_eng_lesson(signature,cluster_n,verify_state,numbers,status,verdict_ref,ts)
		VALUES('t-orph',1,'ai_evaluated','{}','pending','',now()-interval '2 days')`); err != nil {
		t.Fatalf("lesson: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	svc.orphanEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='孤儿账' AND esc_scope='orphan_eng_lesson'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件 n=%d err=%v", n, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_eng_lesson SET status='verdicted',
		verdict_ref='弃——t 销' WHERE signature='t-orph'`); err != nil {
		t.Fatalf("销: %v", err)
	}
	svc.orphanEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%orphan_eng_lesson%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("恢复 n=%d", n)
	}
}

// cognitionEscCheck：未决升件在案+处置行缺席>24h→唤知面死升件；
// 处置行落地→恢复自闭（alerter nil=人推安全跳过）。
func TestCognitionEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope IN ('t_open','cognition_dead') OR event='处置'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','','t 未决','断流','t_open')`); err != nil {
		t.Fatalf("esc: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	svc.cognitionEscCheck(ctx, deadSince, nil)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='唤知' AND esc_scope='cognition_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("唤知死升件 n=%d err=%v", n, err)
	}
	// 处置行落地→唤知面复绿（t_open 未决仍在,恢复语义只看处置行新鲜度）
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席','处置','','t 处置行落地')`); err != nil {
		t.Fatalf("处置: %v", err)
	}
	svc.cognitionEscCheck(ctx, deadSince, nil)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%cognition_dead%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("恢复 n=%d", n)
	}
}

// dataGapEscCheck：unevaluable 同真因≥3 持续→升件 data_gap_*；真因消→恢复。
func TestDataGapEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_opp_eval WHERE inst_id LIKE 'T-GAP%';
		DELETE FROM ops_event WHERE esc_scope='data_gap_t_missing'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
			VALUES($1,'t_fam','gap_carry','unevaluable','[]',
				('{"reason":"t_missing"}')::jsonb)`,
			fmt.Sprintf("T-GAP-%d", i)); err != nil {
			t.Fatalf("eval%d: %v", i, err)
		}
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	svc.dataGapEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='数据缺' AND esc_scope='data_gap_t_missing'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件 n=%d err=%v", n, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_opp_eval WHERE inst_id LIKE 'T-GAP%'`); err != nil {
		t.Fatalf("消: %v", err)
	}
	svc.dataGapEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%data_gap_t_missing%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("恢复 n=%d", n)
	}
}

// escEdge 沿触发：非死→死恰一报,持续死不重报,复绿自闭。
func TestEscEdge(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE esc_scope='t_edge'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	ds := map[string]time.Time{}
	fire := func(d bool) {
		svc.escEdge(ctx, ds, "资金闸", "t_edge", d, func() string { return "t_edge note" })
	}
	fire(false)
	fire(true)
	fire(true) // 持续死不重报
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_scope='t_edge'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("沿触发 n=%d err=%v", n, err)
	}
	fire(false) // 复绿自闭
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%t_edge%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("自闭 n=%d err=%v", n, err)
	}
}

// commercialEscCheck：live 在册而策略面 30d 净产出≤0→升件 commercial_stall；
// 净产出回正→恢复自闭；无 live 在册=静默（空转定性归唤知会话非探针宣判）。
func TestCommercialEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_cashflow WHERE source_ref BETWEEN 990001 AND 990099;
		DELETE FROM fin_strategy WHERE inst_id='T-STALL-USDT';
		DELETE FROM ops_event WHERE esc_scope='commercial_stall'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	ref := 990000
	ins := func(kind string, amt float64) {
		ref++
		if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow(venue,kind,amount,ccy,occurred_at,source_table,source_ref)
			VALUES('okx',$1,$2,'USD',now()-interval '2 days','fin_order_leg',$3)`,
			kind, amt, ref); err != nil {
			t.Fatalf("cf %s: %v", kind, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,entered_at,criteria,created_at,venue)
		VALUES('T-STALL-USDT','funding-hedge','live',50,now(),'{}',now(),'okx')`); err != nil {
		t.Fatalf("strategy: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}

	// 净付型面:fee -3 + funding +0.01=-2.99≤0→升件
	ins("fee", -3.0)
	ins("funding_settle", 0.01)
	svc.commercialEscCheck(ctx, deadSince)
	var n int
	var note string
	if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(max(note),'') FROM ops_event
		WHERE event='升件' AND esc_class='商业产出' AND esc_scope='commercial_stall'`).Scan(&n, &note); err != nil || n != 1 {
		t.Fatalf("stall 升件 n=%d err=%v", n, err)
	}
	if !strings.Contains(note, "-2.9900") {
		t.Fatalf("note 缺实数: %s", note)
	}
	svc.commercialEscCheck(ctx, deadSince) // 未销在案不暴
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='commercial_stall'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("未销抑 n=%d", n)
	}

	// interest 不入账:加 +10 interest 仍≤0→不恢复
	ins("interest", 10.0)
	svc.commercialEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND esc_scope='commercial_stall'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("interest 冒充恢复 n=%d", n)
	}

	// 策略面真回正(trade_pnl +5)→恢复自闭
	ins("trade_pnl", 5.0)
	svc.commercialEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%commercial_stall%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("恢复 n=%d", n)
	}

	// live 退役→无 live 在册=静默不重升（即便面负）
	if _, err := pool.Exec(ctx, `UPDATE fin_strategy SET state='retired'
		WHERE inst_id='T-STALL-USDT'`); err != nil {
		t.Fatalf("retire: %v", err)
	}
	ins("fee", -10.0)
	svc.commercialEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='commercial_stall'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("无 live 静默 n=%d", n)
	}
}
