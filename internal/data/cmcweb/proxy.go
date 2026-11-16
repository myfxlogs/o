// proxy.go —— REST 代理：/api/* → 已登录页内 fetch 代发 connectapi。
// 单会话制实证(SG 2026-11-16 23:04/23:15 乒乓):CMC 个人盘每次 createsession
// 踢掉前一个会话——Go 侧任何 AuthManager 会话都与浏览器会话互踢。故此径
// Go 零会话,REST 全走 app 自己的 fetch(同 JS 上下文/同会话/同指纹);
// 令牌源=CDP 旁录 app WS 发帧 auth 里的真 JWT(周期续期,自动保鲜)。
package cmcweb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// apiProxy /api/* → 页内 fetch connectapi（路径白名单只 GET /v2/*——
// 采集腿全部需求是只读 GET,写径永不放行）。
func (r *Relay) apiProxy(b *Browser) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		path := req.URL.EscapedPath()[len("/api"):] + func() string {
			if req.URL.RawQuery != "" {
				return "?" + req.URL.RawQuery
			}
			return ""
		}()
		jwt := b.AppJWT()
		if jwt == "" {
			http.Error(w, "app JWT 未捕获——浏览器会话未起", http.StatusServiceUnavailable)
			return
		}
		body, status, err := b.PageFetch(req.Context(), "https://connectapi.cmcmarkets.com"+path, jwt)
		if err != nil {
			http.Error(w, "页内 fetch 败: "+err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// PageFetch 页内 fetch——在已登录 SPA JS 上下文里发 connectapi GET
// （与 app 自发请求同栈同指纹;awaitPromise 等回执）。
func (b *Browser) PageFetch(ctx context.Context, url, jwt string) (string, int, error) {
	if b.bctx == nil {
		return "", 0, fmt.Errorf("browser 未起")
	}
	js := fmt.Sprintf(`fetch(%q, {headers: {Authorization: 'Bearer %s'}})
		.then(r => r.text().then(t => JSON.stringify({s: r.status, b: t})))
		.catch(e => JSON.stringify({s: 0, b: String(e)}))`, url, jwt)
	var res string
	fctx, cancel := context.WithTimeout(b.bctx, 30*time.Second)
	defer cancel()
	if err := chromedp.Run(fctx, chromedp.Evaluate(js, &res,
		func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		})); err != nil {
		return "", 0, fmt.Errorf("evaluate: %w", err)
	}
	var out struct {
		S int    `json:"s"`
		B string `json:"b"`
	}
	if err := json.Unmarshal([]byte(res), &out); err != nil {
		return "", 0, fmt.Errorf("fetch 回执解析: %w", err)
	}
	if out.S == 0 {
		return "", 0, fmt.Errorf("fetch 网络败: %s", out.B)
	}
	return out.B, out.S, nil
}
