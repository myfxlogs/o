package mt5

import (
	"testing"
	"time"
)

// O-38639 回归锚：mtapi ExMany timeFrame 域=分钟（实测方言图
// {1=M1,60=H1,240=H4,1440=D1,10080=W1}）——mt5FrameD1 必须=1440，
// 禁回退 MQL5 ENUM_TIMEFRAMES PERIOD_D1=16408（分钟语义下≈11.4d
// 被网关吸附 W1，全簿周K伪 1D 的陈窗签名根因）。
func TestKlineFrameDailyIsMinutes(t *testing.T) {
	if mt5FrameD1 != 1440 {
		t.Fatalf("mt5FrameD1=%d——mtapi ExMany timeFrame 单位=分钟，D1=1440；16408 系 MQL5 枚举误搬（O-38639 周K伪1D根因）", mt5FrameD1)
	}
}

func TestKlineAvgGapSuspect(t *testing.T) {
	base := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	// 真日棒：200 棒跨≈280d（含周末）均值≈1.4d→不疑
	if klineAvgGapSuspect(base.AddDate(0, 0, -280), base, 200) {
		t.Fatal("日棒窗均值 1.4d 被误判周期错配")
	}
	// 周K伪 1D：200 棒跨≈1400d 均值=7d→疑
	if !klineAvgGapSuspect(base.AddDate(0, 0, -1393), base, 200) {
		t.Fatal("周K窗均值 7d 未被检出——O-38639 签名漏检")
	}
	// 棒数不足不判
	if klineAvgGapSuspect(base.AddDate(0, 0, -14), base, 2) {
		t.Fatal("n<3 不应判周期错配")
	}
}
