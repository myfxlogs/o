// universe.go —— 宇宙扩件·符号目录供给件（T-0416）：okx=/public/instruments
// ?instType=SWAP（REST 一次全谱目录枚举）实读+TTL 常驻缓存——一处供给多处
// 消费（collector 订阅扩件 ExpandSymbols/粗扫臂谱域/hunger 探针订阅宇宙/
// 评估臂提名 catalog 闸——同一供给件禁第二套归一，装配见 cmd/core+
// cmd/collector+cmd/research）。票面钦定语义：「全谱」=目录全集非订阅集；
// 目录读败=上抛 fail-closed 不虚构标的集（探针哲学不动——供给件真存在，
// 败情照样升件）。
package okxws

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// UniverseTTL 目录常驻缓存默认 TTL——venue 目录慢变面（新挂/下市分钟级
// 滞无虞），TTL 内命中零重读防 venue 打爆；装配层可另钉。
const UniverseTTL = 10 * time.Minute

// datedFutRe 季度到期期货形（'-' 尾 6 位到期日——T-0189 返修钉：OKX 实挂
// 全 -USD- 币本位，-USDT- 保证金判=空集误诊）。
var datedFutRe = regexp.MustCompile(`-\d{6}$`)

// UniverseCatalog 宇宙目录供给件（TTL 常驻缓存——refresh 败情上抛不污旧
// 账，fail-closed 归调用面自决）。
type UniverseCatalog struct {
	get  ScanGet // REST 公开读件（装配层挂 okxexec.PublicGet/authGetter.Get）
	ttl  time.Duration
	mu   sync.Mutex
	at   time.Time
	swap []string // live -USDT-SWAP 目录全集（粗扫谱域）
	all  []string // swap∪基币现货∪季度期货∪uly 指数键（订阅宇宙）
}

// NewUniverseCatalog 装配（get=venue REST 公开读件；ttl<=0=UniverseTTL）。
func NewUniverseCatalog(get ScanGet, ttl time.Duration) *UniverseCatalog {
	if ttl <= 0 {
		ttl = UniverseTTL
	}
	return &UniverseCatalog{get: get, ttl: ttl}
}

// Symbols 订阅宇宙全谱（collector ExpandSymbols 闭包语义同形——原
// universeExpander 出口面归本件唯一真源）。
func (u *UniverseCatalog) Symbols(ctx context.Context) ([]string, error) {
	if err := u.refresh(ctx); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]string(nil), u.all...), nil
}

// Set 订阅宇宙 map 形（探针∩判据/提名存在性闸同读）。
func (u *UniverseCatalog) Set(ctx context.Context) (map[string]bool, error) {
	if err := u.refresh(ctx); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	m := make(map[string]bool, len(u.all))
	for _, s := range u.all {
		m[s] = true
	}
	return m, nil
}

// SwapSet 粗扫谱域=SWAP 目录全集（live -USDT-SWAP——md_tick_scan 票面钦定
// 「全谱」语义=目录全集）。
func (u *UniverseCatalog) SwapSet(ctx context.Context) (map[string]bool, error) {
	if err := u.refresh(ctx); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	m := make(map[string]bool, len(u.swap))
	for _, s := range u.swap {
		m[s] = true
	}
	return m, nil
}

// refresh TTL 内命中=直返；过期→SWAP 实读（败=上抛不污旧账——fail-closed）
// →FUTURES 季度面败=降级不连带让位（SWAP 全集已得——universeExpander 原
// 语义同守）。锁内取件串行化防 TTL 边界并发双打。
func (u *UniverseCatalog) refresh(ctx context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.all != nil && time.Since(u.at) < u.ttl {
		return nil
	}
	raw, err := u.get(ctx, "/api/v5/public/instruments?instType=SWAP")
	if err != nil {
		return err
	}
	var sw []struct {
		InstID string `json:"instId"`
		State  string `json:"state"`
	}
	if err := json.Unmarshal(raw, &sw); err != nil {
		return err
	}
	var swap, all []string
	seenSpot := map[string]bool{}
	for _, i := range sw {
		if i.State != "live" || !strings.HasSuffix(i.InstID, "-USDT-SWAP") {
			continue
		}
		swap = append(swap, i.InstID)
		all = append(all, i.InstID)
		// T-0129 根修沿用：宇宙放开须含现货腿面——SWAP 基币现货对同入
		// 订阅集（对冲腿现价源非指数价——index-tickers 不可代）。
		base := strings.TrimSuffix(i.InstID, "-SWAP")
		if !seenSpot[base] {
			seenSpot[base] = true
			all = append(all, base)
		}
	}
	if len(swap) == 0 {
		return fmt.Errorf("instruments SWAP live 零行——目录面病灶 fail-closed")
	}
	if raw, err = u.get(ctx, "/api/v5/public/instruments?instType=FUTURES"); err == nil {
		var fut []struct {
			InstID string `json:"instId"`
			Alias  string `json:"alias"`
			Uly    string `json:"uly"`
			State  string `json:"state"`
		}
		if err := json.Unmarshal(raw, &fut); err == nil {
			for _, i := range fut {
				// T-0189 返修沿用：quarter 到期期货全收+uly 剥离键并入集
				//（index-tickers USD 指数行覆盖——basisAnn JOIN 键）。
				if i.State == "live" && i.Alias == "quarter" && datedFutRe.MatchString(i.InstID) {
					all = append(all, i.InstID)
					all = append(all, i.Uly)
				}
			}
		}
	}
	u.swap, u.all, u.at = swap, all, time.Now()
	return nil
}
