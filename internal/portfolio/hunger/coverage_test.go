// coverage_test.go —— T-0392 锚③桩测：腿缺行情→ops_event `覆盖失明` 行+
// 观测盲升件断言；有行情零误报断言；宇宙外虚构键不求供零误报；同窗 24h
// 按键去重。
package hunger

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// coverFixture 探针桩景（fin 迁移——okx.symbols 种 BTC/ETH 现货+SWAP 四键
// 宇宙，fin_0004 在册）。
func coverFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
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

func seedCoverStrategy(t *testing.T, pool *pgxpool.Pool, inst, tmpl, state string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd) VALUES($1,$2,$3,100)`,
		inst, tmpl, state); err != nil {
		t.Fatal(err)
	}
}

// seedTick md_ticks 行定龄（age=0 鲜行；>coverFreshMin 窗外=陈行）。
func seedTick(t *testing.T, pool *pgxpool.Pool, inst string, age time.Duration) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO md_ticks(ts,inst_id,last,bid,ask)
		VALUES(now()-make_interval(secs=>$2),$1,1,1,1)
		ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts`,
		inst, age.Seconds()); err != nil {
		t.Fatal(err)
	}
}

// coverRows 探针行数（覆盖失明行/观测盲升件行——断言公共面）。
func coverRows(t *testing.T, pool *pgxpool.Pool) (cover, esc int) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='覆盖失明' AND ticket='T-0392'`).Scan(&cover); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND ticket='T-0392'`).Scan(&esc); err != nil {
		t.Fatal(err)
	}
	return cover, esc
}

