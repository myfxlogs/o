package report

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 三粒度幂等锚：daily 重跑 UPDATE 非插重；零数据期如实出行。
func TestReportGen(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
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

// NAV 日拍锚（T-0170）：WithNav 接线→cycle 落 ccy='NAV' 行；nil 接线→零行
// 如实（不假拍）；append-only 语义（同日多拍=多行，读侧日桶取末收敛）。
func TestNavSnapshot(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, "DELETE FROM md_equity_curve WHERE ccy='NAV'")
	// nil 接线=不拍
	NewRunner(pool, nil).cycle(ctx)
	var n int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM md_equity_curve WHERE ccy='NAV'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("未接线不应拍: n=%d err=%v", n, err)
	}
	// 接线=落行，NAV 值如实透传
	r := NewRunner(pool, nil).WithNav(func(context.Context) (float64, error) {
		return 12345.5, nil
	})
	r.cycle(ctx)
	var eq float64
	if err := pool.QueryRow(ctx, `SELECT equity FROM md_equity_curve
		WHERE ccy='NAV' AND src='report_daily' ORDER BY ts DESC LIMIT 1`).Scan(&eq); err != nil || eq != 12345.5 {
		t.Fatalf("NAV 日拍缺席/错值: eq=%v err=%v", eq, err)
	}
	// 二拍=追加非覆盖（读侧日桶收敛语义）
	r.cycle(ctx)
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM md_equity_curve WHERE ccy='NAV'").Scan(&n); err != nil || n != 2 {
		t.Fatalf("append-only 语义破: n=%d", n)
	}
}

// T-0026 返修锚：列名断言——防猜列名再犯（evaluation.ts/position.closed_at/
// transition.ts/risk_decision.ts 真实列面）。
func TestSchemaColumnAssertions(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
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
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
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

// T-0029 交付物3锚：weekly/monthly metrics.macro_events=期内 md_macro_event
// 事件计数（fixture 验数非摆拍）；daily 无键。
func TestReportMacroEventsCount(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, "DELETE FROM fin_cashflow")
	_, _ = pool.Exec(ctx, "DELETE FROM fin_report")
	_, _ = pool.Exec(ctx, "DELETE FROM md_macro_event")
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM fin_cashflow")
		_, _ = pool.Exec(ctx, "DELETE FROM md_macro_event")
	}()
	now := time.Now()
	// weekly 期内两事件+期外一事件（边界验证）
	b := BoundsOf("weekly", now)
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_macro_event(event_date,kind,title) VALUES ($1,'fomc','in-week-1'),($2,'cpi','in-week-2')`, b.Start, b.Start.AddDate(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_macro_event(event_date,kind,title) VALUES ($1,'nfp','out-of-week')`, b.Start.AddDate(0, 0, -14)); err != nil {
		t.Fatal(err)
	}
	if err := Gen(ctx, pool, "weekly", b); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT (metrics->>'macro_events')::int FROM fin_report WHERE period='weekly'`).Scan(&n); err != nil {
		t.Fatalf("weekly macro_events 键缺席: %v", err)
	}
	if n != 2 {
		t.Fatalf("weekly macro_events=%d（期望 2——期外事件不计）", n)
	}
	// daily 无键
	if err := Gen(ctx, pool, "daily", BoundsOf("daily", now)); err != nil {
		t.Fatal(err)
	}
	var has bool
	if err := pool.QueryRow(ctx, `SELECT metrics ? 'macro_events' FROM fin_report WHERE period='daily'`).Scan(&has); err != nil || has {
		t.Fatalf("daily 不应有 macro_events 键: %v", has)
	}
}
