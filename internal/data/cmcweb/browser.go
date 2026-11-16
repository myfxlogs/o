// browser.go —— 无头真 Chrome 生命周期+CDP 旁录。跑真 CMC platform：登录态由
// AuthManager 会话 cookie 注入（免 UI 表单自动化），app 自建 connectapi WS，
// 本件只读 webSocketFrameReceived 旁录帧——协议/会话/续期全由官方客户端自理，
// 指纹=真 Chrome（UA/自动化特征件同隐）。
package cmcweb

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

// appJWTRe app WS 发帧 auth 载荷里的 JWT 抽取（{"auth":{"token":"eyJ…"}}）。
var appJWTRe = regexp.MustCompile(`"token"\s*:\s*"(eyJ[^"]+)"`)

// AppJWT app 自发 auth 帧捕获的当前 JWT（REST 代理令牌源;空=未捕获）。
func (b *Browser) AppJWT() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.appJWT
}

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
	appWS     map[network.RequestID]bool // 匹配 WSMatch 的存活 WS 集合——app 常并行两条,一条关≠全死
	appJWT    string                     // app 自发 auth 帧里的真 JWT——REST 代理令牌源
	lastFrame time.Time
	wsURLSeen string

	allocCancel context.CancelFunc
	bctx        context.Context
	bcancel     context.CancelFunc
}

func NewBrowser(cfg BrowserCfg, relay *Relay) *Browser {
	return &Browser{cfg: cfg, relay: relay}
}

// WSAlive app WS 活态（集合非空）。
func (b *Browser) WSAlive() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.appWS) > 0
}

// LastFrame 末帧时刻（探针面）。
func (b *Browser) LastFrame() time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastFrame
}

// Start 起浏览器实例（幂等——旧实例先掐；持久卷 SingletonLock 残件先清，
// 容器硬杀会留锁=新实例 process_singleton_posix 拒起，21:22 实坑）。
func (b *Browser) Start(ctx context.Context) error {
	b.Stop()
	for _, f := range []string{"SingletonLock", "SingletonSocket", "SingletonCookie"} {
		_ = os.Remove(filepath.Join(b.cfg.ProfileDir, f))
	}
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
	b.appWS = map[network.RequestID]bool{}

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
	b.mu.Lock()
	b.appWS = map[network.RequestID]bool{}
	b.mu.Unlock()
	b.relay.SetUpstreamAlive(false)
}

// onEvent CDP 目标域事件分发。
func (b *Browser) onEvent(ev any) {
	switch e := ev.(type) {
	case *network.EventWebSocketCreated:
		log.Printf("[cmcweb] WS created: %s", e.URL) // 全量记录——匹配口径实证面
		if strings.Contains(e.URL, b.cfg.WSMatch) {
			b.mu.Lock()
			b.appWS[e.RequestID] = true
			b.wsURLSeen = e.URL
			n := len(b.appWS)
			b.mu.Unlock()
			b.relay.SetUpstreamAlive(true)
			log.Printf("[cmcweb] app WS up: %s (open=%d)", e.URL, n)
		}
	case *network.EventWebSocketFrameReceived:
		b.mu.Lock()
		ok := b.appWS[e.RequestID]
		if ok {
			b.lastFrame = time.Now()
		}
		b.mu.Unlock()
		if !ok || e.Response.Opcode != 1 { // 只收匹配 WS 集合内的 text 帧——websockbin 等杂流不混
			return
		}
		b.relay.Feed(e.Response.PayloadData)
	case *network.EventWebSocketFrameSent:
		// app 周期 auth 帧携真 JWT——REST 代理令牌源（单会话制:页内 fetch 用
		// app 自己的令牌发 connectapi,与 app 请求不可区分）。
		if _, ok := b.appWS[e.RequestID]; !ok {
			return
		}
		if m := appJWTRe.FindStringSubmatch(e.Response.PayloadData); len(m) == 2 {
			b.mu.Lock()
			b.appJWT = m[1]
			b.mu.Unlock()
		}
	case *network.EventWebSocketClosed:
		b.mu.Lock()
		if _, ok := b.appWS[e.RequestID]; !ok {
			b.mu.Unlock()
			return
		}
		delete(b.appWS, e.RequestID)
		n := len(b.appWS)
		b.mu.Unlock()
		log.Printf("[cmcweb] app WS closed (open=%d)", n)
		if n == 0 {
			b.relay.SetUpstreamAlive(false)
		}
	}
}

// InjectCookies/SessionCookies 径已退役——cookie 注入对 localStorage 型 SPA
// 无效（22:03 实证:注入后仍落登录面），单会话制下 Go 侧会话本就禁建。
// 登录唯一合法径=UI 填表（login.go）。

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

// LoginFormPresent 登录面显形探测——PerformSearch 穿 shadow 找可见 password
// 输入件（新 cmc-login web-component 的字段在 closed shadow,JS 探不到）。
func (b *Browser) LoginFormPresent(ctx context.Context) bool {
	if b.bctx == nil {
		return true
	}
	evalCtx, cancel := context.WithTimeout(b.bctx, 15*time.Second)
	defer cancel()
	var has bool
	_ = chromedp.Run(evalCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		nodes, err := searchNodes(ctx, "input")
		if err != nil {
			return err
		}
		for _, n := range nodes {
			if strings.EqualFold(attrOf(n, "type"), "password") {
				if _, _, ok := boxCenter(ctx, n); ok {
					has = true
					return nil
				}
			}
		}
		return nil
	}))
	return has
}

// Diag 登录/导航失败取证——页面 URL+title+登录面态+截图落 profile 卷
// （排查"登录提交后到底卡在哪屏"的黑盒面）。
func (b *Browser) Diag(ctx context.Context) string {
	if b.bctx == nil {
		return "browser=nil"
	}
	dctx, cancel := context.WithTimeout(b.bctx, 20*time.Second)
	defer cancel()
	var loc, title string
	_ = chromedp.Run(dctx, chromedp.Location(&loc), chromedp.Title(&title))
	hasPwd := b.LoginFormPresent(ctx)
	var shot []byte
	if err := chromedp.Run(dctx, chromedp.CaptureScreenshot(&shot)); err == nil && len(shot) > 0 {
		f := filepath.Join(b.cfg.ProfileDir, fmt.Sprintf("diag-%d.png", time.Now().Unix()))
		_ = os.WriteFile(f, shot, 0o600)
		log.Printf("[cmcweb] diag 截图落 %s", f)
	}
	return fmt.Sprintf("url=%s title=%q pwdForm=%v", loc, title, hasPwd)
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