// TestProbeLegCoverageBlindLeg 形①：funding-hedge live 策略 BTC-USDT-SWAP——
// inst 本键鲜 tick 在、派生现货腿 BTC-USDT 零行→覆盖失明行+观测盲升件
// （T-0392 病灶原型复刻——「有物无人喂」静默盲）。
func TestProbeLegCoverageBlindLeg(t *testing.T) {
	s, pool := coverFixture(t)
	ctx := context.Background()
	seedCoverStrategy(t, pool, "BTC-USDT-SWAP", "funding-hedge", "live")
	seedTick(t, pool, "BTC-USDT-SWAP", 0) // 本键有供——独现货腿盲
	if err := s.ProbeLegCoverage(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	cover, esc := coverRows(t, pool)
	if cover != 1 || esc != 1 {
		t.Fatalf("失明行缺失: 覆盖失明=%d 升件=%d", cover, esc)
	}
	var note, scope string
	if err := pool.QueryRow(ctx, `SELECT note,esc_scope FROM ops_event
		WHERE event='升件' AND esc_class='观测盲'`).Scan(&note, &scope); err != nil {
		t.Fatal(err)
	}
	if scope != "BTC-USDT" {
		t.Fatalf("esc_scope 非盲腿键: %s", scope)
	}
	for _, frag := range []string{"BTC-USDT", "覆盖失明", "md_ticks"} {
		if !strings.Contains(note, frag) {
			t.Fatalf("升件 note 缺 %q: %s", frag, note)
		}
	}
}

// TestProbeLegCoverageFedZeroFalsePositive 形②：腿键全鲜——现货腿与本键
// 双供零误报。
func TestProbeLegCoverageFedZeroFalsePositive(t *testing.T) {
	s, pool := coverFixture(t)
	ctx := context.Background()
	seedCoverStrategy(t, pool, "BTC-USDT-SWAP", "funding-hedge", "live")
	seedTick(t, pool, "BTC-USDT-SWAP", 0)
	seedTick(t, pool, "BTC-USDT", 0)
	if err := s.ProbeLegCoverage(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if cover, esc := coverRows(t, pool); cover != 0 || esc != 0 {
		t.Fatalf("有行情误报: 覆盖失明=%d 升件=%d", cover, esc)
	}
}

// TestProbeLegCoverageFictionalNotRequired 形③：虚构键宇宙外——FAKE9-USDT
// 派生入腿键集但非订阅宇宙员，不求供价不炸（宇宙闸滤除断言）。
func TestProbeLegCoverageFictionalNotRequired(t *testing.T) {
	s, pool := coverFixture(t)
	ctx := context.Background()
	seedCoverStrategy(t, pool, "FAKE9-USDT-SWAP", "funding-hedge", "live")
	// FAKE9-USDT-SWAP 亦宇宙外——本键同滤；零 md_ticks 行零误报。
	if err := s.ProbeLegCoverage(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if cover, esc := coverRows(t, pool); cover != 0 || esc != 0 {
		t.Fatalf("宇宙外误报: 覆盖失明=%d 升件=%d", cover, esc)
	}
}

// TestProbeLegCoverageStaleTick 形④：腿键有行但陈（>coverFreshMin 窗）——
// 陈行=失明同判（行情死非零行才盲）。
func TestProbeLegCoverageStaleTick(t *testing.T) {
	s, pool := coverFixture(t)
	ctx := context.Background()
	seedCoverStrategy(t, pool, "BTC-USDT-SWAP", "funding-hedge", "live")
	seedTick(t, pool, "BTC-USDT-SWAP", 0)
	seedTick(t, pool, "BTC-USDT", 30*time.Minute)
	if err := s.ProbeLegCoverage(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if cover, esc := coverRows(t, pool); cover != 1 || esc != 1 {
		t.Fatalf("陈行失明未捕: 覆盖失明=%d 升件=%d", cover, esc)
	}
}

// TestProbeLegCoverageStarExpanded T-0416 宇宙扩件接线锚：okx.symbols=["*"]
// +供给件挂线→订阅宇宙实读探针复电（病灶形=生产连刷「宇宙扩件探针未挂」；
// 修后同形①判词一致——宇宙外滤除判据不动）。
func TestProbeLegCoverageStarExpanded(t *testing.T) {
	s, pool := coverFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE app_config SET value='["*"]' WHERE key='okx.symbols'`); err != nil {
		t.Fatal(err)
	}
	if err := s.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// 红桩断言（病灶存在复刻）：未挂供给件=* 形 fail-closed。
	if err := s.ProbeLegCoverage(ctx); err == nil ||
		!strings.Contains(err.Error(), "宇宙扩件探针未挂") {
		t.Fatalf("未挂供给件应 fail-closed: %v", err)
	}
	// 挂线实读→探针复电——目录员腿照常判。
	s.AttachUniverse(func(context.Context) ([]string, error) {
		return []string{"BTC-USDT-SWAP", "BTC-USDT"}, nil
	})
	seedCoverStrategy(t, pool, "BTC-USDT-SWAP", "funding-hedge", "live")
	seedTick(t, pool, "BTC-USDT-SWAP", 0)
	if err := s.ProbeLegCoverage(ctx); err != nil {
		t.Fatalf("供给件挂线后应复电: %v", err)
	}
	if cover, esc := coverRows(t, pool); cover != 1 || esc != 1 {
		t.Fatalf("复电后失明腿未捕: 覆盖失明=%d 升件=%d", cover, esc)
	}
}

// TestSubUniverseStarFailClosed 供给件读败=fail-closed 升件不虚构（探针
// 哲学不动——修后语义保留锚）。
func TestSubUniverseStarFailClosed(t *testing.T) {
	s, pool := coverFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE app_config SET value='["*"]' WHERE key='okx.symbols'`); err != nil {
		t.Fatal(err)
	}
	if err := s.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	s.AttachUniverse(func(context.Context) ([]string, error) {
		return nil, errors.New("instruments 面断")
	})
	if _, err := s.subUniverse(ctx); err == nil ||
		!strings.Contains(err.Error(), "宇宙扩读败") {
		t.Fatalf("供给件读败应 fail-closed: %v", err)
	}
}

// TestProbeLegCoverageDedup24h 同窗去重：同盲键再拍——24h 内至多一报。
func TestProbeLegCoverageDedup24h(t *testing.T) {
	s, pool := coverFixture(t)
	ctx := context.Background()
	seedCoverStrategy(t, pool, "BTC-USDT-SWAP", "funding-hedge", "live")
	seedTick(t, pool, "BTC-USDT-SWAP", 0)
	for i := 0; i < 2; i++ {
		if err := s.ProbeLegCoverage(ctx); err != nil {
			t.Fatalf("probe %d: %v", i, err)
		}
	}
	if cover, esc := coverRows(t, pool); cover != 1 || esc != 1 {
		t.Fatalf("同窗去重失效: 覆盖失明=%d 升件=%d", cover, esc)
	}
}
