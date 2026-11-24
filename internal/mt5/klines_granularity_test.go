package mt5

// klines_granularity_test.go —— O-38593 日K粒度自验桩测：网关 timeFrame
// 分钟制实证后 16408 误投产周棒署 '1D' 史病灶——nonDailyBars 末段棒距
// 中位签名判据（周棒=168h 恒中判非日；日棒周末隙不误伤；样本不足不判）。
import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	mt5pb "o/internal/gen/mtapi/mt5"
)

func barsEvery(step time.Duration, n int, start time.Time) []*mt5pb.Bar {
	out := make([]*mt5pb.Bar, n)
	for i := range out {
		out[i] = &mt5pb.Bar{Time: timestamppb.New(start.Add(time.Duration(i) * step))}
	}
	return out
}

func TestNonDailyBarsWeeklySignature(t *testing.T) {
	now := time.Now()
	if !nonDailyBars(barsEvery(168*time.Hour, 10, now)) {
		t.Fatal("周距 168h 恒中应判非日粒度")
	}
	if nonDailyBars(barsEvery(24*time.Hour, 10, now)) {
		t.Fatal("日距 24h 不应判非日")
	}
	// 日棒含单道 >120h 长隙（节假日级）：末 6 对距仅 1 对超阈——过半不
	// 成立不误伤（判据=过半非任一）。
	var ts []time.Time
	ts = append(ts, now.Add(-8*24*time.Hour))
	for i := 1; i <= 4; i++ {
		ts = append(ts, ts[i-1].Add(24*time.Hour))
	}
	ts = append(ts, ts[4].Add(144*time.Hour)) // 单道长隙
	ts = append(ts, ts[5].Add(24*time.Hour))
	ts = append(ts, ts[6].Add(24*time.Hour))
	var bars []*mt5pb.Bar
	for _, t := range ts {
		bars = append(bars, &mt5pb.Bar{Time: timestamppb.New(t)})
	}
	if nonDailyBars(bars) {
		t.Fatal("日棒单道长隙不应判非日（>120h 仅 1/6 对）")
	}
	if nonDailyBars(barsEvery(24*time.Hour, 5, now)) {
		t.Fatal("样本<7 棒证据不足不应判")
	}
}
