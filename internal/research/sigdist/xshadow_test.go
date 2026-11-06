package sigdist

// xshadow_test.go —— 影子评估层蒸馏桩测（T-0408 验收锚②③）：map 外候选对
// （OKX 宇宙×broker 自报目录按 base 匹配−已映射）产 xvenue_shadow_* 独立
// kind 行（真 kind 零染）；map 内对不双产；目录陈（>8h）=候选缩零不产。
import (
	"testing"
	"time"
)

// 锚：ETH 已映射（真行照旧）+XAU 未映射候选（XAUUSDm 目录符+meta.mid 快照）
// →xvenue_shadow_spread/xvenue_shadow_carry 影行落，真 kind 无 XAU 键。
func TestXvenueShadowCandidates(t *testing.T) {
	ctx, r := setup(t)
	if err := r.AttachConfigForTest(ctx, map[string]any{
		"bridge.xvenue.map": map[string]string{"ETH-USDT-SWAP": "ETHUSDm"},
		"okx.symbols":       []string{"ETH-USDT-SWAP", "XAU-USDT-SWAP"},
	}); err != nil {
		t.Fatal(err)
	}
	seed := func(t *testing.T) {
		t.Helper()
		for _, q := range []string{
			// 已映射 ETH 三件套（真径照旧）。
			`INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES(now(),'ETH-USDT-SWAP','mark-price',3000)`,
			`INSERT INTO md_ticks(ts,venue,inst_id,bid,ask) VALUES(now(),'mt5','ETHUSDm',2990,2992)`,
			`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('funding_view','ETH-USDT-SWAP',-0.0001,'{"next_funding_ts":"2099-01-01T00:00:00Z"}',now())`,
			`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('mt5_swap','ETHUSDm',-1000,'{"swap_type":5,"swap_long_ann_bps":-816,"swap_short_ann_bps":-50}',now())`,
			// 未映射 XAU 候选（目录粗扫行——meta.mid 快照价无 tick 流）。
			`INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES(now(),'XAU-USDT-SWAP','mark-price',2405)`,
			`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('funding_view','XAU-USDT-SWAP',0,'{"next_funding_ts":"2099-01-01T00:00:00Z"}',now())`,
			`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('mt5_swap','XAUUSDm',-44500,'{"swap_type":5,"swap_long_ann_bps":-44500,"swap_short_ann_bps":-2000,"contract_size":100,"max_volume":20,"mid":2400}',now())`,
		} {
			if _, err := r.pool.Exec(ctx, q); err != nil {
				t.Fatal(err)
			}
		}
	}
	seed(t)
	if err := r.Xvenue(ctx); err != nil {
		t.Fatal(err)
	}
	// 真域：ETH 行照旧。
	var spread float64
	if err := r.pool.QueryRow(ctx, `SELECT val FROM md_signal
		WHERE kind='xvenue_spread' AND key='ETH-USDT-SWAP'`).Scan(&spread); err != nil {
		t.Fatal("真 xvenue_spread 应落:", err)
	}
	// 影域：XAU 候选双 kind 影行（快照 mid=2400 供 spread+carry）。
	var shSpread, shCarry float64
	var meta map[string]any
	if err := r.pool.QueryRow(ctx, `SELECT val, meta FROM md_signal
		WHERE kind='xvenue_shadow_spread' AND key='XAU-USDT-SWAP'`).Scan(&shSpread, &meta); err != nil {
		t.Fatal("xvenue_shadow_spread 应落:", err)
	}
	// (2405−2400)/2400×10⁴≈20.83bps。
	if shSpread < 20 || shSpread > 22 {
		t.Fatalf("shadow spread=%v（快照 mid 径）", shSpread)
	}
	if meta["shadow"] != true || meta["mid_src"] != "catalog_quote" {
		t.Fatalf("影行 meta 可辨钉: %v", meta)
	}
	if err := r.pool.QueryRow(ctx, `SELECT val, meta FROM md_signal
		WHERE kind='xvenue_shadow_carry' AND key='XAU-USDT-SWAP'`).Scan(&shCarry, &meta); err != nil {
		t.Fatal("xvenue_shadow_carry 应落:", err)
	}
	// funding=0→long_perp_short_cfd：diff=−0+ss=−2000（方向净负如实）。
	if shCarry != -2000 {
		t.Fatalf("shadow carry=%v want=-2000", shCarry)
	}
	if meta["direction"] != "long_perp_short_cfd" || meta["shadow"] != true {
		t.Fatalf("影 carry meta direction/shadow 钉: %v", meta)
	}
	// 真 kind 零染（XAU 禁混真径）。
	var n int
	for _, k := range []string{"xvenue_spread", "carry_diff"} {
		r.pool.QueryRow(ctx, `SELECT count(*) FROM md_signal WHERE kind=$1 AND key='XAU-USDT-SWAP'`, k).Scan(&n)
		if n != 0 {
			t.Fatalf("影行不得入真 kind=%s", k)
		}
	}
	// 已映射符不入影域（ETH 零影行——不双产）。
	r.pool.QueryRow(ctx, `SELECT count(*) FROM md_signal
		WHERE kind='xvenue_shadow_carry' AND key='ETH-USDT-SWAP'`).Scan(&n)
	if n != 0 {
		t.Fatal("map 内对不得产影行")
	}
}

