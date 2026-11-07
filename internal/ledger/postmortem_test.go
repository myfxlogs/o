package ledger

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"o/internal/platform/config"
)

// pmPos 结案判定用仓 fixture：execution+position（可指定开/平时点），
// guard 子面行/ops_decision close 理由行按需落。
func pmPos(t *testing.T, ctx context.Context, r *Runner, notional float64, opened, closed time.Time) int64 {
	t.Helper()
	var xid, pid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('X',$1,'closed') RETURNING id`, notional).Scan(&xid); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,opened_at,closed_at)
		VALUES($1,'X',$2,$3,$4) RETURNING id`, xid, notional, opened, closed).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid
}

func pmGuardClose(t *testing.T, ctx context.Context, r *Runner, pid int64, trig string) {
	t.Helper()
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_guard_close(position_id,state,trigger_kind,closed_at)
		VALUES($1,'closed',$2,now())`, pid, trig); err != nil {
		t.Fatal(err)
	}
}

func pmDecisionClose(t *testing.T, ctx context.Context, r *Runner, pid int64, trig, reason string) {
	t.Helper()
	if _, err := r.pool.Exec(ctx, `SELECT ops_decision_write('fin','guard','0','close',$1,
		jsonb_build_object('position_id',$2::float8,'trigger',$3::text))`, reason, pid, trig); err != nil {
		t.Fatal(err)
	}
}

func pmCash(t *testing.T, ctx context.Context, r *Runner, pid int64, amt float64) {
	t.Helper()
	if _, err := Post(ctx, r.pool, Entry{Kind: "trade_pnl", Venue: "okx", Ccy: "USDT",
		Amount: amt, PositionRef: &pid, SourceRef: 1, SourceTable: "fin_order_leg"}); err != nil {
		t.Fatal(err)
	}
}

func pmLessonCount(t *testing.T, ctx context.Context, r *Runner, pid int64) int {
	t.Helper()
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_econ_lesson
		WHERE kind='trade_postmortem' AND entity_ref=$1 AND status='pending'`,
		fmt.Sprintf("position#%d", pid)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func pmOutcome(t *testing.T, ctx context.Context, r *Runner, pid int64) (int, string) {
	t.Helper()
	var n int
	var verdict string
	if err := r.pool.QueryRow(ctx, `SELECT count(*),COALESCE(max(payload->>'verdict'),'')
		FROM ops_decision WHERE domain='fin' AND action='outcome'
			AND payload->>'position_id'=$1`,
		fmt.Sprint(pid)).Scan(&n, &verdict); err != nil {
		t.Fatal(err)
	}
	return n, verdict
}

// 红桩①：15s 即平+|settled| 超阈（-49% 实证类）→监察案入裁定席 pending 集。
func TestPostmortemFastCloseBigLossCase(t *testing.T) {
	ctx, r := setup(t)
	now := time.Now()
	pid := pmPos(t, ctx, r, 100, now.Add(-15*time.Second), now)
	pmGuardClose(t, ctx, r, pid, "funding_flip")
	pmDecisionClose(t, ctx, r, pid, "funding_flip", "funding_flip：latest=-0.01×sign=1 |率|100bps>flip_bps=5")
	pmCash(t, ctx, r, pid, -49)
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := pmLessonCount(t, ctx, r, pid); n != 1 {
		t.Fatalf("监察案行=%d（应 1 pending）", n)
	}
	var nums string
	if err := r.pool.QueryRow(ctx, `SELECT numbers::text FROM fin_econ_lesson
		WHERE kind='trade_postmortem' AND entity_ref=$1`,
		fmt.Sprintf("position#%d", pid)).Scan(&nums); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"fast_close", "settled_over"} {
		if !strings.Contains(nums, code) {
			t.Fatalf("监察案 anomalies 缺 %s: %s", code, nums)
		}
	}
	if n, _ := pmOutcome(t, ctx, r, pid); n != 0 {
		t.Fatalf("异常件不得落 outcome 行（n=%d）", n)
	}
	if r.rej.Snapshot()["ledger/postmortem_anomaly"] == 0 {
		t.Fatal("异常件未计门")
	}
}

