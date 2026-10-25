// run_test.go —— 文法枚举锚（T-0120）：幂等结构键不重生+面关=族不生+
// 缺 executor 族假想卡自动落。
package grammar

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
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, NewRunner(pool, cfg)
}

// 锚①：文法枚举幂等——双拍同结果不重生行（UNIQUE 结构键撞行幂等）。
func TestEnumerateIdempotent(t *testing.T) {
	ctx, r := setup(t)
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n1 int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_template
		WHERE state='candidate'`).Scan(&n1); err != nil {
		t.Fatal(err)
	}
	if n1 == 0 {
		t.Fatal("文法枚举零候选——面在族不生")
	}
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n2 int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_template
		WHERE state='candidate'`).Scan(&n2); err != nil {
		t.Fatal(err)
	}
	if n1 != n2 {
		t.Fatalf("幂等违例: 首拍 %d 行次拍 %d 行", n1, n2)
	}
}

// 锚②：面关=族不生——mt5.catalog enabled=false 时 mt5_swap_carry 不落。
func TestDisabledFaceNoFamily(t *testing.T) {
	ctx, r := setup(t)
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_template
		WHERE template='mt5_swap_carry' AND state='candidate'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("mt5 catalog 关态——mt5_swap_carry 候选不应落")
	}
	// 假想卡锚：缺 executor 族自动落 requirements.executor。
	var nH int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_hypothesis
		WHERE requirements->>'executor' IS NOT NULL`).Scan(&nH); err != nil {
		t.Fatal(err)
	}
	if nH < 3 {
		t.Fatalf("缺 executor 假想卡应 ≥3 族（lending/earn/quarter_basis/vol_sell）: %d", nH)
	}
}
