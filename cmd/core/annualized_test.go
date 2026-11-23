// annualized_test.go —— 工单#34703 桩测：deployedBasis 时加权分母
// 三案（单仓全窗/半窗开合/零部署→"—"）+ DCD 快照腿 snap 标。
package main

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestDeployedBasisFullWindow(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal("md migrate:", err)
	}
	// 单仓全窗：opened 40d 前 notional=1000 未平——30d 窗全覆盖→均值=1000
	var execID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution
		(inst_id,notional_usd,state) VALUES('ETH-USDT-SWAP',1000,'legged')
		RETURNING id`).Scan(&execID); err != nil {
		t.Fatal("execution 桩:", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position
		(execution_id,inst_id,notional_usd,opened_at,closed_at)
		VALUES($1,'ETH-USDT-SWAP',1000,now()-interval '40 days',NULL)`, execID); err != nil {
		t.Fatal("position 桩:", err)
	}
	c := coreService{pool: pool}
	mean, ok, snap := c.deployedBasis(ctx, time.Now().Add(-30*24*time.Hour), time.Now())
	if !ok {
		t.Fatal("deployedBasis 不可得")
	}
	if snap {
		t.Fatal("无 dcd 腿不应 snap")
	}
	if math.Abs(mean-1000) > 0.01 {
		t.Fatalf("全窗单仓 mean=%v want 1000", mean)
	}
	// 年化算式同源：net=30 dep=1000 days=30 → 36.5%
	ann, aok := annualizedPct(30, 30, mean)
	if !aok || math.Abs(ann-36.5) > 0.001 {
		t.Fatalf("annualizedPct=%v ok=%v want 36.5", ann, aok)
	}
}

func TestDeployedBasisPartialWindow(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal("md migrate:", err)
	}
	// 半窗开合：opened 20d 前、closed 10d 前 notional=2000——
	// 交叠 10d/30d → 2000×1/3≈666.67
	var execID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution
		(inst_id,notional_usd,state) VALUES('BTC-USDT-SWAP',2000,'legged')
		RETURNING id`).Scan(&execID); err != nil {
		t.Fatal("execution 桩:", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position
		(execution_id,inst_id,notional_usd,opened_at,closed_at,state)
		VALUES($1,'BTC-USDT-SWAP',2000,now()-interval '20 days',
			now()-interval '10 days','closed')`, execID); err != nil {
		t.Fatal("position 桩:", err)
	}
	c := coreService{pool: pool}
	mean, ok, _ := c.deployedBasis(ctx, time.Now().Add(-30*24*time.Hour), time.Now())
	if !ok {
		t.Fatal("deployedBasis 不可得")
	}
	if math.Abs(mean-2000.0/3) > 0.5 {
		t.Fatalf("半窗开合 mean=%v want ~%.2f", mean, 2000.0/3)
	}
}

func TestDeployedBasisZeroDeploy(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal("md migrate:", err)
	}
	c := coreService{pool: pool}
	mean, ok, snap := c.deployedBasis(ctx, time.Now().Add(-30*24*time.Hour), time.Now())
	if !ok {
		t.Fatal("deployedBasis 不可得")
	}
	if mean != 0 || snap {
		t.Fatalf("零部署 mean=%v snap=%v want 0/false", mean, snap)
	}
	// 零部署→年化"—"：annualizedPct 拒算 + 读面签显 $0
	if _, aok := annualizedPct(5, 30, mean); aok {
		t.Fatal("零部署不应出年化值")
	}
	if tag := annualDepTag(mean, ok, snap); tag != "（部署$0）" {
		t.Fatalf("零部署读面签=%q want （部署$0）", tag)
	}
}

func TestDeployedBasisDCDSnap(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal("md migrate:", err)
	}
	// DCD 产品仓：dcd_baseline 凭据 amt=100 TRX + tick 0.3 → 快照腿=30
	var decID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_risk_decision
		(intent,decision,reason,config_gen,kind)
		VALUES('{"op":"dcd_baseline","venue":"okx","ccy":"TRX","amt":"100"}'::jsonb,
		       'allow','allow',1,'open') RETURNING id`).Scan(&decID); err != nil {
		t.Fatal("decision 桩:", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now(),'okx','TRX-USDT',0.3)`); err != nil {
		t.Fatal("tick 桩:", err)
	}
	c := coreService{pool: pool}
	mean, ok, snap := c.deployedBasis(ctx, time.Now().Add(-30*24*time.Hour), time.Now())
	if !ok {
		t.Fatal("deployedBasis 不可得")
	}
	if !snap {
		t.Fatal("dcd 腿在场应 snap=true")
	}
	if math.Abs(mean-30) > 0.5 {
		t.Fatalf("dcd 快照 mean=%v want ~30", mean)
	}
	if tag := annualDepTag(mean, ok, snap); tag != "（部署$30）"+annualSnapMark {
		t.Fatalf("快照读面签=%q", tag)
	}
}

// —— O-34963 下限补丁桩测：annualJudgment 判据输入两态
// （部署≥floor→部署口径；<floor→NAV 回退）四案+floor 可配。

func annJudgeSetup(t *testing.T) (context.Context, *coreService, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal("md migrate:", err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal("cfg load:", err)
	}
	return ctx, &coreService{pool: pool, cfg: cfg}, pool
}

func annJudgeNav(t *testing.T, ctx context.Context, pool *pgxpool.Pool, nav float64) {
	t.Helper()
	if _, err := pool.Exec(ctx, `INSERT INTO md_equity_curve(ts,venue,ccy,equity,src)
		VALUES(now(),'okx','NAV',$1,'test')`, nav); err != nil {
		t.Fatal("NAV 桩:", err)
	}
}

