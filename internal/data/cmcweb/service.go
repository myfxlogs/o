// service.go —— cmcweb 编排面：AuthManager 登录 → 会话 cookie 注入 → 真 Chrome
// 跑 platform → app WS 旁录 → 本地中继喂 o-core WsClient。看门狗：页死重启、
// 被登出重注、连续败退避+条件请求。凭据不出进程（Go 侧只用于 cookie 源）。
package cmcweb

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/data/cmc"
	"o/internal/platform/condreq"
)

const (
	watchdogInterval  = 30 * time.Second
	loginRetryMax     = 3
	loginRetryBackoff = 2 * time.Minute
	deadRestartMin    = 10 * time.Minute // 页活但 app WS 久死 → 整页重载重启会话
)

var (
	errLoginForm = errors.New("登录面显形——会话 cookie 失签")
	errNoWS      = errors.New("app WS 未建立（90s 超时）")
)

// Service 浏览器采集中继服务。
type Service struct {
	auth      *cmc.AuthManager
	browser   *Browser
	pool      *pgxpool.Pool
	accountID int64
	email     string
	pass      string
}

func NewService(auth *cmc.AuthManager, accountID int64, bcfg BrowserCfg, pool *pgxpool.Pool, email, pass string) *Service {
	return &Service{
		auth:      auth,
		browser:   NewBrowser(bcfg, NewRelay(bcfg.RelayAddr)),
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
	go s.auth.StartRefresh(ctx) // Go 侧会话续命——cookie 源恒温

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

// session 一轮完整会话建立：登录→起浏览器→注 cookie→导航→等 WS。
func (s *Service) session(ctx context.Context) error {
	if err := s.auth.Login(ctx); err != nil {
		return err
	}
	if err := s.browser.Start(ctx); err != nil {
		return err
	}
	cookies := s.auth.SessionCookies(s.browser.cfg.PlatformBase)
	if err := s.browser.InjectCookies(s.browserCtx(), cookies); err != nil {
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

// watchdog 活态环：页死→重启会话；app WS 久死→重载页。
func (s *Service) watchdog(ctx context.Context) {
	deadSince := time.Time{}
	for {
		select {
		case <-ctx.Done():
			s.browser.Stop()
			return
		case <-time.After(watchdogInterval): // sched-exempt-ok：看门狗拍
		}
		if !s.browser.Alive() {
			log.Printf("[cmcweb] 页死——重启会话")
			if err := s.session(ctx); err != nil {
				log.Printf("[cmcweb] 重启败: %v", err)
			}
			deadSince = time.Time{}
			continue
		}
		if !s.browser.WSAlive() {
			if deadSince.IsZero() {
				deadSince = time.Now()
			} else if time.Since(deadSince) > deadRestartMin {
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
				deadSince = time.Time{}
			}
		} else {
			deadSince = time.Time{}
		}
	}
}
