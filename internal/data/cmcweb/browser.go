// browser.go —— 无头真 Chrome 生命周期+CDP 旁录。跑真 CMC platform：登录态由
// AuthManager 会话 cookie 注入（免 UI 表单自动化），app 自建 connectapi WS，
// 本件只读 webSocketFrameReceived 旁录帧——协议/会话/续期全由官方客户端自理，
// 指纹=真 Chrome（UA/自动化特征件同隐）。
package cmcweb

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	browserWSWait    = 90 * time.Second // 导航后等 app WS 建立上限
	browserNavTimout = 60 * time.Second
)

// BrowserCfg 浏览器装配键。
type BrowserCfg struct {
	PlatformBase string // https://platform.cmcmarkets.com
	WSMatch      string // app WS URL 匹配子串（connectapi.cmcmarkets.com/ws）
	ChromePath   string // chromium 可执行件
	ProfileDir   string // user-data-dir 持久化目录
	UserAgent    string // 真 Chrome UA（隐 HeadlessChrome 字样）
	RelayAddr    string // 本地中继监听址（docker 内网名消费）
}

// Browser 无头实例+旁录态。
type Browser struct {
	cfg   BrowserCfg
	relay *Relay

	mu        sync.Mutex
	wsAlive   bool
	lastFrame time.Time
	wsURLSeen string

	allocCancel context.CancelFunc
	bctx        context.Context
	bcancel     context.CancelFunc
}

func NewBrowser(cfg BrowserCfg, relay *Relay) *Browser {
	return &Browser{cfg: cfg, relay: relay}
}

// WSAlive app WS 活态。
func (b *Browser) WSAlive() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.wsAlive
}

// LastFrame 末帧时刻（探针面）。
func (b *Browser) LastFrame() time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastFrame
}

// Start 起浏览器实例（幂等——旧实例先掐）。
func (b *Browser) Start(ctx context.Context) error {
	b.Stop()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(b.cfg.ChromePath),
		chromedp.Flag("headless", "new"),
		chromedp.Flag("no-sandbox", true), // 容器内无 userns——钉死
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("user-data-dir", b.cfg.ProfileDir),
		chromedp.Flag("disable-blink-features", "AutomationControlled"), // navigator.webdriver 隐件
		chromedp.Flag("user-agent", b.cfg.UserAgent),
		chromedp.Flag("lang", "en-GB"),
		chromedp.WindowSize(1440, 900),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	bctx, bcancel := chromedp.NewContext(allocCtx)
	b.allocCancel, b.bctx, b.bcancel = allocCancel, bctx, bcancel

	chromedp.ListenTarget(bctx, b.onEvent)
	if err := chromedp.Run(bctx, network.Enable(), page.Enable(), runtime.Enable()); err != nil {
		b.Stop()
		return fmt.Errorf("browser boot: %w", err)
	}
	log.Printf("[cmcweb] browser up profile=%s", b.cfg.ProfileDir)
	return nil
}

// Stop 掐实例（幂等）。
func (b *Browser) Stop() {
	if b.bcancel != nil {
		b.bcancel()
		b.bcancel = nil
	}
	if b.allocCancel != nil {
		b.allocCancel()
		b.allocCancel = nil
	}
	b.setAlive(false, "")
}

// onEvent CDP 目标域事件分发。
func (b *Browser) onEvent(ev any) {
	switch e := ev.(type) {
	case *network.EventWebSocketCreated:
		if strings.Contains(e.URL, b.cfg.WSMatch) {
			b.setAlive(true, e.URL)
			log.Printf("[cmcweb] app WS up: %s", e.URL)
		}
	case *network.EventWebSocketFrameReceived:
		b.mu.Lock()
		alive := b.wsAlive
		b.mu.Unlock()
		if !alive || e.Response.Opcode != 1 { // 只收本端 WS 的 text 帧
			return
		}
		b.mu.Lock()
		b.lastFrame = time.Now()
		b.mu.Unlock()
		b.relay.Feed(e.Response.PayloadData)
	case *network.EventWebSocketClosed:
		b.mu.Lock()
		if b.wsAlive {
			b.mu.Unlock()
			b.setAlive(false, "")
			log.Printf("[cmcweb] app WS closed")
		} else {
			b.mu.Unlock()
		}
	}
}

