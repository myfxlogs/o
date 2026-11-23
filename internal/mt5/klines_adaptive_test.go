package mt5

// klines_adaptive_test.go —— O-35587 日K批写自适应拆批桩测：网关 504
// （载荷=符×棒数乘积超时）下整批零落→对半拆至单符地板恢复落行；
// 单符真败=样本上抛不再拆；部分落行（n>0 带 why）不拆。
import (
	"context"
	"strings"
	"testing"
)

func TestKlineWriteAdaptiveSplitOnTotalFail(t *testing.T) {
	c := &BridgeCollector{}
	// 桩：批>2 全灭、≤2 全活——模拟 504 载荷阈。
	c.klineWr = func(ctx context.Context, cl *Client, batch []string) (int, string) {
		if len(batch) > 2 {
			return 0, "rpc error: code = Unavailable desc = 504 Gateway Timeout"
		}
		return len(batch) * 10, ""
	}
	syms := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	n, why := c.klineWriteAdaptive(context.Background(), nil, syms)
	if why != "" {
		t.Fatalf("拆批后不应有败样: %s", why)
	}
	if n != 80 {
		t.Fatalf("8 符×10 棒=80，实得 %d", n)
	}
}

func TestKlineWriteAdaptiveSingleSymbolFloor(t *testing.T) {
	c := &BridgeCollector{}
	var calls int
	c.klineWr = func(ctx context.Context, cl *Client, batch []string) (int, string) {
		calls++
		if len(batch) > 1 {
			return 0, "504"
		}
		if batch[0] == "BAD" {
			return 0, "symbol exception"
		}
		return 10, ""
	}
	syms := []string{"A", "BAD", "C"}
	n, why := c.klineWriteAdaptive(context.Background(), nil, syms)
	// A/C 落行、BAD 单符真败上抛——拆到单符地板为止。
	if n != 20 {
		t.Fatalf("A+C=20 棒，实得 %d", n)
	}
	if why == "" || !strings.Contains(why, "symbol exception") {
		t.Fatalf("BAD 单符败样应上抛，实得 %q", why)
	}
	// 调用次数=整批1+左半1+右半递归…单调递减拆至地板，BAD 不再重试。
	if calls < 5 {
		t.Fatalf("拆批调用过少 calls=%d", calls)
	}
}

func TestKlineWriteAdaptivePartialWriteNoSplit(t *testing.T) {
	c := &BridgeCollector{}
	var calls int
	c.klineWr = func(ctx context.Context, cl *Client, batch []string) (int, string) {
		calls++
		return 5, "kline_write_err mid-batch"
	}
	n, why := c.klineWriteAdaptive(context.Background(), nil, []string{"A", "B", "C", "D"})
	if calls != 1 {
		t.Fatalf("部分落行不应拆批，calls=%d", calls)
	}
	if n != 5 || why == "" {
		t.Fatalf("n=5 why 非空应直通，实得 n=%d why=%q", n, why)
	}
}

func TestKlineWriteAdaptiveCleanPass(t *testing.T) {
	c := &BridgeCollector{}
	var calls int
	c.klineWr = func(ctx context.Context, cl *Client, batch []string) (int, string) {
		calls++
		return len(batch) * 3, ""
	}
	n, why := c.klineWriteAdaptive(context.Background(), nil, []string{"A", "B"})
	if calls != 1 || n != 6 || why != "" {
		t.Fatalf("净批应单发直通 calls=%d n=%d why=%q", calls, n, why)
	}
}
