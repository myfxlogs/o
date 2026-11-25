// shared.go —— 按户共享会话行（<venue>.session_token）：一账户一会话
// 架构锚（O-37448——键=venue 寻址非 cred_ns：venue=账户身份 token 一身三任；
// 桥腿在 PG-md 无 fin_venue_roster 读权不可解 cred_ns，venue 名即消费腿
// 唯一可声明身份——design D2「按 venue 参数化复挂」的直实现；
// 'mt5'→'mt5.session_token' 存量键不动=Exness 兼容零迁移窗，
// 'xmglobal'→'xmglobal.session_token'）。
// 实测（2026-11-15 跨机实证 mt5tokprobe）：token=后端 MT5 会话句柄——
// 不绑 gRPC 连接不绑出口 IP；两台主机各自连接递同一 token 并发读全绿，
// 单边断连不连坐。角色制：汇市巡=owner 独占 ConnectEx（凭据在 app_secrets，
// 按户各持 owner 权）；OCI 桥腿=纯消费腿（读共享行→SetSharedToken→
// CheckAlive，永不 ConnectEx 永不 Disconnect）；OpenSession 短调用面共享
// 先行（零新 login 减 churn）。
// 共享会话不归任何消费端登出（Disconnect shared 幂等空转——误登=连坐
// 杀共享会话坑）；僵尸界=券商 TTL 自然收尸。
// 键名含 token=credentialRE 自动 owner 硬拒域——免册零闸改（会话句柄
// 按凭证形管；bridge.mt5.password app_config 明文先例同构）。
package mt5

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// sharedSessionKey 存量共享会话键（venue='exness'——Exness 兼容锚 O-39440①；
// 新户键=<venue>.session_token 由 VenueAcct.SessionKey 派生）。
const sharedSessionKey = "exness.session_token"

// sharedSession 共享会话行载荷（app_config 单键 JSON 原子装——token 按
// 网关域有效故 addr 必携；ts=owner 建会话时刻，消费腿可据此判换血）。
type sharedSession struct {
	Addr  string `json:"addr"`
	Token string `json:"token"`
	Ts    string `json:"ts"`
}

// sharedSessionRead 直读共享行（直走 pool 非 cfg snap——token 是易变会话
// 态非配置态，snap 经 Watch 刷新有窗口；缺/坏=err 消费腿 fail-closed 等 owner）。
// key=<venue>.session_token 按户寻址（O-37448——venue=账户身份消费腿唯一可声明面）。
func sharedSessionRead(ctx context.Context, pool *pgxpool.Pool, key string) (sharedSession, error) {
	var raw string
	if err := pool.QueryRow(ctx, `SELECT value FROM app_config WHERE key=$1`,
		key).Scan(&raw); err != nil {
		return sharedSession{}, err
	}
	var ss sharedSession
	if err := json.Unmarshal([]byte(raw), &ss); err != nil {
		return sharedSession{}, fmt.Errorf("共享会话行坏体: %w", err)
	}
	if ss.Addr == "" || ss.Token == "" {
		return sharedSession{}, fmt.Errorf("共享会话行缺 addr/token")
	}
	return ss, nil
}

// sharedSessionWrite owner 会话建立后落共享行（幂等——复挂同值覆写仅 ts 变；
// 写败不炸会话=消费腿晚一拍拿新值）。直写径与 cfg.Set 同 SQL 但免 snap 依赖。
func sharedSessionWrite(ctx context.Context, pool *pgxpool.Pool, key, addr, tok string) {
	b, _ := json.Marshal(sharedSession{Addr: addr, Token: tok,
		Ts: time.Now().Format(time.RFC3339Nano)})
	if _, err := pool.Exec(ctx,
		`INSERT INTO app_config(key,value,updated_at,updated_by) VALUES($1,$2,now(),'o-core/mt5sess')
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value,updated_at=now(),
		updated_by=EXCLUDED.updated_by`, key, string(b)); err != nil {
		log.Printf("WARN [mt5] 共享会话行 %s 写败（消费腿晚拍拿——会话本体不炸）: %v", key, err)
	}
}

// sharedAttach 共享会话复挂（消费腿/OpenSession 共件）：读行→dial→
// SetSharedToken→CheckAlive；活=返 client（shared 标置——调用侧 Disconnect
// 幂等空转不伤共享会话），死=关 conn 返 err 等 owner 重建。消费腿永不 ConnectEx
// （owner 独占 login 权——双 login=跨腿互撞坑）。key=按户共享行键。
func sharedAttach(ctx context.Context, pool *pgxpool.Pool, dial func(string) (*Client, error), key string) (*Client, error) {
	ss, err := sharedSessionRead(ctx, pool, key)
	if err != nil {
		return nil, fmt.Errorf("共享会话未立（等汇市巡建 key=%s）: %w", key, err)
	}
	cl, err := dial(ss.Addr)
	if err != nil {
		return nil, fmt.Errorf("共享会话 dial %s: %w", ss.Addr, err)
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	err = func() error {
		defer cancel()
		cl.SetSharedToken(ss.Token)
		return cl.CheckAlive(cctx)
	}()
	if err != nil {
		cl.Close()
		return nil, fmt.Errorf("共享会话已死（等汇市巡重建 key=%s）: %w", key, err)
	}
	return cl, nil
}