// 锚：目录陈（mt5_swap ts>8h 界外）→候选缩零→影行不产（扫死 fail-closed）；
// 宇宙缺 okx.symbols→影子域让位（真域不受影响）。
func TestXvenueShadowStaleCatalog(t *testing.T) {
	ctx, r := setup(t)
	if err := r.AttachConfigForTest(ctx, map[string]any{
		"bridge.xvenue.map": map[string]string{},
		"okx.symbols":       []string{"XAU-USDT-SWAP"},
	}); err != nil {
		t.Fatal(err)
	}
	md(t, ctx, r, `INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES(now(),'XAU-USDT-SWAP','mark-price',2405)`)
	md(t, ctx, r, `INSERT INTO md_signal(kind,key,val,meta,ts) VALUES
		('funding_view','XAU-USDT-SWAP',0,'{"next_funding_ts":"2099-01-01T00:00:00Z"}',now())`)
	md(t, ctx, r, `INSERT INTO md_signal(kind,key,val,meta,ts) VALUES
		('mt5_swap','XAUUSDm',-44500,'{"swap_type":5,"mid":2400}',now()-interval '9 hours')`)
	if err := r.Xvenue(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	r.pool.QueryRow(ctx, `SELECT count(*) FROM md_signal
		WHERE kind LIKE 'xvenue_shadow%'`).Scan(&n)
	if n != 0 {
		t.Fatalf("目录陈应零影行 got %d", n)
	}
	// 宇宙缺（okx.symbols 撤）→影子域让位真域无扰。
	if err := r.AttachConfigForTest(ctx, map[string]any{
		"bridge.xvenue.map": map[string]string{"ETH-USDT-SWAP": "ETHUSDm"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.Xvenue(ctx); err != nil {
		t.Fatal(err)
	}
	r.pool.QueryRow(ctx, `SELECT count(*) FROM md_signal
		WHERE kind LIKE 'xvenue_shadow%'`).Scan(&n)
	if n != 0 {
		t.Fatalf("宇宙缺应零影行 got %d", n)
	}
}

// 锚：tick 鲜行优先于快照（瞬订阅符 mid_src=tick）；快照血统 ts=swap 行 ts。
func TestXvenueShadowMidPriority(t *testing.T) {
	ctx, r := setup(t)
	if err := r.AttachConfigForTest(ctx, map[string]any{
		"bridge.xvenue.map": map[string]string{},
		"okx.symbols":       []string{"XAU-USDT-SWAP"},
	}); err != nil {
		t.Fatal(err)
	}
	swapTs := time.Now().Add(-2 * time.Hour)
	md(t, ctx, r, `INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES(now(),'XAU-USDT-SWAP','mark-price',2500)`)
	md(t, ctx, r, `INSERT INTO md_ticks(ts,venue,inst_id,bid,ask) VALUES(now(),'mt5','XAUUSDm',2400,2402)`)
	md(t, ctx, r, `INSERT INTO md_signal(kind,key,val,meta,ts) VALUES
		('mt5_swap','XAUUSDm',-44500,'{"swap_type":5,"mid":2300}',$1)`, swapTs)
	md(t, ctx, r, `INSERT INTO md_signal(kind,key,val,meta,ts) VALUES
		('funding_view','XAU-USDT-SWAP',0,'{"next_funding_ts":"2099-01-01T00:00:00Z"}',now())`)
	if err := r.Xvenue(ctx); err != nil {
		t.Fatal(err)
	}
	var meta map[string]any
	var spread float64
	if err := r.pool.QueryRow(ctx, `SELECT val, meta FROM md_signal
		WHERE kind='xvenue_shadow_spread' AND key='XAU-USDT-SWAP'`).Scan(&spread, &meta); err != nil {
		t.Fatal(err)
	}
	if meta["mid_src"] != "tick" {
		t.Fatalf("tick 鲜行应优先 mid_src: %v", meta)
	}
	// spread=(2500−2401)/2401×10⁴≈412bps（tick mid=2401 非快照 2300）。
	if spread < 400 || spread > 425 {
		t.Fatalf("tick mid 优先 spread=%v", spread)
	}
}
