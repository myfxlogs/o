package banrecheck

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// migrateBoth 测试面跨域（fin_opp_template/fin_capability_surface=md 链，
// ops_event/ops_esc_pending=fin 链——两链并集态 md 先 fin 后=历史重放方向）。
func migrateBoth(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
}

// seedBan 禁前提行+能力面注册（md_1044 已种 okx/dcd 能力面——ON CONFLICT
// 幂等；模板行 upsert——md_1103 回填钉定行已在册，同键 UPDATE 不撞
// uq_opp_template_struct；桩件语义=「该行当前禁因形」非新增行）。
func seedBan(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	template, surface, kind string) {
	t.Helper()
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_capability_surface(venue,surface,endpoint_family,product_family,rw_class,cost_attrs,enabled)
		 VALUES('okx',$1,'/api/v5/finance/sfp/dcd','dual-investment','read','{}',true)
		 ON CONFLICT (venue,surface) DO NOTHING`, surface); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_opp_template(template,venue,surface,params,enabled,state,
			disabled_reason,ban_premise,ban_premise_kind,banned_at)
		 VALUES($1,'okx',$2,'{}',false,'disabled','测试禁因——桩件',
			'测试前提命题：venue 面缺类',$3, now()-interval '10 days')
		 ON CONFLICT (template,venue,COALESCE(surface,'')) DO UPDATE SET
			enabled=false,state='disabled',disabled_reason=EXCLUDED.disabled_reason,
			ban_premise=EXCLUDED.ban_premise,ban_premise_kind=EXCLUDED.ban_premise_kind,
			banned_at=EXCLUDED.banned_at`,
		template, surface, kind); err != nil {
		t.Fatal(err)
	}
}

func newTestRunner(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	check CheckFunc) *Runner {
	t.Helper()
	return NewRunner(pool, config.New(pool)).
		AttachEscPool(pool).AttachCheck(check)
}

func countRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	where string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE `+where, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// 证伪→升件复审（非自动解禁——enabled/state 原样不动）。
func TestFalsifiedEscalatesNoAutoUnban(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedBan(t, ctx, pool, "dcd_put", "dcd", "venue_capability")

	// 迁移回填钉定行同场在册——核径按模板判别：目标行证伪、余行 unknown
	// 不产行（断言面只认目标模板的升件/回写）。
	r := newTestRunner(t, ctx, pool, func(_ context.Context, rw Row) (Result, error) {
		if rw.Template != "dcd_put" {
			return Result{Verdict: "unknown"}, nil
		}
		return Result{Verdict: "falsified", Detail: "currency-pair 在架对=12——缺类前提证伪"}, nil
	})
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var ev, scope, note string
	if err := pool.QueryRow(ctx,
		`SELECT event,coalesce(esc_scope,''),note FROM ops_event
		 WHERE seat='research/banrecheck' AND esc_scope='TMPLBAN-dcd_put-okx/dcd'`).
		Scan(&ev, &scope, &note); err != nil {
		t.Fatalf("升件行缺: %v", err)
	}
	if ev != "升件" || scope != "TMPLBAN-dcd_put-okx/dcd" {
		t.Fatalf("升件形错: event=%s scope=%s", ev, scope)
	}
	if !strings.Contains(note, "前提证伪") || !strings.Contains(note, "在架对=12") {
		t.Fatalf("判词实证缺: %s", note)
	}
	// 非自动解禁——enabled/state 原样。
	var enabled bool
	var state string
	if err := pool.QueryRow(ctx,
		`SELECT enabled,state FROM fin_opp_template WHERE template='dcd_put'`).
		Scan(&enabled, &state); err != nil {
		t.Fatal(err)
	}
	if enabled || state != "disabled" {
		t.Fatalf("复核拍越权解禁: enabled=%v state=%s", enabled, state)
	}
	// 结论回写名册键——ban_last_verdict='falsified'（前提成不成立同答）。
	var verdict *string
	if err := pool.QueryRow(ctx,
		`SELECT ban_last_verdict FROM fin_opp_template WHERE template='dcd_put'`).
		Scan(&verdict); err != nil {
		t.Fatal(err)
	}
	if verdict == nil || *verdict != "falsified" {
		t.Fatalf("结论回写缺: %v", verdict)
	}
	// 去重——二拍不重复升件（同 scope 窗内抑）。
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, ctx, pool,
		`seat='research/banrecheck' AND event='升件' AND esc_scope='TMPLBAN-dcd_put-okx/dcd'`); n != 1 {
		t.Fatalf("升件去重败: %d 行", n)
	}
}

// 仍立→复核行续命（72h 窗内不重报）。
func TestUpholdReportsDedupWindow(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedBan(t, ctx, pool, "vol_sell_strangle", "dcd", "venue_capability")

	// 钉定行同场——核径按模板判别：目标行仍立、余行 unknown 不产行。
	r := newTestRunner(t, ctx, pool, func(_ context.Context, rw Row) (Result, error) {
		if rw.Template != "vol_sell_strangle" {
			return Result{Verdict: "unknown"}, nil
		}
		return Result{Verdict: "upheld", Detail: "currency-pair 业务拒 code=51001——面仍拒"}, nil
	})
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var note string
	if err := pool.QueryRow(ctx,
		`SELECT note FROM ops_event WHERE seat='research/banrecheck' AND event='记事'
		 AND note LIKE '%vol_sell_strangle%' ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil {
		t.Fatalf("复核行缺: %v", err)
	}
	if !strings.Contains(note, "前提仍立") || !strings.Contains(note, "禁龄=10.0d") {
		t.Fatalf("复核行形错: %s", note)
	}
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, ctx, pool,
		`seat='research/banrecheck' AND event='记事' AND note LIKE '%vol_sell_strangle%'`); n != 1 {
		t.Fatalf("复核行窗内重报: %d 行", n)
	}
}

