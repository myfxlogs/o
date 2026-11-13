// audit_1113_guard2_test.go —— 审计/风控哨戒(二巡) RED 锚：driftTriggered
// 绝对兜底闸对裸 imbNow 先判于基线——粒度底>500bps 的仓基线机制死码化
// （相对闸永不可达=恒假阳）。默认 skip；显跑：
// AUDIT_RED=1 go test ./internal/guard -run TestAuditDriftAbsCapGranularityFloor
package guard

import (
	"os"
	"testing"
)

func auditRedGateGuard2(t *testing.T) {
	t.Helper()
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit RED 锚——修复期显跑: AUDIT_RED=1")
	}
}

// TestAuditDriftAbsCapGranularityFloor 不变量（T-0064+canon
// lotsz-granularity-vs-hedge-band）：漂移判词=对开时基线偏移，
// 「出生粒度残差不杀」。现实现 facts.go:155 对裸 |imbNow|>500bps
// 无条件 return true，先于基线读/种——两破洞：
// ①基线已播种 9800bps 的仓（KB 实证粒度底：$24 腿×0.01 张网格≈9800bps），
//
//	稳定 9810（偏移 10bps<band 20）仍被兜底杀——相对闸成永不可达死支；
//
// ②基线 NULL 首拍残差 9800 不播种直接杀——「出生残差入基线」钦定被架空。
// 正确语义=残差于该仓自身粒度底内不算「裸腿级失守」（兜底阈≥该仓
// 粒度底，或播种态残差豁免裸值兜底）——修好本测试自然转绿。
func TestAuditDriftAbsCapGranularityFloor(t *testing.T) {
	auditRedGateGuard2(t)
	ctx, w, _ := setup(t)
	var stratID, execID, posID int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		 VALUES('BTC-USDT-SWAP','guard-test','live',24,'okx') RETURNING id`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		 VALUES($1,'BTC-USDT-SWAP',24,'open') RETURNING id`, stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	// 面①：基线=9800bps 已播种（该仓粒度底——KB 实证域），当前 9810 稳定。
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,open_imbalance_bps)
		 VALUES($1,$2,'BTC-USDT-SWAP',24,9800) RETURNING id`, execID, stratID).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	if w.driftTriggered(ctx, posID, 9810, 20) {
		t.Fatal("面①：基线 9800/现 9810（偏移 10bps<band 20）稳定残差被裸值 abs-cap>500 杀——" +
			"粒度底>500bps 仓的相对漂移闸死码化（恒假阳，T-0064 基线语义架空）")
	}
	// 面②：基线 NULL 首拍——残差应播种入列而非即杀。
	var pos2 int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
		 VALUES($1,$2,'BTC-USDT-SWAP',24) RETURNING id`, execID, stratID).Scan(&pos2); err != nil {
		t.Fatal(err)
	}
	if w.driftTriggered(ctx, pos2, 9800, 20) {
		t.Fatal("面②：基线 NULL 首拍残差 9800bps 未播种即杀——出生残差入基线判词被 abs-cap 前置短路")
	}
	var seeded *float64
	if err := w.pool.QueryRow(ctx,
		"SELECT open_imbalance_bps FROM fin_position WHERE id=$1", pos2).Scan(&seeded); err != nil || seeded == nil {
		t.Fatal("面②：首拍后基线仍 NULL——播种径未走")
	}
}
