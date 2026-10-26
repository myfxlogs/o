// run_test.go —— 假想官桩测（T-0106 验收锚）：gap→假想卡产出/幂等/家族归类/
// 零越权写/gap 链接通（pending→hypothesized 卡引用回填）/花名册活。
package hypothesis

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
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

func count(t *testing.T, ctx context.Context, r *Runner, q string, args ...any) int {
	t.Helper()
	var n int
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// seedGap 直植 gap 行（分类器产物形——family 已归类 pending）。
func seedGap(t *testing.T, ctx context.Context, r *Runner, venue, surface, family string, products int) int64 {
	t.Helper()
	var id int64
	meta := `{"family":"` + family + `","products":` + strconv.Itoa(products) + `,"yield_bearing":true,"kind":"locked-yield"}`
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_surface_gap(venue,surface,state,metadata)
		VALUES($1,$2,'pending',$3::jsonb) RETURNING id`, venue, surface, meta).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// 锚链：pending gap→假想卡产出（family/词汇表闭集/需求清单/evidence）→
// gap 翻 hypothesized 卡引用回填；重扫幂等不重生。
func TestHypothesisDraftFromGap(t *testing.T) {
	ctx, r := setup(t)
	gapID := seedGap(t, ctx, r, "okx", "earn-flex", "carry", 5)
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var family, state string
	var prims, reqs, ev []byte
	var hyID int64
	if err := r.pool.QueryRow(ctx, `SELECT family,leg_primitives,requirements,state,evidence
		FROM fin_opp_hypothesis WHERE venue='okx' AND surface='earn-flex'`).
		Scan(&family, &prims, &reqs, &state, &ev); err != nil {
		t.Fatal("假想卡应落表")
	}
	if family != "carry" || state != "proposed" {
		t.Fatalf("family=%s state=%s", family, state)
	}
	if !strings.Contains(string(prims), "earn_flexible_park") ||
		!strings.Contains(string(prims), "locked_ccy_sell_call") {
		t.Fatalf("腿原语应出闭集词汇表: %s", prims)
	}
	if strings.Contains(string(prims), "invent") {
		t.Fatalf("词汇表外原语不得出现: %s", prims)
	}
	if !strings.Contains(string(reqs), "evaluator") || !strings.Contains(string(reqs), "risk_thresholds") {
		t.Fatalf("需求清单应齐: %s", reqs)
	}
	if !strings.Contains(string(ev), `"provenance": "hypothesis_officer"`) {
		t.Fatalf("证据应标起草席来源: %s", ev)
	}
	// gap 链接通：state=hypothesized+卡引用回填。
	if err := r.pool.QueryRow(ctx, `SELECT state,hypothesis_id FROM fin_surface_gap
		WHERE id=$1`, gapID).Scan(&state, &hyID); err != nil || state != "hypothesized" || hyID == 0 {
		t.Fatalf("gap 应翻 hypothesized+卡引用: %v state=%s hyID=%d", err, state, hyID)
	}
	var back int64
	if err := r.pool.QueryRow(ctx, `SELECT id FROM fin_opp_hypothesis WHERE id=$1`, hyID).Scan(&back); err != nil || back != hyID {
		t.Fatalf("卡引用应可回查: %v back=%d hyID=%d", err, back, hyID)
	}
	// 幂等：重扫不重生（撞卡返既有 id）。
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_opp_hypothesis
		WHERE venue='okx' AND surface='earn-flex'`); n != 1 {
		t.Fatalf("重扫应幂等 1 卡: %d", n)
	}
	// 花名册活（T-0078 判据：名单岗必须 runtime 在行）。
	if err := r.enroll(ctx); err != nil {
		t.Fatal(err)
	}
	var title string
	if err := r.pool.QueryRow(ctx, `SELECT title FROM fin_employee_registry
		WHERE name='假想官' AND status='active'`).Scan(&title); err != nil {
		t.Fatal("假想官应 runtime 在行")
	}
}

// 锚：零越权写——起草前后 fin_opp_template/fin_proposal 零增；不可起草族
// （unclassified）与产品数下限下不起草。
func TestHypothesisZeroWriteAndGuards(t *testing.T) {
	ctx, r := setup(t)
	tplBefore := count(t, ctx, r, `SELECT count(*) FROM fin_opp_template`)
	propBefore := count(t, ctx, r, `SELECT count(*) FROM fin_proposal`)
	seedGap(t, ctx, r, "okx", "f-good", "hedge", 3)
	seedGap(t, ctx, r, "okx", "f-unc", "unclassified", 9)
	seedGap(t, ctx, r, "okx", "f-thin", "carry", 0)
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_opp_template`); n != tplBefore {
		t.Fatalf("假想官触模板表=越权（%d→%d）", tplBefore, n)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_proposal`); n != propBefore {
		t.Fatalf("假想官触提案表=越权（%d→%d）", propBefore, n)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_opp_hypothesis
		WHERE surface='f-good'`); n != 1 {
		t.Fatalf("合格 gap 应立卡: %d", n)
	}
	for _, s := range []string{"f-unc", "f-thin"} {
		if n := count(t, ctx, r, `SELECT count(*) FROM fin_opp_hypothesis WHERE surface=$1`, s); n != 0 {
			t.Fatalf("%s 不应立卡", s)
		}
		var state string
		if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_surface_gap WHERE surface=$1`, s).Scan(&state); err != nil || state != "pending" {
			t.Fatalf("%s 应留 pending: %v %s", s, err, state)
		}
	}
}

// T-0139 消费闸锚：信号读面挂上→新鲜信号入卡 evidence.signals；
// 读闸（陈=缺席）在 sigdist.Fresh 侧——stub 直证附据形。
func TestHypothesisEvidenceSignals(t *testing.T) {
	ctx, r := setup(t)
	r.AttachSignals(func(ctx context.Context, kind, key string) (Sig, bool, error) {
		if kind == "funding_view" && key == "okx-USDT-SWAP" {
			return Sig{Kind: kind, Key: key, Val: 0.0001}, true, nil
		}
		return Sig{}, false, nil
	})
	gapID := seedGap(t, ctx, r, "okx", "USDT-SWAP-sig", "hedge", 5)
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var ev map[string]any
	if err := r.pool.QueryRow(ctx, `SELECT evidence FROM fin_opp_hypothesis
		WHERE venue='okx' AND surface='USDT-SWAP-sig'`).Scan(&ev); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(ev["signals"])
	var sigs []Sig
	if err := json.Unmarshal(raw, &sigs); err != nil || len(sigs) != 1 || sigs[0].Val != 0.0001 {
		t.Fatalf("evidence.signals 应含新鲜信号: %s err=%v", raw, err)
	}
	_ = gapID
}
