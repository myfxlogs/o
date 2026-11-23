// annualized_test.go —— 工单#34703 桩测：deployedBasis 时加权分母
// 三案（单仓全窗/半窗开合/零部署→"—"）+ DCD 快照腿 snap 标。
package main

import (
	"context"
	"math"
	"testing"
	"time"

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
