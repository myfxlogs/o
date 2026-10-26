// rejects_test.go —— T-0148 锚：depth5 缺 ctval 拒因注册面——缺 ctval 行
// 计数累加且零 WARN（未注册键运行期落 WARN=审计门完整性破+日志洪泛，
// o-core 每分 ~77 行实案形）。
package okxws

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"o/internal/platform/rejects"
)

// 锚①：生产门构造含 depth5_no_ctval（构造面直绑——注册漏即红）。
func TestOkxwsDoorHasDepth5NoCtval(t *testing.T) {
	c := NewCollector(nil, nil)
	if _, ok := c.rej.Snapshot()["okxws/depth5_no_ctval"]; !ok {
		t.Fatal("生产门应注册 depth5_no_ctval（未注册=每次 Inc 落 WARN 洪泛）")
	}
}

// 锚②（票面桩锚）：books5 帧缺 ctval→计数累加+零 WARN。
func TestDepth5NoCtvalCountNoWarn(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	c := &Collector{rej: rejects.New("okxws", "depth5_no_ctval"),
		ctvCache: map[string]float64{}, queue: make(chan rowItem, 4)}
	frame := []byte(`{"arg":{"channel":"books5","instId":"BTC-USDT-SWAP"},"data":[{}]}`)
	c.handle(frame)
	c.handle(frame) // 二帧证累加（非一次性位）
	if n := c.rej.Snapshot()["okxws/depth5_no_ctval"]; n != 2 {
		t.Fatalf("缺 ctval 行应计数累加 got %d", n)
	}
	if strings.Contains(buf.String(), "未注册门键") {
		t.Fatalf("注册键不得落 WARN: %s", buf.String())
	}
}
