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
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, sg, "md"); err != nil { // T-0214：SG 单侧库双链（earn 写 md_earn_rate 实证）——反向桥正本面须含 md 域
		t.Fatalf("migrate md: %v", err)
	}
	r, err := New(md, sg, nil, "serve")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r.tsMinPass = nil // 桩零化过拍闸——连拍召唤语义同现态（节拍=现测锚非产闸）
	return r, md, sg
}

func runnerDual(t *testing.T) (*Runner, *pgxpool.Pool, *pgxpool.Pool) {
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
	if err := pgstore.Migrate(ctx, sg, "md"); err != nil { // T-0214：SG 单侧库双链
		t.Fatalf("migrate md: %v", err)
	}
	r, err := New(md, sg, nil, "dualwrite")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r.tsMinPass = nil // 同 runner——桩零化过拍闸
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

// TestEvalNaturalKeyCrossWriter T-0211 验收锚①+②：跨写者同键异 id（宿主
// 迁移/重灌重演）——对账不崩、SG 行按 ts 新鲜赢、提案 eval_id 译 SG 发号。
func TestEvalNaturalKeyCrossWriter(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runnerDual(t)
	// SG 存 dev 时代行：同键旧 id=100、旧判词。
	if _, err := sg.Exec(ctx, `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES (100,now()-interval '1 hour','SOL-USDT-SWAP','funding-hedge','short_perp_long_spot',0.05,'rejected','[]','{}')`); err != nil {
		t.Fatal(err)
	}
	// md 新宿主同键发新 id=200+提案引用（迁期实案形）。
	seedProposal(t, md, 1, 200, "SOL-USDT-SWAP")
	if err := r.projectEvals(ctx); err != nil {
		t.Fatalf("同键异 id eval 投影应过（旧形此处撞 fin_opp_eval_key 崩）: %v", err)
	}
	var id int64
	var verdict string
	if err := sg.QueryRow(ctx, `SELECT id,verdict FROM fin_opp_eval
		WHERE inst_id='SOL-USDT-SWAP'`).Scan(&id, &verdict); err != nil {
		t.Fatal(err)
	}
	if id != 100 || verdict != "proposed" {
		t.Fatalf("SG 行应持自有 id=100 按 ts 新鲜赢（verdict 翻 proposed）: id=%d verdict=%s", id, verdict)
	}
	// 提案投影：eval_id 译 SG 发号 100（不可落 md 发号 200——FK 指 SG 行）。
	if err := r.projectNewProposals(ctx); err != nil {
		t.Fatalf("提案投影: %v", err)
	}
	var evalID int64
	if err := sg.QueryRow(ctx, "SELECT eval_id FROM fin_proposal WHERE id=1").Scan(&evalID); err != nil {
		t.Fatal(err)
	}
	if evalID != 100 {
		t.Fatalf("提案 eval_id 应译 SG 发号=100（md 发号 200 不落）: %d", evalID)
	}
	// 幂等重演：重复投影/对账不炸不漂（重启回环语义锚）。
	if err := r.projectEvals(ctx); err != nil {
		t.Fatalf("重演应幂等: %v", err)
	}
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatalf("全量对账重演应过: %v", err)
	}
	if n := count(t, sg, "SELECT count(*) FROM fin_opp_eval WHERE inst_id='SOL-USDT-SWAP'"); n != 1 {
		t.Fatalf("同键恒一行: %d", n)
	}
}

