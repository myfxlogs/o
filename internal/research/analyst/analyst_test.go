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
