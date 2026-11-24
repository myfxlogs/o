package ledger

import (
	"context"
	"math"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// O-36327 锚桩（构性残差豁免跟踪化；O-36422 摘 dcd_mtm_drift 面）：
// ①earn_accrual_gap=interest 应计累计；
// ②|跟踪−锚|>max(2×锚,$25)→破带贡献归零+升件；
// ③resolver 未注册（含退役键）/静态行照旧求和。

func trkFixture(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return pool
}

func trkRunner(pool *pgxpool.Pool) *Runner {
	return &Runner{pool: pool,
		rej: rejects.New("ledger", "diff_alert", "missing_param", "bal_missing", "write_err",
			"recon_probe_failed", "postmortem_anomaly", "diag_task_err")}
}

func seedResid(t *testing.T, pool *pgxpool.Pool, venue, ccy string, amt float64, resolver string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_known_residual(venue,ccy,amount_usd,reason,ticket_ref,expires_at,resolver)
		 VALUES($1,$2,$3,'test','t',now()+interval '30 days',$4)`,
		venue, ccy, amt, resolver); err != nil {
		t.Fatal(err)
	}
}

// O-36422：dcd_mtm_drift resolver 退役（custody 纯币量裁示——USD 折算
// 漂移退 NAV 层）——DCD 基线/mark 桩件随件摘除；earn_accrual_gap/静态/
// 破带/回退语义面不变。

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestTrackedExemptEarnGap(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	r := trkRunner(pool)
	seedResid(t, pool, "tkv", "TKB", 10, "earn_accrual_gap")
	seedCash(t, pool, "tkv", "TKB", "interest", 3.5)
	seedCash(t, pool, "tkv", "TKB", "interest", 2.25)
	seedCash(t, pool, "tkv", "TKB", "fee", -1) // 非 interest 不计
	total, parts, err := r.trackedExempt(ctx, "tkv", "TKB")
	if err != nil {
		t.Fatal(err)
	}
	if parts[0].Mode != "tracked" || !near(parts[0].Value, 5.75) {
		t.Fatalf("earn 跟踪值应=5.75 got mode=%s v=%.6f", parts[0].Mode, parts[0].Value)
	}
	if !near(total, 5.75) {
		t.Fatalf("豁免额应=5.75 got=%.6f", total)
	}
}

func TestTrackedExemptBreach(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	r := trkRunner(pool)
	// 锚 $10 跟踪 $40：|40-10|=30 > max(20,25)=25 → 破带归零+升件
	seedResid(t, pool, "tkv", "TKC", 10, "earn_accrual_gap")
	seedCash(t, pool, "tkv", "TKC", "interest", 40)
	total, parts, err := r.trackedExempt(ctx, "tkv", "TKC")
	if err != nil {
		t.Fatal(err)
	}
	if parts[0].Mode != "breach" || parts[0].Value != 0 {
		t.Fatalf("破带应归零 got mode=%s v=%.4f", parts[0].Mode, parts[0].Value)
	}
	if total != 0 {
		t.Fatalf("破带豁免额应=0 got=%.4f", total)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='对账'
		AND esc_scope='residual_model_breach_tkv_TKC_earn_accrual_gap'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("破带升件未落 n=%d", n)
	}
}

func TestTrackedExemptFallback(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	r := trkRunner(pool)
	// 未注册键（含退役 dcd_mtm_drift 旧键形）+静态行→双双回退静态锚
	seedResid(t, pool, "tkv", "TKD", 7, "no_such_resolver")
	seedResid(t, pool, "tkv", "TKD", 9, "dcd_mtm_drift:NOPE") // O-36422 退役键=未注册回退
	seedResid(t, pool, "tkv", "TKD", 3, "")                   // 静态行
	total, parts, err := r.trackedExempt(ctx, "tkv", "TKD")
	if err != nil {
		t.Fatal(err)
	}
	if total != 19 {
		t.Fatalf("回退+静态合计应=19 got=%.4f", total)
	}
	if parts[0].Mode != "fallback" || parts[1].Mode != "fallback" || parts[2].Mode != "static" {
		t.Fatalf("mode 序错: %+v", parts)
	}
}

func seedCash(t *testing.T, pool *pgxpool.Pool, venue, ccy, kind string, amt float64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_cashflow(venue,ccy,kind,amount,source_ref,source_table,ts,occurred_at)
		 VALUES($1,$2,$3,$4,1,'test',now(),now())`, venue, ccy, kind, amt); err != nil {
		t.Fatal(err)
	}
}
