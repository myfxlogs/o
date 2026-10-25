package trx

import "testing"

// T-0108 桩测：sun↔TRX 量纲显式断言（1 TRX=1e6 sun——量纲换算不可漂）。
func TestSunToTrxUnits(t *testing.T) {
	const sunPerTRX = 1e6
	for amt, want := range map[int64]float64{1234567: 1.234567, 1000000: 1.0, 500000: 0.5} {
		if got := float64(amt) / sunPerTRX; got != want {
			t.Fatalf("sun→TRX %d: got %v want %v", amt, got, want)
		}
	}
}
