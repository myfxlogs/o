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

// T-0384 锚①桩测三形：信号活+评估零=病灶行落 / 信号活+评估有=零行 /
// 信号死=不报（源死非本探针域）；锚③现族零误报（生产谱面复刻——
// 唯 xvenue-carry 病灶，funding-hedge 等活族不报）；同窗 24h 去重。

func silenceFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
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

// seedSignal md_signal 状态行（age=血统龄——0 即鲜行；>7d 即窗内死）。
func seedSignal(t *testing.T, pool *pgxpool.Pool, kind, key string, age time.Duration) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES($1,$2,1.0,'{}',now()-make_interval(secs=>$3))`,
		kind, key, age.Seconds()); err != nil {
		t.Fatal(err)
	}
}

// seedPulseAt 脉冲行定龄（T-0485 改读面：age=0 窗内活脉冲；>7d 脉冲陈——
// 族在全集但窗内死。fin_opp_eval 镜像行不再播——SG 恒零谱形即本桩底）。
func seedPulseAt(t *testing.T, pool *pgxpool.Pool, template string, age time.Duration) {
	t.Helper()
	n24 := 0
	if age < 24*time.Hour {
		n24 = 1
	}
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO md_eval_pulse(template,last_eval_ts,eval_n_24h,refreshed_at)
		VALUES($1,now()-make_interval(secs=>$2),$3,now())`,
		template, age.Seconds(), n24); err != nil {
		t.Fatal(err)
	}
}

// seedHypoNeeds 假想卡信号需求（族注册表链素材——requirements.data_needs）。
func seedHypoNeeds(t *testing.T, pool *pgxpool.Pool, surface, family string, dataNeeds []string) {
	t.Helper()
	needs := `["` + strings.Join(dataNeeds, `","`) + `"]`
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_opp_hypothesis(venue,surface,family,requirements)
		VALUES('okx',$1,$2,('{"data_needs":`+needs+`}')::jsonb)`,
		surface, family); err != nil {
		t.Fatal(err)
	}
}

// lesionRows 病灶两形行数（族沉默行/升件行——断言公共面）。
func lesionRows(t *testing.T, pool *pgxpool.Pool) (silence, esc int) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='族沉默' AND ticket='T-0384'`).Scan(&silence); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲'`).Scan(&esc); err != nil {
		t.Fatal(err)
	}
	return silence, esc
}

