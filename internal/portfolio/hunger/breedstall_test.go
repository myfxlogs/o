package hunger

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// O-39330 锚桩（族卡断流探针——「族卡→评估臂」无主转换感知器）：
// ①candidate 卡零产出+假设锚超龄→升件行落（note 携归因卡+建议语）
// ②方言名归一（roster 'dialect_card' ↔ eval 'dialect-card'）→不误报
// ③有策略行→豁免；④live/disabled 态→不查；⑤24h 同窗去重；
// ⑥窗键缺→fail-closed 报错不升件（防哑针回归锚）；⑦缺假设锚照报载「锚缺」。

func breedStallFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
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

func seedBreedTemplate(t *testing.T, pool *pgxpool.Pool, template, venue, surface, state string, enabled bool) {
	t.Helper()
	disabledReason := ""
	if state == "disabled" {
		disabledReason = "test-ban"
	}
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_opp_template(template,venue,surface,enabled,state,disabled_reason)
		 VALUES($1,$2,$3,$4,$5,nullif($6,''))`,
		template, venue, surface, enabled, state, disabledReason); err != nil {
		t.Fatal(err)
	}
}

func seedBreedHypothesis(t *testing.T, pool *pgxpool.Pool, family string, ageDays float64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_opp_hypothesis(venue,surface,family,ts)
		 VALUES('okx',$3,$1,now()-make_interval(days=>$2))`,
		family, ageDays, "surf-"+family); err != nil {
		t.Fatal(err)
	}
}

func seedBreedEval(t *testing.T, pool *pgxpool.Pool, template string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		 VALUES('TEST-USDT-SWAP',$1,'short_perp_long_spot','rejected','{}','{}')`,
		template); err != nil {
		t.Fatal(err)
	}
}

func seedBreedStrategy(t *testing.T, pool *pgxpool.Pool, templateRef string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		 VALUES('TEST-USDT-SWAP',$1,'paper',100)`, templateRef); err != nil {
		t.Fatal(err)
	}
}

