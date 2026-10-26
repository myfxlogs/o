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
	// T-0167 分链：本包测试面跨域（研究域表+判据键）——两链并集态（md 先 fin 后
	// =历史重放方向：md 链瞬态超表层先落，fin 链状态化/守卫后到）。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
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
	r.AttachSignals(func(ctx context.Context, venue, surface string) ([]Sig, bool, error) {
		if surface == "USDT-SWAP-sig" {
			return []Sig{{Kind: "funding_view", Key: "BTC-USDT-SWAP", Val: 0.0001}}, true, nil
		}
		return nil, false, nil
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

// T-0140 锚：信号驱动卡——funding 背离超阈→provenance=hypothesis_officer+
// signals 域实证；阈内/均值缺席→零卡；陈信号（停桥）→零卡不误报。
func TestSignalDrivenCards(t *testing.T) {
	ctx, r := setup(t)
	seedTh := func(t *testing.T) {
		t.Helper()
		for _, kv := range [][2]string{
			{"research.signal.funding_div_ratio", "3"},
			{"research.signal.spread_bps", "50"},
		} {
			if _, err := r.pool.Exec(ctx, `INSERT INTO app_config(key,value) VALUES($1,$2)
				ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, kv[0], kv[1]); err != nil {
				t.Fatal(err)
			}
		}
		if err := r.cfg.Load(ctx); err != nil {
			t.Fatal(err)
		}
	}
	seedCard := func(t *testing.T, kind, key string, val float64, meta string, age string) {
		t.Helper()
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
			VALUES($1,$2,$3,$4::jsonb, now()-interval '`+age+`')
			ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val, meta=EXCLUDED.meta, ts=EXCLUDED.ts`,
			kind, key, val, meta); err != nil {
			t.Fatal(err)
		}
	}
	// ① 背离 5×>3 阈→carry 卡（provenance/signals/trigger 全验）
	seedTh(t)
	seedCard(t, "funding_view", "BTC-USDT-SWAP", 0.0005,
		`{"avg_7d":0.0001}`, "0 seconds")
	if err := r.ScanSignals(ctx); err != nil {
		t.Fatal(err)
	}
	var prov string
	var ev map[string]any
	if err := r.pool.QueryRow(ctx, `SELECT evidence->>'provenance', evidence
		FROM fin_opp_hypothesis WHERE venue='okx' AND surface='market-funding-rate'`).
		Scan(&prov, &ev); err != nil {
		t.Fatal("背离卡应立:", err)
	}
	if prov != "hypothesis_officer" || ev["trigger"] != "funding_divergence" {
		t.Fatalf("卡 provenance/trigger=%s/%v", prov, ev["trigger"])
	}
	raw, _ := json.Marshal(ev["signals"])
	var sigs []Sig
	json.Unmarshal(raw, &sigs)
	if len(sigs) != 1 || sigs[0].Key != "BTC-USDT-SWAP" {
		t.Fatalf("卡 signals 域应含源信号: %s", raw)
	}
	// ② 阈内背离→零新卡（幂等重扫也不重生）
	seedCard(t, "funding_view", "ETH-USDT-SWAP", 0.00011,
		`{"avg_7d":0.0001}`, "0 seconds")
	if err := r.ScanSignals(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_hypothesis WHERE surface='market-funding-rate'`).Scan(&n)
	if n != 1 {
		t.Fatalf("阈内零卡（同面幂等）got %d", n)
	}
	// ③ depth5 spread 超阈→hedge 卡
	seedCard(t, "depth5", "BTC-USDT-SWAP", 80, `{"best_bid":100}`, "0 seconds")
	if err := r.ScanSignals(ctx); err != nil {
		t.Fatal(err)
	}
	var fam string
	if err := r.pool.QueryRow(ctx, `SELECT family FROM fin_opp_hypothesis
		WHERE venue='okx' AND surface='books5'`).Scan(&fam); err != nil {
		t.Fatal("spread 突变卡应立:", err)
	}
	if fam != "hedge" {
		t.Fatalf("spread 卡 family=%s", fam)
	}
	// ④ 陈信号（停桥 5min）→零卡不误报
	seedTh(t)
	if _, err := r.pool.Exec(ctx, `DELETE FROM fin_opp_hypothesis WHERE surface='market-funding-rate'`); err != nil {
		t.Fatal(err)
	}
	seedCard(t, "funding_view", "BTC-USDT-SWAP", 0.0005,
		`{"avg_7d":0.0001}`, "5 minutes") // ① 行翻陈——全表现陈（停桥形）
	seedCard(t, "funding_view", "SOL-USDT-SWAP", 0.001,
		`{"avg_7d":0.0001}`, "5 minutes")
	if err := r.ScanSignals(ctx); err != nil {
		t.Fatal(err)
	}
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_hypothesis WHERE surface='market-funding-rate'`).Scan(&n)
	if n != 0 {
		t.Fatalf("陈信号应零卡不误报 got %d", n)
	}
}
