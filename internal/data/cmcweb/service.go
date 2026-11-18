// service.go —— cmcweb 编排面：真 Chrome 跑 platform → UI 登录 → app WS
// 旁录 → 本地中继喂 o-core WsClient + /api/ 页内 fetch REST 代理。看门狗：
// 页死重启、被登出重登、连续败退避+条件请求。单会话制——Go 侧零 AuthManager
// (任何 createsession 踢浏览器会话,SG 乒乓实证;UI 登录是唯一合法入口)。
package cmcweb

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/condreq"
)

const (
	watchdogInterval  = 30 * time.Second
	watchdogTickMax   = 90 * time.Second // 单拍硬帽——chromedp 阻塞调用楔死实证（11-18 假活8h）
	loginRetryMax     = 3
	loginRetryBackoff = 2 * time.Minute
	deadRestartMin    = 10 * time.Minute // 页活但 app WS 久死 → 整页重载重启会话
)

var (
	errLoginForm = errors.New("登录面显形——UI 登录未过")
	errNoWS      = errors.New("app WS 未建立（90s 超时）")
)

// Service 浏览器采集中继服务。
type Service struct {
	browser     *Browser
	pool        *pgxpool.Pool
	accountID   int64
	email       string
	pass        string
	wsDeadSince atomic.Int64 // app WS 死窗起点 unixnano（0=无——弃置拍与续拍并发写安全面）
}

func NewService(accountID int64, bcfg BrowserCfg, pool *pgxpool.Pool, email, pass string) *Service {
	relay := NewRelay(bcfg.RelayAddr)
	b := NewBrowser(bcfg, relay)
	relay.BindBrowser(b)
	return &Service{
		browser:   b,
		pool:      pool,
		accountID: accountID,
		email:     email,
		pass:      pass,
	}
}

// Relay 中继句柄（main 接线 HTTP 监听用）。
func (s *Service) Relay() *Relay { return s.browser.relay }

// Run 编排主环（阻塞）。
func (s *Service) Run(ctx context.Context) {
	bootOK := false
	for i := 0; i < loginRetryMax; i++ {
		if err := s.session(ctx); err != nil {
			log.Printf("[cmcweb] session 建失败(%d/%d): %v", i+1, loginRetryMax, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(loginRetryBackoff): // sched-exempt-ok：登录重试退避
			}
			continue
		}
		bootOK = true
		break
	}
	if !bootOK {
		log.Printf("[cmcweb] 登录面连三败——条件请求+冷却续看")
		_ = condreq.Request(ctx, s.pool, "credential", "cmcweb", "cmc 登录面人工过验",
			`{"missing":"浏览器登录会话建立失败(验证/凭据过期?)","impact":"CMC 浏览器采集中继停摆"}`)
	}
	s.watchdog(ctx)
}

// session 一轮完整会话建立：起浏览器→导航→页态机(登录面填表/选择页点行)→等 WS。
func (s *Service) session(ctx context.Context) error {
	if err := s.browser.Start(ctx); err != nil {
		return err
	}
	if err := s.browser.Navigate(s.browserCtx()); err != nil {
		return err
	}
	if !s.waitForApp(ctx) {
		log.Printf("[cmcweb] WS 未建 diag: %s", s.browser.Diag(ctx))
		if s.browser.LoginFormPresent(ctx) {
			return errLoginForm
		}
		return errNoWS
	}
	log.Printf("[cmcweb] session 建立——app WS 在录 acct=%d", s.accountID)
	return nil
}

// waitForApp 统一等待环——按页态分派:账户选择页→点行(15s 节流);登录页→
// UI 填表(30s 节流);WS 起→true。页态单拍采样不可靠(SPA 异步渲染,22:22
// 实坑:/#/login 已显而 form 未渲),轮询态机全覆盖。
func (s *Service) waitForApp(ctx context.Context) bool {
	deadline := time.Now().Add(browserWSWait)
	var lastClick, lastLogin time.Time
	for time.Now().Before(deadline) {
		if s.browser.WSAlive() {
			return true
		}
		switch {
		case s.browser.OnAccountSelection(ctx):
			if time.Since(lastClick) > 15*time.Second {
				res, err := s.browser.SelectAccount(ctx, s.accountID)
				if err != nil {
					log.Printf("[cmcweb] 账户选择点击败: %v", err)
				} else {
					log.Printf("[cmcweb] 账户选择页 click: %s", res)
				}
				lastClick = time.Now()
			}
		case s.browser.LoginFormPresent(ctx):
			if time.Since(lastLogin) > 30*time.Second {
				log.Printf("[cmcweb] 登录面显形——UI 登录")
				if err := s.browser.UILogin(ctx, s.email, s.pass); err != nil {
					log.Printf("[cmcweb] UILogin 败 diag: %s", s.browser.Diag(ctx))
				}
				lastLogin = time.Now()
			}
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(3 * time.Second): // sched-exempt-ok：WS 等待轮询
		}
	}
	return s.browser.WSAlive()
}

func (s *Service) browserCtx() context.Context {
	if s.browser.bctx == nil {
		return context.Background()
	}
	return s.browser.bctx
}

// watchdog 活态环：页死→重启会话；app WS 久死→重载页。每拍喂 relay
// 心跳（healthz 拍龄判源）+整拍包 watchdogTickMax 硬帽——环内任一调用
// 楔死不冻环（11-18 产线 8h 零日志零自愈实证：单调用楔死=全环陪葬）。
func (s *Service) watchdog(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.browser.Stop()
			return
		case <-time.After(watchdogInterval): // sched-exempt-ok：看门狗拍
		}
		s.browser.relay.SetWatchdogBeat()
		tctx, cancel := context.WithTimeout(ctx, watchdogTickMax)
		done := make(chan struct{})
		go func() { s.watchdogTick(tctx); close(done) }()
		select {
		case <-done:
		case <-tctx.Done():
			log.Printf("[cmcweb] watchdog 拍超 %v——掐浏览器解楔+弃置续拍", watchdogTickMax)
			s.browser.Stop() // 掐 bctx——楔死 chromedp 调用 ctx 解阻塞出 goroutine；下拍页死径整会话重建
		}
		cancel()
	}
}

// watchdogTick 单拍体：页死→重启会话；app WS 久死→重载页（ctx=拍级时限）。
func (s *Service) watchdogTick(ctx context.Context) {
	if !s.browser.Alive() {
		log.Printf("[cmcweb] 页死——重启会话")
		if err := s.session(ctx); err != nil {
			log.Printf("[cmcweb] 重启败: %v", err)
		}
		return
	}
	if s.browser.WSAlive() {
		s.wsDeadSince.Store(0)
		return
	}
	if s.wsDeadSince.Load() == 0 {
		s.wsDeadSince.Store(time.Now().UnixNano())
		return
	}
	if time.Since(time.Unix(0, s.wsDeadSince.Load())) <= deadRestartMin {
		return
	}
	log.Printf("[cmcweb] app WS 死超 %v——重载页", deadRestartMin)
	if err := s.browser.Navigate(s.browserCtx()); err != nil {
		log.Printf("[cmcweb] 重载败: %v", err)
	}
	if !s.waitForApp(ctx) && s.browser.LoginFormPresent(ctx) {
		log.Printf("[cmcweb] 重载后登录面显形——整会话重建")
		if err := s.session(ctx); err != nil {
			log.Printf("[cmcweb] 会话重建败: %v", err)
		}
	}
	s.wsDeadSince.Store(0)
}
