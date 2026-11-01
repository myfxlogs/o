package funding

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestScanE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	// T-0167 分链：本包测试面跨域（研究域表+判据键）——两链并集态（md 先 fin 后
	// =历史重放方向：md 链瞬态超表层先落，fin 链状态化/守卫后到）。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"research.funding.taker_fee_bps": "5", "research.funding.slippage_bps": "3",
		"research.funding.fail_cost_bps": "2", "research.funding.window_loss_bps": "4",
		"research.funding.min_net_apr": "0.05", "research.funding.settles_per_day": "3"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, "DELETE FROM md_funding WHERE inst_id LIKE 'TEST%'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM fin_proposal WHERE inst_id LIKE 'TEST%'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM fin_opp_eval WHERE inst_id LIKE 'TEST%'"); err != nil {
		t.Fatal(err)
	}
	// 鲜 funding 行：高率（应提案）+负率（应拒）
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now(),'TEST-USDT-SWAP',0.01),(now(),'TEST2-USDT-SWAP',-0.02)`); err != nil {
		t.Fatal(err)
	}
	// 陈行不评估
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now()-interval '40 hours','TEST3-USDT-SWAP',0.05)`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var verdict string
	var propN int
	if err := pool.QueryRow(ctx,
		"SELECT verdict FROM fin_opp_eval WHERE inst_id='TEST-USDT-SWAP' ORDER BY id DESC LIMIT 1").Scan(&verdict); err != nil || verdict != "proposed" {
		t.Fatalf("高率卡 verdict=%s err=%v", verdict, err)
	}
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE inst_id='TEST-USDT-SWAP'").Scan(&propN); err != nil || propN != 1 {
		t.Fatalf("提案行=%d err=%v", propN, err)
	}
	if err := pool.QueryRow(ctx,
		"SELECT verdict FROM fin_opp_eval WHERE inst_id='TEST2-USDT-SWAP' ORDER BY id DESC LIMIT 1").Scan(&verdict); err != nil || verdict != "rejected" {
		t.Fatalf("负率卡 verdict=%s err=%v", verdict, err)
	}
	var n int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_opp_eval WHERE inst_id='TEST3-USDT-SWAP'").Scan(&n)
	if n != 0 {
		t.Fatal("陈 funding 行被评估——新鲜度门失效")
	}
	// 判据缺位锚：删一键→scan 应 fail-closed
	if _, err := pool.Exec(ctx, "DELETE FROM app_config WHERE key='research.funding.min_net_apr'"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err == nil {
		t.Fatal("判据缺位应 fail-closed")
	}
	if got := r.rej.Snapshot()["research/missing_params"]; got == 0 {
		t.Fatal("missing_params 门未计")
	}
}

// TestScanEarnFloorE2E 动态底扫描径三态：Earn 现率新鲜行→below_earn 拒；行缺/陈→
// 静态底降级+计数（降级不阻断）。
func TestScanEarnFloorE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	// T-0167 分链：本包测试面跨域（研究域表+判据键）——两链并集态（md 先 fin 后
	// =历史重放方向：md 链瞬态超表层先落，fin 链状态化/守卫后到）。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// net=0.06 锚定形判据（taker 1500+slip 50bps；rate 0.001×1 settle×365）
	for k, v := range map[string]string{
		"research.funding.taker_fee_bps": "1500", "research.funding.slippage_bps": "50",
		"research.funding.fail_cost_bps": "0", "research.funding.window_loss_bps": "0",
		"research.funding.min_net_apr": "0.05", "research.funding.settles_per_day": "1",
		"research.earn.stale_hours": "2"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cleanup := func() {
		for _, q := range []string{
			"DELETE FROM fin_proposal WHERE inst_id LIKE 'TESTE%'",
			"DELETE FROM fin_opp_eval WHERE inst_id LIKE 'TESTE%'",
			"DELETE FROM md_funding WHERE inst_id LIKE 'TESTE%'",
			"DELETE FROM md_earn_rate WHERE ccy='USDT'"} {
			_, _ = pool.Exec(ctx, q)
		}
	}
	cleanup()
	defer cleanup()
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	// ①鲜 Earn 行 0.08（>net 0.06）→TESTE1 below_earn 拒；TESTE2 net≈10.9 过
	if _, err := pool.Exec(ctx, `INSERT INTO md_earn_rate(ts,venue,ccy,rate) VALUES(now(),'okx','USDT',0.08)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now(),'TESTE1-USDT-SWAP',0.001),(now(),'TESTE2-USDT-SWAP',0.01)`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var reason string
	if err := pool.QueryRow(ctx, `SELECT evidence->>'reason' FROM fin_opp_eval WHERE inst_id='TESTE1-USDT-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&reason); err != nil || len(reason) < 10 || reason[:10] != "below_earn" {
		t.Fatalf("between 态 reason=%q err=%v（应 below_earn）", reason, err)
	}
	var verdict string
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval WHERE inst_id='TESTE2-USDT-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&verdict); err != nil || verdict != "proposed" {
		t.Fatalf("高于 Earn 底 verdict=%s err=%v", verdict, err)
	}
	// ②Earn 行缺→静态底降级：TESTE3（同 0.001，net 0.06≥0.05）→proposed+计数
	if _, err := pool.Exec(ctx, "DELETE FROM md_earn_rate WHERE ccy='USDT'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now(),'TESTE3-USDT-SWAP',0.001)`); err != nil {
		t.Fatal(err)
	}
	before := r.rej.Snapshot()["research/earn_rate_degraded"]
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["research/earn_rate_degraded"] <= before {
		t.Fatal("降级计数门未计")
	}
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval WHERE inst_id='TESTE3-USDT-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&verdict); err != nil || verdict != "proposed" {
		t.Fatalf("降级态 verdict=%s err=%v（应静态底放行）", verdict, err)
	}
	// ③陈 Earn 行（5h>2h 门）→同样降级
	if _, err := pool.Exec(ctx, `INSERT INTO md_earn_rate(ts,venue,ccy,rate) VALUES
		(now()-interval '5 hours','okx','USDT',0.08)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now(),'TESTE4-USDT-SWAP',0.001)`); err != nil {
		t.Fatal(err)
	}
	before = r.rej.Snapshot()["research/earn_rate_degraded"]
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["research/earn_rate_degraded"] <= before {
		t.Fatal("陈现率降级计数门未计")
	}
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval WHERE inst_id='TESTE4-USDT-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&verdict); err != nil || verdict != "proposed" {
		t.Fatalf("陈现率降级 verdict=%s err=%v", verdict, err)
	}
}