// 红桩②：正常 funding_flip 平仓（10h 持+小赚）→expected 判词落 ops_decision 不产案。
func TestPostmortemExpectedOutcome(t *testing.T) {
	ctx, r := setup(t)
	now := time.Now()
	pid := pmPos(t, ctx, r, 100, now.Add(-10*time.Hour), now)
	pmGuardClose(t, ctx, r, pid, "funding_flip")
	pmDecisionClose(t, ctx, r, pid, "funding_flip", "funding_flip：latest=-0.01×sign=1 |率|100bps>flip_bps=5")
	pmCash(t, ctx, r, pid, 1.2)
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := pmLessonCount(t, ctx, r, pid); n != 0 {
		t.Fatalf("正常件产案（n=%d）", n)
	}
	n, verdict := pmOutcome(t, ctx, r, pid)
	if n != 1 || verdict != "expected" {
		t.Fatalf("outcome 行=%d verdict=%s（应 1/expected）", n, verdict)
	}
}

// 触发认知集外+残袋——两异常类各立案；手平判据内=expected（trigger 归一 manual）。
func TestPostmortemAnomalyClasses(t *testing.T) {
	ctx, r := setup(t)
	now := time.Now()
	// ① 认知集外触发（guard 子面词不在册——正常持时/小额仍异常）
	pid1 := pmPos(t, ctx, r, 100, now.Add(-2*time.Hour), now)
	pmGuardClose(t, ctx, r, pid1, "weird_close")
	pmCash(t, ctx, r, pid1, 0.5)
	// ② 残袋>0（手平+残袋——bare_bag 异常类）
	pid2 := pmPos(t, ctx, r, 100, now.Add(-2*time.Hour), now)
	pmCash(t, ctx, r, pid2, -0.5)
	if _, err := r.pool.Exec(ctx, `UPDATE fin_position SET bare_bag_ccy='CORE',
		bare_bag_qty=100,bare_bag_usd=50 WHERE id=$1`, pid2); err != nil {
		t.Fatal(err)
	}
	// ③ 手平正常件（无 guard 行无决策行——trigger 归一 manual 在认知集）
	pid3 := pmPos(t, ctx, r, 100, now.Add(-2*time.Hour), now)
	pmCash(t, ctx, r, pid3, 0.5)
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	var nums string
	if err := r.pool.QueryRow(ctx, `SELECT numbers::text FROM fin_econ_lesson
		WHERE entity_ref=$1`, fmt.Sprintf("position#%d", pid1)).Scan(&nums); err != nil {
		t.Fatalf("认知集外案缺: %v", err)
	}
	if !strings.Contains(nums, "trigger_unknown") {
		t.Fatalf("案①缺 trigger_unknown: %s", nums)
	}
	if err := r.pool.QueryRow(ctx, `SELECT numbers::text FROM fin_econ_lesson
		WHERE entity_ref=$1`, fmt.Sprintf("position#%d", pid2)).Scan(&nums); err != nil {
		t.Fatalf("残袋案缺: %v", err)
	}
	if !strings.Contains(nums, "bare_bag") {
		t.Fatalf("案②缺 bare_bag: %s", nums)
	}
	n, verdict := pmOutcome(t, ctx, r, pid3)
	if n != 1 || verdict != "expected" {
		t.Fatalf("手平正常件 outcome=%d/%s", n, verdict)
	}
	var trig string
	if err := r.pool.QueryRow(ctx, `SELECT payload->>'trigger' FROM ops_decision
		WHERE action='outcome' AND payload->>'position_id'=$1`, fmt.Sprint(pid3)).Scan(&trig); err != nil {
		t.Fatal(err)
	}
	if trig != "manual" {
		t.Fatalf("手平 trigger 归一败=%s", trig)
	}
}

// 判据键缺=fail-closed 跳判+missing_param 计数（不产案不落 outcome 不炸结算）。
func TestPostmortemMissingParamFailClosed(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx, `DELETE FROM app_config WHERE key='guard.postmortem.min_hold_minutes'`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(r.pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r.cfg = cfg
	now := time.Now()
	pid := pmPos(t, ctx, r, 100, now.Add(-15*time.Second), now) // 秒级即平本必异常——键缺跳判不产案
	pmCash(t, ctx, r, pid, -49)
	if err := r.settle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := pmLessonCount(t, ctx, r, pid); n != 0 {
		t.Fatalf("键缺误产案 n=%d", n)
	}
	if n, _ := pmOutcome(t, ctx, r, pid); n != 0 {
		t.Fatalf("键缺误落 outcome n=%d", n)
	}
	if r.rej.Snapshot()["ledger/missing_param"] == 0 {
		t.Fatal("键缺未计 missing_param")
	}
}