// TestProbeFamiliesLesionOnLiveSignalZeroEval 形①：xvenue-carry 评估窗内零
// +carry_diff 鲜行→族沉默行+观测盲升件（T-0164 病灶原型复刻）。
func TestProbeFamiliesLesionOnLiveSignalZeroEval(t *testing.T) {
	s, pool := silenceFixture(t)
	ctx := context.Background()
	seedHypoNeeds(t, pool, "xvenue-carry", "xvenue_carry",
		[]string{"carry_diff", "xvenue_spread", "mt5_swap", "depth5", "md_ticks mt5"})
	seedPulseAt(t, pool, "xvenue-carry", 10*24*time.Hour) // 全集在列、近窗零
	seedSignal(t, pool, "carry_diff", "BTC-USDT-SWAP", 0)
	seedSignal(t, pool, "carry_diff", "ETH-USDT-SWAP", 0)
	seedSignal(t, pool, "xvenue_spread", "BTC-USDT-SWAP", 0)
	seedSignal(t, pool, "mt5_swap", "BTCUSDm", 0)
	seedSignal(t, pool, "depth5", "BTC-USDT-SWAP", 0)
	if err := s.ProbeFamilies(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	silence, esc := lesionRows(t, pool)
	if silence != 1 || esc != 1 {
		t.Fatalf("病灶行缺失: 族沉默=%d 升件=%d", silence, esc)
	}
	var note, cls, scope string
	if err := pool.QueryRow(ctx, `SELECT note,esc_class,esc_scope FROM ops_event
		WHERE event='升件' AND esc_class='观测盲'`).Scan(&note, &cls, &scope); err != nil {
		t.Fatal(err)
	}
	if scope != "xvenue-carry" {
		t.Fatalf("esc_scope 非族名: %s", scope)
	}
	for _, frag := range []string{"xvenue-carry", "carry_diff", "鲜行=2"} {
		if !strings.Contains(note, frag) {
			t.Fatalf("升件 note 缺 %q: %s", frag, note)
		}
	}
	// 族沉默行四列钦定面（template/信号kind/信号行数/证据锚）。
	var snote string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='族沉默'`).Scan(&snote); err != nil {
		t.Fatal(err)
	}
	for _, frag := range []string{"template=xvenue-carry", "信号kind=carry_diff",
		"信号行数=2", "评估行数=0", "锚=md_signal 7d{carry_diff:2"} {
		if !strings.Contains(snote, frag) {
			t.Fatalf("族沉默 note 缺 %q: %s", frag, snote)
		}
	}
}

// TestProbeFamiliesQuietOnFreshEval 形②：信号活+评估有=零行
// （funding-hedge 命名规约链→funding_view 实谱零误报）。
// T-0485 钦定桩：rejected-only 谱形=SG fin_opp_eval 镜像恒零+脉冲行新鲜
// →探针静默（修前红=观测盲日频误报六案同签名形，修后绿）。
func TestProbeFamiliesQuietOnFreshEval(t *testing.T) {
	s, pool := silenceFixture(t)
	ctx := context.Background()
	seedPulseAt(t, pool, "funding-hedge", 0)                // 脉冲窗内活
	seedSignal(t, pool, "funding_view", "BTC-USDT-SWAP", 0) // 源活
	// rejected-only 谱形断言：SG 镜像恒零（本桩零播种即谓词域外景）。
	var mirrorN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE template='funding-hedge'`).Scan(&mirrorN); err != nil {
		t.Fatal(err)
	}
	if mirrorN != 0 {
		t.Fatalf("rejected-only 谱形不成立（镜像行=%d 应 0）", mirrorN)
	}
	if err := s.ProbeFamilies(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if silence, esc := lesionRows(t, pool); silence != 0 || esc != 0 {
		t.Fatalf("活族误报: 族沉默=%d 升件=%d", silence, esc)
	}
}

// TestProbeFamiliesLesionOnPulseStale 真死面保留（T-0485 验收锚③）：
// 脉冲行陈旧=真死可报——rejected-only 族 eval 链死探测能力不因改读失明。
func TestProbeFamiliesLesionOnPulseStale(t *testing.T) {
	s, pool := silenceFixture(t)
	ctx := context.Background()
	seedHypoNeeds(t, pool, "xvenue-carry", "xvenue_carry", []string{"carry_diff"})
	seedPulseAt(t, pool, "xvenue-carry", 10*24*time.Hour) // 脉冲陈=窗内零
	seedSignal(t, pool, "carry_diff", "BTC-USDT-SWAP", 0)
	if err := s.ProbeFamilies(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if silence, esc := lesionRows(t, pool); silence != 1 || esc != 1 {
		t.Fatalf("真死病灶行缺失: 族沉默=%d 升件=%d", silence, esc)
	}
}

// TestProbeFamiliesSilentOnDeadSignal 形③：驱动信号死=不报（源死归
// ops_face 活性面非本探针域——从件鲜不翻判）。
func TestProbeFamiliesSilentOnDeadSignal(t *testing.T) {
	s, pool := silenceFixture(t)
	ctx := context.Background()
	seedHypoNeeds(t, pool, "xvenue-carry", "xvenue_carry",
		[]string{"carry_diff", "xvenue_spread"})
	seedPulseAt(t, pool, "xvenue-carry", 10*24*time.Hour)
	seedSignal(t, pool, "carry_diff", "BTC-USDT-SWAP", 10*24*time.Hour) // 驱动陈>7d
	seedSignal(t, pool, "xvenue_spread", "BTC-USDT-SWAP", 0)            // 从件鲜——不翻判
	if err := s.ProbeFamilies(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if silence, esc := lesionRows(t, pool); silence != 0 || esc != 0 {
		t.Fatalf("源死误报: 族沉默=%d 升件=%d", silence, esc)
	}
}

// TestProbeFamiliesDedup24h 同窗去重：病灶在而再拍——24h 内同族至多一报。
func TestProbeFamiliesDedup24h(t *testing.T) {
	s, pool := silenceFixture(t)
	ctx := context.Background()
	seedHypoNeeds(t, pool, "xvenue-carry", "xvenue_carry", []string{"carry_diff"})
	seedPulseAt(t, pool, "xvenue-carry", 10*24*time.Hour)
	seedSignal(t, pool, "carry_diff", "BTC-USDT-SWAP", 0)
	for i := 0; i < 2; i++ {
		if err := s.ProbeFamilies(ctx); err != nil {
			t.Fatalf("probe %d: %v", i, err)
		}
	}
	if silence, esc := lesionRows(t, pool); silence != 1 || esc != 1 {
		t.Fatalf("同窗去重失效: 族沉默=%d 升件=%d", silence, esc)
	}
}

// TestProbeFamiliesProdLandscapeZeroFalsePositive 锚③现族零误报：
// 生产谱面复刻——唯 xvenue-carry 落病灶（锚②修前态断言：T-0383 未合
// =病灶行在账，合后信号侧自然解）；funding-hedge/quarter_basis_long/
// perp_quarter_spread 窗内活不报，dcd/vol_sell/basis_short/earn/
// lending/surface-gap 族源非 md_signal 出域不报。
func TestProbeFamiliesProdLandscapeZeroFalsePositive(t *testing.T) {
	s, pool := silenceFixture(t)
	ctx := context.Background()
	seedHypoNeeds(t, pool, "xvenue-carry", "xvenue_carry",
		[]string{"carry_diff", "xvenue_spread", "mt5_swap", "depth5", "md_ticks mt5"})
	// 活族窗内行（funding-hedge/quarter_basis_long/perp_quarter_spread）。
	for _, tm := range []string{"funding-hedge", "quarter_basis_long", "perp_quarter_spread"} {
		seedPulseAt(t, pool, tm, 0)
	}
	// 窗内零行族（xvenue-carry 病灶 + 非信号源族出域静默）。
	for _, tm := range []string{"xvenue-carry", "quarter_basis_short", "dcd_call",
		"dcd_put", "vol_sell_strangle", "vol_sell_single_call",
		"earn_term_spread", "lending_spread", "surface-gap"} {
		seedPulseAt(t, pool, tm, 10*24*time.Hour)
	}
	// 信号谱全鲜（生产实态六 kind）。
	for _, k := range []struct{ kind, key string }{
		{"carry_diff", "BTC-USDT-SWAP"}, {"carry_diff", "ETH-USDT-SWAP"},
		{"funding_view", "BTC-USDT-SWAP"}, {"xvenue_spread", "BTC-USDT-SWAP"},
		{"mt5_swap", "BTCUSDm"}, {"depth5", "BTC-USDT-SWAP"}, {"basis_ann", "BTC-USD-261226"},
	} {
		seedSignal(t, pool, k.kind, k.key, 0)
	}
	if err := s.ProbeFamilies(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	silence, esc := lesionRows(t, pool)
	if silence != 1 || esc != 1 {
		t.Fatalf("生产谱面病灶数非 1: 族沉默=%d 升件=%d", silence, esc)
	}
	var scope string
	if err := pool.QueryRow(ctx, `SELECT esc_scope FROM ops_event
		WHERE event='升件' AND esc_class='观测盲'`).Scan(&scope); err != nil {
		t.Fatal(err)
	}
	if scope != "xvenue-carry" {
		t.Fatalf("误报他族: scope=%s", scope)
	}
}

// TestDeriveKinds 映射两链派生单测（零硬编码——①假想卡注册表链
// ②命名规约链；驱动 kind=①链首元）。
func TestDeriveKinds(t *testing.T) {
	kindSet := map[string]bool{
		"carry_diff": true, "xvenue_spread": true, "mt5_swap": true,
		"depth5": true, "funding_view": true, "basis_ann": true,
	}
	needs := map[string][]string{
		"xvenue_carry":            {"carry_diff", "xvenue_spread", "mt5_swap", "depth5", "md_ticks mt5"},
		"stake_carry_trx":         {"fin_capability_surface staking/swap-carry", "trx_onchain|md_earn_rate", "cmc_realized|mt5_swap|funding_view"},
		"lending_spread_template": {}, // 文法卡无 data_needs 键=零贡献（本表滤后缺席）
	}
	cases := []struct {
		template string
		want     []string
	}{
		{"xvenue-carry", []string{"carry_diff", "xvenue_spread", "mt5_swap", "depth5"}}, // ①链驱动序首=carry_diff
		{"stake-carry", []string{"mt5_swap", "funding_view"}},                           // 实例面前缀同认+|替代表求交
		{"funding-hedge", []string{"funding_view"}},                                     // ②链首词元
		{"lending_spread", nil},                                                         // 无 kind 链=出域
		{"dcd_call", nil},
		{"surface-gap", nil},
	}
	for _, c := range cases {
		got := deriveKinds(c.template, kindSet, needs)
		if len(got) != len(c.want) {
			t.Fatalf("%s kinds=%v 非期 %v", c.template, got, c.want)
		}
		for i, k := range c.want {
			if got[i] != k {
				t.Fatalf("%s kinds=%v 首元非 %s", c.template, got, k)
			}
		}
	}
}
