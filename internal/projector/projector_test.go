// projector_test.go —— T-0168 桩锚（O_TEST_PG_DSN 5434 独立实例）：
// 活跃集外零投影/内投影、signal 谓词、双向回写+state 列组禁投影覆盖、
// 杀连全表对账自愈、macro 全列覆写 diff、跨库切序守卫、镜像陈→sizing fail-closed。
// 双链并集前置：md 池=Migrate(md)、sg 池=Migrate(fin)（独立 schema=独立库语义）。
package projector

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"o/internal/trading/execution"
)

func runner(t *testing.T) (*Runner, *pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	md := testutil.Pool(t)
	sg := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, md, "md"); err != nil {
		t.Fatalf("migrate md: %v", err)
	}
	if err := pgstore.Migrate(ctx, sg, "fin"); err != nil {
		t.Fatalf("migrate fin: %v", err)
	}
	r, err := New(md, sg, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r, md, sg
}

// seedActive SG 侧造活跃集（live 策略 BTC；ETH 无任何证据行=集外）。
func seedActive(t *testing.T, sg *pgxpool.Pool) {
	t.Helper()
	_, err := sg.Exec(context.Background(), `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		VALUES ('BTC-USDT-SWAP','funding-hedge','live',1000)`)
	if err != nil {
		t.Fatal(err)
	}
}

