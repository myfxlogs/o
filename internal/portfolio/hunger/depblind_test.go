package hunger

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0394 桩测三锚（票面钦）：①植假失明对→「依赖对失明」行+观测盲升件断言
// （信号对/腿对/缺席消费三形同测）；②真对零误报（信号活+评估有/腿有鲜
// tick/信号死三形皆不报——产者死归活性面非本域）；③枚举器空集=
// fail-closed 报错（扫不到对=病灶非零失明）。补面：清单落账谱变才重落；
// 注册豁免落「依赖对豁免」行非暗吞；签名去重吸收特例探针同窗报。

// depFixture 探针桩景（fin 迁移——okx.symbols 种 BTC/ETH 现货+SWAP 四键
// 宇宙 fin_0004 在册；seed 助手复用 silence_test 同包件）。
func depFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return New(pool, cfg), pool
}

// seedDepStrategy 策略行（腿对枚举素材——activeSet 策略臂）。
func seedDepStrategy(t *testing.T, pool *pgxpool.Pool, inst, tmpl, state string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd) VALUES($1,$2,$3,100)`,
		inst, tmpl, state); err != nil {
		t.Fatal(err)
	}
}

// seedDepTick md_ticks 行定龄（age=0 鲜行；>depblindLegFreshMin 窗外=陈行）。
func seedDepTick(t *testing.T, pool *pgxpool.Pool, inst string, age time.Duration) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO md_ticks(ts,inst_id,last,bid,ask)
		VALUES(now()-make_interval(secs=>$2),$1,1,1,1)
		ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts`,
		inst, age.Seconds()); err != nil {
		t.Fatal(err)
	}
}

// seedDepScan 粗扫行定龄（T-0401③——scan 哨兵对素材；age=0 鲜行=粗扫臂活，
// 不种=scan 对失明形）。
func seedDepScan(t *testing.T, pool *pgxpool.Pool, age time.Duration) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO md_tick_scan(ts,inst_id,last,bid,ask)
		VALUES(now()-make_interval(secs=>$1),'ZZZ-SCAN-USDT-SWAP',1,1,1)
		ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts`,
		age.Seconds()); err != nil {
		t.Fatal(err)
	}
}

// depRows 探针行数（失明行/升件行——断言公共面；esc_scope 限定本件签名域）。
func depRows(t *testing.T, pool *pgxpool.Pool) (blind, esc int) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='依赖对失明' AND ticket='T-0394'`).Scan(&blind); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND ticket='T-0394'`).Scan(&esc); err != nil {
		t.Fatal(err)
	}
	return blind, esc
}

// TestProbeDepsBlindPairs 锚①：植假失明三形同拍——
//
//	信号对：xvenue-carry 评估近窗零（旧行在全集保枚举面）+carry_diff 鲜
//	  行→signal:carry_diff→xvenue-carry 失明；
//	缺席对：orphan_k 鲜行而两链派生零命中（无卡册无命名归族）→
//	  signal:orphan_k→∅ 产而活消费缺席；
//	腿对：funding-hedge live 策略 BTC-USDT 现货腿零 tick→leg:BTC-USDT 盲。
//
// 断言：三对「依赖对失明」行 kind=dep_blind key=<签名> 齐+三观测盲升件
// esc_scope=对签名。
func TestProbeDepsBlindPairs(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	seedHypoNeeds(t, pool, "xvenue-carry", "xvenue_carry",
		[]string{"carry_diff", "xvenue_spread"})
	seedEvalAt(t, pool, "xvenue-carry", 10*24*time.Hour) // 全集在列、近窗零
	seedSignal(t, pool, "carry_diff", "BTC-USDT-SWAP", 0)
	seedSignal(t, pool, "orphan_k", "BTC-USDT-SWAP", 0)
	seedDepStrategy(t, pool, "BTC-USDT", "funding-hedge", "live")
	// T-0401③：粗扫表不种=scan 哨兵对失明（第四形——粗扫产者死=筛选层全盲）。
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	blind, esc := depRows(t, pool)
	if blind != 4 || esc != 4 {
		t.Fatalf("失明行/升件数不齐: 失明=%d 升件=%d", blind, esc)
	}
	for _, sig := range []string{
		"signal:carry_diff→xvenue-carry", "signal:orphan_k→∅", "leg:BTC-USDT",
		"scan:md_tick_scan→discovery"} {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE event='依赖对失明' AND note LIKE '%key='||$1||' %'`,
			sig).Scan(&n); err != nil || n != 1 {
			t.Fatalf("失明行缺/键错位 %s: n=%d err=%v", sig, n, err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE event='升件' AND esc_class='观测盲' AND esc_scope=$1`,
			sig).Scan(&n); err != nil || n != 1 {
			t.Fatalf("升件缺/scope 错位 %s: n=%d err=%v", sig, n, err)
		}
	}
}

// TestProbeDepsRealPairsNoFalsePositive 锚②：真对零误报——
//
//	信号活+评估窗内有→不报；
//	腿有鲜 tick→不报；
//	信号死（>7d）+评估零→不报（产者死=源死归活性面非本探针域）。
//
// 断言：失明行=0 升件=0；清单行照落（枚举非空——谱面证据在账）。
func TestProbeDepsRealPairsNoFalsePositive(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	seedHypoNeeds(t, pool, "xvenue-carry", "xvenue_carry", []string{"carry_diff"})
	seedEvalAt(t, pool, "xvenue-carry", 0)                              // 评估窗内活
	seedSignal(t, pool, "carry_diff", "BTC-USDT-SWAP", 0)               // 信号活+评估有
	seedSignal(t, pool, "dead_kind", "BTC-USDT-SWAP", 200*24*time.Hour) // 死源
	seedDepStrategy(t, pool, "ETH-USDT", "funding-hedge", "live")
	seedDepTick(t, pool, "ETH-USDT", 0) // 腿有鲜 tick
	seedDepScan(t, pool, 0)             // T-0401③：粗扫臂活（不种=scan 对失明误报）
	// ③族钉定谱零误报位：窗内 eval 有产须配提案消费（缺=eval→proposal 失明）；
	// ops_event→链对=探针自写行链覆写在即天然有消。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		SELECT id,inst_id,direction,0 FROM fin_opp_eval`); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	blind, esc := depRows(t, pool)
	if blind != 0 || esc != 0 {
		t.Fatalf("真对误报: 失明=%d 升件=%d", blind, esc)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='依赖对清单' AND ticket='T-0394'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("清单行缺: n=%d err=%v", n, err)
	}
}

