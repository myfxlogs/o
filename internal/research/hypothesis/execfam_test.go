package hypothesis

// execfam_test.go —— T-0330 提案闸前置拒桩测（验收锚①②）：执行径未建族
// 全谱过阈也 direct eval rejected/exec_family_missing——提案零行不入队列
// 淤积（族码归因=template 列+reason 机读）。
import (
	"testing"
)

func TestExecFamilyMissingFrontReject(t *testing.T) {
	ctx, r := xsetup(t)
	xseed(t, ctx, r, map[string]string{"research.signal.xvenue_carry_diff_bps": "300"})
	// mt5 执行面在册 enabled——全谱闸开（能力面在≠前置拒豁免；锚定「拒因=
	// 执行径缺」而非能力面缺）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_capability_surface
		(venue,surface,endpoint_family,product_family,rw_class,enabled)
		VALUES('mt5','trade-order','trade-order','cfd','write',true)`); err != nil {
		t.Fatal(err)
	}
	// 信号三件套同 TestXvenueCarryChain（diff=-532 负 funding 亮牌→净年化过阈）。
	for _, q := range []string{
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('carry_diff','ETH-USDT-SWAP',-532,
			'{"funding_ann_bps":-1095,"mt5_swap_ann_bps":0,"mt5_symbol":"ETHUSDm","contract_size":1,"max_volume":100}',now())`,
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('xvenue_spread','ETH-USDT-SWAP',20,'{"mt5_symbol":"ETHUSDm"}',now())`,
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('depth5','ETH-USDT-SWAP',5,'{"bid_usd":[50000,40000,30000,20000,10000]}',now())`,
		`INSERT INTO md_ticks(ts,venue,inst_id,bid,ask) VALUES(now(),'mt5','ETHUSDm',2990,2992)`,
	} {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	r.xvenueCarryScan(ctx)
	// 锚①：eval 前置拒——rejected/exec_family_missing（族码=template 列归因）。
	var verdict, reason string
	if err := r.pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id='ETH-USDT-SWAP' AND template='xvenue-carry'`).Scan(&verdict, &reason); err != nil {
		t.Fatal("前置拒评估行应落:", err)
	}
	if verdict != "rejected" || reason != "exec_family_missing" {
		t.Fatalf("verdict/reason=%s/%s（执行径未建族应前置拒）", verdict, reason)
	}
	// 锚②：提案零行——不进 fin_proposed/fin_armed 队列（淤积断源）。
	var n int
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal`).Scan(&n)
	if n != 0 {
		t.Fatalf("未建族提案应零行 got %d", n)
	}
}
