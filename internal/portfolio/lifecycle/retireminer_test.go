// retireminer_test.go —— 退役验尸产线桩锚（fin_0172）：退役 transition
// 无验尸案→产 pending；幂等（重拍不重产/案在跳喂）；pending 水位闸
// ≥20 停喂；35d 窗外不补；非退役 transition 不产。
package lifecycle

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func retireRig(t *testing.T) *Runner {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return &Runner{pool: pool, rej: rejects.New("lifecycle-t")}
}

// seedDead 种一死策略：策略行+退役 transition 留痕（Transition 正径走
// 合法迁——paper→retired 合法位）。
func seedDead(t *testing.T, r *Runner, inst string) int64 {
	t.Helper()
	ctx := context.Background()
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction)
		VALUES($1,'funding-hedge','paper',100,'short_perp_long_spot') RETURNING id`, inst).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	if err := Transition(ctx, r.pool, sid, "retired", "paper_ttl_expired", false); err != nil {
		t.Fatal(err)
	}
	return sid
}

// TestRetireLessonMineProducesOnce 退役即产案；重拍幂等不重产。
func TestRetireLessonMineProducesOnce(t *testing.T) {
	r := retireRig(t)
	ctx := context.Background()
	sid := seedDead(t, r, "RM-A-USDT-SWAP")
	r.retireLessonMine(ctx)
	var kind, ref, status string
	var nums string
	if err := r.pool.QueryRow(ctx, `SELECT kind,entity_ref,status,numbers::text
		FROM fin_econ_lesson WHERE entity_ref=$1`, "strategy#"+strconv.FormatInt(sid, 10)).
		Scan(&kind, &ref, &status, &nums); err != nil {
		t.Fatalf("验尸案缺: %v", err)
	}
	if kind != "retire_postmortem" || status != "pending" {
		t.Fatalf("案类/态错位: %s %s", kind, status)
	}
	for _, k := range []string{"paper_ttl_expired", "funding-hedge", "paper_net_usd", "realized_usd", "age_days"} {
		if !strings.Contains(nums, k) {
			t.Fatalf("numbers 载缺 %s: %s", k, nums)
		}
	}
	r.retireLessonMine(ctx) // 重拍——UNIQUE 幂等不重产
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_econ_lesson
		WHERE kind='retire_postmortem'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("重拍重产: n=%d err=%v", n, err)
	}
}

// TestRetireLessonMinePendingCap pending 水位 ≥20 停喂——不淹矿面。
func TestRetireLessonMinePendingCap(t *testing.T) {
	r := retireRig(t)
	ctx := context.Background()
	for i := 0; i < retireMinePendingCap; i++ {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_econ_lesson(kind,entity_ref)
			VALUES('retire_postmortem',$1)`, "strategy#cap-"+strconv.Itoa(i)); err != nil {
			t.Fatal(err)
		}
	}
	seedDead(t, r, "RM-CAP-USDT-SWAP")
	r.retireLessonMine(ctx)
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_econ_lesson
		WHERE kind='retire_postmortem'`).Scan(&n); err != nil || n != retireMinePendingCap {
		t.Fatalf("水位闸失效: n=%d err=%v", n, err)
	}
}

// TestRetireLessonMineWindow 窗外死案不补——35d 前退役不淹 pending（考古
// 归裁定席直查 transition 账）。
func TestRetireLessonMineWindow(t *testing.T) {
	r := retireRig(t)
	ctx := context.Background()
	sid := seedDead(t, r, "RM-OLD-USDT-SWAP")
	if _, err := r.pool.Exec(ctx, `UPDATE fin_strategy_transition SET ts=now()-interval '40 days'
		WHERE strategy_id=$1`, sid); err != nil {
		t.Fatal(err)
	}
	r.retireLessonMine(ctx)
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_econ_lesson
		WHERE kind='retire_postmortem'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("窗外死案被补: n=%d err=%v", n, err)
	}
}

// TestRetireLessonMineNoRetireNoLesson 非退役 transition（paper→probe）不产案。
func TestRetireLessonMineNoRetireNoLesson(t *testing.T) {
	r := retireRig(t)
	ctx := context.Background()
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction)
		VALUES('RM-LIVE-USDT-SWAP','funding-hedge','paper',100,'short_perp_long_spot') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	if err := Transition(ctx, r.pool, sid, "probe", "判据满", false); err != nil {
		t.Fatal(err)
	}
	r.retireLessonMine(ctx)
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_econ_lesson
		WHERE kind='retire_postmortem'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("非退役产案: n=%d err=%v", n, err)
	}
}