// TestProbeDepsEmptyEnumFailClosed 锚③（T-0405 语义重钉）：③④族启用后
// 枚举地板=钉定表对+app_config 键域对——净库零信号/零腿仍有 ③④对，
// 全谱空集结构性不可达（len(pairs)==0 闸仍守）；改为断言净库枚举谱=
// 钉定表对+键域对两族齐入列（config 域皆读面册内=册外零盲）。
func TestProbeDepsEmptyEnumFailClosed(t *testing.T) {
	s, _ := depFixture(t)
	pairs, err := s.enumerateDeps(context.Background())
	if err != nil {
		t.Fatalf("enumerate: %v", err)
	}
	have := map[string]bool{}
	for _, p := range pairs {
		have[p.sig] = true
	}
	for _, spec := range depTablePairs {
		if !have[spec.sig] {
			t.Fatalf("③族钉定对缺席: %s", spec.sig)
		}
	}
	var cfgPairs int
	for _, p := range pairs {
		if p.family == "config" {
			cfgPairs++
			if p.consumerAbsent {
				t.Fatalf("种键域读面册外误标 orphan: %s", p.sig)
			}
		}
	}
	if cfgPairs == 0 {
		t.Fatal("④族键域对缺席——app_config 键域枚举未入列")
	}
}

