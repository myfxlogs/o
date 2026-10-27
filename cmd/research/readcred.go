// readcred.go —— T-0170 认证读面 dev 端（径 A 终裁）/ T-0174 venue 束：
// boot 经隧道取 venue-map（READCRED_URL——仅内存零磁盘副本）+ 401 自重取
// （轮换自愈零值守——venue 认证拒→readcred 重取一次→按本臂 venue 提件换件
// 重试；再败=原错上抛 fail-closed 与无凭据现态同形让位）。map 级粒度：一束
// HTTP 取全所，臂按 venue 各提各件——哪个 venue 401 重取哪件。凭据客户端=
// NewClientReadOnly（D-0013 只读镜像结构闸——research 面 GET 之外写径不可达）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"o/internal/trading/okxexec"
)

// authGetter 签名 GET 窄面（*okxexec.Client 与 *authClient 同形——装配闭包
// 对凭据件透明）。
type authGetter interface {
	Get(ctx context.Context, pathWithQuery string) (json.RawMessage, error)
}

// authClient 只读凭据持守件（401 自重取换件；凭据仅存内存）。venue=本臂
// 取件键——venue-map 回包按此分发。
type authClient struct {
	mu      sync.Mutex
	cli     authGetter
	venue   string
	url     string
	httpc   *http.Client
	makeCli func(okxexec.Credentials) authGetter
}

// newAuthClient 装配：boot 拉取一次（败=WARN 降级无凭据件——臂 fail-closed
// 让位，401 自重取自愈，不炸 boot）。
func newAuthClient(url, venue string) *authClient {
	a := &authClient{
		cli:   okxexec.NewClientReadOnly(okxexec.Credentials{}, false),
		venue: venue,
		url:   url,
		httpc: &http.Client{Timeout: 10 * time.Second},
	}
	a.makeCli = func(c okxexec.Credentials) authGetter { return okxexec.NewClientReadOnly(c, false) }
	if err := a.refetch(context.Background()); err != nil {
		log.Printf("WARN [readcred] boot 拉取败（%v）——无凭据降级，401 自重取自愈", err)
	}
	return a
}

// refetch 取 venue-map 按本臂 venue 提件换件（boot 单发或持锁调）。
func (a *authClient) refetch(ctx context.Context) error {
	m, err := fetchReadcred(ctx, a.httpc, a.url)
	if err != nil {
		return err
	}
	creds, err := venueCreds(m, a.venue)
	if err != nil {
		return err
	}
	a.cli = a.makeCli(creds)
	return nil
}

// Get 签名 GET（认证拒→重取一次→换件重试；再败原样上抛 fail-closed）。
func (a *authClient) Get(ctx context.Context, path string) (json.RawMessage, error) {
	a.mu.Lock()
	cli := a.cli
	a.mu.Unlock()
	raw, err := cli.Get(ctx, path)
	if !isAuthErr(err) {
		return raw, err
	}
	a.mu.Lock()
	if a.cli == cli { // 他 goroutine 已换件则免重取直试新件（并发拍单飞）
		if e := a.refetch(ctx); e != nil {
			a.mu.Unlock()
			return nil, err // 重取败=原 401 上抛（fail-closed 原样让位）
		}
	}
	fresh := a.cli
	a.mu.Unlock()
	return fresh.Get(ctx, path)
}

// isAuthErr venue 认证拒判（HTTP 401 或信封 50103——unauth/钥失效同形；
// 轮换自愈唯一触发判据，他错不触发重取）。
func isAuthErr(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "http 401") {
		return true
	}
	var ae *okxexec.APIError
	return errors.As(err, &ae) && ae.Code == "50103"
}

// fetchReadcred GET /v1/readcred → venue-map（响应仅进内存——禁落盘禁日志）。
// 形：{<venue>:{<field>:<明文>}}——venue 形面校验归消费臂（venueCreds）。
func fetchReadcred(ctx context.Context, httpc *http.Client, url string) (map[string]map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("readcred http %d", resp.StatusCode)
	}
	var m map[string]map[string]string
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&m); err != nil {
		return nil, fmt.Errorf("readcred parse: %w", err)
	}
	return m, nil
}

// venueCreds venue-map → 臂凭据（缺所/缺件/未支持 venue=错 fail-closed，
// 禁造空凭据冒充）。venue 凭据形面知识在此——新所臂接入先扩此表。
func venueCreds(m map[string]map[string]string, venue string) (okxexec.Credentials, error) {
	f, ok := m[venue]
	if !ok {
		return okxexec.Credentials{}, fmt.Errorf("readcred: venue %q 缺——fail-closed", venue)
	}
	switch venue {
	case "okx":
		c := okxexec.Credentials{Key: f["api_key"], Secret: f["api_secret"], Passphrase: f["passphrase"]}
		if c.Key == "" || c.Secret == "" || c.Passphrase == "" {
			return okxexec.Credentials{}, fmt.Errorf("readcred: venue %q 缺件——fail-closed", venue)
		}
		return c, nil
	default:
		return okxexec.Credentials{}, fmt.Errorf("readcred: venue %q 凭据形未支持——fail-closed", venue)
	}
}
