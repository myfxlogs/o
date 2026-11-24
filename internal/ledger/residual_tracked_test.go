package ledger

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// O-36327 锚桩（构性残差豁免跟踪化）：
// ①resolver 行豁免额=现算非锚（venue 名义量差×mark 随 BTC 漂→跟踪值随动）；
// ②earn_accrual_gap=interest 应计累计；
// ③|跟踪−锚|>max(2×锚,$25)→破带贡献归零+升件；
// ④resolver 未注册/读败/读件缺席→回退静态锚；⑤静态行照旧求和；
// ⑥reconcile 级：跟踪值随 mark 动——带内静默、破带照报。

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

func seedDCDBaseline(t *testing.T, pool *pgxpool.Pool, ordID, venue, ccy string, amt float64) {
	t.Helper()
	intent := `{"op":"dcd_baseline","ordId":"` + ordID + `","venue":"` + venue +
		`","ccy":"` + ccy + `","amt":` + fmtFloat(amt) + `,"kind":"transfer"}`
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		 VALUES($1::jsonb,'allow','allow',1)`, intent); err != nil {
		t.Fatal(err)
	}
}

func fmtFloat(f float64) string {
	return strings.TrimRight(strings.TrimRight(
		strconv.FormatFloat(f, 'f', 8, 64), "0"), ".")
}

func seedMark(t *testing.T, pool *pgxpool.Pool, inst string, px float64, ageMin int) {
	t.Helper()
	// md_ticks PK=(venue,inst_id) 状态表——upsert 复演「最新 mark」语义。
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now()-make_interval(mins=>$3),'okx',$1,$2)
		 ON CONFLICT (venue,inst_id) DO UPDATE SET last=EXCLUDED.last,ts=EXCLUDED.ts`,
		inst, px, ageMin); err != nil {
		t.Fatal(err)
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// dcdOrdStub venue dcd 单桩——ordID→(notionalSz,err)；nil map=读件缺席。
func dcdOrdStub(ords map[string]string, err error) func(context.Context) ([]DcdOrderView, error) {
	return func(ctx context.Context) ([]DcdOrderView, error) {
		if err != nil {
			return nil, err
		}
		out := make([]DcdOrderView, 0, len(ords))
		for id, sz := range ords {
			out = append(out, DcdOrderView{OrdID: id, State: "live",
				NotionalCcy: "BTC", NotionalSz: sz})
		}
		return out, nil
	}
}

func TestTrackedExemptDCDDrift(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	r := trkRunner(pool)
	// 锚 $100；基线 0.5BTC，venue 名义 0.6BTC→币量差 0.1×mark。
	// mark=1000→跟踪=$100（带内）；mark=500→跟踪=$50——BTC 价漂→跟踪随动。
	seedResid(t, pool, "tkv", "TKA", 100, "dcd_mtm_drift:ORDX")
	seedDCDBaseline(t, pool, "ORDX", "tkv", "BTC", 0.5)
	seedMark(t, pool, "BTC-USDT", 1000, 1)
	r.dcdOrd = dcdOrdStub(map[string]string{"ORDX": "0.6"}, nil)

	_, parts, err := r.trackedExempt(ctx, "tkv", "TKA")
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 || parts[0].Mode != "tracked" {
		t.Fatalf("mode 非 tracked: %+v", parts)
	}
	if !near(parts[0].Value, 100) {
		t.Fatalf("mark=1000 跟踪值应=100 got=%.6f", parts[0].Value)
	}
	// BTC 价漂→跟踪值随动=$50（0.1×500）
	seedMark(t, pool, "BTC-USDT", 500, 0)
	total, parts, err := r.trackedExempt(ctx, "tkv", "TKA")
	if err != nil {
		t.Fatal(err)
	}
	if parts[0].Mode != "tracked" || !near(parts[0].Value, 50) {
		t.Fatalf("价漂后跟踪值应=50 got mode=%s v=%.6f", parts[0].Mode, parts[0].Value)
	}
	// 破带检查：|50-100|=50 ≤ max(200,25)=200——带内照豁
	if !near(total, 50) {
		t.Fatalf("豁免额应=跟踪值50 got=%.6f", total)
	}
}

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
	// 未注册键+已注册但源缺（无 dcd 基线）→双双回退静态锚
	seedResid(t, pool, "tkv", "TKD", 7, "no_such_resolver")
	seedResid(t, pool, "tkv", "TKD", 9, "dcd_mtm_drift:NOPE")
	seedResid(t, pool, "tkv", "TKD", 3, "") // 静态行
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

func TestTrackedExemptDCDReadFail(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	// resolver 读败三态全回退静态锚：读件缺席/venue 读 err/单缺席。
	r := trkRunner(pool)
	seedResid(t, pool, "tkv", "TKE", 11, "dcd_mtm_drift:ORDY")
	seedDCDBaseline(t, pool, "ORDY", "tkv", "BTC", 0.5)
	seedMark(t, pool, "BTC-USDT", 1000, 1)

	// ①读件未挂接（nil）→回退
	_, parts, err := r.trackedExempt(ctx, "tkv", "TKE")
	if err != nil || parts[0].Mode != "fallback" || parts[0].Value != 11 {
		t.Fatalf("读件缺席应回退锚11: %+v err=%v", parts, err)
	}
	// ②venue 读 err→回退
	r.dcdOrd = dcdOrdStub(nil, fmt.Errorf("venue 读败"))
	_, parts, err = r.trackedExempt(ctx, "tkv", "TKE")
	if err != nil || parts[0].Mode != "fallback" {
		t.Fatalf("venue 读败应回退: %+v err=%v", parts, err)
	}
	// ③单缺席→回退
	r.dcdOrd = dcdOrdStub(map[string]string{"OTHER": "1"}, nil)
	_, parts, err = r.trackedExempt(ctx, "tkv", "TKE")
	if err != nil || parts[0].Mode != "fallback" {
		t.Fatalf("单缺席应回退: %+v err=%v", parts, err)
	}
	// ④非名义在持态→回退（桩恒 live——直测 resolver 态闸）
	r.dcdOrd = func(ctx context.Context) ([]DcdOrderView, error) {
		return []DcdOrderView{{OrdID: "ORDY", State: "settled",
			NotionalCcy: "BTC", NotionalSz: "0.6"}}, nil
	}
	_, parts, err = r.trackedExempt(ctx, "tkv", "TKE")
	if err != nil || parts[0].Mode != "fallback" {
		t.Fatalf("settled 态应回退: %+v err=%v", parts, err)
	}
}

// 验收①+③合并锚：reconcile 级——跟踪值随 mark 动：mark 高位豁免盖差
// 静默，mark 下移跟踪值缩水破阈照报（不告警→告警转换=跟踪随动实证，
// 非静态锚恒盖亦非恒报）。
func TestReconcileTrackedDCDFollowsMark(t *testing.T) {
	ctx, r := setup(t)
	pool := r.pool
	if _, err := pool.Exec(ctx, "DELETE FROM fin_known_residual"); err != nil {
		t.Fatal(err)
	}
	// 基线 0.5BTC venue=okx；venue 名义 0.6BTC→差 0.1BTC×mark。
	seedDCDBaseline(t, pool, "ORDZ", "okx", "BTC", 0.5)
	seedMark(t, pool, "BTC-USDT", 1000, 1)
	seedResid(t, pool, "okx", "USDT", 100, "dcd_mtm_drift:ORDZ")
	var ven map[[2]string]float64
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, VenueParts, error) { return ven, nil, nil })
	r.WithDcdOrders(dcdOrdStub(map[string]string{"ORDZ": "0.6"}, nil))
	// 账本 USDT 净=100；venue USDT=40→diff=60。跟踪豁免=0.1×1000=$100
	// →60≤0.5+100 静默（静态锚 100 同值——改 mark 见真章）。
	if _, err := Post(ctx, pool, Entry{Kind: "deposit", Venue: "okx", Ccy: "USDT",
		Amount: 100, SourceRef: 1, SourceTable: "t"}); err != nil {
		t.Fatal(err)
	}
	ven = map[[2]string]float64{{"okx", "USDT"}: 40, {"okx", "BTC"}: 0.5}
	r.rej = rejects.New("ledger", "diff_alert", "missing_param", "bal_missing", "write_err",
		"recon_probe_failed", "postmortem_anomaly", "diag_task_err")
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if n := r.rej.Snapshot()["ledger/diff_alert"]; n != 0 {
		t.Fatalf("mark=1000 跟踪豁免应盖差静默 diff_alert=%d", n)
	}
	// BTC 价漂 mark=500→跟踪=0.1×500=$50→豁免缩到 50.5 阈下→diff60 照报。
	seedMark(t, pool, "BTC-USDT", 500, 0)
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if n := r.rej.Snapshot()["ledger/diff_alert"]; n != 1 {
		t.Fatalf("mark=500 跟踪值缩水应照报 diff_alert=%d", n)
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