func (b *Browser) setAlive(alive bool, u string) {
	b.mu.Lock()
	b.wsAlive, b.wsURLSeen = alive, u
	b.mu.Unlock()
	b.relay.SetUpstreamAlive(alive)
}

// InjectCookies 登录态注入——AuthManager 会话 cookie 罐写进浏览器 cookie 库。
func (b *Browser) InjectCookies(ctx context.Context, cookies []*http.Cookie) error {
	if len(cookies) == 0 {
		return fmt.Errorf("cookie 罐空")
	}
	u, err := url.Parse(b.cfg.PlatformBase)
	if err != nil {
		return err
	}
	for _, c := range cookies {
		dom := c.Domain
		if dom == "" {
			dom = u.Host
		}
		p := network.SetCookie(c.Name, c.Value).
			WithDomain(dom).
			WithPath(defaultStr(c.Path, "/")).
			WithSecure(true).
			WithHTTPOnly(c.HttpOnly)
		if err := chromedp.Run(ctx, p); err != nil {
			return fmt.Errorf("setCookie %s: %w", c.Name, err)
		}
	}
	return nil
}

func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// Navigate 导航 platform 根（app 自启自连 WS）。
func (b *Browser) Navigate(ctx context.Context) error {
	nctx, cancel := context.WithTimeout(ctx, browserNavTimout)
	defer cancel()
	return chromedp.Run(nctx, chromedp.Navigate(b.cfg.PlatformBase+"/"))
}

// WaitWS 轮询等 app WS 建立（≤browserWSWait）。
func (b *Browser) WaitWS(ctx context.Context) bool {
	deadline := time.Now().Add(browserWSWait)
	for time.Now().Before(deadline) {
		if b.WSAlive() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(2 * time.Second): // sched-exempt-ok：WS 建立等待轮询
		}
	}
	return false
}

// LoginFormPresent 登录面显形探测（password 输入框在=被登出）。
func (b *Browser) LoginFormPresent(ctx context.Context) bool {
	if b.bctx == nil {
		return true
	}
	var has bool
	evalCtx, cancel := context.WithTimeout(b.bctx, 15*time.Second)
	defer cancel()
	if err := chromedp.Run(evalCtx, chromedp.Evaluate(
		`!!document.querySelector('input[type="password"]')`, &has)); err != nil {
		return true // 页死视同登出——走重登径
	}
	return has
}

// UILogin 登录面兜底——cookie 注入不生效时填表单（CMC loginForm=AngularJS:
// #username type=text + #password + input[type=submit] ng-click=login(),
// SG dump-dom 实证 2026-11-16；OneTrust 横幅可能盖提交钮先接受再填）。
func (b *Browser) UILogin(ctx context.Context, email, pass string) error {
	if b.bctx == nil {
		return fmt.Errorf("browser 未起")
	}
	lctx, cancel := context.WithTimeout(b.bctx, 45*time.Second)
	defer cancel()
	userSel := `#username, input[name="username"], input[type="email"], input[autocomplete="username"]`
	// OneTrust 同意横幅先撤（best-effort——不在场=3s 过，不占登录预算）
	bctx, bcancel := context.WithTimeout(lctx, 3*time.Second)
	_ = chromedp.Run(bctx, chromedp.Click(`#onetrust-accept-btn-handler`, chromedp.ByQuery))
	bcancel()
	return chromedp.Run(lctx,
		chromedp.WaitVisible(userSel, chromedp.ByQuery),
		chromedp.Clear(userSel, chromedp.ByQuery),
		chromedp.SendKeys(userSel, email, chromedp.ByQuery),
		chromedp.SendKeys(`#password`, pass, chromedp.ByQuery),
		chromedp.Click(`input[type="submit"], button[type="submit"]`, chromedp.ByQuery),
	)
}

// Alive 页活性探针（target 死=Run 败）。
func (b *Browser) Alive() bool {
	if b.bctx == nil {
		return false
	}
	var v int
	evalCtx, cancel := context.WithTimeout(b.bctx, 10*time.Second)
	defer cancel()
	return chromedp.Run(evalCtx, chromedp.Evaluate(`1`, &v)) == nil
}
