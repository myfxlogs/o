package report

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0422 统一实现盈亏口径锚：开仓腿不入（置换流量非亏）/平仓腿按 closed_at
// 归属+裸袋挂账冲回/rollback_cost 入现金钉式/移动与校准类全出/窗界生效。
func TestRealizedCaliber(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for _, tbl := range []string{"fin_cashflow", "fin_position", "fin_execution"} {
		if _, err := pool.Exec(ctx, "DELETE FROM "+tbl); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for _, tbl := range []string{"fin_cashflow", "fin_position", "fin_execution"} {
			_, _ = pool.Exec(ctx, "DELETE FROM "+tbl)
		}
	}()
	now := time.Now()
	y, m, d := now.UTC().Date()
	today0 := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	yesterday := today0.Add(-24 * time.Hour)
	// 三仓夹具：A=今日平仓无袋（腿差 -0.1）；B=今日平仓带袋 $20（腿差 -24.7
	// 冲回后实亏 -4.7）；C=持仓中（腿 -50 全出——置换流量非亏）。
	mkpos := func(closed *time.Time, bag float64) int64 {
		var eid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
			VALUES('T-USDT-SWAP',25,'closed') RETURNING id`).Scan(&eid); err != nil {
			t.Fatal(err)
		}
		var pid int64
		err := pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd,closed_at,bare_bag_ccy,bare_bag_qty,bare_bag_usd)
			VALUES($1,'T-USDT-SWAP',25,$2,CASE WHEN $3>0 THEN 'T' END,CASE WHEN $3>0 THEN 1 END,CASE WHEN $3>0 THEN $3 END)
			RETURNING id`, eid, closed, bag).Scan(&pid)
		if err != nil {
			t.Fatal(err)
		}
		return pid
	}
	posA := mkpos(&now, 0)
	posB := mkpos(&now, 20)
	posC := mkpos(nil, 0)
	type cf struct {
		kind string
		amt  float64
		pos  *int64
		at   time.Time
	}
	rows := []cf{
		{"interest", 1.5, nil, now}, {"fee", -0.2, nil, now},
		{"rollback_cost", -0.5, nil, now},
		{"deposit", 1000, nil, now}, {"earn_out", -50, nil, now},
		{"interest", 0.3, nil, yesterday},
		{"spot_buy", -25, &posA, yesterday}, {"spot_sell", 24.9, &posA, now},
		{"spot_buy", -25, &posB, now}, {"spot_sell", 0.3, &posB, now},
		{"spot_buy", -50, &posC, now},
	}
	for i, r := range rows {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
			VALUES($1,'okx','USDT',$2,$3,$4,'test',$5)`, r.kind, r.amt, r.pos, i+1, r.at); err != nil {
			t.Fatal(err)
		}
	}
	// 今日窗：现金 0.8（1.5-0.2-0.5）+今日平仓腿 -24.8（A -0.1+B -24.7，
	// 腿发生日不计——B 昨买腿不入今日窗）+袋冲回 20=-4.0；
	// 开仓 C 腿全出。
	v, err := Realized(ctx, pool, today0, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if want := -4.0; v != want {
		t.Fatalf("今日实现盈亏=%v（期望 %v）", v, want)
	}
	// 累计窗（左无界）：+昨日 interest 0.3=-3.7；deposit/earn_out 全出。
	v, err = Realized(ctx, pool, time.Time{}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if want := -3.7; v != want {
		t.Fatalf("累计实现盈亏=%v（期望 %v）", v, want)
	}
	// 逐日序列：昨桶=0.3（现金昨项——A 昨买腿按平仓日归今桶）、今桶=-4.0。
	ds, err := RealizedDaily(ctx, pool, yesterday)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 || ds[0] != 0.3 || ds[1] != -4.0 {
		t.Fatalf("逐日序列=%v（期望 [0.3 -4.0]）", ds)
	}
}
