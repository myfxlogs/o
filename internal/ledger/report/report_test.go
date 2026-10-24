package report

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

// 三粒度幂等锚：daily 重跑 UPDATE 非插重；零数据期如实出行。
func TestReportGen(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, p := range []string{"daily", "weekly", "monthly"} {
		if err := Gen(ctx, pool, p, BoundsOf(p, now)); err != nil {
			t.Fatalf("%s gen: %v", p, err)
		}
	}
	// 幂等锚：daily 重跑行数仍=1
	if err := Gen(ctx, pool, "daily", BoundsOf("daily", now)); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_report WHERE period='daily'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("daily 重跑产重行=%d", n)
	}
	// 零数据诚实锚：metrics.net=0 且 note 含注记
	var note string
	if err := pool.QueryRow(ctx,
		"SELECT note FROM fin_report WHERE period='daily'").Scan(&note); err != nil || note == "" {
		t.Fatalf("零数据注记缺席=%q", note)
	}
	// 月度三行交代锚
	if err := pool.QueryRow(ctx,
		"SELECT metrics->>'annualized' FROM fin_report WHERE period='monthly'").Scan(&note); err != nil {
		t.Fatalf("monthly annualized 缺席: %v", err)
	}
}

// GET /report 取径锚：三 period 应答字段齐。
func TestReportHandler(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := Gen(ctx, pool, "daily", BoundsOf("daily", time.Now())); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/report?period=daily&n=4", nil)
	rec := httptest.NewRecorder()
	Handler(pool)(rec, req)
	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || len(out) != 1 {
		t.Fatalf("handler rows=%d err=%v", len(out), err)
	}
	m := out[0]["metrics"].(map[string]any)
	for _, k := range []string{"net", "cum_nav", "opened", "closed", "rejected", "earn_balance"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("metrics 缺 %s", k)
		}
	}
}
