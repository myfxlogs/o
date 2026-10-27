package sigdist

// xvenue_test.go —— 跨所信号桩测（T-0141 验收锚）：xvenue_spread/carry_diff
// 数值形+单边陈/缺不产如实+行数有界+T-0164 血统闸（派生 ts=最旧上游观测）。
import (
	"testing"
	"time"
)

// 锚：双鲜边→两族行；okx/mt5 单边陈→该对缺席；mt5_swap val NULL→carry_diff
// 缺席（换算缺件如实）；同拍重扫行数有界。
func TestXvenue(t *testing.T) {
	ctx, r := setup(t)
	if err := r.AttachConfigForTest(ctx, map[string]any{
		"bridge.xvenue.map": map[string]string{"ETH-USDT-SWAP": "ETHUSDm"},
	}); err != nil {
		t.Fatal(err)
	}
	seed := func(t *testing.T) {
		t.Helper()
		for _, q := range []string{
			`INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES(now(),'ETH-USDT-SWAP','mark-price',3000)`,
			`INSERT INTO md_ticks(ts,venue,inst_id,bid,ask) VALUES(now(),'mt5','ETHUSDm',2990,2992)`,
			`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('funding_view','ETH-USDT-SWAP',-0.0001,'{}',now())`,
			`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('mt5_swap','ETHUSDm',-1000,'{"swap_type":5}',now())`,
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
	var spread, diff float64
	if err := r.pool.QueryRow(ctx, `SELECT val FROM md_signal WHERE kind='xvenue_spread' AND key='ETH-USDT-SWAP'`).Scan(&spread); err != nil {
		t.Fatal("xvenue_spread 应落:", err)
	}
	// (3000−2991)/2991×10⁴≈30.09 bps
	if spread < 29 || spread > 31 {
		t.Fatalf("spread_bps=%v", spread)
	}
	if err := r.pool.QueryRow(ctx, `SELECT val FROM md_signal WHERE kind='carry_diff' AND key='ETH-USDT-SWAP'`).Scan(&diff); err != nil {
		t.Fatal("carry_diff 应落:", err)
	}
	// funding −0.0001×3×365×10⁴=−1095 bps 年化；mt5_swap=−1000 → diff=−95
	if diff > -94.1 || diff < -95.9 {
		t.Fatalf("carry_diff=%v（funding_ann−swap_ann）", diff)
	}
	// 单边陈：mt5 tick 翻 5min→两族该对均缺席更新（行 ts 陈化不再刷新）
	if _, err := r.pool.Exec(ctx, `UPDATE md_ticks SET ts=now()-interval '5 minutes' WHERE venue='mt5'`); err != nil {
		t.Fatal(err)
	}
	if err := r.Xvenue(ctx); err != nil {
		t.Fatal(err)
	}
	var ts1, ts2 time.Time
	r.pool.QueryRow(ctx, `SELECT ts FROM md_signal WHERE kind='xvenue_spread' AND key='ETH-USDT-SWAP'`).Scan(&ts1)
	if err := r.Xvenue(ctx); err != nil {
		t.Fatal(err)
	}
	r.pool.QueryRow(ctx, `SELECT ts FROM md_signal WHERE kind='xvenue_spread' AND key='ETH-USDT-SWAP'`).Scan(&ts2)
	if !ts1.Equal(ts2) {
		t.Fatal("mt5 单边陈应不再刷新（ts 不变）")
	}
}

// T-0171 单键域桩锚：'mark-price' WS 行唯一键域（REST 轮询方言已退役）——
// mark 陈行（150s>120s 闸）单边陈→该对缺席如实（mt5 虽鲜不救）。
func TestXvenueMarkStale(t *testing.T) {
	ctx, r := setup(t)
	if err := r.AttachConfigForTest(ctx, map[string]any{
		"bridge.xvenue.map": map[string]string{"SOL-USDT-SWAP": "SOLUSDm"},
	}); err != nil {
		t.Fatal(err)
	}
	seed := []string{
		`INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES(now()-interval '150 seconds','SOL-USDT-SWAP','mark-price',100)`,
		`INSERT INTO md_ticks(ts,venue,inst_id,bid,ask) VALUES(now(),'mt5','SOLUSDm',200,202)`,
	}
	for _, q := range seed {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Xvenue(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_signal
		WHERE kind='xvenue_spread' AND key='SOL-USDT-SWAP'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("mark 单边陈应缺席如实（不产行）")
	}
	// mark 转鲜→行落（闸窗语义复核）
	if _, err := r.pool.Exec(ctx, `UPDATE md_mark_index SET ts=now() WHERE inst_id='SOL-USDT-SWAP'`); err != nil {
		t.Fatal(err)
	}
	if err := r.Xvenue(ctx); err != nil {
		t.Fatal(err)
	}
	var val float64
	if err := r.pool.QueryRow(ctx, `SELECT val FROM md_signal
		WHERE kind='xvenue_spread' AND key='SOL-USDT-SWAP'`).Scan(&val); err != nil {
		t.Fatal("mark 鲜行应撑起 xvenue_spread:", err)
	}
	// 双陈→缺席如实（新拍零刷新）
	if _, err := r.pool.Exec(ctx, `UPDATE md_ticks SET ts=now()-interval '5 minutes' WHERE venue='mt5'`); err != nil {
		t.Fatal(err)
	}
	var ts1, ts2 time.Time
	r.pool.QueryRow(ctx, `SELECT ts FROM md_signal WHERE kind='xvenue_spread' AND key='SOL-USDT-SWAP'`).Scan(&ts1)
	if err := r.Xvenue(ctx); err != nil {
		t.Fatal(err)
	}
	r.pool.QueryRow(ctx, `SELECT ts FROM md_signal WHERE kind='xvenue_spread' AND key='SOL-USDT-SWAP'`).Scan(&ts2)
	if !ts1.Equal(ts2) {
		t.Fatal("双陈应不刷新（缺席如实不变）")
	}
}

// T-0164 血统闸桩锚（派单钦定：上游陈→派生行即陈→评估面视缺席）：
// swap 观测 2h 陈（8h 推导窗内）+funding 鲜→carry_diff 行落但 ts=swap 观测
// ts（2h 前，非写拍 now）→Fresh 120s 消费闸读=缺席——行存在而消费不可见，
// 禁合成面自盖章冒充鲜。spread 同律（ts=最旧边 tick 观测 30s 前）。
func TestXvenueLineage(t *testing.T) {
	ctx, r := setup(t)
	if err := r.AttachConfigForTest(ctx, map[string]any{
		"bridge.xvenue.map": map[string]string{"ETH-USDT-SWAP": "ETHUSDm"},
	}); err != nil {
		t.Fatal(err)
	}
	md(t, ctx, r, `INSERT INTO md_mark_index(ts,inst_id,channel,px)
		VALUES(now()-interval '10 seconds','ETH-USDT-SWAP','mark-price',3000)`)
	md(t, ctx, r, `INSERT INTO md_ticks(ts,venue,inst_id,bid,ask)
		VALUES(now()-interval '30 seconds','mt5','ETHUSDm',2990,2992)`)
	md(t, ctx, r, `INSERT INTO md_signal(kind,key,val,meta,ts) VALUES
		('funding_view','ETH-USDT-SWAP',-0.0001,'{}',now()-interval '5 seconds'),
		('mt5_swap','ETHUSDm',-1000,'{"swap_type":5}',now()-interval '2 hours')`)
	if err := r.Xvenue(ctx); err != nil {
		t.Fatal(err)
	}
	var spreadTs, carryTs time.Time
	if err := r.pool.QueryRow(ctx, `SELECT ts FROM md_signal
		WHERE kind='xvenue_spread' AND key='ETH-USDT-SWAP'`).Scan(&spreadTs); err != nil {
		t.Fatal("xvenue_spread 应落:", err)
	}
	// spread ts=最旧边 mt5 tick 观测（30s 前——非写拍 now()）。
	if age := time.Since(spreadTs); age < 25*time.Second || age > 60*time.Second {
		t.Fatalf("spread ts 应=最旧上游观测（tick 30s 前）got age=%v", age)
	}
	if err := r.pool.QueryRow(ctx, `SELECT ts FROM md_signal
		WHERE kind='carry_diff' AND key='ETH-USDT-SWAP'`).Scan(&carryTs); err != nil {
		t.Fatal("carry_diff 应落:", err)
	}
	// carry ts=最旧上游 mt5_swap 观测（2h 前——funding 虽鲜不救）。
	if age := time.Since(carryTs); age < 110*time.Minute {
		t.Fatalf("carry ts 应=swap 观测（2h 前）got age=%v", age)
	}
	// 桩锚：行存在而评估面（Fresh 120s 闸）视缺席。
	if _, ok, _ := Fresh(ctx, r.pool, "carry_diff", "ETH-USDT-SWAP"); ok {
		t.Fatal("swap 上游陈（2h）→carry_diff 消费面应视缺席（血统层禁自盖章）")
	}
}
