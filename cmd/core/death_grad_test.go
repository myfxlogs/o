package main

import (
	"math"
	"testing"
)

// deathFactor 三窗年化→能力因子四档：≥8% 满力/0~8% 半力/30d 负短窗
// 回正留级/三窗皆负=0.10 临床底（非零——复愈通道不冻死）。
func TestDeathFactor(t *testing.T) {
	for _, c := range []struct {
		ann7, ann14, ann30, want float64
	}{
		{12, 10, 9, 1.0},      // 饱——超成年线
		{8, 8, 8, 1.0},        // 恰线=满力
		{5, 6, 4, 0.5},        // 饿——正但未达线
		{2, -3, 1, 0.5},       // 30d 微正仍饿档
		{1, 2, -0.5, 0.25},    // 30d 负但短窗回正=病中回暖
		{0, 0, -1, 0.25},      // 短窗零界（非负）=留级非临床
		{-1, -0.5, -0.4, 0.1}, // 三窗皆负=临床底
		{-5, -4, -3, 0.1},     // 深亏同底
	} {
		if got := deathFactor(c.ann7, c.ann14, c.ann30); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("deathFactor(%v,%v,%v)=%v want %v", c.ann7, c.ann14, c.ann30, got, c.want)
		}
	}
}
