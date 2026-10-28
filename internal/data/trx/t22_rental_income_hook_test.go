package trx

// T-0220 验收锚：新租金入账事件——仅增量段 rental_income 触发回调；
// 全史回填/首拉（lastTs=0）/非租金 kind/未接线皆静默（防历史段轰炸）。

import (
	"context"
	"testing"

	"o/internal/store/pgstore"
)

func TestEmitRentalIncome_IncrementalGuard(t *testing.T) {
	rental := pgstore.TRXTx{TxID: "tx-rent-1", Kind: "rental_income", AmountSun: 28_340_000}
	deposit := pgstore.TRXTx{TxID: "tx-dep-1", Kind: "deposit", AmountSun: 28_340_000}

	cases := []struct {
		name        string
		rec         pgstore.TRXTx
		incremental bool
		wantFired   bool
	}{
		{"增量段租金→发", rental, true, true},
		{"全史回填租金→静默", rental, false, false},
		{"增量段非租金→静默", deposit, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewTxsCollector("addr", nil, "http://unused", func() string { return "" }, "", nil, 0)
			var fired int
			var gotTxID string
			var gotSun int64
			c.SetOnRentalIncome(func(_ context.Context, txID string, amountSun int64) {
				fired++
				gotTxID, gotSun = txID, amountSun
			})
			c.emitRentalIncome(context.Background(), tc.rec, tc.incremental)
			if tc.wantFired {
				if fired != 1 {
					t.Fatalf("回调应恰发一次, got %d", fired)
				}
				if gotTxID != rental.TxID || gotSun != rental.AmountSun {
					t.Fatalf("载荷不符: txID=%s sun=%d", gotTxID, gotSun)
				}
			} else if fired != 0 {
				t.Fatalf("应静默, 却发了 %d 次", fired)
			}
		})
	}
}

// 未接线（nil 回调）零 panic——缺件降级径。
func TestEmitRentalIncome_NilHook(t *testing.T) {
	c := NewTxsCollector("addr", nil, "http://unused", func() string { return "" }, "", nil, 0)
	c.emitRentalIncome(context.Background(),
		pgstore.TRXTx{TxID: "tx", Kind: "rental_income", AmountSun: 1}, true) // 不应 panic
}