// TestProbeDepsDedupAbsorbsSibling 签名去重吸收面：0384 特例探针同窗已报
// template=xvenue-carry（esc_scope=族名）→本件 signal:*→xvenue-carry 对全
// 吸收不重复升；0392 同款 esc_scope=inst 吸腿对（产者键=inst 同键）。
func TestProbeDepsDedupAbsorbsSibling(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	seedHypoNeeds(t, pool, "xvenue-carry", "xvenue_carry", []string{"carry_diff"})
	seedEvalAt(t, pool, "xvenue-carry", 10*24*time.Hour)
	seedSignal(t, pool, "carry_diff", "BTC-USDT-SWAP", 0)
	seedDepStrategy(t, pool, "BTC-USDT", "funding-hedge", "live")
	seedDepScan(t, pool, 0) // T-0401③：粗扫臂活
	// 特例探针同窗报（0384 族沉默=esc_scope template；0392 腿覆盖=esc_scope inst）
	for _, scope := range []string{"xvenue-carry", "BTC-USDT"} {
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
			VALUES('决策席','升件','T-0384','先例报','观测盲',$1)`, scope); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	blind, esc := depRows(t, pool)
	if blind != 0 || esc != 0 {
		t.Fatalf("签名去重未吸收: 失明=%d 升件=%d", blind, esc)
	}
}

// TestProbeDepsWhitelistExemptNotSilent 注册豁免面：合法产而未读对注册
// 豁免→失明/升件零行+「依赖对豁免」行携理由键同窗落一行（注册式不静默
// ——豁免本身留账非暗吞）；同窗再跑不重复落。
func TestProbeDepsWhitelistExemptNotSilent(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	seedSignal(t, pool, "orphan_k", "BTC-USDT-SWAP", 0)
	seedDepScan(t, pool, 0) // T-0401③：粗扫臂活
	old := depBlindWhitelist
	depBlindWhitelist = map[string]string{"signal:orphan_k→∅": "纯展示面预留——人工基线"}
	defer func() { depBlindWhitelist = old }()
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	blind, esc := depRows(t, pool)
	if blind != 0 || esc != 0 {
		t.Fatalf("豁免对误报: 失明=%d 升件=%d", blind, esc)
	}
	var n int
	var note string
	if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(max(note),'') FROM ops_event
		WHERE event='依赖对豁免' AND ticket='T-0394'`).Scan(&n, &note); err != nil || n != 1 {
		t.Fatalf("豁免行缺/重复: n=%d err=%v", n, err)
	}
	for _, frag := range []string{"key=signal:orphan_k→∅", "理由键=纯展示面预留"} {
		if !strings.Contains(note, frag) {
			t.Fatalf("豁免行缺 %q: %s", frag, note)
		}
	}
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("再拍: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='依赖对豁免'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("豁免同窗重复落行: n=%d", n)
	}
}

// TestProbeDepsSubreqStale 晋级滞留形（T-0401③）：open 请求超龄（>10min）
// 而同键 md_ticks 零新鲜→subreq 对失明+升件；同键鲜 tick 在/请求未超龄/
// closed 三形均不报（提名正常在采/未到期/已退订）。
func TestProbeDepsSubreqStale(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	seedDepScan(t, pool, 0)
	for _, x := range []struct {
		inst string
		age  time.Duration
	}{
		{"QQQ-USDT-SWAP", 20 * time.Minute}, // 超龄未采→滞留盲
		{"WWW-USDT-SWAP", 20 * time.Minute}, // 超龄但有鲜 tick→不报
		{"EEE-USDT-SWAP", 1 * time.Minute},  // 未超龄→不报（拾取在途期）
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_sub_request(venue,inst_id,requester,state,ts)
			VALUES('okx',$1,'xvenue-carry','open',now()-make_interval(secs=>$2))`,
			x.inst, x.age.Seconds()); err != nil {
			t.Fatal(err)
		}
	}
	seedDepTick(t, pool, "WWW-USDT-SWAP", 0)
	if _, err := pool.Exec(ctx, `INSERT INTO fin_sub_request(venue,inst_id,requester,state,ts)
		VALUES('okx','RRR-USDT-SWAP','xvenue-carry','closed',now()-interval '1 hour'),
		      ('okx','TTT-USDT-SWAP','user_intent','open',now()-interval '20 minutes')`); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='依赖对失明' AND note LIKE '%key=subreq:%'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("滞留失明行数错位（应恰 2=QQQ+TTT）: n=%d err=%v", n, err)
	}
	for _, scope := range []string{"subreq:xvenue-carry:QQQ-USDT-SWAP",
		"subreq:user_intent:TTT-USDT-SWAP"} { // T-0412 用户意图滞留同枚举面罩
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE event='升件' AND esc_class='观测盲'
			AND esc_scope=$1`, scope).Scan(&n); err != nil || n != 1 {
			t.Fatalf("滞留升件缺/scope 错位 %s: n=%d err=%v", scope, n, err)
		}
	}
}

// TestProbeDepsCensusStableNoDup 清单幂等面：谱面未变二拍不重复落
// 「依赖对清单」行（枚举器产出留账≠每拍轰炸）。
func TestProbeDepsCensusStableNoDup(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	seedSignal(t, pool, "carry_diff", "BTC-USDT-SWAP", 0)
	seedEvalAt(t, pool, "xvenue-carry", 0) // template 在全集但无 kind 映射（无卡册）
	for i := 0; i < 2; i++ {
		if err := s.ProbeDeps(ctx); err != nil {
			t.Fatalf("probe%d: %v", i, err)
		}
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='依赖对清单'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("清单重复落行: n=%d", n)
	}
}

