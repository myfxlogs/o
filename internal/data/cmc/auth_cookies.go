// auth_cookies.go —— 登录态 cookie 罐导出件（从 auth.go 拆出，保 ≤300 行律）。
package cmc

import (
	"net/http"
	"net/url"
)

// SessionCookies 导出登录态 cookie 罐（cmcweb 浏览器臂注件——同源会话直接注入，
// 省 UI 表单自动化；凭据不出进程）。
func (a *AuthManager) SessionCookies(rawURL string) []*http.Cookie {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.httpClient.Jar.Cookies(u)
}
