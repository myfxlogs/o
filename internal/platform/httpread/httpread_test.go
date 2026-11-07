// httpread_test.go —— T-0440③b 共享读件桩测：帽内原样返回/恰帽边界不误报/
// 超帽显式 ErrOverCap（非静默截断）/帽非法拒装/读败原样上抛。
package httpread

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestBodyWithinCap(t *testing.T) {
	b, err := Body(bytes.NewReader([]byte(`{"code":"0"}`)), 1<<20)
	if err != nil || string(b) != `{"code":"0"}` {
		t.Fatalf("帽内读应原样返回: %v %q", err, b)
	}
}

func TestBodyExactCapBoundary(t *testing.T) {
	body := strings.Repeat("a", 1<<20) // 恰帽=合法不超
	b, err := Body(bytes.NewReader([]byte(body)), 1<<20)
	if err != nil || len(b) != 1<<20 {
		t.Fatalf("恰帽边界应放行: %v len=%d", err, len(b))
	}
}

func TestBodyOverCapExplicit(t *testing.T) {
	body := strings.Repeat("a", (1<<20)+2) // 帽+2——多读一字节探到截断
	_, err := Body(bytes.NewReader([]byte(body)), 1<<20)
	if !errors.Is(err, ErrOverCap) {
		t.Fatalf("超帽应显式 ErrOverCap（非静默截断）: %v", err)
	}
	if !strings.Contains(err.Error(), "响应超帽") {
		t.Fatalf("错文应携「响应超帽」锚: %v", err)
	}
}

func TestBodyReadErrPropagates(t *testing.T) {
	_, err := Body(io.NopCloser(io.MultiReader(bytes.NewReader([]byte("ab")), &errReader{})), 1<<20)
	if err == nil || errors.Is(err, ErrOverCap) {
		t.Fatalf("读败应原样上抛非超帽错: %v", err)
	}
}

type errReader struct{}

func (*errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestBodyCapMustBePositive(t *testing.T) {
	if _, err := Body(bytes.NewReader(nil), 0); err == nil {
		t.Fatal("帽<1 应拒装")
	}
}
