package hunger

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// canaryFixture 雷探针桩景（fin 迁移——ops_event/fin_* 全链在册）。
func canaryFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	old := canaryClaims
	t.Cleanup(func() { canaryClaims = old })
	return New(pool, cfg), pool
}

// escNote 升件行 note 读回（按签名模糊取）。
func escNote(t *testing.T, pool *pgxpool.Pool, like string) string {
	t.Helper()
	var n string
	if err := pool.QueryRow(context.Background(),
		`SELECT note FROM ops_event WHERE event='升件' AND note LIKE $1
		 ORDER BY id DESC LIMIT 1`, like).Scan(&n); err != nil {
		return ""
	}
	return n
}

// TestCanaryPlantsSeed 净库首拍→种一雷（ref=AC- 签名+账疑类+pend 域键）。
func TestCanaryPlantsSeed(t *testing.T) {
	s, pool := canaryFixture(t)
	ctx := context.Background()
	canaryClaims = []canaryClaim{{
		id:        "t0",
		note:      "账疑：桩命题 ref=%s",
		verifySQL: `SELECT 0`,
		realScope: "t_real",
	}}
	if err := s.ProbeAuditCanary(ctx); err != nil {
		t.Fatal(err)
	}
	n := escNote(t, pool, "%ref=AC-%")
	if n == "" {
		t.Fatal("未播种")
	}
	var cls, scope string
	if err := pool.QueryRow(ctx,
		`SELECT esc_class,esc_scope FROM ops_event WHERE event='升件'
		 AND note LIKE '%ref=AC-%' ORDER BY id DESC LIMIT 1`).Scan(&cls, &scope); err != nil {
		t.Fatal(err)
	}
	if cls != "账疑" || scope != "acct_suspect" {
		t.Fatalf("雷件类/域错形: %s/%s", cls, scope)
	}
}

// TestCanaryVerifyTrueGoesReal 命题核实为真→真升件非雷（不占雷位）。
func TestCanaryVerifyTrueGoesReal(t *testing.T) {
	s, pool := canaryFixture(t)
	ctx := context.Background()
	canaryClaims = []canaryClaim{{
		id:        "t0",
		note:      "账疑：桩命题 ref=%s",
		verifySQL: `SELECT 1`,
		realScope: "t_real",
	}}
	if err := s.ProbeAuditCanary(ctx); err != nil {
		t.Fatal(err)
	}
	if n := escNote(t, pool, "%ref=AC-%"); n != "" {
		t.Fatalf("命题为真仍种雷: %s", n)
	}
	if n := escNote(t, pool, "%REAL%"); n == "" {
		t.Fatal("命题为真未走真升件")
	}
	var scope string
	if err := pool.QueryRow(ctx,
		`SELECT esc_scope FROM ops_event WHERE event='升件' AND note LIKE '%REAL%'`).Scan(&scope); err != nil {
		t.Fatal(err)
	}
	if scope != "t_real" {
		t.Fatalf("真升件域错: %s", scope)
	}
}

// TestCanaryCatchWithVerifyWords 处置含核实词→审计雷检出行（时延在账）。
func TestCanaryCatchWithVerifyWords(t *testing.T) {
	s, pool := canaryFixture(t)
	ctx := context.Background()
	canaryClaims = []canaryClaim{{
		id: "t0", note: "账疑：桩命题 ref=%s",
		verifySQL: `SELECT 0`, realScope: "t_real",
	}}
	if err := s.ProbeAuditCanary(ctx); err != nil {
		t.Fatal(err)
	}
	var seedID int64
	if err := pool.QueryRow(ctx,
		`SELECT id FROM ops_event WHERE event='升件' AND note LIKE '%ref=AC-%'`).Scan(&seedID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note)
		 VALUES('处置席','处置','',$1)`,
		fmt.Sprintf("升件#%d 销——核实桩命题=0条，申报不实", seedID)); err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) } // 过冷却
	if err := s.ProbeAuditCanary(ctx); err != nil {
		t.Fatal(err)
	}
	var catch string
	if err := pool.QueryRow(ctx,
		`SELECT note FROM ops_event WHERE event='审计雷' ORDER BY id DESC LIMIT 1`).Scan(&catch); err != nil {
		t.Fatalf("检出行缺: %v", err)
	}
	if want := fmt.Sprintf("雷#%d 检出", seedID); !strings.HasPrefix(catch, want) {
		t.Fatalf("检出行错形: %s", catch)
	}
	// 过冷却+检出后应补种下一雷（种序轮转——同 claim 桩面仍 AC-）。
	if escNote(t, pool, "%AC-"+s.now().Format("20060102")+"%") == "" {
		t.Fatal("检出+过冷却不补种")
	}
}

// TestCanaryRubberStamp 处置无核实词→橡皮章嫌疑升件。
func TestCanaryRubberStamp(t *testing.T) {
	s, pool := canaryFixture(t)
	ctx := context.Background()
	canaryClaims = []canaryClaim{{
		id: "t0", note: "账疑：桩命题 ref=%s",
		verifySQL: `SELECT 0`, realScope: "t_real",
	}}
	if err := s.ProbeAuditCanary(ctx); err != nil {
		t.Fatal(err)
	}
	var seedID int64
	if err := pool.QueryRow(ctx,
		`SELECT id FROM ops_event WHERE event='升件' AND note LIKE '%ref=AC-%'`).Scan(&seedID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note)
		 VALUES('处置席','处置','',$1)`,
		fmt.Sprintf("升件#%d 销", seedID)); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeAuditCanary(ctx); err != nil {
		t.Fatal(err)
	}
	if n := escNote(t, pool, "%橡皮章%"); n == "" {
		t.Fatal("橡皮章嫌疑未升件")
	}
}

// TestCanaryNoDoubleSeed 在飞雷不补种（未处置=等结局）。
func TestCanaryNoDoubleSeed(t *testing.T) {
	s, pool := canaryFixture(t)
	ctx := context.Background()
	canaryClaims = []canaryClaim{{
		id: "t0", note: "账疑：桩命题 ref=%s",
		verifySQL: `SELECT 0`, realScope: "t_real",
	}}
	if err := s.ProbeAuditCanary(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeAuditCanary(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='升件' AND note LIKE '%ref=AC-%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("在飞雷复种: %d", n)
	}
}
