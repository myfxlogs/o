package notify

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func setupRegime(t *testing.T) (context.Context, *Regime, *fakeSender) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_regime_fact(id,dim,stance,confidence,evidence,valid_until)
		VALUES(1,'crypto','risk_off',0.6,'{"ctx_max_id":52}',now()+interval '6 hours')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_regime_memo(id,fact_id,memo) VALUES(1,1,'加密偏谨慎引证据#31')`); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSender{}
	r := NewRegime(pool, fs, func() string { return "c1" }, func() bool { return true })
	return ctx, r, fs
}

// 锚：开态 Scan 发一条含 stance+memo；push 锚落→重扫零推不重推。
func TestRegimePush(t *testing.T) {
	ctx, r, fs := setupRegime(t)
	if err := r.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 || !strings.Contains(fs.sent[0], "风险偏好下行") || !strings.Contains(fs.sent[0], "加密偏谨慎") {
		t.Fatalf("推送文不符: %+v", fs.sent)
	}
	// 重扫零推（幂等锚已落）
	if err := r.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 {
		t.Fatalf("重扫不应重推: %d", len(fs.sent))
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_regime_push`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("push 锚应=1: %d", n)
	}
}

// 锚：关态 Scan 落锚不发（fail-closed 语义——下拍不重扫）。
func TestRegimeOff(t *testing.T) {
	ctx, r, fs := setupRegime(t)
	r.on = func() bool { return false }
	if err := r.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 0 {
		t.Fatalf("关态不应发: %d", len(fs.sent))
	}
}