// —— T-0405 ③④族+面清单 diff 桩测（票面钦④桩测：③族 paper 零 fill 检出/
//   合法未读白名单豁/面清单新增行升件三态）——

// TestProbeDepsTablePairPaperZeroFill ③族钉定对检出：paper 策略在册逾窗
// （entered_at>7d——应产 paper 记账）而 fin_paper_fill 窗内零行→
// table:fin_strategy[paper]→fin_paper_fill 失明行+观测盲升件。同窗新生
// paper（entered_at<窗）不构产者——误报闸位断言。
func TestProbeDepsTablePairPaperZeroFill(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd,entered_at)
		VALUES('BTC-USDT','funding-hedge','paper',100,now()-interval '10 days')`); err != nil {
		t.Fatal(err)
	}
	seedDepStrategy(t, pool, "ETH-USDT", "funding-hedge", "paper") // 新生 paper 不构产者
	seedDepTick(t, pool, "BTC-USDT", 0)                            // 腿对有供——隔③族单点断言
	seedDepTick(t, pool, "ETH-USDT", 0)
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='依赖对失明' AND note LIKE '%key=table:fin_strategy[paper]→fin_paper_fill %'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("③族失明行缺: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND esc_scope='table:fin_strategy[paper]→fin_paper_fill'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("③族升件缺: n=%d err=%v", n, err)
	}
}

// TestProbeDepsTablePairWhitelistExempt ③族白名单豁：合法「产而未读」对
// 注册豁免→失明/升件零行+「依赖对豁免」行携理由键（豁面照旧机制）。
func TestProbeDepsTablePairWhitelistExempt(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd,entered_at)
		VALUES('BTC-USDT','funding-hedge','paper',100,now()-interval '10 days')`); err != nil {
		t.Fatal(err)
	}
	seedDepTick(t, pool, "BTC-USDT", 0) // 腿对行情有供——隔③族单点断言
	seedDepScan(t, pool, 0)             // T-0401③：粗扫臂活（合流补种——不种=scan 对失明误报）
	old := depBlindWhitelist
	depBlindWhitelist = map[string]string{
		"table:fin_strategy[paper]→fin_paper_fill": "paper 记账面待建——人工基线"}
	defer func() { depBlindWhitelist = old }()
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	blind, esc := depRows(t, pool)
	if blind != 0 || esc != 0 {
		t.Fatalf("豁免对误报: 失明=%d 升件=%d", blind, esc)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='依赖对豁免' AND note LIKE 'key=table:fin_strategy%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("豁免行缺: n=%d err=%v", n, err)
	}
}

// TestProbeDepsConfigOrphanDomain ④族死键检出：app_config 册外域键在册
// （产者已写）而读面册外→config:<域>→∅ orphan 失明行+升件；册内域恒不盲。
func TestProbeDepsConfigOrphanDomain(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO app_config(key,value) VALUES('zzdead.foo','1')`); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='依赖对失明' AND note LIKE '%key=config:zzdead→∅ %'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("④族死键失明行缺: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND esc_scope='config:zzdead→∅'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("④族死键升件缺: n=%d err=%v", n, err)
	}
}

// TestProbeFaceDiffNewFaceEscalation 面清单 diff 三态：首拍=基线建册零
// 升件→新键域入册后二拍=新面行+观测盲升件（gate_domain+dep_pair 两形
// 同新）→三拍谱稳=零新增（差集空+同窗去重双闸）；快照仅留最近两拍批。
func TestProbeFaceDiffNewFaceEscalation(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	// 一拍=基线（零新面升件）
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe1: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='面清单新面'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("基线拍误升新面: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(DISTINCT batch) FROM ops_face_snapshot`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("基线批缺: n=%d err=%v", n, err)
	}
	// 新面入册（gate_domain:newdom+dep_pair:config:newdom→∅ 两形同新）
	if _, err := pool.Exec(ctx,
		`INSERT INTO app_config(key,value) VALUES('newdom.x','1')`); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe2: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='面清单新面'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("新面行数错位: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:gate_domain:newdom'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("gate_domain 新面升件缺: n=%d err=%v", n, err)
	}
	// 三拍谱稳：差集空零新增（同窗去重同吸）；快照批≤2
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe3: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='面清单新面'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("谱稳拍误升: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(DISTINCT batch) FROM ops_face_snapshot`).Scan(&n); err != nil || n > 2 {
		t.Fatalf("快照批超留: n=%d err=%v", n, err)
	}
}