// owner_order=业主线不机核——check 缝零调用仅龄报。
func TestOwnerOrderNoMachineCheck(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedBan(t, ctx, pool, "owner_ban", "dcd", "owner_order")

	var calls int
	r := newTestRunner(t, ctx, pool, func(_ context.Context, rw Row) (Result, error) {
		if rw.Template == "owner_ban" {
			calls++ // owner_order 应被 checkRow 拦截零调用——到达即越权
		}
		return Result{Verdict: "unknown"}, nil
	})
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("owner_order 触机核: calls=%d", calls)
	}
	var note string
	if err := pool.QueryRow(ctx,
		`SELECT note FROM ops_event WHERE seat='research/banrecheck' AND event='记事'
		 ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil {
		t.Fatalf("业主令龄报行缺: %v", err)
	}
	if !strings.Contains(note, "业主令类前提——机核跳过仅龄报") {
		t.Fatalf("业主令径形错: %s", note)
	}
	// owner_order 独立结论键回写（名册可机读「业主线不机核」）。
	var verdict *string
	if err := pool.QueryRow(ctx,
		`SELECT ban_last_verdict FROM fin_opp_template WHERE template='owner_ban'`).
		Scan(&verdict); err != nil {
		t.Fatal(err)
	}
	if verdict == nil || *verdict != "owner_order" {
		t.Fatalf("owner_order 结论键错: %v", verdict)
	}
}

// unknown=核径不可用不产行（不误升不误报）。
func TestUnknownNoRows(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedBan(t, ctx, pool, "unk_t", "dcd", "apr_regime")

	r := newTestRunner(t, ctx, pool, func(context.Context, Row) (Result, error) {
		return Result{Verdict: "unknown", Detail: "核径未装配"}, nil
	})
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, ctx, pool, `seat='research/banrecheck'`); n != 0 {
		t.Fatalf("unknown 产行: %d", n)
	}
}

// 复核对象集闸——enabled=t/candidate/无前提行不入拍。
func TestRowsFilter(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedBan(t, ctx, pool, "dcd_call", "dcd", "venue_capability")
	// 对照行：enabled=true 带前提（不该入）、candidate 无前提（不该入）、
	// enabled=false 但 ban_premise NULL（失考行——spec 钉「禁因失考」显式
	// 值但 premise 可空=无前提可核不入拍）。
	for _, q := range []string{
		`INSERT INTO fin_opp_template(template,venue,surface,params,enabled,state,
			ban_premise,ban_premise_kind) VALUES('live_t','okx','dcd','{}',true,'enabled','x','venue_capability')`,
		`INSERT INTO fin_opp_template(template,venue,surface,params,enabled,state)
		 VALUES('cand_t','okx','dcd','{}',false,'candidate')`,
		`INSERT INTO fin_opp_template(template,venue,surface,params,enabled,state,
			disabled_reason,ban_premise_kind) VALUES('nopr_t','okx','dcd','{}',false,'disabled','禁因失考','none')`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	r := newTestRunner(t, ctx, pool, func(context.Context, Row) (Result, error) {
		return Result{Verdict: "unknown"}, nil
	})
	rs, err := r.rows(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 迁移回填钉定行同场在册（dcd_put/dcd_call/vol_sell×2 venue_capability
	// 前提合法入拍）——断言面=目标行在集+对照三行全不在集，非全等计数。
	var sawCall, sawBad bool
	for _, rw := range rs {
		if rw.Template == "dcd_call" {
			sawCall = true
		}
		if rw.Template == "live_t" || rw.Template == "cand_t" || rw.Template == "nopr_t" {
			sawBad = true
		}
	}
	if !sawCall || sawBad {
		t.Fatalf("复核对象集错: %+v", rs)
	}
}

// 写径缺席=守位不核（escPool nil 不炸不产行）。
func TestNoEscPoolGuard(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedBan(t, ctx, pool, "dcd_put", "dcd", "venue_capability")
	r := NewRunner(pool, config.New(pool)).AttachCheck(
		func(context.Context, Row) (Result, error) {
			return Result{Verdict: "falsified"}, nil
		})
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, ctx, pool, `seat='research/banrecheck'`); n != 0 {
		t.Fatalf("守位产行: %d", n)
	}
	_ = time.Now // 保占位——banned_at 断言形备查
}
