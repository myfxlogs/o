package analyst

import (
	"context"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *Runner) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, NewRunner(pool, cfg)
}

// 锚②LLM 缝：桩 LLM 产判词→md_regime_fact+memo 落；枚举域外词拒行不落。
func TestAnalyzeStubLLM(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_ctx_fact(source,kind,payload)
		VALUES('manual','news','{"t":"fomc 鹰派"}')`); err != nil {
		t.Fatal(err)
	}
	r.llm = func(ctx context.Context, sys, user string) (string, error) {
		return `[{"dim":"rates","stance":"risk_off","confidence":0.7,"evidence":[1],"memo":"CPI 超预期引证据#1——数据边界：仅一条"},
			{"dim":"junk","stance":"moon","confidence":9,"evidence":[],"memo":"枚举域外词应拒"}]`, nil
	}
	if err := r.analyze(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_regime_fact WHERE stance='risk_off'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("risk_off 判词应=1（枚举域外拒行）: %d", n)
	}
	var memo string
	if err := r.pool.QueryRow(ctx, `SELECT memo FROM md_regime_memo LIMIT 1`).Scan(&memo); err != nil || memo == "" {
		t.Fatalf("memo 行缺: %v", err)
	}
}

// 锚③LLM 缝缺=守位：采集照常、产出行缺席如实。
func TestNoLLMSilent(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_ctx_fact(source,kind,payload) VALUES('manual','news','{}')`); err != nil {
		t.Fatal(err)
	}
	if err := r.analyze(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_regime_fact`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("缝缺应零产出: %d", n)
	}
}

// TestAnalyzeBacklogFIFO O-30635 游标跳读根修锚：积压>50 时 DESC+LIMIT 会把
// 水位钉到最新 50 底、中段行永跳；ASC 头吃=批内 id 序=消费序，水位连贯推进、
// 积压下拍续扫。55 行两拍应覆盖首尾水位锚（ctx_max_id=第50行id 与 max id）。
func TestAnalyzeBacklogFIFO(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_ctx_fact(source,kind,payload)
		SELECT 't','news','{}' FROM generate_series(1,55)`); err != nil {
		t.Fatal(err)
	}
	var id50, idMax int64
	if err := r.pool.QueryRow(ctx,
		`SELECT id FROM md_ctx_fact ORDER BY id OFFSET 49 LIMIT 1`).Scan(&id50); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT max(id) FROM md_ctx_fact`).Scan(&idMax); err != nil {
		t.Fatal(err)
	}
	r.llm = func(ctx context.Context, sys, user string) (string, error) {
		return `[{"dim":"rates","stance":"neutral","confidence":0.5,"evidence":[],"memo":"m"}]`, nil
	}
	if err := r.analyze(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.analyze(ctx); err != nil {
		t.Fatal(err)
	}
	// 两拍各产一判词行——evidence.ctx_max_id 应分别=第50行 id（首拍水位）
	// 与 max id（次拍水位）：中段 51..54 行由次拍拾回=无跳读。
	var n50, nMax int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_regime_fact
		WHERE (evidence->>'ctx_max_id')::bigint=$1`, id50).Scan(&n50); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_regime_fact
		WHERE (evidence->>'ctx_max_id')::bigint=$1`, idMax).Scan(&nMax); err != nil {
		t.Fatal(err)
	}
	if n50 != 1 || nMax != 1 {
		t.Fatalf("两拍水位锚缺: id50=%d→%d行 idMax=%d→%d行（DESC 跳读=中段永丢）",
			id50, n50, idMax, nMax)
	}
	// 第三拍应空扫（55<50+50 全消化）——零新判词行。
	before := 0
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_regime_fact`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err := r.analyze(ctx); err != nil {
		t.Fatal(err)
	}
	var after int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_regime_fact`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("积压清零后第三拍不应再产出: %d→%d", before, after)
	}
}
