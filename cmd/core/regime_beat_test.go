// regimeBeat 世模环境帧：确认棒宇宙横切→regime 标签量表行；
// 同纹抑/类变重报/thin/stale 如实报。
package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestRegimeBeat(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM md_klines`, `DELETE FROM ops_event WHERE ticket='regime'`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	svc := coreService{pool: pool}
	last := func() string {
		var s string
		if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
			WHERE ticket='regime' ORDER BY id DESC LIMIT 1`).Scan(&s); err != nil {
			return ""
		}
		return s
	}
	rowN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE ticket='regime'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	seed := func(drift float64, ageDays, n int) {
		if _, err := pool.Exec(ctx, `DELETE FROM md_klines`); err != nil {
			t.Fatal(err)
		}
		for s := 0; s < n; s++ {
			inst := fmt.Sprintf("RG%d-USDT-SWAP", s)
			for i := 20; i >= 0; i-- {
				c := 100 * math.Exp(drift*float64(20-i)) * (1 + 0.02*math.Sin(float64(i*3+s)))
				ts := time.Now().Add(-time.Duration(i+ageDays) * 24 * time.Hour)
				if _, err := pool.Exec(ctx, `INSERT INTO md_klines
					(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm)
					VALUES('okx',$1,'1D',$2,$3,$3,$3,$3,1,1,true)`,
					inst, ts, c); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	seed(0.01, 0, 3) // thin:足窗品种<5 如实报
	svc.regimeBeat(ctx)
	if s := last(); !strings.Contains(s, "regime=thin") {
		t.Fatalf("thin 未报: %q", s)
	}

	seed(0.01, 0, 6) // up:宇宙上行→up_*vol
	svc.regimeBeat(ctx)
	if s := last(); !strings.Contains(s, "regime=up_") {
		t.Fatalf("up 未报: %q", s)
	}

	svc.regimeBeat(ctx) // 同纹抑:同数据再拍=零新行
	if n := rowN(); n != 2 {
		t.Fatalf("同纹未抑: n=%d", n)
	}

	seed(-0.02, 0, 6) // 类变重报:宇宙转跌→down_*vol 新行
	svc.regimeBeat(ctx)
	if s := last(); !strings.Contains(s, "regime=down_") {
		t.Fatalf("down 未报: %q", s)
	}

	seed(0.01, 5, 6) // stale:末确认棒>96h=数据面死如实报
	svc.regimeBeat(ctx)
	if s := last(); !strings.Contains(s, "regime=stale") {
		t.Fatalf("stale 未报: %q", s)
	}
}
