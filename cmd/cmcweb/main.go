// cmcweb —— CMC 浏览器采集中继（业主裁 2026-11-16：爬虫面换真 Chrome 指纹）。
// 真 CMC platform 跑在无头 Chrome 里，app 自建 connectapi WS，本件 CDP 旁录帧
// → 本地中继冒充 WS 端点 → o-core WsClient 指 ws://o-cmcweb:8080 原样消费。
// 登录态=AuthManager 会话 cookie 注入（凭据不出进程，免 UI 表单自动化）。
package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/data/cmc"
	"o/internal/data/cmcweb"
	"o/internal/platform/condreq"
	"o/internal/platform/config"
	"o/internal/platform/pgguard"
	"o/internal/platform/roster"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		log.Fatal("PG_DSN 缺——配置/名册/条件请求面无落点 fail-closed")
	}
	pcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("PG_DSN 解析败: %v", err)
	}
	pgguard.GuardPool(pcfg)
	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		log.Fatalf("PG 连接败: %v", err)
	}
	defer pool.Close()
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		log.Fatalf("config load: %v", err)
	}

	sec := func(k string) string {
		b, err := cfg.Secret(ctx, k)
		if err != nil {
			return ""
		}
		return string(b)
	}
	cmc.SetCreateTplHex(sec("cmc.create_tpl"))
	email, pass := sec("cmc.email"), sec("cmc.password")
	acctS, _ := cfg.GetString("cmc.account_id")
	acct, _ := strconv.ParseInt(acctS, 10, 64)
	if email == "" || pass == "" || acct == 0 {
		_ = condreq.Request(ctx, pool, "credential", "cmc", "cmc.email+cmc.password+cmc.account_id",
			`{"missing":"凭据/账户号三件","impact":"CMC 浏览器采集中继停摆"}`)
		log.Fatal("cmc 凭据键缺——条件请求已落，让位")
	}

	get := func(k, d string) string {
		v, _ := cfg.GetString(k)
		if v == "" {
			return d
		}
		return v
	}
	bcfg := cmcweb.BrowserCfg{
		PlatformBase: get("cmcweb.platform_base", cmc.DefaultPlatformBase),
		WSMatch:      get("cmcweb.ws_match", "connectapi.cmcmarkets.com/ws"),
		ChromePath:   get("cmcweb.chrome_path", "/usr/bin/chromium-browser"),
		ProfileDir:   get("cmcweb.profile_dir", "/var/lib/cmcweb/profile"),
		UserAgent:    get("cmcweb.user_agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/157.0.0.0 Safari/537.36"),
		RelayAddr:    get("cmcweb.relay_addr", ":8080"),
	}

	if err := roster.Register(ctx, pool, "CMC 浏览器中继", "cmc", "真Chrome采集中继岗(指纹/会话官侧自理)", nil); err != nil {
		log.Fatalf("roster register: %v", err)
	}
	errc := make(chan error, 4)
	go roster.BeatLoop(ctx, pool, "CMC 浏览器中继", 30*time.Second, errc)

	auth := cmc.NewAuthManager(email, pass, acct, bcfg.PlatformBase)
	svc := cmcweb.NewService(auth, acct, bcfg, pool, email, pass)
	go func() {
		if err := svc.Relay().ListenAndServe(ctx); err != nil {
			errc <- err
		}
	}()
	go svc.Run(ctx)
	log.Printf("[cmcweb] 浏览器采集中继上岗 acct=%d relay=%s", acct, bcfg.RelayAddr)
	log.Fatal(<-errc)
}