func breedEscCount(t *testing.T, pool *pgxpool.Pool, scope string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ops_event
		 WHERE event='升件' AND esc_class='breed_stall' AND esc_scope=$1`,
		scope).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestBreedStallFires(t *testing.T) {
	s, pool := breedStallFixture(t)
	ctx := context.Background()
	seedBreedTemplate(t, pool, "stall_card_a", "okx", "stall-card-a", "candidate", false)
	seedBreedHypothesis(t, pool, "stall_card_a", 3) // 超 2d 阈
	if err := s.ProbeBreedStall(ctx); err != nil {
		t.Fatal(err)
	}
	if n := breedEscCount(t, pool, "stall_card_a"); n != 1 {
		t.Fatalf("族卡断流升件未落 n=%d", n)
	}
	var note, class, scope string
	if err := pool.QueryRow(ctx,
		`SELECT note,esc_class,esc_scope FROM ops_event
		 WHERE event='升件' AND esc_class='breed_stall'`).Scan(&note, &class, &scope); err != nil {
		t.Fatal(err)
	}
	if scope != "stall_card_a" || class != "breed_stall" {
		t.Fatalf("升件键形错 scope=%q class=%q", scope, class)
	}
	for _, want := range []string{"stall_card_a", "candidate在册", "龄=", "evals=0",
		"strategies=0", "proposals=0", "臂建工单需立", "探针只报不裁"} {
		if !strings.Contains(note, want) {
			t.Fatalf("归因卡缺段 %q —— note=%s", want, note)
		}
	}
	// 同窗去重——二拍零新行
	if err := s.ProbeBreedStall(ctx); err != nil {
		t.Fatal(err)
	}
	if n := breedEscCount(t, pool, "stall_card_a"); n != 1 {
		t.Fatalf("同窗复火 n=%d", n)
	}
}

func TestBreedStallDialectNormalize(t *testing.T) {
	s, pool := breedStallFixture(t)
	ctx := context.Background()
	// roster 下划线方言 ↔ eval 连字符方言——归一命中=产出已立不误报
	seedBreedTemplate(t, pool, "dialect_card", "okx", "dialect-card", "candidate", false)
	seedBreedEval(t, pool, "dialect-card")
	seedBreedHypothesis(t, pool, "dialect_card", 9)
	if err := s.ProbeBreedStall(ctx); err != nil {
		t.Fatal(err)
	}
	if n := breedEscCount(t, pool, "dialect_card"); n != 0 {
		t.Fatalf("方言归一误报 n=%d", n)
	}
	// 反向同验：roster 连字符 ↔ eval 下划线
	seedBreedTemplate(t, pool, "rev-dialect", "okx", "rev-dialect", "candidate", false)
	seedBreedEval(t, pool, "rev_dialect")
	seedBreedHypothesis(t, pool, "rev-dialect", 9)
	if err := s.ProbeBreedStall(ctx); err != nil {
		t.Fatal(err)
	}
	if n := breedEscCount(t, pool, "rev-dialect"); n != 0 {
		t.Fatalf("反向方言归一误报 n=%d", n)
	}
}

func TestBreedStallStrategyExempt(t *testing.T) {
	s, pool := breedStallFixture(t)
	ctx := context.Background()
	seedBreedTemplate(t, pool, "exempt_card", "okx", "exempt-card", "candidate", false)
	seedBreedStrategy(t, pool, "exempt_card") // 策略行在=臂建转换已发生
	seedBreedHypothesis(t, pool, "exempt_card", 9)
	if err := s.ProbeBreedStall(ctx); err != nil {
		t.Fatal(err)
	}
	if n := breedEscCount(t, pool, "exempt_card"); n != 0 {
		t.Fatalf("有策略行误报 n=%d", n)
	}
}

func TestBreedStallSilentNonCandidate(t *testing.T) {
	s, pool := breedStallFixture(t)
	ctx := context.Background()
	// live 态（enabled=true→视图 std_state='live'）+ disabled 态——均不查
	seedBreedTemplate(t, pool, "live_card", "okx", "live-card", "enabled", true)
	seedBreedTemplate(t, pool, "dead_card", "okx", "dead-card", "disabled", false)
	seedBreedHypothesis(t, pool, "live_card", 9)
	seedBreedHypothesis(t, pool, "dead_card", 9)
	if err := s.ProbeBreedStall(ctx); err != nil {
		t.Fatal(err)
	}
	if n := breedEscCount(t, pool, "live_card") + breedEscCount(t, pool, "dead_card"); n != 0 {
		t.Fatalf("非 candidate 态误报 n=%d", n)
	}
}

func TestBreedStallNoAnchorStillFires(t *testing.T) {
	s, pool := breedStallFixture(t)
	ctx := context.Background()
	// 零产出+无假设锚——照报载「锚缺」不虚构龄（spec S0.4 降档先例）
	seedBreedTemplate(t, pool, "anchorless_card", "okx", "anchorless", "candidate", false)
	if err := s.ProbeBreedStall(ctx); err != nil {
		t.Fatal(err)
	}
	if n := breedEscCount(t, pool, "anchorless_card"); n != 1 {
		t.Fatalf("锚缺照报未落 n=%d", n)
	}
	var note string
	if err := pool.QueryRow(ctx,
		`SELECT note FROM ops_event WHERE esc_scope='anchorless_card'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "锚缺") {
		t.Fatalf("锚缺措辞缺席 note=%s", note)
	}
}

func TestBreedStallWithinAge(t *testing.T) {
	s, pool := breedStallFixture(t)
	ctx := context.Background()
	// 锚龄 1d < 2d 阈——未超龄静默
	seedBreedTemplate(t, pool, "young_card", "okx", "young-card", "candidate", false)
	seedBreedHypothesis(t, pool, "young_card", 1)
	if err := s.ProbeBreedStall(ctx); err != nil {
		t.Fatal(err)
	}
	if n := breedEscCount(t, pool, "young_card"); n != 0 {
		t.Fatalf("未超龄误报 n=%d", n)
	}
}

func TestBreedStallKeyMissing(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`DELETE FROM app_config WHERE key='dm.hunger.breed_stall_days'`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	s := New(pool, cfg)
	if err := s.ProbeBreedStall(ctx); err == nil {
		t.Fatal("窗键缺未 fail-closed")
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE esc_class='breed_stall'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("fail-closed 仍升件 n=%d", n)
	}
}