func annJudgePosition(t *testing.T, ctx context.Context, pool *pgxpool.Pool, notional float64) {
	t.Helper()
	var execID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution
		(inst_id,notional_usd,state) VALUES('ETH-USDT-SWAP',$1,'legged')
		RETURNING id`, notional).Scan(&execID); err != nil {
		t.Fatal("execution 桩:", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_position
		(execution_id,inst_id,notional_usd,opened_at,closed_at)
		VALUES($1,'ETH-USDT-SWAP',$2,now()-interval '40 days',NULL)`,
		execID, notional); err != nil {
		t.Fatal("position 桩:", err)
	}
}

// 案一：零部署→dep=0<floor→NAV 回退（净$30/NAV$10000→3.65%——
// 非部署口径的 0 除，读面签带「<下限→NAV口径」）。
func TestAnnualJudgmentZeroDeploy(t *testing.T) {
	ctx, c, pool := annJudgeSetup(t)
	annJudgeNav(t, ctx, pool, 10000)
	pct, ok, tag := c.annualJudgment(ctx,
		time.Now().Add(-30*24*time.Hour), time.Now(), 30, 30)
	if !ok {
		t.Fatal("零部署+NAV 在场应出 NAV 口径值")
	}
	if math.Abs(pct-3.65) > 0.001 {
		t.Fatalf("零部署 NAV 口径 pct=%v want ~3.65", pct)
	}
	if !strings.Contains(tag, "NAV口径") || !strings.Contains(tag, "部署$0") {
		t.Fatalf("零部署读面签=%q want 部署$0<下限→NAV口径", tag)
	}
}

// 案二：低于 floor（实证场景部署$76/净$20.68/NAV$10000→≈2.52%
// NAV 口径，非部署口径 330%）。
func TestAnnualJudgmentBelowFloor(t *testing.T) {
	ctx, c, pool := annJudgeSetup(t)
	annJudgeNav(t, ctx, pool, 10000)
	annJudgePosition(t, ctx, pool, 76)
	pct, ok, tag := c.annualJudgment(ctx,
		time.Now().Add(-30*24*time.Hour), time.Now(), 20.68, 30)
	if !ok {
		t.Fatal("低部署+NAV 在场应出 NAV 口径值")
	}
	want := 20.68 * 365 / 30 / 10000 * 100
	if math.Abs(pct-want) > 0.001 {
		t.Fatalf("低部署 pct=%v want NAV 口径 %.4f（非部署口径 %.1f）", pct, want,
			20.68*365/30/76*100)
	}
	if !strings.Contains(tag, "NAV口径") || !strings.Contains(tag, "部署$76") {
		t.Fatalf("低部署读面签=%q", tag)
	}
}

// 案三：高于 floor（部署$1000→部署口径照常 #34702 语义不变）。
func TestAnnualJudgmentAboveFloor(t *testing.T) {
	ctx, c, pool := annJudgeSetup(t)
	annJudgePosition(t, ctx, pool, 1000)
	pct, ok, tag := c.annualJudgment(ctx,
		time.Now().Add(-30*24*time.Hour), time.Now(), 30, 30)
	if !ok {
		t.Fatal("高部署应出部署口径值")
	}
	if math.Abs(pct-36.5) > 0.001 {
		t.Fatalf("高部署 pct=%v want 36.5", pct)
	}
	if tag != "（部署$1000）" {
		t.Fatalf("高部署读面签=%q want （部署$1000）", tag)
	}
}

// 案四：恰等 floor（部署$500==下限→部署口径——<floor 才回退）。
func TestAnnualJudgmentAtFloor(t *testing.T) {
	ctx, c, pool := annJudgeSetup(t)
	annJudgePosition(t, ctx, pool, 500)
	pct, ok, tag := c.annualJudgment(ctx,
		time.Now().Add(-30*24*time.Hour), time.Now(), 30, 30)
	if !ok {
		t.Fatal("恰等 floor 应出部署口径值")
	}
	if math.Abs(pct-73) > 0.001 {
		t.Fatalf("恰等 floor pct=%v want 73（30×365/30/500×100）", pct)
	}
	if tag != "（部署$500）" {
		t.Fatalf("恰等 floor 读面签=%q want （部署$500）", tag)
	}
}

// floor 键可配+读败/非正 fail-closed 缺省 500。
func TestAnnualDeployFloorConfig(t *testing.T) {
	if got := annualDeployFloor(nil); got != 500 {
		t.Fatalf("nil cfg floor=%v want 500", got)
	}
	ctx, c, pool := annJudgeSetup(t)
	if got := annualDeployFloor(c.cfg); got != 500 {
		t.Fatalf("种子 floor=%v want 500", got)
	}
	// 可配：改 100→读 100（部署$76≥100→部署口径翻转实证）
	if _, err := pool.Exec(ctx, `UPDATE app_config
		SET value='100' WHERE key='annualized.deploy_floor_usd'`); err != nil {
		t.Fatal("floor 改写:", err)
	}
	if err := c.cfg.Load(ctx); err != nil {
		t.Fatal("cfg reload:", err)
	}
	if got := annualDeployFloor(c.cfg); got != 100 {
		t.Fatalf("可配 floor=%v want 100", got)
	}
	// 非正值→缺省 500（0 floor=部署口径恒生效=病灶复辟口）
	if _, err := pool.Exec(ctx, `UPDATE app_config
		SET value='0' WHERE key='annualized.deploy_floor_usd'`); err != nil {
		t.Fatal("floor 清零:", err)
	}
	if err := c.cfg.Load(ctx); err != nil {
		t.Fatal("cfg reload2:", err)
	}
	if got := annualDeployFloor(c.cfg); got != 500 {
		t.Fatalf("非正 floor=%v want 500 缺省", got)
	}
}