// TestEvalPKCollisionSeqRewind T-0211 验收锚③：md 发号撞 SG 异键行 id
// （重灌序列回掷场景）——pick CTE 另发号落行，PK 不炸、异键行不被覆写。
func TestEvalPKCollisionSeqRewind(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runnerDual(t)
	if _, err := sg.Exec(ctx, `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES (300,now(),'AAA-USDT-SWAP','funding-hedge','short_perp_long_spot',0.05,'rejected','[]','{}')`); err != nil {
		t.Fatal(err)
	}
	// md 重灌后发号回掷：id=300 被 SG AAA 行占用，但本行键=BBB。
	seedProposal(t, md, 1, 300, "BBB-USDT-SWAP")
	if err := r.projectNewProposals(ctx); err != nil {
		t.Fatalf("异键撞号应另发号落行（旧形 PK 违例崩）: %v", err)
	}
	var verdict string
	if err := sg.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval WHERE id=300`).Scan(&verdict); err != nil {
		t.Fatal(err)
	}
	if verdict != "rejected" {
		t.Fatalf("SG id=300 的 AAA 行不得被覆写: %s", verdict)
	}
	var newID int64
	if err := sg.QueryRow(ctx, `SELECT id FROM fin_opp_eval WHERE inst_id='BBB-USDT-SWAP'`).Scan(&newID); err != nil {
		t.Fatal(err)
	}
	if newID == 300 {
		t.Fatalf("撞号行应另发号（≠300）: %d", newID)
	}
	var evalID int64
	if err := sg.QueryRow(ctx, "SELECT eval_id FROM fin_proposal WHERE id=1").Scan(&evalID); err != nil {
		t.Fatal(err)
	}
	if evalID != newID {
		t.Fatalf("提案 eval_id 应指另发号行 %d: %d", newID, evalID)
	}
}

// TestEvalTsNewerWins T-0211 验收锚④：同键异 id 陈帧不覆鲜行
// （WHERE EXCLUDED.ts>ts 单调语义在自然键下不动）。
func TestEvalTsNewerWins(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runnerDual(t)
	if _, err := sg.Exec(ctx, `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES (100,now(),'SOL-USDT-SWAP','funding-hedge','short_perp_long_spot',0.08,'proposed','[]','{}')`); err != nil {
		t.Fatal(err)
	}
	// md 同键异 id 但 ts 更陈（乱序帧/旧宿主残帧形）。
	if _, err := md.Exec(ctx, `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES (200,now()-interval '2 hours','SOL-USDT-SWAP','funding-hedge','short_perp_long_spot',0.05,'rejected','[]','{}')`); err != nil {
		t.Fatal(err)
	}
	if err := r.projectEvals(ctx); err != nil {
		t.Fatal(err)
	}
	var verdict string
	if err := sg.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval WHERE id=100`).Scan(&verdict); err != nil {
		t.Fatal(err)
	}
	if verdict != "proposed" {
		t.Fatalf("陈帧不得覆鲜行（verdict 应仍 proposed）: %s", verdict)
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
	// md 序列 last_value=1（显式 id 插入不推进序）≤ sg max=1 →
	// T-0195 撞平自抬：md 正本域两序列钦定自抬径（重启掷骰崩环根修）。
	if err := r.guards(ctx); err != nil {
		t.Fatalf("撞平自抬后应过: %v", err)
	}
	var raised int64
	if err := md.QueryRow(ctx, "SELECT last_value FROM fin_proposal_id_seq").Scan(&raised); err != nil {
		t.Fatal(err)
	}
	if raised < 2 { // sg max=1→setval≥1001（自抬安全余量锚）
		t.Fatalf("自抬应 ≥sgMax+余量: %d", raised)
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

// TestDualwriteBypassPredicate 对拍旁写泵：无谓词全量搬运（集外行也投影——
// SG 旧行为保真，对拍口径一致）。
func TestDualwriteBypassPredicate(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runnerDual(t)
	seedActive(t, sg) // BTC 集内；ETH 集外——dualwrite 不分
	seedMdFacts(t, md)
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_ticks"); n != 2 {
		t.Fatalf("dualwrite 集外也投影（应 2 行）: %d", n)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_signal WHERE kind='basis_ann'"); n != 1 {
		t.Fatalf("dualwrite 全 kinds（basis_ann 应在）: %d", n)
	}
}

// TestConfigReplicaSync 第九桥副本：正本 diff 副本+删陈+热启帧回灌语义（行变才通知）。
func TestConfigReplicaSync(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runner(t)
	if _, err := sg.Exec(ctx, `INSERT INTO app_config(key,value) VALUES('probe.tick_age','90')`); err != nil {
		t.Fatal(err)
	}
	if err := r.syncConfig(ctx); err != nil {
		t.Fatal(err)
	}
	var v string
	if err := md.QueryRow(ctx, "SELECT value::text FROM app_config WHERE key='probe.tick_age'").Scan(&v); err != nil {
		t.Fatalf("副本应落: %v", err)
	}
	if v != "90" {
		t.Fatalf("副本值应 verbatim: %s", v)
	}
	// 正本改值→副本覆写；正本删键→副本删陈。
	if _, err := sg.Exec(ctx, `UPDATE app_config SET value='"120"' WHERE key='probe.tick_age'`); err != nil {
		t.Fatal(err)
	}
	if _, err := sg.Exec(ctx, `INSERT INTO app_config(key,value) VALUES('tmp.k','"x"')`); err != nil {
		t.Fatal(err)
	}
	if err := r.syncConfig(ctx); err != nil {
		t.Fatal(err)
	}
	if err := md.QueryRow(ctx, "SELECT value::text FROM app_config WHERE key='probe.tick_age'").Scan(&v); err != nil || v != `"`+"120"+`"` {
		t.Fatalf("副本应覆写（jsonb 文本形含引号）: %s %v", v, err)
	}
	if n := count(t, md, "SELECT count(*) FROM app_config WHERE key='tmp.k'"); n != 1 {
		t.Fatal("新键应副本")
	}
	if _, err := sg.Exec(ctx, "DELETE FROM app_config WHERE key='tmp.k'"); err != nil {
		t.Fatal(err)
	}
	if err := r.syncConfig(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, md, "SELECT count(*) FROM app_config WHERE key='tmp.k'"); n != 0 {
		t.Fatal("正本删键副本应删陈")
	}
}

// TestFundingRecvWatermark funding 水印=recv_ts（ts=未来结算拍非单调——
// 同结算键补写新 recv 行不得被 ts 水印吞，T-0169 对拍期实证修）。
func TestFundingRecvWatermark(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runnerDual(t)
	seedActive(t, sg)
	if err := r.refreshSets(ctx); err != nil {
		t.Fatal(err)
	}
	settle := time.Now().Add(time.Hour).Truncate(time.Second) // 未来结算拍键
	if _, err := md.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,recv_ts)
		VALUES ($1,'BTC-USDT-SWAP',0.0001,$2)`, settle, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := r.projectTsTable(ctx, "ev_funding"); err != nil {
		t.Fatal(err)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_funding WHERE inst_id='BTC-USDT-SWAP'"); n != 1 {
		t.Fatalf("首拍应投影: %d", n)
	}
	// 同结算键（ts 不变）以更新 recv_ts 补写——ts 水印形会吞；recv 水印形必达。
	if _, err := md.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,recv_ts)
		VALUES (now()+interval '1 hour','BTC-USDT-SWAP',0.0002,now())
		ON CONFLICT (venue,inst_id,ts) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if n := count(t, md, "SELECT count(*) FROM md_funding WHERE inst_id='BTC-USDT-SWAP'"); n != 2 {
		t.Fatalf("md 侧应两行（追加形）: %d", n)
	}
	if err := r.projectTsTable(ctx, "ev_funding"); err != nil {
		t.Fatal(err)
	}
	var n int
	var rate *string
	if err := sg.QueryRow(ctx, `SELECT count(*), max(funding_rate::text) FROM md_funding
		WHERE inst_id='BTC-USDT-SWAP'`).Scan(&n, &rate); err != nil {
		t.Fatal(err)
	}
	if n < 2 || rate == nil || *rate != "0.0002" {
		t.Fatalf("recv 水印应达新补写行（SG 侧最新 0.0002）: n=%d rate=%v", n, rate)
	}
}

// TestReverseBridge fin_strategy/fin_venue_untradeable 反向桥（T-0172）：
// SG 正本→PG-md 副本全表 diff；行变再 diff 只搬差行（state 覆写实证）。
func TestReverseBridge(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runner(t)
	if _, err := sg.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,venue,direction)
		VALUES ('BTC-USDT-SWAP','funding-hedge',NULL,'paper',100,'okx','short_perp_long_spot'),
		       ('ETH-USDT-SWAP','dcd_call',NULL,'live',200,'okx','short_dcd')`); err != nil {
		t.Fatal(err)
	}
	if _, err := sg.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
		VALUES ('FAKE-USDT-SWAP','51001',' region-blocked')`); err != nil {
		t.Fatal(err)
	}
	if err := r.reverseStrategies(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.reverseUntradeable(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, md, "SELECT count(*) FROM fin_strategy"); n != 2 {
		t.Fatalf("策略副本应 2 行: %d", n)
	}
	if n := count(t, md, "SELECT count(*) FROM fin_venue_untradeable WHERE inst_id='FAKE-USDT-SWAP'"); n != 1 {
		t.Fatalf("黑名单副本应 1 行: %d", n)
	}
	// 正本行变（retire）→再 diff 副本覆写；未变行零重复不炸（幂等）。
	if _, err := sg.Exec(ctx, `UPDATE fin_strategy SET state='retired' WHERE inst_id='BTC-USDT-SWAP'`); err != nil {
		t.Fatal(err)
	}
	if err := r.reverseStrategies(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := md.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE inst_id='BTC-USDT-SWAP'").Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "retired" {
		t.Fatalf("副本应覆写 retired: %s", state)
	}
	// 副本零漂移：再 diff 后行数不变（upsert-only——正本无删径）。
	if err := r.reverseStrategies(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, md, "SELECT count(*) FROM fin_strategy"); n != 2 {
		t.Fatalf("幂等再 diff 应仍 2 行: %d", n)
	}
}

// TestTsPassThrottle T-0193 桩锚：ts 表族过拍最小间隔闸——两连拍第二拍
// 间隔内跳拍（串行 worker 挤兑根修语义锚；Elapsed 过闸后追平照常）。
func TestTsPassThrottle(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runnerDual(t)
	r.tsMinPass = map[string]time.Duration{"ev_funding": time.Hour} // 大阈=第二拍必跳
	seedActive(t, sg)
	if err := r.refreshSets(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := md.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,recv_ts)
		VALUES (now(),'BTC-USDT-SWAP',0.0001,now())`); err != nil {
		t.Fatal(err)
	}
	if err := r.projectTsTable(ctx, "ev_funding"); err != nil {
		t.Fatal(err)
	}
	var n1 int
	if err := sg.QueryRow(ctx, "SELECT count(*) FROM md_funding").Scan(&n1); err != nil || n1 != 1 {
		t.Fatalf("首拍应投影 1 行: %d %v", n1, err)
	}
	if _, err := md.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,recv_ts)
		VALUES (now()+interval '1 hour','ETH-USDT-SWAP',0.0002,now())`); err != nil {
		t.Fatal(err)
	}
	if err := r.projectTsTable(ctx, "ev_funding"); err != nil {
		t.Fatal(err)
	}
	var n2 int
	if err := sg.QueryRow(ctx, "SELECT count(*) FROM md_funding").Scan(&n2); err != nil || n2 != 1 {
		t.Fatalf("间隔内应跳拍（仍 1 行）: %d %v", n2, err)
	}
}

// TestActiveSetNormalizedLegs T-0250 桩锚（T-0216 根因②）：refreshSets 归一
// 双臂入集——basis 策略期货键剥 -YYMMDD→剥 -USD→-USDT 现货腿（paper/live
// 均含）；dcd 策略经 SG 镜像 md_dcd_product JOIN 取 uly 剥 -USD(G)→-USDT；
// retired 与非钦定模板不入归一臂；端到端现货 tick 落 SG 镜像。
func TestActiveSetNormalizedLegs(t *testing.T) {
	ctx := context.Background()
	r, md, sg := runner(t)
	if _, err := sg.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd) VALUES
		('BTC-USD-270626','quarter_basis_long','paper',1000),
		('ETH-USD-270626','perp_quarter_spread','live',1000),
		('SOL-USD-270626','quarter_basis_long','retired',1000),
		('XRP-USD-270626','funding-hedge','live',1000),
		('DCD-AVAX-1','dcd_call','paper',500),
		('DCD-LTC-2','dcd_put','live',500),
		('DCD-DOT-3','dcd_call','retired',500)`); err != nil {
		t.Fatal(err)
	}
	// SG 镜像 md_dcd_product（投影落面——uly 方言 -USDG/-USD 两形各一）。
	if _, err := sg.Exec(ctx, `INSERT INTO md_dcd_product(ts,venue,surface,uly,product_id,direction,raw) VALUES
		(now(),'okx','dcd','AVAX-USDG','DCD-AVAX-1','C','{}'),
		(now(),'okx','dcd','LTC-USD','DCD-LTC-2','P','{}'),
		(now(),'okx','dcd','DOT-USD','DCD-DOT-3','C','{}')`); err != nil {
		t.Fatal(err)
	}
	// 现货腿 tick 实物：BTC-USDT 应落镜像（旧形裸键谓词滤桥外=T-0216 根因②），
	// SOL-USDT（retired 腿）零投影。
	if _, err := md.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last,recv_ts) VALUES
		(now(),'okx','BTC-USDT',66000,now()),(now(),'okx','SOL-USDT',150,now())`); err != nil {
		t.Fatal(err)
	}
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	have := map[string]bool{}
	for _, s := range r.activeList() {
		have[s] = true
	}
	for _, want := range []string{"BTC-USDT", "ETH-USDT", "AVAX-USDT", "LTC-USDT", "XRP-USD-270626"} {
		if !have[want] {
			t.Fatalf("归一臂入集缺 %s", want)
		}
	}
	for _, reject := range []string{"SOL-USDT", "SOL-USD-270626", "XRP-USDT", "DOT-USDT"} {
		if have[reject] {
			t.Fatalf("离集键误入 %s", reject)
		}
	}
	if n := count(t, sg, "SELECT count(*) FROM md_ticks WHERE inst_id='BTC-USDT'"); n != 1 {
		t.Fatalf("basis 现货腿 tick 应投影: %d", n)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_ticks WHERE inst_id='SOL-USDT'"); n != 0 {
		t.Fatalf("retired 现货腿 tick 零投影: %d", n)
	}
}