// T-0029 macro 注解锚：近窗事件→evidence.macro 在；无事件→缺省；判据不受影响。
func TestMacroAnnotate(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	// T-0167 分链：本包测试面跨域（研究域表+判据键）——两链并集态（md 先 fin 后
	// =历史重放方向：md 链瞬态超表层先落，fin 链状态化/守卫后到）。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 种子锚：kind 三值齐+行数≥15
	var kinds, n int
	if err := pool.QueryRow(ctx,
		"SELECT count(DISTINCT kind),count(*) FROM md_macro_event").Scan(&kinds, &n); err != nil || kinds != 3 || n < 15 {
		t.Fatalf("种子行 kinds=%d n=%d", kinds, n)
	}
	// 注一个明日事件→evidence.macro 应在
	if _, err := pool.Exec(ctx,
		"INSERT INTO md_macro_event(event_date,kind,title) VALUES(current_date+1,'fomc','dbg-event') ON CONFLICT DO NOTHING"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	e := &Eval{InstID: "M-USDT-SWAP", Evidence: map[string]any{}}
	r.annotateMacro(ctx, e)
	if _, ok := e.Evidence["macro"]; !ok {
		t.Fatal("近窗事件 macro 注解缺席")
	}
	// 无近窗事件（horizon=0 域内）→不注解
	e2 := &Eval{InstID: "M2", Evidence: map[string]any{}}
	if _, err := pool.Exec(ctx,
		"UPDATE app_config SET value='\"0\"' WHERE key='research.macro.horizon_days'"); err != nil {
		t.Fatal(err)
	}
	_ = cfg.Load(ctx)
	r.annotateMacro(ctx, e2)
	if _, ok := e2.Evidence["macro"]; ok {
		t.Fatal("horizon=0 不应注解")
	}
	// 缺键 fail-open：删键不炸不注解
	if _, err := pool.Exec(ctx, "DELETE FROM app_config WHERE key='research.macro.horizon_days'"); err != nil {
		t.Fatal(err)
	}
	_ = cfg.Load(ctx)
	e3 := &Eval{InstID: "M3", Evidence: map[string]any{}}
	r.annotateMacro(ctx, e3) // fail-open 无注解不炸
}

// T-0131 锚：提案去重升级——同键 active 策略在册→persist 不再插提案行
// （洪泛期病灶：armed/live 消化后下拍重插灌穿提案表）。
func TestActiveStrategyBlocksProposal(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	// T-0167 分链：本包测试面跨域（研究域表+判据键）——两链并集态（md 先 fin 后
	// =历史重放方向：md 链瞬态超表层先落，fin 链状态化/守卫后到）。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"research.funding.taker_fee_bps": "5", "research.funding.slippage_bps": "3",
		"research.funding.fail_cost_bps": "2", "research.funding.window_loss_bps": "4",
		"research.funding.min_net_apr": "0.05", "research.funding.settles_per_day": "3"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate)
		VALUES(now(),'DUP-USDT-SWAP',0.01)`); err != nil {
		t.Fatal(err)
	}
	// 同键 active 策略在册——新提案应被闸。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction)
		VALUES('DUP-USDT-SWAP','funding-hedge','paper',100,'short_perp_long_spot')`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='DUP-USDT-SWAP'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("同键 active 策略在册应零提案 n=%d err=%v", n, err)
	}
	// 触发器闸（迁移 0054）：同键 proposed/armed 直插也被跳。
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('TRG-USDT-SWAP','funding-hedge','short_perp_long_spot',0.08,'proposed','{}','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
			VALUES($1,'TRG-USDT-SWAP','short_perp_long_spot',1.0)`, eid); err != nil {
			t.Fatal(err)
		}
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='TRG-USDT-SWAP'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("去重触发器应同键只留一行 n=%d err=%v", n, err)
	}
}

// T-0150 起草闸桩锚：perp-only 合成 inst（无 spot 对）proposed 评估→
// infeasible/no_spot_leg 零提案；正常现货对不受影响。
func TestSpotLegPrecheckGate(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	// T-0167 分链：本包测试面跨域（研究域表+判据键）——两链并集态（md 先 fin 后
	// =历史重放方向：md 链瞬态超表层先落，fin 链状态化/守卫后到）。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	// SPOT 目录桩：只实有 TEST-USDT（合成器 SHEIN 类不在）。
	r.AttachSpotFeed(func(ctx context.Context) ([]byte, error) {
		return []byte(`[{"instId":"TEST-USDT"},{"instId":"BTC-USDT"}]`), nil
	})
	for _, inst := range []string{"SHEIN-USDT-SWAP", "TEST-USDT-SWAP"} {
		var eid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
			VALUES($1,'funding-hedge','short_perp_long_spot',9.9,8.8,'proposed','[]','{}') RETURNING id`, inst).Scan(&eid); err != nil {
			t.Fatal(err)
		}
		var e Eval
		var legsB, evB []byte
		if err := pool.QueryRow(ctx, `SELECT inst_id,direction,gross_apr,net_apr,verdict,legs,evidence
			FROM fin_opp_eval WHERE id=$1`, eid).Scan(&e.InstID, &e.Direction, &e.GrossAPR, &e.NetAPR,
			&e.Verdict, &legsB, &evB); err != nil {
			t.Fatal(err)
		}
		json.Unmarshal(legsB, &e.Legs)
		json.Unmarshal(evB, &e.Evidence)
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			return r.persist(ctx, tx, e, Params{})
		}); err != nil {
			t.Fatal(err)
		}
	}
	var v string
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval WHERE inst_id='SHEIN-USDT-SWAP'`).Scan(&v); err != nil || v != "rejected" {
		t.Fatalf("合成 inst 应 rejected(no_spot_leg) got %q err=%v", v, err)
	}
	var reason string
	pool.QueryRow(ctx, `SELECT evidence->>'reason' FROM fin_opp_eval WHERE inst_id='SHEIN-USDT-SWAP'`).Scan(&reason)
	if reason != "no_spot_leg" {
		t.Fatalf("拒因=%q", reason)
	}
	var pn int
	pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal WHERE inst_id='SHEIN-USDT-SWAP'`).Scan(&pn)
	if pn != 0 {
		t.Fatalf("合成 inst 应零提案 got %d", pn)
	}
	pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal WHERE inst_id='TEST-USDT-SWAP'`).Scan(&pn)
	if pn != 1 {
		t.Fatalf("正常现货对应照常提案 got %d", pn)
	}
	// 黑名单同读面：TEST-USDT-SWAP 入册→后续 persist 拒 blacklisted。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
		VALUES('TEST-USDT-SWAP','51001','stub') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	var eid int64
	pool.QueryRow(ctx, `SELECT id FROM fin_opp_eval WHERE inst_id='TEST-USDT-SWAP'`).Scan(&eid)
	var e Eval
	var legsB, evB []byte
	if err := pool.QueryRow(ctx, `SELECT inst_id,direction,gross_apr,net_apr,verdict,legs,evidence
		FROM fin_opp_eval WHERE id=$1`, eid).Scan(&e.InstID, &e.Direction, &e.GrossAPR, &e.NetAPR,
		&e.Verdict, &legsB, &evB); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(legsB, &e.Legs)
	json.Unmarshal(evB, &e.Evidence)
	e.Verdict = "proposed"
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return r.persist(ctx, tx, e, Params{})
	}); err != nil {
		t.Fatal(err)
	}
	pool.QueryRow(ctx, `SELECT evidence->>'reason' FROM fin_opp_eval WHERE inst_id='TEST-USDT-SWAP'`).Scan(&reason)
	if reason != "blacklisted" {
		t.Fatalf("黑名单拒因应 blacklisted got %q", reason)
	}
}
