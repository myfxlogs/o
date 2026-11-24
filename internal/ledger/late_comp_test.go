package ledger

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// O-37868 锚桩（迟簿双计自愈闸）：armed 键上 occurred_at<arm_ts ∧
// ts>arm_ts 的 custody 现金流逐行补 native_baseline 负腿；早账/新账/
// earn 内部划转/未武装键/重放幂等全锚。

func seedBaseline(t *testing.T, pool *pgxpool.Pool, venue, ccy string, amt float64, ts time.Time) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO fin_risk_decision
		(intent,decision,reason,config_gen,kind,ts)
		VALUES(jsonb_build_object('op','native_baseline','venue',$1::text,'ccy',$2::text,'amt',$3::float8),
			'allow','allow',1,'open',$4)`, venue, ccy, amt, ts); err != nil {
		t.Fatal(err)
	}
}

func seedCashAt(t *testing.T, pool *pgxpool.Pool, venue, ccy, kind string, amt float64, ts, occurred time.Time) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO fin_cashflow
		(venue,ccy,kind,amount,source_ref,source_table,ts,occurred_at)
		VALUES($1,$2,$3,$4,1,'test',$5,$6) RETURNING id`,
		venue, ccy, kind, amt, ts, occurred).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func baselineSum(t *testing.T, pool *pgxpool.Pool, venue, ccy string) float64 {
	t.Helper()
	var s float64
	if err := pool.QueryRow(context.Background(), `SELECT COALESCE(sum((intent->>'amt')::numeric),0)
		FROM fin_risk_decision WHERE intent->>'op'='native_baseline'
		  AND intent->>'venue'=$1 AND intent->>'ccy'=$2`, venue, ccy).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLateBaselineComp(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	r := trkRunner(pool)
	arm := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	before := arm.Add(-24 * time.Hour)
	after := arm.Add(time.Hour)

	// 武装键 tkv/TKL：基线 +100（arm_ts=arm）
	seedBaseline(t, pool, "tkv", "TKL", 100, arm)
	// 迟到账（occurred<arm, ts>arm）→ 应补 -50 负腿
	lateID := seedCashAt(t, pool, "tkv", "TKL", "deposit", 50, after, before)
	// 早到账（ts<arm——武装时已入 net 不双计）→ 不补
	seedCashAt(t, pool, "tkv", "TKL", "deposit", 7, arm.Add(-time.Minute), before)
	// 新账（occurred>arm——venue 同步增长非双计）→ 不补
	seedCashAt(t, pool, "tkv", "TKL", "deposit", 9, after, after)
	// earn_in 迟到账（arm/reconcile cf 口径排除域）→ 不补
	seedCashAt(t, pool, "tkv", "TKL", "earn_in", 30, after, before)
	// 未武装键 tkv/TKU 迟到账 → 不补（无基线可双计）
	seedCashAt(t, pool, "tkv", "TKU", "deposit", 60, after, before)

	if err := r.compensateLateBaselineBooks(ctx); err != nil {
		t.Fatal(err)
	}
	if got := baselineSum(t, pool, "tkv", "TKL"); got != 50 {
		t.Fatalf("补偿后基线Σ应=50（100-50）got=%.4f", got)
	}
	// 补偿腿 lateSrc 锚=迟到账 id；腿数恰 1
	var legs, srcOK int
	if err := pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE intent->>'lateSrc'=$1)
		FROM fin_risk_decision WHERE intent->>'op'='native_baseline'
		  AND intent->>'venue'='tkv' AND intent->>'ccy'='TKL'`, fmt.Sprint(lateID)).Scan(&legs, &srcOK); err != nil {
		t.Fatal(err)
	}
	if legs != 2 || srcOK != 1 {
		t.Fatalf("补偿腿应恰1条且lateSrc锚在案 legs=%d srcOK=%d", legs, srcOK)
	}
	// 未武装键零补偿
	if got := baselineSum(t, pool, "tkv", "TKU"); got != 0 {
		t.Fatalf("未武装键不应有基线腿 got=%.4f", got)
	}
	// 重放幂等——再跑零新增
	if err := r.compensateLateBaselineBooks(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_risk_decision
		WHERE intent->>'op'='native_baseline' AND intent->>'venue'='tkv' AND intent->>'ccy'='TKL'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("重放应幂等腿数仍=2 got=%d", n)
	}
}

func TestLateBaselineCompMulti(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	r := trkRunner(pool)
	arm := time.Now().Add(-3 * time.Hour).UTC().Truncate(time.Second)
	before := arm.Add(-48 * time.Hour)
	after := arm.Add(30 * time.Minute)

	seedBaseline(t, pool, "tkv", "TKM", 1000, arm)
	// 两笔迟到账异号（deposit+/withdraw-）→ 各补反号腿 Σ=1000-(50-20)=970
	seedCashAt(t, pool, "tkv", "TKM", "deposit", 50, after, before)
	seedCashAt(t, pool, "tkv", "TKM", "withdraw", -20, after, before)

	if err := r.compensateLateBaselineBooks(ctx); err != nil {
		t.Fatal(err)
	}
	if got := baselineSum(t, pool, "tkv", "TKM"); got != 970 {
		t.Fatalf("异号迟到账补偿Σ应=970 got=%.4f", got)
	}
}
