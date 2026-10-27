// TRX 采集器 HTTP 辅助（TronGrid API 请求）。
package trx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// httpClient 可被测试替换（httptest.NewServer + SetHTTPClient）。
var httpClient = &http.Client{Timeout: 30 * time.Second}

// SetHTTPClient 替换默认 HTTP 客户端（测试用 httptest.Server）。
func SetHTTPClient(c *http.Client) { httpClient = c }

// tronPost 发送 TronGrid POST 请求（getaccount/listwitnesses 等）。
func tronPost(ctx context.Context, base, apiKey, path, body string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, "POST", base+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", apiKey)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("tron POST %s: status %d", path, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// doWithRetry 幂等 GET 瞬态重试（T-0182：容器 embedded DNS 间歇 NXDOMAIN
// 病灶对症件——no such host 瞬态层重试≤3 次指数退避自愈合；
// 幂等 GET 语义无副作用风险）。
func doWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	for try := 0; try < 4; try++ {
		resp, err = httpClient.Do(req)
		if err == nil {
			return resp, nil
		}
		if !strings.Contains(err.Error(), "no such host") {
			return nil, err // 非 DNS 层败不重试（401/超时等语义直返）
		}
		if try == 3 {
			break
		}
		time.Sleep(time.Duration(1<<try) * 500 * time.Millisecond) // sched-exempt-ok：DNS 瞬态退避
	}
	return nil, err
}

// tronGet 发送 TronGrid GET 请求（/v1/transactions 等）。
func tronGet(ctx context.Context, base, apiKey, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", apiKey)
	}
	resp, err := doWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("tron GET: status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
