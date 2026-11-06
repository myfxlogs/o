package hypothesis

// xshadow_test.go —— 影子评估消费臂桩测（T-0408 验收锚②③④）：xvenue_shadow_
// carry 候选对→fin_opp_eval 三态判词（template='xvenue-carry-shadow' 可辨）；
// proposed→condreq「map 闸建议」携证据呈业主不落闸（fin_proposal 零行/map
// 键零写）；rejected/unevaluable→零闸动留账。金案形（swap_long=-445%）
// 影子域判词拒对拍。
import (
	"context"
	"encoding/json"
	"testing"
)

func shadowSeed(t *testing.T, ctx context.Context, r *Runner, inst string, diff float64, meta string) {
	t.Helper()
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
		VALUES('xvenue_shadow_carry',$1,$2,$3::jsonb,now())
		ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val, meta=EXCLUDED.meta, ts=EXCLUDED.ts`,
		inst, diff, meta); err != nil {
		t.Fatal(err)
	}
}

// 锚：金案形判词拒——XAU 候选（funding=0、swap_long=-445%/yr 成本实锤、
// swap_short 同为负）方向净 carry=−2000<0→feasible=false→rejected/
// net_ann_economics 如实（-445% 对拍）+零 condreq 零提案零闸动。
func TestXvenueShadowRejectGoldLike(t *testing.T) {
	ctx, r := xsetup(t)
	r.AttachFin(r.pool) // 桩测同池（fin 链已迁）
	shadowSeed(t, ctx, r, "XAU-USDT-SWAP", -2000,
		`{"funding_ann_bps":0,"mt5_swap_ann_bps":-2000,"mt5_symbol":"XAUUSDm",
		"direction":"long_perp_short_cfd","swap_long_ann_bps":-44500,"swap_short_ann_bps":-2000,
		"contract_size":100,"max_volume":20,"mid":2400,"shadow":true}`)
	r.xvenueShadowScan(ctx)
	var verdict, reason, tmpl string
	if err := r.pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason', template FROM fin_opp_eval
		WHERE inst_id='XAU-USDT-SWAP'`).Scan(&verdict, &reason, &tmpl); err != nil {
		t.Fatal("影域评估行应落:", err)
	}
	if tmpl != shadowTemplate {
		t.Fatalf("影域 template 可辨应=%s got %s", shadowTemplate, tmpl)
	}
	if verdict != "rejected" || reason != "net_ann_economics" {
		t.Fatalf("金案形应判词拒 got %s/%s（-445%% 成本对拍）", verdict, reason)
	}
	// 零打扰：condreq/提案/map 键全零动。
	var n int
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_condition_gap`).Scan(&n)
	if n != 0 {
		t.Fatalf("判词假应零 condreq got %d", n)
	}
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal`).Scan(&n)
	if n != 0 {
		t.Fatalf("影域禁入提案径 got %d", n)
	}
	r.pool.QueryRow(ctx, `SELECT count(*) FROM app_config WHERE key LIKE 'bridge.xvenue.map%'`).Scan(&n)
	if n != 0 {
		t.Fatalf("map owner 域应零写 got %d", n)
	}
}

// 锚：判词真全链——净正候选（funding 正+swap_long 成本可控+depth5 容量在
// +mt5 能力面在）→proposed→condreq pending「map 闸建议」携证据（pairs/
// direction/net/eval_id）呈业主；map/提案仍零写（只呈不落闸）。
func TestXvenueShadowProposedToCondreq(t *testing.T) {
	ctx, r := xsetup(t)
	r.AttachFin(r.pool)
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_capability_surface
		(venue,surface,endpoint_family,product_family,rw_class,enabled)
		VALUES('mt5','trade-order','trade-order','cfd','write',true)`); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('xvenue_shadow_carry','SOL-USDT-SWAP',1200,
			'{"funding_ann_bps":2000,"mt5_swap_ann_bps":-800,"mt5_symbol":"SOLUSDm",
			"direction":"short_perp_long_cfd","swap_long_ann_bps":-800,"contract_size":1,"max_volume":500,"mid":200,"shadow":true}',now())`,
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('xvenue_shadow_spread','SOL-USDT-SWAP',15,'{"mt5_symbol":"SOLUSDm","mid_src":"catalog_quote"}',now())`,
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('depth5','SOL-USDT-SWAP',5,'{"bid_usd":[90000,80000]}',now())`,
	} {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	r.xvenueShadowScan(ctx)
	var verdict, reason string
	var evalID int64
	if err := r.pool.QueryRow(ctx, `SELECT id, verdict, evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id='SOL-USDT-SWAP' AND template='xvenue-carry-shadow'`).
		Scan(&evalID, &verdict, &reason); err != nil {
		t.Fatal("影域评估行应落:", err)
	}
	if verdict != "proposed" {
		t.Fatalf("净正+数据齐应判词真 got %s/%s", verdict, reason)
	}
	// condreq pending 呈——map 闸建议携证据。
	var surface, state, payload string
	if err := r.pool.QueryRow(ctx, `SELECT surface, state, payload::text FROM fin_condition_gap
		WHERE kind='permission' AND venue='mt5'`).Scan(&surface, &state, &payload); err != nil {
		t.Fatal("condreq 呈批行应落:", err)
	}
	if surface != "bridge.xvenue.map.shadow" || state != "pending" {
		t.Fatalf("surface/state=%s/%s", surface, state)
	}
	var pl map[string]any
	if err := json.Unmarshal([]byte(payload), &pl); err != nil {
		t.Fatal(err)
	}
	pairs, _ := pl["suggested_pairs"].(map[string]any)
	if pairs["SOL-USDT-SWAP"] != "SOLUSDm" || pl["eval_id"] != float64(evalID) {
		t.Fatalf("condreq 载荷应携候选对+评估证据: %s", payload)
	}
	// 不落闸实证：map 键零写+fin_proposal 零行（影子层只呈）。
	var n int
	r.pool.QueryRow(ctx, `SELECT count(*) FROM app_config WHERE key LIKE 'bridge.xvenue.map%'`).Scan(&n)
	if n != 0 {
		t.Fatalf("map owner 域应零写 got %d", n)
	}
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal`).Scan(&n)
	if n != 0 {
		t.Fatalf("影域禁入提案径 got %d", n)
	}
}

// 锚：数据缺=unevaluable（depth5 未采≠簿薄——capacity_missing 不可判如实
// 非误拒）+fin 池未挂=呈批让位评估行仍落（WARN 径非静默）。
func TestXvenueShadowUnevaluableAndNoFin(t *testing.T) {
	ctx, r := xsetup(t)
	shadowSeed(t, ctx, r, "XAU-USDT-SWAP", 500,
		`{"funding_ann_bps":500,"mt5_swap_ann_bps":0,"mt5_symbol":"XAUUSDm",
		"direction":"short_perp_long_cfd","swap_long_ann_bps":0,"contract_size":100,"max_volume":20,"shadow":true}`)
	r.xvenueShadowScan(ctx)
	var verdict, reason string
	if err := r.pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id='XAU-USDT-SWAP' AND template='xvenue-carry-shadow'`).Scan(&verdict, &reason); err != nil {
		t.Fatal("评估行应落（fin 未挂亦落账）:", err)
	}
	if verdict != "unevaluable" || reason != "capacity_missing" {
		t.Fatalf("depth5 缺应 unevaluable/capacity_missing got %s/%s", verdict, reason)
	}
}
