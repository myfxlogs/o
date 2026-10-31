// retry.go —— 私有 REST 读径 429 指数退避（T-0272④）：GET 限定——venue 现读
// 径免疫单点 429 炸拍；≤3 次重试、总窗 ≤10s 防拍阻塞，尽仍败=err 上抛不假零；
// 写径与非 429 败零重试零新增延迟；禁做成「读败返陈值」。
package okxexec

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"
)

// HTTPStatusError HTTP 非 2xx（429 退避判需码位——do() 层消费；
// Error() 文本保持 http %d 原口径不变）。
type HTTPStatusError struct {
	Code int
	Body string
}

func (e *HTTPStatusError) Error() string {
	return "http " + strconv.Itoa(e.Code) + ": " + e.Body
}

const (
	doRetryMax    = 3                      // 重试上限（首发外 ≤3 次）
	doRetryBase   = 500 * time.Millisecond // 退避底——500ms/1s/2s 指数
	doRetryWindow = 10 * time.Second       // 总窗上限——防事件拍阻塞
)

// do 签名请求入口（ro 写径硬闸+GET+429 指数退避包装——单发体=doOnce）。
func (c *Client) do(ctx context.Context, method, pathWithQuery string, body []byte) ([]byte, error) {
	if c.ro && method != http.MethodGet {
		return nil, ErrReadOnly // T-0104：写径签名前硬闸——非 GET 一律不达 venue
	}
	deadline := time.Now().Add(doRetryWindow)
	for attempt := 0; ; attempt++ {
		b, err := c.doOnce(ctx, method, pathWithQuery, body)
		var se *HTTPStatusError
		if err == nil || method != http.MethodGet ||
			!errors.As(err, &se) || se.Code != http.StatusTooManyRequests ||
			attempt >= doRetryMax || time.Now().After(deadline) {
			return b, err
		}
		wait := doRetryBase << attempt
		if rem := time.Until(deadline); wait > rem {
			wait = rem
		}
		t := time.NewTimer(wait) // sched-exempt-ok：429 退避期界等待（非轮询——venue 限流自愈窗）
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, ctx.Err()
		case <-t.C:
		}
	}
}
