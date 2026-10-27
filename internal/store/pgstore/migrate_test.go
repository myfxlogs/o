package pgstore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/migrations"
	"strings"
)

// TestMigrateChains 两链各空库全过+幂等（T-0167 桩锚——PG-md/SG 建库即用前提）。
// 独立 schema=独立库语义（testutil 包级隔离）：md 链/fin 链各占一 schema 互不染。
func TestMigrateChains(t *testing.T) {
	ctx := context.Background()
	for _, domain := range []string{"md", "fin"} {
		t.Run(domain, func(t *testing.T) {
			pool := testutil.Pool(t)
			if err := Migrate(ctx, pool, domain); err != nil {
				t.Fatalf("migrate %s: %v", domain, err)
			}
			if err := Migrate(ctx, pool, domain); err != nil {
				t.Fatalf("migrate %s 二次幂等: %v", domain, err)
			}
			want := 0
			for _, m := range migrationTable {
				if strings.HasPrefix(m.Name, domain+"_") {
					want++
				}
			}
			var n int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != want {
				t.Fatalf("%s 链 schema_migrations=%d 登记=%d 不齐", domain, n, want)
			}
		})
	}
}

// TestMigrateDomainFailClosed 非法域拒（分域选链 fail-closed——不选链=不执行）。
func TestMigrateDomainFailClosed(t *testing.T) {
	pool := testutil.Pool(t)
	if err := Migrate(context.Background(), pool, ""); err == nil {
		t.Fatal("空域应拒")
	}
	if err := Migrate(context.Background(), pool, "both"); err == nil {
		t.Fatal("非法域应拒")
	}
}

// TestMigrateChainSurface 两链各自的域面在位、他域面缺席（D-0030 §三 每表一归属）。
func TestMigrateChainSurface(t *testing.T) {
	ctx := context.Background()
	t.Run("md", func(t *testing.T) {
		pool := testutil.Pool(t)
		if err := Migrate(ctx, pool, "md"); err != nil {
			t.Fatalf("migrate md: %v", err)
		}
		for _, tbl := range []string{"md_ticks", "md_signal", "md_funding", "fin_opp_eval", "fin_opp_hypothesis", "fin_forward_check"} {
			if !tableExists(t, pool, tbl) {
				t.Fatalf("md 链缺表 %s", tbl)
			}
		}
		for _, tbl := range []string{"fin_position", "fin_cashflow", "fin_active_set"} {
			if tableExists(t, pool, tbl) {
				t.Fatalf("md 链不应有 fin 域表 %s", tbl)
			}
		}
		// app_config=T-0169 第九桥只读副本（bridge——md 链应有；app_secrets 仍 fin 独占）。
		if !tableExists(t, pool, "app_config") {
			t.Fatal("md 链应有 app_config 第九桥副本")
		}
		if tableExists(t, pool, "app_secrets") {
			t.Fatal("md 链不应有 app_secrets（密文域不出 SG）")
		}
	})
	t.Run("fin", func(t *testing.T) {
		pool := testutil.Pool(t)
		if err := Migrate(ctx, pool, "fin"); err != nil {
			t.Fatalf("migrate fin: %v", err)
		}
		for _, tbl := range []string{"fin_position", "fin_cashflow", "app_config", "md_ticks", "md_macro_event", "fin_eval_reject", "fin_active_set"} {
			if !tableExists(t, pool, tbl) {
				t.Fatalf("fin 链缺表 %s", tbl)
			}
		}
		if tableExists(t, pool, "fin_opp_hypothesis") {
			t.Fatal("fin 链不应有 md 域表 fin_opp_hypothesis")
		}
	})
}

// TestMigrateFundingUnique 幂等键落位+去重幂等（T-0167 桩锚/D-0032 §三-2）：
// 键在（同键二次插拒）；人为拆键造重后重放 0066 件=去重留 recv_ts 最新+键复建。
func TestMigrateFundingUnique(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate fin: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,venue,inst_id,funding_rate)
		VALUES ('2026-10-26 08:00:00+00','okx','BTC-USDT-SWAP',0.0001)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,venue,inst_id,funding_rate)
		VALUES ('2026-10-26 08:00:00+00','okx','BTC-USDT-SWAP',0.0002)`); err == nil {
		t.Fatal("UNIQUE(venue,inst_id,ts) 未生效——同键二次插应拒")
	}
	// 拆键造重→重放 0066 件：去重留 recv_ts 最新行+键复建（迁移幂等可重放）。
	if _, err := pool.Exec(ctx, "DROP INDEX md_funding_venue_inst_ts_uq"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,venue,inst_id,funding_rate,recv_ts)
		VALUES ('2026-10-26 08:00:00+00','okx','BTC-USDT-SWAP',0.0003,now()+interval '5 seconds'),
		       ('2026-10-26 08:00:00+00','okx','BTC-USDT-SWAP',0.0004,now()+interval '9 seconds')`); err != nil {
		t.Fatal(err)
	}
	body, err := migrations.FS.ReadFile("fin_0066_md_funding_uq.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(body)); err != nil {
		t.Fatalf("0066 重放: %v", err)
	}
	var n int
	var rate float64
	if err := pool.QueryRow(ctx, `SELECT count(*), COALESCE(max(funding_rate),0) FROM md_funding
		WHERE venue='okx' AND inst_id='BTC-USDT-SWAP'`).Scan(&n, &rate); err != nil {
		t.Fatal(err)
	}
	if n != 1 || rate != 0.0004 {
		t.Fatalf("去重留最新未达：n=%d rate=%v（应 1 行 0.0004）", n, rate)
	}
}

