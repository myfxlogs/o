package lifecycle

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0426 毕业闸红桩：probe→live 判据增「执行径演习通过」——判据件缺/演习败
// =fail-closed 不晋（业主令 2026-11-07「每策略毕业前就该检查订单会不会出错」）。
func TestGraduationGateExecDrillFailClosed(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`DELETE FROM fin_exec_drill WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref='drill-gate-test')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`DELETE FROM fin_strategy_transition WHERE strategy_id IN
		   (SELECT id FROM fin_strategy WHERE template_ref='drill-gate-test')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_strategy WHERE template_ref='drill-gate-test'`); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"invest.lifecycle.paper_min_hours": "0", "invest.lifecycle.paper_min_net_usd": "0.001",
		"invest.lifecycle.paper_max_dd_usd": "5", "invest.lifecycle.live_switch": "true",
		"invest.paper.notional_usd": "100"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// probe 态候选：观察窗/净回报判据全满（经济判据域绿——只差演习凭据）。
	var sid int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		VALUES('DG-USDT-SWAP','drill-gate-test','probe',100,'okx') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,ts)
		VALUES($1,'funding','DG-USDT-SWAP',1,now())`, sid); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	// ①判据件缺（无演习凭据）→不晋（fail-closed 同经济判据缺位待遇）。
	if err := r.promote(ctx, 0, 0.001, 5, true); err != nil {
		t.Fatal(err)
	}
	assertState(t, pool, sid, "probe")
	// ②演习败→不晋（fail-closed）。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_exec_drill(strategy_ref,inst_id,verdict,rules_version,detail)
		VALUES($1,'DG-USDT-SWAP','fail','simvenue/1','账实不平')`, sid); err != nil {
		t.Fatal(err)
	}
	if err := r.promote(ctx, 0, 0.001, 5, true); err != nil {
		t.Fatal(err)
	}
	assertState(t, pool, sid, "probe")
	// ③演习绿→晋 live（判据链全过+留痕携演习词）。
	if _, err := pool.Exec(ctx, `UPDATE fin_exec_drill SET verdict='pass' WHERE strategy_ref=$1`, sid); err != nil {
		t.Fatal(err)
	}
	if err := r.promote(ctx, 0, 0.001, 5, true); err != nil {
		t.Fatal(err)
	}
	assertState(t, pool, sid, "live")
	var reason string
	if err := pool.QueryRow(ctx, `SELECT reason FROM fin_strategy_transition
		WHERE strategy_id=$1 AND to_state='live' ORDER BY id DESC LIMIT 1`, sid).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reason, "执行径演习绿") {
		t.Fatalf("迁移留痕应携演习判据词: %q", reason)
	}
}

func assertState(t *testing.T, pool *pgxpool.Pool, sid int64, want string) {
	t.Helper()
	var got string
	if err := pool.QueryRow(context.Background(),
		"SELECT state FROM fin_strategy WHERE id=$1", sid).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("策略态应 %s（fail-closed 闸）: got %s", want, got)
	}
}
