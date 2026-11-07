// client_cap_test.go —— T-0440①②桩测：公开读面帽 8MB——大响应 >1MB<8MB
// 解析过（旧 1MB 帽必截断→「resp parse: unexpected end of JSON input」伪装
// 病回归锚）+>8MB 显式 ErrOverCap（超帽错非 JSON 语法错）。
package okxexec

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"o/internal/platform/httpread"
)

// bigEnvelope 造累计体量达 target 的合法信封（逐行追加实长控制）。
func bigEnvelope(target int) string {
	var b strings.Builder
	b.WriteString(`{"code":"0","msg":"","data":[`)
	for i := 0; b.Len() < target; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"instId":"X-%06d-USDT"}`, i)
	}
	b.WriteString(`]}`)
	return b.String()
}

func TestPublicGetLargeResponseParses(t *testing.T) {
	body := bigEnvelope(1<<20 + 512<<10) // ≈1.5MB——>旧 1MB 帽 <8MB 新帽
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	c := withBase(NewClient(Credentials{}, false), srv.URL)
	raw, err := c.PublicGet(context.Background(), "/api/v5/public/instruments?instType=SPOT")
	if err != nil {
		t.Fatalf("帽升 8MB 后 >1MB 响应应解析过（旧帽截断病灶回归锚）: %v", err)
	}
	if len(raw) <= 1<<20 {
		t.Fatalf("data 应全量非截断（>1MB 锚）: %d", len(raw))
	}
}

func TestPublicGetOverCapExplicitErr(t *testing.T) {
	body := bigEnvelope(8<<20 + 512<<10) // ≈8.5MB——超新帽
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	c := withBase(NewClient(Credentials{}, false), srv.URL)
	_, err := c.PublicGet(context.Background(), "/api/v5/public/instruments?instType=SPOT")
	if !errors.Is(err, httpread.ErrOverCap) {
		t.Fatalf("超帽应显式 ErrOverCap 非 JSON 语法错伪装: %v", err)
	}
}