// TestFinActiveSetTrigger 活跃集发布面（T-0167 桩锚/D-0030 §四）：
// 非 retired 策略入集；策略退役+无在仓仓位=出集；在仓仓位入集、平仓出集。
func TestFinActiveSetTrigger(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate fin: %v", err)
	}
	inSet := func(t *testing.T, inst string) bool {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx,
			"SELECT count(*) FROM fin_active_set WHERE inst_id=$1", inst).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n > 0
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		VALUES ('BTC-USDT-SWAP','funding-hedge','live',1000)`); err != nil {
		t.Fatal(err)
	}
	if !inSet(t, "BTC-USDT-SWAP") {
		t.Fatal("live 策略应入活跃集")
	}
	if _, err := pool.Exec(ctx,
		"UPDATE fin_strategy SET state='retired' WHERE inst_id='BTC-USDT-SWAP'"); err != nil {
		t.Fatal(err)
	}
	if inSet(t, "BTC-USDT-SWAP") {
		t.Fatal("策略退役且无仓位应出活跃集")
	}
	// 仓位在仓=入集；closed_at 落=出集（与 opener/run.go 未平仓谓词同口径）。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		VALUES ('ETH-USDT-SWAP','funding-hedge','live',1000)`); err != nil {
		t.Fatal(err)
	}
	var strat int64
	if err := pool.QueryRow(ctx,
		"SELECT id FROM fin_strategy WHERE inst_id='ETH-USDT-SWAP'").Scan(&strat); err != nil {
		t.Fatal(err)
	}
	var exec int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES ($1,'ETH-USDT-SWAP',1000,'open') RETURNING id`, strat).Scan(&exec); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		VALUES ($1,$2,'ETH-USDT-SWAP',1000)`, exec, strat); err != nil {
		t.Fatal(err)
	}
	if !inSet(t, "ETH-USDT-SWAP") {
		t.Fatal("在仓仓位应入活跃集")
	}
	if _, err := pool.Exec(ctx,
		"UPDATE fin_position SET closed_at=now() WHERE inst_id='ETH-USDT-SWAP'"); err != nil {
		t.Fatal(err)
	}
	if !inSet(t, "ETH-USDT-SWAP") {
		t.Fatal("仓位平仓但策略仍 live 应留活跃集")
	}
	if _, err := pool.Exec(ctx,
		"UPDATE fin_strategy SET state='retired' WHERE inst_id='ETH-USDT-SWAP'"); err != nil {
		t.Fatal(err)
	}
	if inSet(t, "ETH-USDT-SWAP") {
		t.Fatal("仓位平仓且策略退役应出活跃集")
	}
}

// TestFinEvalRejectKey 拆表幂等键（T-0167 桩锚/D-0032 §三-3）：
// 同 (inst,template,direction,reason) 覆写不重行，异 reason 各行留痕。
func TestFinEvalRejectKey(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate fin: %v", err)
	}
	ins := `INSERT INTO fin_eval_reject(inst_id,template,direction,reason,evidence)
		VALUES ('BTC-USDT-SWAP','funding-hedge','short_perp_long_spot',$1,'{}')
		ON CONFLICT (inst_id,template,direction,reason) DO UPDATE SET ts=now(), evidence=EXCLUDED.evidence`
	for i := 0; i < 2; i++ {
		if _, err := pool.Exec(ctx, ins, "breakeven_beyond_hold"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, ins, "lifetime_below_min"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_eval_reject WHERE inst_id='BTC-USDT-SWAP'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("fin_eval_reject 行数=%d（应 2：同因覆写+异因各行）", n)
	}
}

func tableExists(t *testing.T, pool *pgxpool.Pool, tbl string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name=$1 AND table_schema=current_schema())", tbl).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	return exists
}
