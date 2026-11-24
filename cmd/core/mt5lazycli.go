// mt5lazycli.go —— T-0145 mt5 懒连会话件（自 xvenue.go 拆件——棘轮 D-588；
// O-37449 venue 参数化：venue=名册户 token，空/'mt5'=Exness 兼容）。
package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/platform/config"
)

// mt5LazyCli 懒连会话件（mtexec.Cli 产线实现+Suspecter——T-0145 装配钦定：
// 常驻会话跨重启/断链自愈——首用 OpenSession（凭据缺=取用点 err 显形非
// boot 炸）；RPC 败→Suspect() 标失效下次 ensure 重连）。
type mt5LazyCli struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	venue string // O-37448——服务哪户（空/'mt5'=Exness 兼容；xvenue carry 腿=Exness 户）
	mu    sync.Mutex
	c     *mt5.Client
	dead  bool
	acct  string
}

func (l *mt5LazyCli) ensure(ctx context.Context) (*mt5.Client, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.c != nil && !l.dead {
		return l.c, nil
	}
	if l.c != nil {
		l.c.Disconnect(ctx)
		l.c.Close()
		l.c, l.acct = nil, ""
	}
	c, err := mt5.OpenSession(ctx, l.pool, l.cfg, l.venue)
	if err != nil {
		l.dead = false // 未建连=非失效标（下拍续试不连坐既有会话）
		return nil, err
	}
	l.c, l.dead = c, false
	return c, nil
}

// Suspect 会话失效标记（mtexec.Suspecter——RPC 传输败径调用；下拍重连自愈）。
func (l *mt5LazyCli) Suspect() {
	l.mu.Lock()
	l.dead = true
	l.mu.Unlock()
}

// account 户号实读（方言 account 键——AccountFacts 会话内现读一次后钉；
// 会话未建=ensure 先连，凭据缺=err fail-closed 禁猜户号）。
func (l *mt5LazyCli) account(ctx context.Context) (string, error) {
	c, err := l.ensure(ctx)
	if err != nil {
		return "", err
	}
	l.mu.Lock()
	acct := l.acct
	l.mu.Unlock()
	if acct != "" {
		return acct, nil
	}
	login, _, _, err := c.AccountFacts(ctx)
	if err != nil {
		l.Suspect()
		return "", fmt.Errorf("mt5 AccountFacts 户号实读: %w", err)
	}
	acct = strconv.FormatInt(login, 10)
	l.mu.Lock()
	l.acct = acct
	l.mu.Unlock()
	return acct, nil
}

func (l *mt5LazyCli) Authed(ctx context.Context) context.Context {
	if c, err := l.ensure(ctx); err == nil {
		return c.Authed(ctx)
	}
	return ctx // 会话未建=无 token——RPC 败上浮 executor 显形（err 路径 Suspect 自愈）
}
func (l *mt5LazyCli) Conn() *grpc.ClientConn {
	if c, err := l.ensure(context.Background()); err == nil {
		return c.Conn()
	}
	return nil // 懒连败=nil conn——取用点 nil 守 err 非 panic（grpc v1.84 nil conn 调用=cc.dopts nil deref 实证，T-0145 返修②）
}
func (l *mt5LazyCli) MT5() mt5pb.MT5Client {
	if c, err := l.ensure(context.Background()); err == nil {
		return c.MT5()
	}
	return nil // 懒连败=nil——取用点 nil 守 err 非 panic（NewMT5Client(nil) 调用=nil deref 病灶根修，T-0145 返修②）
}
