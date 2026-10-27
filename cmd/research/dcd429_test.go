// dcd429_test.go —— T-0180 桩锚：逐对容错（单对恒 429→余对照常枚举+rej
// 计数）+>50% 对败→整面让位分型。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"o/internal/platform/rejects"
	"o/internal/research/dcd"
)

// fakeGetter 路径编程 getter（429 对可点名）。
type dcdFakeGetter struct{ always429 map[string]bool }

func (f dcdFakeGetter) Get(ctx context.Context, path string) (json.RawMessage, error) {
	if strings.Contains(path, "currency-pair") {
		return json.RawMessage(`[{"baseCcy":"BTC","quoteCcy":"USDT","optType":"C"},{"baseCcy":"ETH","quoteCcy":"USDT","optType":"C"},{"baseCcy":"SOL","quoteCcy":"USDT","optType":"C"}]`), nil
	}
	if f.always429[strings.Split(path, "baseCcy=")[1]] {
		return nil, fmt.Errorf("http 429: too many")
	}
	return json.RawMessage(`{"products":[{"productId":"BTC-USDT-1-C","baseCcy":"BTC","quoteCcy":"USDT","optType":"C","strike":"95000","expiry":"1735000000000","annualizedYield":"0.15","minSz":"0.1","maxSz":"5","spot":"88000"}]}`), nil
}

func TestDcdFeedPairTolerance(t *testing.T) {
	rej := rejects.New("research", "dcd_pair_429")
	feed := dcdFeedFor(dcdFakeGetter{always429: map[string]bool{"ETH&quoteCcy=USDT&optType=C": true}}, rej)
	ps, err := feed(context.Background(), dcd.Surface{EndpointFamily: "/x"})
	if err != nil {
		t.Fatalf("单对 429 不应整面败: %v", err)
	}
	if len(ps) != 2 {
		t.Fatalf("应 2 对枚举到: %d", len(ps))
	}
	if rej.Snapshot()["research/dcd_pair_429"] != 1 {
		t.Fatalf("rej 应 1: %v", rej.Snapshot())
	}
}

func TestDcdFeedHalfFailYields(t *testing.T) {
	rej := rejects.New("research", "dcd_pair_429")
	feed := dcdFeedFor(dcdFakeGetter{always429: map[string]bool{
		"BTC&quoteCcy=USDT&optType=C": true, "ETH&quoteCcy=USDT&optType=C": true}}, rej)
	if _, err := feed(context.Background(), dcd.Surface{EndpointFamily: "/x"}); err == nil {
		t.Fatal(">50% 对败应整面让位")
	}
}
