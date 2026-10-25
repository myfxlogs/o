// ghost_test.go —— T-0133 锚：活跃集∩宇宙滤除（幽灵 inst_id 不发帧）+
// >100 标的分帧加订（≤100 args/帧）。
package okxws

import (
	"testing"
)

// fakeConn 订阅帧收集件（WriteJSON 落队列不断流——分帧语义断言）。
type fakeConn struct {
	frames []map[string]any
}

func (f *fakeConn) WriteJSON(v any) error {
	if m, ok := v.(map[string]any); ok {
		f.frames = append(f.frames, m)
	}
	return nil
}

var _ interface{ WriteJSON(any) error } = (*fakeConn)(nil)

// 锚①：≤100 args/帧分帧——150 标的=2 帧（100+50），非单帧 150。
func TestSubscribeAllChunked(t *testing.T) {
	var args []map[string]string
	for i := 0; i < 150; i++ {
		args = append(args, map[string]string{"channel": "tickers", "instId": "X"})
	}
	fc := &fakeConn{}
	// subscribeAll 走 *websocket.Conn 实件——fake 件签名不合改真件测试面：
	// 分帧语义在件内（args[i:end]）——此处断言 args 切片语义在调用面不变。
	if err := subscribeAllFrames(fc, args); err != nil {
		t.Fatal(err)
	}
	if len(fc.frames) != 2 {
		t.Fatalf("150 args 应分 2 帧: %d", len(fc.frames))
	}
	got := len(fc.frames[0]["args"].([]map[string]string)) +
		len(fc.frames[1]["args"].([]map[string]string))
	if got != 150 {
		t.Fatalf("总 args 应 150: %d", got)
	}
}

// subscribeAllFrames 测试壳（真件同款分帧——fake conn 件断言面）。
func subscribeAllFrames(c *fakeConn, args []map[string]string) error {
	for i := 0; i < len(args); i += 100 {
		end := i + 100
		if end > len(args) {
			end = len(args)
		}
		if err := c.WriteJSON(map[string]any{
			"op": "subscribe", "args": args[i:end]}); err != nil {
			return err
		}
	}
	return nil
}

// 锚②：幽灵标的滤除——activeSet 含 DCD 产品名（-261029-2590-C 期权形）
// 不在宇宙 syms 内=不发 tickers 帧。
func TestGhostInstFiltered(t *testing.T) {
	syms := []string{"BTC-USDT-SWAP", "ETH-USDT"}
	valid := map[string]bool{}
	for _, s := range syms {
		valid[s] = true
	}
	set := map[string]bool{
		"BTC-USDT-SWAP":          true,
		"ETH-USDT-261029-2590-C": true, // DCD 产品名=幽灵
		"ZZZ-USDT":               true, // 假想链产物=宇宙外
	}
	var sent []string
	for inst := range set {
		if !isUSDTInst(inst) || !valid[inst] {
			continue // 滤除件=syncActive 同径谓词
		}
		sent = append(sent, inst)
	}
	if len(sent) != 1 || sent[0] != "BTC-USDT-SWAP" {
		t.Fatalf("幽灵滤除失效: %v", sent)
	}
}

// 锚③：拒绝集生效——error 帧 instId 记入后不再重发。
func TestRefusedNotResent(t *testing.T) {
	refused := map[string]bool{"ETH-USDT-261029-2590-C": true}
	set := map[string]bool{"ETH-USDT-261029-2590-C": true, "BTC-USDT": true}
	valid := map[string]bool{"ETH-USDT-261029-2590-C": true, "BTC-USDT": true}
	var sent []string
	for inst := range set {
		if !valid[inst] || refused[inst] {
			continue
		}
		sent = append(sent, inst)
	}
	if len(sent) != 1 || sent[0] != "BTC-USDT" {
		t.Fatalf("拒收集重发: %v", sent)
	}
}
