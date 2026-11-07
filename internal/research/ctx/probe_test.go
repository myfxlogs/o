package ctx

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0429 断供探针桩（钦定形=T-0410 feedstale——row-age 巡检+feed断更行+
// 观测盲升件 esc_scope=CTX-<源>，24h 同窗去重）。病灶原型：FRED 止 id976
// 零升件静默死+ffcal 零行史死（判词六维饥荒实证 2026-11-07）。

func probeFixture(t *testing.T) (*Runner, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool).AttachEscPool(pool)
	return r, pool
}

// seedCtx 源行定龄（ageHours——>阈即陈；age<0=不种行=零行史形）。
func seedCtx(t *testing.T, pool *pgxpool.Pool, source string, ageHours float64) {
	t.Helper()
	ctx := context.Background()
	if ageHours < 0 {
		return // 零行史形——不种
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_ctx_fact(source,kind,payload,received_ts)
		VALUES($1,'probe','{}',now()-make_interval(hours=>$2))`, source, ageHours); err != nil {
		t.Fatal(err)
	}
}

// staleRows 病灶两形行数（feed断更/升件——按 scope 断言公共面）。
func staleRows(t *testing.T, pool *pgxpool.Pool, scope string) (lesion, esc int) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='feed断更' AND ticket='T-0429' AND note LIKE '%`+scope+`%'`).Scan(&lesion); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='观测盲' AND esc_scope=$1`, scope).Scan(&esc); err != nil {
		t.Fatal(err)
	}
	return lesion, esc
}

// TestProbeStaleLesionOnStaleRow 形①：fred 末行龄 100h>阈 96h→病灶行+升件。
// 其余源种鲜行（探针按源独立——只报陈源不误报鲜源）。
func TestProbeStaleLesionOnStaleRow(t *testing.T) {
	r, pool := probeFixture(t)
	ctx := context.Background()
	seedCtx(t, pool, "fred", 100)
	for _, s := range []string{"ffcal", "frankfurter", "cftc", "okx_public"} {
		seedCtx(t, pool, s, 1)
	}
	if err := r.ProbeStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if lesion, esc := staleRows(t, pool, "CTX-FRED"); lesion != 1 || esc != 1 {
		t.Fatalf("病灶行缺失: feed断更=%d 升件=%d", lesion, esc)
	}
	for _, scope := range []string{"CTX-FFCAL", "CTX-FX", "CTX-COT", "CTX-OI"} {
		if lesion, esc := staleRows(t, pool, scope); lesion != 0 || esc != 0 {
			t.Fatalf("鲜源误报 %s: feed断更=%d 升件=%d", scope, lesion, esc)
		}
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_scope='CTX-FRED'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "断供") || !strings.Contains(note, "received_ts") {
		t.Fatalf("升件 note 缺断供/received_ts 锚: %s", note)
	}
}

// TestProbeStaleQuietOnFreshRows 形②：全源鲜行=零行（进水口在拍非病灶）。
func TestProbeStaleQuietOnFreshRows(t *testing.T) {
	r, pool := probeFixture(t)
	ctx := context.Background()
	for _, s := range []string{"ffcal", "fred", "frankfurter", "cftc", "okx_public"} {
		seedCtx(t, pool, s, 1)
	}
	if err := r.ProbeStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	for _, scope := range []string{"CTX-FRED", "CTX-FFCAL", "CTX-FX", "CTX-COT", "CTX-OI"} {
		if lesion, esc := staleRows(t, pool, scope); lesion != 0 || esc != 0 {
			t.Fatalf("鲜源误报 %s: feed断更=%d 升件=%d", scope, lesion, esc)
		}
	}
}

// TestProbeStaleZeroRowsForm 形③：零行史=断供极端形同报（feedstale 形④同例
// ——frankfurter 首拍拉败即零行史，死在出生时应可见）。
func TestProbeStaleZeroRowsForm(t *testing.T) {
	r, pool := probeFixture(t)
	ctx := context.Background()
	for _, s := range []string{"ffcal", "fred", "cftc", "okx_public"} {
		seedCtx(t, pool, s, 1)
	}
	// frankfurter 零行——ageHours<0 不种
	seedCtx(t, pool, "frankfurter", -1)
	if err := r.ProbeStale(ctx); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if lesion, esc := staleRows(t, pool, "CTX-FX"); lesion != 1 || esc != 1 {
		t.Fatalf("零行史病灶缺失: feed断更=%d 升件=%d", lesion, esc)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='feed断更' AND note LIKE '%CTX-FX%'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "零行史") {
		t.Fatalf("零行史 note 未显形: %s", note)
	}
}

// TestProbeStaleDedup24h 形④同窗去重：病灶在而再拍——24h 内至多一报。
func TestProbeStaleDedup24h(t *testing.T) {
	r, pool := probeFixture(t)
	ctx := context.Background()
	seedCtx(t, pool, "fred", 100)
	for _, s := range []string{"ffcal", "frankfurter", "cftc", "okx_public"} {
		seedCtx(t, pool, s, 1)
	}
	for i := 0; i < 2; i++ {
		if err := r.ProbeStale(ctx); err != nil {
			t.Fatalf("probe %d: %v", i, err)
		}
	}
	if lesion, esc := staleRows(t, pool, "CTX-FRED"); lesion != 1 || esc != 1 {
		t.Fatalf("同窗去重失效: feed断更=%d 升件=%d", lesion, esc)
	}
}

// TestProbeStale守位 形⑤：escPool 未挂=守位不探不炸（装配缺件如实缺升件——
// 与 analyst AttachLLM nil=守位同形；采集成果不得被观测臂拖崩）。
func TestProbeStale守位(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	seedCtx(t, pool, "fred", 100)
	r := NewRunner(pool) // 无 AttachEscPool
	if err := r.ProbeStale(ctx); err != nil {
		t.Fatalf("escPool 缺应守位非炸: %v", err)
	}
}
