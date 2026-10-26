// run_test.go —— 差扫岗+面分类器桩测（T-0102 v3/v4 锚）：目录差扫对拍注册表
// 写 gap 行（幂等）；分类命中→候选生成（provenance=surface_gap）走毕业线；
// 死面 autorejected+拒因；已注册面零差；终态不机器翻案；爆面上限。
package gapscan

import (
	"context"
	"errors"
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

// 锚：目录差扫——已注册面（dcd）零差；未注册面（earn-flex）gap 行立→分类
// carry→候选生成（provenance=surface_gap）走毕业线；幂等重扫不重生。
func TestGapScanClassifyGenerate(t *testing.T) {
	ctx, r := setup(t)
	r.AttachProbe(func(_ context.Context, f ProbeFace) (ProbeResult, error) {
		return ProbeResult{Products: 3, YieldBearing: true}, nil
	})
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_surface_gap
		WHERE venue='okx' AND surface='dcd'`); n != 0 {
		t.Fatal("已注册面不应立 gap 行")
	}
	var state string
	var metaB []byte
	if err := r.pool.QueryRow(ctx, `SELECT state,metadata FROM fin_surface_gap
		WHERE venue='okx' AND surface='earn-flex'`).Scan(&state, &metaB); err != nil {
		t.Fatal("未注册面应立 gap 行")
	}
	if state != "pending" {
		t.Fatalf("gap 态=%s（应 pending——候选已进毕业线仍待注册裁）", state)
	}
	meta := string(metaB)
	for _, frag := range []string{`"family": "carry"`, `"provenance": "surface_gap"`, `"gen": "candidate"`} {
		if !strings.Contains(meta, frag) {
			t.Fatalf("元数据缺 %s: %s", frag, meta)
		}
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_opp_eval
		WHERE inst_id='okx/earn-flex' AND direction='gap_carry' AND verdict='proposed'`); n != 1 {
		t.Fatalf("面候选评估应 1: %d", n)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='okx/earn-flex' AND direction='gap_carry' AND state='fin_proposed'`); n != 1 {
		t.Fatalf("面候选提案应 1: %d", n)
	}
	// 幂等：重扫不重生（同面候选在途）。
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='okx/earn-flex'`); n != 1 {
		t.Fatalf("重扫应幂等不重生: %d", n)
	}
	// 卡面露出：待裁计数=1。
	if n, err := PendingCount(ctx, r.pool); err != nil || n != 1 {
		t.Fatalf("PendingCount=%d err=%v（应 1）", n, err)
	}
}

// 锚：死面闸——读面不通（路由缺席）→ autorejected+拒因；零产品同裁。
func TestGapDeadFaceAutoreject(t *testing.T) {
	ctx, r := setup(t)
	r.AttachProbe(func(_ context.Context, f ProbeFace) (ProbeResult, error) {
		if f.Surface == "dcd" {
			return ProbeResult{}, &FaceAbsentError{Status: 404}
		}
		return ProbeResult{Products: 0}, nil
	})
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var reason string
	if err := r.pool.QueryRow(ctx, `SELECT reason FROM fin_surface_gap
		WHERE venue='okx' AND surface='dcd' AND state='autorejected'`).Scan(&reason); err != nil || reason != "http 404" {
		t.Fatalf("死面应 autorejected+拒因: %v reason=%q", err, reason)
	}
	if err := r.pool.QueryRow(ctx, `SELECT reason FROM fin_surface_gap
		WHERE venue='okx' AND surface='earn-flex' AND state='autorejected'`).Scan(&reason); err != nil || reason != "zero_products" {
		t.Fatalf("零产品应 autorejected: %v reason=%q", err, reason)
	}
}

// 锚：瞬态读败 fail-closed 不裁死（本拍让位，行不立不翻）。
func TestGapTransientSkip(t *testing.T) {
	ctx, r := setup(t)
	r.AttachProbe(func(context.Context, ProbeFace) (ProbeResult, error) {
		return ProbeResult{}, errors.New("dial tcp: timeout")
	})
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_surface_gap`); n != 0 {
		t.Fatalf("瞬态读败不应立行: %d", n)
	}
}

// 锚：终态不机器翻案（registered 行重探保持）+autorejected 复活回 pending。
func TestGapTerminalStates(t *testing.T) {
	ctx, r := setup(t)
	r.AttachProbe(func(context.Context, ProbeFace) (ProbeResult, error) {
		return ProbeResult{Products: 2, YieldBearing: true}, nil
	})
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_surface_gap(venue,surface,state)
		VALUES('okx','earn-flex','registered')`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_surface_gap
		WHERE venue='okx' AND surface='earn-flex'`).Scan(&state); err != nil || state != "registered" {
		t.Fatalf("终态应不翻案: %v state=%s", err, state)
	}
	// 复活：autorejected 行重探通 → 回 pending。
	if _, err := r.pool.Exec(ctx, `UPDATE fin_surface_gap SET state='autorejected',
		reason='http 404' WHERE venue='okx' AND surface='earn-flex'`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_surface_gap
		WHERE venue='okx' AND surface='earn-flex'`).Scan(&state); err != nil || state != "pending" {
		t.Fatalf("复活面应回 pending: %v state=%s", err, state)
	}
	// T-0106：hypothesized（卡已立）重探不翻案——假想官消费态守恒。
	if _, err := r.pool.Exec(ctx, `UPDATE fin_surface_gap SET state='hypothesized',
		hypothesis_id=1 WHERE venue='okx' AND surface='earn-flex'`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_surface_gap
		WHERE venue='okx' AND surface='earn-flex'`).Scan(&state); err != nil || state != "hypothesized" {
		t.Fatalf("已起草面重探应守恒: %v state=%s", err, state)
	}
}

// 锚：爆面上限（gen_cap=1——第二面候选排队不生成）。
func TestGapGenCap(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO app_config(key,value) VALUES('research.gap.gen_cap','1')
		 ON CONFLICT(key) DO UPDATE SET value='1'`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(r.pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r.cfg = cfg
	r.AttachProbe(func(context.Context, ProbeFace) (ProbeResult, error) {
		return ProbeResult{Products: 2, YieldBearing: true}, nil
	})
	// 预置一个在途 gap_carry 候选（占满上限）。
	var eid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('okx/pre','surface-gap','gap_carry','proposed','{}','{"provenance":"surface_gap"}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,'okx/pre','gap_carry',0)`, eid); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='okx/earn-flex'`); n != 0 {
		t.Fatalf("达上限应排队不生成: %d", n)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_surface_gap
		WHERE surface='earn-flex' AND state='pending'`); n != 1 {
		t.Fatalf("排队面应留 pending: %d", n)
	}
}

// classifyFace 纯函数谱（v4 规则全覆盖——含未触发面的未来族）。
func TestClassifyFace(t *testing.T) {
	for _, tc := range []struct {
		meta map[string]any
		want string
	}{
		{map[string]any{"yield_bearing": true}, "carry"},
		{map[string]any{"funding_bearing": true}, "hedge"},
		{map[string]any{"swap_sign": 0.5}, "carry"},
		{map[string]any{"swap_sign": -0.5}, "hedge"},
		{map[string]any{"options": true}, "vol"},
		{map[string]any{}, "unclassified"},
	} {
		if got := classifyFace(tc.meta); got != tc.want {
			t.Fatalf("classifyFace(%v)=%s 应 %s", tc.meta, got, tc.want)
		}
	}
}
