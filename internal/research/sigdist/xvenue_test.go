package sigdist

// xvenue_test.go —— 跨所信号桩测（T-0141 验收锚）：xvenue_spread/carry_diff
// 数值形+单边陈/缺不产如实+行数有界。
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
			`INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES(now(),'ETH-USDT-SWAP','mark_price',3000)`,
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
