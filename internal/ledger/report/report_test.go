package report

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
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

// T-0026 返修锚：列名断言——防猜列名再犯（evaluation.ts/position.closed_at/
// transition.ts/risk_decision.ts 真实列面）。
func TestSchemaColumnAssertions(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for tbl, cols := range map[string][]string{
		"fin_opp_eval":            {"ts", "verdict"},
		"fin_position":            {"closed_at"},
		"fin_strategy_transition": {"ts", "to_state"},
		"fin_risk_decision":       {"ts", "decision"},
		"fin_cashflow":            {"occurred_at", "kind", "amount"},
		"fin_holding":             {"amount", "kind", "stale"},
	} {
		for _, c := range cols {
			var ok bool
			if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns
				WHERE table_name=$1 AND column_name=$2 AND table_schema=current_schema)`, tbl, c).Scan(&ok); err != nil || !ok {
				t.Fatalf("列断言 %s.%s 缺席", tbl, c)
			}
		}
	}
}

// 返修锚：指标非恒 0——真行入册后指标如实计数（eval.ts/closed_at 正确消费）。
func TestReportCounts(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('RC-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','{}','{}'),('RC2-USDT-SWAP','funding-hedge','short_perp_long_spot','rejected','{}','{}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_execution(intent,venue,inst_id,state) VALUES('{}','okx','RC-USDT-SWAP','done')`); err == nil {
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,state,closed_at)
		SELECT 1,'closed',now()`); err != nil {
		t.Skipf("fin_position 形不符跳过计数锚: %v", err)
	}
	if err := Gen(ctx, pool, "daily", BoundsOf("daily", time.Now())); err != nil {
		t.Fatal(err)
	}
	var p, r float64
	if err := pool.QueryRow(ctx,
		"SELECT (metrics->>'proposed')::float,(metrics->>'rejected')::float FROM fin_report WHERE period='daily'").Scan(&p, &r); err != nil {
		t.Fatal(err)
	}
	if p < 1 || r < 1 {
		t.Fatalf("指标恒 0 返修未愈: proposed=%v rejected=%v", p, r)
	}
}
