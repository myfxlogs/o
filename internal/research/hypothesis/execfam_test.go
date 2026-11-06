package hypothesis

// execfam_test.go —— 提案闸族册桩测：T-0330 前置拒语义（未建族=eval direct
// rejected/exec_family_missing，提案零行不入队列）+T-0377 登记翻转（xvenue-carry
// 建族=paper 径贯通——全闸开时链进至提案 fin_proposed，不再淤积拒）。
import (
	"testing"
)

// T-0377 翻转锚：xvenue-carry 入 execfam 册（T-0144 双腿开户链可达）——
// 能力面/容量/黑名单/净年化全闸开 → verdict=proposed+提案立（不再
// exec_family_missing 前置拒）；真单径仍 opener 永拒兜底（lifecycle 域测试钉）。
func TestExecFamilyRegisteredChainProceeds(t *testing.T) {
	ctx, r := xsetup(t)
	xseed(t, ctx, r, map[string]string{"research.signal.xvenue_carry_diff_bps": "300"})
	// mt5 执行面在册 enabled——全谱闸开。
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_capability_surface
		(venue,surface,endpoint_family,product_family,rw_class,enabled)
		VALUES('mt5','trade-order','trade-order','cfd','write',true)`); err != nil {
		t.Fatal(err)
	}
	// 信号三件套（diff=+532 方向净 carry 亮牌→净年化过阈——T-0408 新约定
	// val=当值方向净 carry（正=好）+meta.direction 钉，非旧约 funding−swap 差）。
	for _, q := range []string{
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('carry_diff','ETH-USDT-SWAP',532,
			'{"funding_ann_bps":-1095,"mt5_swap_ann_bps":0,"mt5_symbol":"ETHUSDm","direction":"long_perp_short_cfd","contract_size":1,"max_volume":100}',now())`,
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('xvenue_spread','ETH-USDT-SWAP',20,'{"mt5_symbol":"ETHUSDm"}',now())`,
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('depth5','ETH-USDT-SWAP',5,'{"bid_usd":[50000,40000,30000,20000,10000]}',now())`,
		`INSERT INTO md_ticks(ts,venue,inst_id,bid,ask) VALUES(now(),'mt5','ETHUSDm',2990,2992)`,
	} {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	r.xvenueCarryScan(ctx)
	// 锚①：族册在——评估不前置拒（proposed）。
	var verdict string
	if err := r.pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval
		WHERE inst_id='ETH-USDT-SWAP' AND template='xvenue-carry'`).Scan(&verdict); err != nil {
		t.Fatal("评估行应落:", err)
	}
	if verdict != "proposed" {
		t.Fatalf("族册在应放行 got %s（exec_family_missing=登记回退信号）", verdict)
	}
	// 锚②：提案立（fin_proposed 入队列——lifecycle 双腿开户消费）。
	var n int
	var dir *string
	if err := r.pool.QueryRow(ctx, `SELECT count(*), max(direction) FROM fin_proposal`).Scan(&n, &dir); err != nil {
		t.Fatal(err)
	}
	if n != 1 || dir == nil || *dir != "long_perp_short_cfd" {
		t.Fatalf("建族后提案应立 1 行 long_perp_short_cfd got %d/%v rej=%v", n, dir, r.rej.Snapshot())
	}
}