func seedMdFacts(t *testing.T, md *pgxpool.Pool) {
	t.Helper()
	_, err := md.Exec(context.Background(), `INSERT INTO md_ticks(ts,venue,inst_id,last,recv_ts) VALUES
		(now(),'okx','BTC-USDT-SWAP',50000,now()),
		(now(),'okx','ETH-USDT-SWAP',3000,now())`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = md.Exec(context.Background(), `INSERT INTO md_mark_index(ts,venue,inst_id,channel,px,recv_ts) VALUES
		(now(),'okx','BTC-USDT-SWAP','mark-price',50001,now()),
		(now(),'okx','ETH-USDT-SWAP','mark-price',3001,now())`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = md.Exec(context.Background(), `INSERT INTO md_signal(kind,key,val,ts) VALUES
		('carry_diff','X-PAIR',0.004,now()),
		('xvenue_spread','Y-PAIR',0.002,now()),
		('funding_view','BTC-USDT-SWAP',0.0001,now()),
		('funding_view','ETH-USDT-SWAP',0.0002,now()),
		('basis_ann','BTC-USDT-270626',0.03,now())`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = md.Exec(context.Background(), `INSERT INTO md_dcd_product(ts,venue,surface,uly,product_id,direction,annualized_yield,expiry,raw)
		VALUES (now(),'okx','dcd','BTC-USDT','DCD-1','C',0.11,now()+interval '30 days','{}')`)
	if err != nil {
		t.Fatal(err)
	}
}

func count(t *testing.T, pool *pgxpool.Pool, q string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestActiveSetPredicate 活跃集外零投影/集内投影（ticks/mark_index/signal 三谓词形）。
func TestActiveSetPredicate(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runner(t)
	seedActive(t, sg)
	seedMdFacts(t, md)
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_ticks WHERE inst_id='BTC-USDT-SWAP'"); n != 1 {
		t.Fatalf("集内 BTC tick 应投影 1 行: %d", n)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_ticks WHERE inst_id='ETH-USDT-SWAP'"); n != 0 {
		t.Fatalf("集外 ETH tick 零投影: %d", n)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_mark_index WHERE inst_id='BTC-USDT-SWAP'"); n != 1 {
		t.Fatalf("集内 mark 行应投影: %d", n)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_mark_index WHERE inst_id='ETH-USDT-SWAP'"); n != 0 {
		t.Fatalf("集外 mark 零投影: %d", n)
	}
	// signal 谓词：carry_diff/xvenue_spread 全量；funding_view 活跃集键；basis_ann 不在面。
	if n := count(t, sg, "SELECT count(*) FROM md_signal"); n != 3 {
		t.Fatalf("signal 应 3 行（carry_diff+xvenue_spread+funding_view@BTC）: %d", n)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_signal WHERE kind='funding_view' AND key='ETH-USDT-SWAP'"); n != 0 {
		t.Fatal("集外 funding_view 零投影")
	}
	if n := count(t, sg, "SELECT count(*) FROM md_signal WHERE kind='basis_ann'"); n != 0 {
		t.Fatal("basis_ann 不在投影谓词")
	}
	// dcd 谓词：无 dcd_call 活策略→零投影；补活策略→投影。
	if n := count(t, sg, "SELECT count(*) FROM md_dcd_product"); n != 0 {
		t.Fatal("无 dcd_call 活策略应零投影")
	}
	if _, err := sg.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		VALUES ('DCD-1','dcd_call','paper',500)`); err != nil {
		t.Fatal(err)
	}
	if err := r.Reconcile(ctx, false); err != nil {
		t.Fatalf("reconcile2: %v", err)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_dcd_product WHERE product_id='DCD-1'"); n != 1 {
		t.Fatalf("dcd_call 活策略 product 行应投影: %d", n)
	}
}

// seedProposal md 正本侧造评估+提案（funding-hedge 判例形）。
func seedProposal(t *testing.T, md *pgxpool.Pool, id, evalID int64, inst string) {
	t.Helper()
	_, err := md.Exec(context.Background(), `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES ($1,now(),$2,'funding-hedge','short_perp_long_spot',0.08,'proposed','[]','{}')`, evalID, inst)
	if err != nil {
		t.Fatal(err)
	}
	_, err = md.Exec(context.Background(), `INSERT INTO fin_proposal(id,eval_id,inst_id,direction,net_apr,created_at)
		VALUES ($1,$2,$3,'short_perp_long_spot',0.08,now())`, id, evalID, inst)
	if err != nil {
		t.Fatal(err)
	}
}

// TestProposalBidirectional 正向投影→SG 回写三列→md 收敛；前向 SQL 不触 state 列组。
func TestProposalBidirectional(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runner(t)
	seedProposal(t, md, 1, 100, "SOL-USDT-SWAP")
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n := count(t, sg, "SELECT count(*) FROM fin_proposal WHERE id=1 AND inst_id='SOL-USDT-SWAP'"); n != 1 {
		t.Fatalf("提案应投影: %d", n)
	}
	if n := count(t, sg, "SELECT count(*) FROM fin_opp_eval WHERE id=100"); n != 1 {
		t.Fatalf("引用 eval 应随件投影: %d", n)
	}
	// SG lifecycle 改态（armed——ev_proposal_sg 触发器在位由迁移保证；此处直调回写径）。
	if _, err := sg.Exec(ctx, "UPDATE fin_proposal SET state='fin_armed' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err := r.repairProposalStates(ctx); err != nil {
		t.Fatalf("回写: %v", err)
	}
	var state string
	if err := md.QueryRow(ctx, "SELECT state FROM fin_proposal WHERE id=1").Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "fin_armed" {
		t.Fatalf("SG 改态应回写 md: %s", state)
	}
	// state 列组归属断言：全量对账重跑（补投影径）后——SG armed 不被任何
	// 正向径覆盖、md armed 不被前向径改写（前向 SQL 无 state 列组——构造性）。
	// 注：同 id 前向重投影在产线不存在（提案 insert-only+SG dedupe 闸），
	// 补投影仅对 SG 缺行 id 生效。
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatal(err)
	}
	var sgState, mdState string
	if err := sg.QueryRow(ctx, "SELECT state FROM fin_proposal WHERE id=1").Scan(&sgState); err != nil {
		t.Fatal(err)
	}
	if err := md.QueryRow(ctx, "SELECT state FROM fin_proposal WHERE id=1").Scan(&mdState); err != nil {
		t.Fatal(err)
	}
	if sgState != "fin_armed" || mdState != "fin_armed" {
		t.Fatalf("对账重跑后 state 列组应各自持正本: sg=%s md=%s", sgState, mdState)
	}
	// 退役三列回写（retire_reason/retired_at——D-0030 state 列组全组）。
	if _, err := sg.Exec(ctx, `UPDATE fin_proposal SET state='fin_retired',
		retire_reason='armed_ttl_expired', retired_at=now() WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err := r.repairProposalStates(ctx); err != nil {
		t.Fatal(err)
	}
	var reason *string
	var at *time.Time
	if err := md.QueryRow(ctx, "SELECT state,retire_reason,retired_at FROM fin_proposal WHERE id=1").
		Scan(&state, &reason, &at); err != nil {
		t.Fatal(err)
	}
	if state != "fin_retired" || reason == nil || *reason != "armed_ttl_expired" || at == nil {
		t.Fatalf("退役三列回写未达: %s %v %v", state, reason, at)
	}
}

// TestReconcileSelfHeal 杀连自愈：镜像行丢失+断窗新提案→全表对账集齐（验收锚）。
func TestReconcileSelfHeal(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runner(t)
	seedActive(t, sg)
	seedMdFacts(t, md)
	seedProposal(t, md, 1, 100, "AVAX-USDT-SWAP")
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	// 「杀连丢镜像」：proposal/eval/tick 行清掉+断窗内 md 新提案（异 inst 避 dedupe）。
	for _, q := range []string{
		"DELETE FROM fin_proposal", "DELETE FROM fin_opp_eval",
		"DELETE FROM md_ticks WHERE inst_id='BTC-USDT-SWAP'",
	} {
		if _, err := sg.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	seedProposal(t, md, 2, 101, "DOT-USDT-SWAP")
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile2: %v", err)
	}
	if n := count(t, sg, "SELECT count(*) FROM fin_proposal WHERE id IN (1,2)"); n != 2 {
		t.Fatalf("杀连丢行+断窗新提案应全数补齐: %d", n)
	}
	if n := count(t, sg, "SELECT count(*) FROM fin_opp_eval WHERE id IN (100,101)"); n != 2 {
		t.Fatalf("引用 eval 应补齐: %d", n)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_ticks WHERE inst_id='BTC-USDT-SWAP'"); n != 1 {
		t.Fatalf("ts 面重刷应回填: %d", n)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_ticks WHERE inst_id='ETH-USDT-SWAP'"); n != 0 {
		t.Fatal("自愈不得越活跃集谓词")
	}
}

// TestMacroOverwriteDiff macro 无 ts 全列覆写 diff（变行才写）。
func TestMacroOverwriteDiff(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runner(t)
	if _, err := md.Exec(ctx, `INSERT INTO md_macro_event(event_date,kind,title) VALUES
		('2027-06-01','fomc','FOMC 特别会')`); err != nil {
		t.Fatal(err)
	}
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatal(err)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_macro_event WHERE event_date='2027-06-01'"); n != 1 {
		t.Fatalf("macro 应投影: %d", n)
	}
	if _, err := md.Exec(ctx, `UPDATE md_macro_event SET title='FOMC 利率决议（改期）' WHERE event_date='2027-06-01'`); err != nil {
		t.Fatal(err)
	}
	if err := r.Reconcile(ctx, false); err != nil {
		t.Fatal(err)
	}
	var title string
	if err := sg.QueryRow(ctx, "SELECT title FROM md_macro_event WHERE event_date='2027-06-01'").Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "FOMC 利率决议（改期）" {
		t.Fatalf("macro 变行应覆写: %s", title)
	}
}

// TestGuardSequenceCollision 跨库 id 键空间切序守卫（T-0169 setval 前置——
// md 新序列撞 SG legacy id=镜像覆写 legacy 行实害）。
func TestGuardSequenceCollision(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runner(t)
	_ = sg

	seedProposal(t, md, 1, 100, "LINK-USDT-SWAP")
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatal(err)
	}
	// md 序列 last_value=1（显式 id 插入不推进序）≤ sg max=1 → 守卫红。
	if err := r.guards(ctx); err == nil {
		t.Fatal("切序前置未满足应拒（md next ≤ SG legacy max）")
	}
	// setval 抬过 legacy max → 守卫过（T-0169 切换径同形）。
	if _, err := md.Exec(ctx, "SELECT setval('fin_proposal_id_seq', 1000000, false)"); err != nil {
		t.Fatal(err)
	}
	if _, err := md.Exec(ctx, "SELECT setval('fin_opp_eval_id_seq', 1000000, false)"); err != nil {
		t.Fatal(err)
	}
	if err := r.guards(ctx); err != nil {
		t.Fatalf("setval 后守卫应过: %v", err)
	}
}

// TestMirrorStaleSizingFailClosed 镜像陈→sizing 陈闸拒（T-0128 闸复用——
// projector 停=ts 停进=60s 阈即拒开仓，D-0030 §七失效语义实证）。
func TestMirrorStaleSizingFailClosed(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runner(t)
	seedActive(t, sg)
	if err := r.refreshSets(ctx); err != nil {
		t.Fatal(err)
	}
	seedMdFacts(t, md)
	if err := r.projectTsTable(ctx, "ev_ticks"); err != nil {
		t.Fatal(err)
	}
	if _, err := execution.LatestPx(ctx, sg, "BTC-USDT-SWAP"); err != nil {
		t.Fatalf("鲜镜像应可取价: %v", err)
	}
	if _, err := sg.Exec(ctx, "UPDATE md_ticks SET ts=now()-interval '120 seconds' WHERE inst_id='BTC-USDT-SWAP'"); err != nil {
		t.Fatal(err)
	}
	if _, err := execution.LatestPx(ctx, sg, "BTC-USDT-SWAP"); err == nil {
		t.Fatal("镜像 ts 陈>60s 应拒（sizing fail-closed）")
	}
}