// TestProbeFaceDiffBaselineExempt 面清单基线册豁免面（T-0448）：合法新面
// 人工过基线登记 faceBaselineWhitelist——册内面差集照落「面清单新面」
// 行（携基线册理由键留账）但升件免（间歇写席 ops_event 7d 活跃窗出入
// 摆动不重复炸票/起草）；册外新面照升件（拒册=不登记等人工判——对照
// 面同拍断言豁免不误伤）。
func TestProbeFaceDiffBaselineExempt(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe1 基线拍: %v", err)
	}
	// 新席 ops_event 自报行=service 面新面素材（whsvc 册内/badsvc 册外对照）
	for _, seat := range []string{"whsvc", "badsvc"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'自报','T-0448','桩')`, seat); err != nil {
			t.Fatal(err)
		}
	}
	old := faceBaselineWhitelist
	faceBaselineWhitelist = map[string]string{
		"face:service:whsvc": "桩基线——合法新面登记（T-0448 同式）"}
	defer func() { faceBaselineWhitelist = old }()
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("probe2: %v", err)
	}
	var n int
	// 册内面：新面行照落留账+升件零（豁免生效——修前红位）
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='面清单新面' AND note LIKE '%key=face:service:whsvc%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("册内新面行缺: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='面清单新面' AND note LIKE '%基线册=%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("册内新面行缺基线册理由键: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='face:service:whsvc'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("册内面升件未免: n=%d err=%v", n, err)
	}
	// 册外面对照：新面行+升件齐（豁免不误伤）
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='face:service:badsvc'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("册外面升件缺: n=%d err=%v", n, err)
	}
}

// TestDepLegKeysStarExpanded T-0416 依赖对探针宇宙接线锚：okx.symbols=["*"]
// +供给件挂线→depLegKeys 实读目录员腿入集（病灶形=生产「宇宙读败:
// 宇宙扩件探针未挂」依赖对探针瞎——修后复电同判据）。
func TestDepLegKeysStarExpanded(t *testing.T) {
	s, pool := depFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE app_config SET value='["*"]' WHERE key='okx.symbols'`); err != nil {
		t.Fatal(err)
	}
	if err := s.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// 红桩（未挂=* 形 fail-closed——两探针同根病灶复刻）。
	if _, err := s.depLegKeys(ctx); err == nil {
		t.Fatal("未挂供给件应 fail-closed")
	}
	s.AttachUniverse(func(context.Context) ([]string, error) {
		return []string{"AAA-USDT-SWAP", "AAA-USDT", "BTC-USDT-SWAP"}, nil
	})
	seedDepStrategy(t, pool, "AAA-USDT-SWAP", "funding-hedge", "live")
	keys, err := s.depLegKeys(ctx)
	if err != nil {
		t.Fatalf("供给件挂线后应复电: %v", err)
	}
	got := map[string]bool{}
	for _, k := range keys {
		got[k] = true
	}
	if !got["AAA-USDT-SWAP"] || !got["AAA-USDT"] {
		t.Fatalf("目录员腿应入所需集: %v", keys)
	}
}

// TestFaceBaselineLegZECUSDTRegistered T-0453 先红桩（T-0388 修复类律——
// 病灶存在断言修前红+同断言修后绿）：签名面 face:dep_pair:leg:ZEC-USDT
// 必须在人工基线册。根因锚：ZEC-USDT 现货腿=fin_strategy funding-hedge
// probe 行派生（state<>'retired' 落 FundingHedgeSpotArm 谓词——ActiveSetUnion
// 同源表达式枚举本体，零硬编码）；提案/策略活性期进出=间歇面形（T-0448 册
// 语义罩位），册外面窗摆回重复升件炸起草拍（生产实证 #3454）。修前未登记
// =红；入册后同断言绿且理由键非空（注册式不静默）。
func TestFaceBaselineLegZECUSDTRegistered(t *testing.T) {
	scope := "face:dep_pair:leg:ZEC-USDT"
	reason, ok := faceBaselineExempt(scope)
	if !ok {
		t.Fatalf("病灶存在：%s 未入人工基线册——窗摆回将重复升件（T-0453）", scope)
	}
	if strings.TrimSpace(reason) == "" {
		t.Fatalf("%s 理由键空——注册式不静默（depblind 豁免同律）", scope)
	}
}
