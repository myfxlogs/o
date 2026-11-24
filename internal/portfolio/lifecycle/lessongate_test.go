package lifecycle

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
)

// O-38942 验收锚：尸签命中→提案 fin_retired(dead_signature)+「教训消费」
// 审计行+零策略行；尸签不命中（签名异/净额非负/闸关）→放行 armed；
// 统计面读败=fail-open。真库实证径同构——本测即消费点端到端实证。

// mkCorpseL 验尸案桩——verdicted retire_postmortem 行（numbers 机读面
// 照 retireminer 产线形：template_ref/inst_id/direction/reason/
// realized_usd/paper_net_usd/age_days）。
func mkCorpseL(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	inst, reason string, realized, paperNet float64) {
	t.Helper()
	nums, _ := json.Marshal(map[string]any{
		"template_ref": "funding-hedge", "inst_id": inst,
		"direction": "short_perp_long_spot", "reason": reason,
		"realized_usd": realized, "paper_net_usd": paperNet, "age_days": 10})
	if _, err := pool.Exec(ctx, `INSERT INTO fin_econ_lesson(kind,entity_ref,numbers,status)
		VALUES('retire_postmortem',$1,$2::jsonb,'verdicted')`,
		"strategy#"+inst+"#"+reason, string(nums)); err != nil {
		t.Fatal(err)
	}
}

func mkDSProp(t *testing.T, ctx context.Context, pool *pgxpool.Pool, inst string) int64 {
	t.Helper()
	var eid, pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES($1,'funding-hedge','short_perp_long_spot',0.08,'proposed','{}','{}') RETURNING id`, inst).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,$2,'short_perp_long_spot',0.11) RETURNING id`, eid, inst).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid
}

func dsPropState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, pid int64) (string, string) {
	t.Helper()
	var st, rr string
	if err := pool.QueryRow(ctx, `SELECT state,COALESCE(retire_reason,'') FROM fin_proposal WHERE id=$1`, pid).
		Scan(&st, &rr); err != nil {
		t.Fatal(err)
	}
	return st, rr
}

// 命中→抑制：同签同因净亏尸在案→提案 fin_retired+审计行+零策略行；
// 异签提案同拍放行（闸只抑命中签不连坐）。
func TestDeadSignatureSuppress(t *testing.T) {
	ctx, pool, cfg := setup(t)
	r := NewRunner(pool, cfg)

	mkCorpseL(t, ctx, pool, "DS-DEAD-USDT-SWAP", "dir_stop_loss", -40, 0)
	deadPID := mkDSProp(t, ctx, pool, "DS-DEAD-USDT-SWAP")
	freePID := mkDSProp(t, ctx, pool, "DS-FREE-USDT-SWAP")

	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	st, rr := dsPropState(t, ctx, pool, deadPID)
	if st != "fin_retired" || rr != "dead_signature" {
		t.Fatalf("命中提案应 fin_retired/dead_signature: %s/%s", st, rr)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy WHERE inst_id='DS-DEAD-USDT-SWAP'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("命中签不应建策略行 n=%d err=%v", n, err)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='教训消费' AND note LIKE '%DS-DEAD-USDT-SWAP%' ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil {
		t.Fatalf("教训消费审计行缺席: %v", err)
	}
	for _, want := range []string{"dead_signature", "dir_stop_loss", "尸"} {
		if !strings.Contains(note, want) {
			t.Fatalf("审计行缺要素 %q: %s", want, note)
		}
	}
	if st, _ := dsPropState(t, ctx, pool, freePID); st != "fin_armed" {
		t.Fatalf("异签提案应放行 armed: %s", st)
	}
	// 族尸统计面同迁移在案——scan 内节流刷新后可查
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM v_lesson_family_stats`).Scan(&rows); err != nil {
		t.Fatalf("v_lesson_family_stats 不可查: %v", err)
	}
}

// 净额非负尸不抑（尸在但没亏钱=非死签）；异模板尸不误伤。
func TestDeadSignatureNetNonNegPass(t *testing.T) {
	ctx, pool, cfg := setup(t)
	r := NewRunner(pool, cfg)

	mkCorpseL(t, ctx, pool, "DS-POS-USDT-SWAP", "paper_ttl_expired", 0, 30)
	mkDSProp(t, ctx, pool, "DS-POS-USDT-SWAP")
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy WHERE inst_id='DS-POS-USDT-SWAP'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("净额非负尸不应抑——策略行应建 n=%d err=%v", n, err)
	}
}

// 闸关=回滚不撤码：lifecycle.lesson_dead_sig=false→同签尸在案照放行。
func TestDeadSignatureGateOff(t *testing.T) {
	ctx, pool, _ := setup(t)
	if _, err := pool.Exec(ctx, `INSERT INTO app_config(key,value) VALUES('lifecycle.lesson_dead_sig','false'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value='false'::jsonb`); err != nil {
		t.Fatal(err)
	}
	cfg := configFor(t, pool, ctx)
	r := NewRunner(pool, cfg)

	mkCorpseL(t, ctx, pool, "DS-OFF-USDT-SWAP", "dir_stop_loss", -40, 0)
	pid := mkDSProp(t, ctx, pool, "DS-OFF-USDT-SWAP")
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if st, _ := dsPropState(t, ctx, pool, pid); st != "fin_armed" {
		t.Fatalf("闸关后应放行 armed: %s", st)
	}
}

// 统计面读败=fail-open：判据查询注入败→放行不抑（禁「查尸败=全抑」）。
func TestDeadSignatureReadFailOpen(t *testing.T) {
	ctx, pool, cfg := setup(t)
	r := NewRunner(pool, cfg)
	mkCorpseL(t, ctx, pool, "DS-FAIL-USDT-SWAP", "dir_stop_loss", -40, 0)

	badCtx, cancel := context.WithCancel(ctx)
	cancel() // 注销 ctx——corpse 查询必败
	if r.deadSigSuppress(badCtx, 999999, "funding-hedge", "DS-FAIL-USDT-SWAP", "short_perp_long_spot") {
		t.Fatal("读败应 fail-open 放行（返 false）")
	}
}

// configFor 重载快照（闸键写后须重载——config.Client 走内存快照）。
func configFor(t *testing.T, pool *pgxpool.Pool, ctx context.Context) *config.Client {
	t.Helper()
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return cfg
}
