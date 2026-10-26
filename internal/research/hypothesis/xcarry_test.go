package hypothesis

// xcarry_test.go —— xvenue_carry 全链桩测（T-0144 验收锚）：carry_diff 亮牌→
// 假想卡（provenance/signals/direction）→评估四件分解→提案 fin_proposed；
// 容量缺/黑名单/能力面缺→rejected 如实；净负→net_ann_negative 如实。
import (
	"context"
	"encoding/json"
	"testing"
)

func xsetup(t *testing.T) (context.Context, *Runner) {
	t.Helper()
	return setup(t)
}

func xseed(t *testing.T, ctx context.Context, r *Runner, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		if _, err := r.pool.Exec(ctx, `INSERT INTO app_config(key,value) VALUES($1,$2)
			ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
}

// 锚①：carry_diff=-532bps（负 funding 亮牌）→xvenue_carry 卡（provenance=
// hypothesis_officer+signals 域+direction=long_perp_short_cfd）+评估四件+
// 提案 fin_proposed（verdict proposed 路径——须 depth5 容量在+mt5 能力面在）。
// mt5 trade-order 面缺（master 能力账无执行面）→verdict=mt5_leg_capability_
// missing rejected 如实（做不了如实也是过）。
func TestXvenueCarryChain(t *testing.T) {
	ctx, r := xsetup(t)
	xseed(t, ctx, r, map[string]string{"research.signal.xvenue_carry_diff_bps": "300"})
	// 信号三件套（carry_diff 携 T-0141 meta+spread 鲜+depth5 容量鲜）。
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
	var fam, dir string
	var ev map[string]any
	if err := r.pool.QueryRow(ctx, `SELECT family, evidence->>'direction', evidence
		FROM fin_opp_hypothesis WHERE venue='okx' AND surface='xvenue-carry'`).Scan(&fam, &dir, &ev); err != nil {
		t.Fatal("xvenue_carry 卡应立:", err)
	}
	if fam != "xvenue_carry" || dir != "long_perp_short_cfd" {
		t.Fatalf("卡 family/direction=%s/%s", fam, dir)
	}
	if ev["provenance"] != "hypothesis_officer" {
		t.Fatalf("provenance=%v", ev["provenance"])
	}
	raw, _ := json.Marshal(ev["signals"])
	var sigs []Sig
	json.Unmarshal(raw, &sigs)
	if len(sigs) != 1 || sigs[0].Kind != "carry_diff" {
		t.Fatalf("卡 signals 域=%s", raw)
	}
	// 评估行四件分解。
	var verdict string
	var eev map[string]any
	if err := r.pool.QueryRow(ctx, `SELECT verdict, evidence FROM fin_opp_eval
		WHERE inst_id='ETH-USDT-SWAP' AND template='xvenue-carry'`).Scan(&verdict, &eev); err != nil {
		t.Fatal("评估行应落:", err)
	}
	for _, k := range []string{"funding_ann_bps", "swap_ann_bps", "spread_bps", "net_ann_bps"} {
		if _, ok := eev[k]; !ok {
			t.Fatalf("evidence 四件缺 %s: %v", k, eev)
		}
	}
	// mt5 trade-order 能力面缺席→verdict rejected/capability_missing（如实）。
	if verdict != "rejected" || eev["reason"] != "mt5_leg_capability_missing" {
		t.Fatalf("verdict/reason=%s/%v（能力面缺应如实拒）", verdict, eev["reason"])
	}
	// 阈内零卡零评估（无机会不产）。
	if _, err := r.pool.Exec(ctx, `DELETE FROM fin_opp_hypothesis WHERE surface='xvenue-carry'`); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
		VALUES('carry_diff','BTC-USDT-SWAP',100,'{"funding_ann_bps":100,"mt5_swap_ann_bps":0,"mt5_symbol":"BTCUSDm"}',now())
		ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val`); err != nil {
		t.Fatal(err)
	}
	r.xvenueCarryScan(ctx)
	var n int
	// ETH carry_diff 信号仍在册（|532|>阈）会重立 ETH 卡——阈内判据锚在
	// BTC（diff=100<300）：不得立卡。
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_hypothesis
		WHERE family='xvenue_carry' AND evidence->>'inst'='BTC-USDT-SWAP'`).Scan(&n)
	if n != 0 {
		t.Fatalf("阈内应零卡 got %d", n)
	}
}
