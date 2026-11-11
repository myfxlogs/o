// audit_1111_lifecycle_test.go —— 审计/生命周期 RED 锚（audit/1111-lifecycle）：
// 断言正确语义常驻——双锚即修复面回归。
package lifecycle

import (
	"context"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func auditSetup(t *testing.T) (context.Context, *Runner) {
	t.Helper()
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`DELETE FROM fin_exec_drill WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref='audit-ttl-test')`,
		`DELETE FROM fin_strategy WHERE template_ref='audit-ttl-test'`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, NewRunner(pool, cfg)
}

// ① armed_ttl 活动锚漏毕业凭据：retireStaleArmedStrategies 判据锚=
// entered_at/fin_position/fin_execution/fin_eval_reject 四件——独漏
// fin_exec_drill（probe→live 钦定毕业凭据产能证据）。非 funding 族
// probe 拿不到 eval_reject 锚（配模板只含 execfam.FundingHedgeList）、
// 又不可有 execution/position（开仓在 live 后）→恒死于 entered_at+
// armed_ttl（产线配置 4h），演习凭据再新也救不回=毕业径 4h 黑洞
// （产线 probe 态零行、非 funding paper 滞 57h+ 实态同型）。正确语义=
// 毕业凭据行（strategy_ref 同柄新鲜 drill 行）计入活动锚。
func TestAuditArmedTtlIgnoresDrillCred(t *testing.T) {
	ctx, r := auditSetup(t)
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd,venue,entered_at)
		VALUES('AD-USDT-261204-66000-P','audit-ttl-test','probe',100,'okx',
			now()-interval '5 hours') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	// 毕业凭据 1h 前新落——probe 在积极毕业（凭据产能证据当计活动）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_exec_drill
		(strategy_ref,inst_id,verdict,rules_version,detail,ts)
		VALUES($1,'AD-USDT-261204-66000-P','pass','a','{}',now()-interval '1 hour')`, sid); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleArmedStrategies(ctx, 4); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "probe" {
		t.Fatalf("probe 携 1h 新毕业凭据仍被 armed_ttl 退役（state=%s）——drill 凭据未计活动锚=非 funding probe 4h 黑洞", st)
	}
}

// ② 毕业闸凭据无龄界无哨段属：execDrillPass 只 ORDER BY ts DESC——陈凭据
// （早于本哨段 entered_at 甚至数日前）照认。对拍件 mine.sh R3 判词钦定
// 「probe>2d 无晚于 entered_at-2d 的 pass 凭据=凭据产线断」——闸与侦测面
// 对同一行判词互斥（闸收而侦测报缺）。正确语义=凭据须晚于
// entered_at-2d（本哨段产能证据，R3 同口径）。
func TestAuditDrillCredStaleCrossStint(t *testing.T) {
	ctx, r := auditSetup(t)
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd,venue,entered_at)
		VALUES('AD2-USDT-261204-66000-P','audit-ttl-test','probe',100,'okx',
			now()-interval '1 hour') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	// 凭据产在 10 天前（本哨段前的旧凭据）——闸当认凭据断。
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_exec_drill
		(strategy_ref,inst_id,verdict,rules_version,detail,ts)
		VALUES($1,'AD2-USDT-261204-66000-P','pass','a','{}',now()-interval '10 days')`, sid); err != nil {
		t.Fatal(err)
	}
	pass, drilled, err := r.execDrillPass(ctx, sid)
	if err != nil {
		t.Fatal(err)
	}
	if drilled && pass {
		t.Fatalf("10 天前陈凭据被毕业闸认收——凭据无哨段龄界（R3 判词:晚于 entered_at-2d 才算在案）")
	}
}
