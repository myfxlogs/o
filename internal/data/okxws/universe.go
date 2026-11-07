// universe.go —— 宇宙扩件·符号目录供给件（T-0416）：okx=/public/instruments
// 实读+TTL 常驻缓存——一处供给多处消费（collector 订阅扩件 ExpandSymbols/
// 粗扫臂谱域/hunger 探针订阅宇宙/评估臂提名 catalog 闸——同一供给件禁第二
// 套归一，装配见 cmd/core+cmd/collector+cmd/research）。票面钦定语义：
// 「全谱」=目录全集非订阅集；目录读败=上抛 fail-closed 不虚构标的集
// （探针哲学不动——供给件真存在，败情照样升件）。
// T-0427 粗扫扩全宇宙：目录 instType 参数化三族（SWAP/SPOT/FUTURES——
// 同供给件同 TTL 同 fail-closed；OPTION/MARGIN 票面不收——无消费面）。
// 族读败=族级记账（FamSet 该族上抛让位，Symbols 面降级不连带——粗扫
// fail-closed 粒度到族的供给侧同构）；FamSet 即 T-0426 仿真层未来语义径
// 的目录接口位（/public/instruments 实读——按族取子集）。
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
	swap []string // live -USDT-SWAP 目录全集（粗扫 SWAP 族谱域）
	spot []string // live -USDT 现货目录全集（T-0427 SPOT 族谱域——USDC 杂币键域滤除同律）
	fut  []string // live quarter 期货目录全集（T-0427 FUTURES 族谱域——沿用 T-0189 quarter 形）
	all  []string // swap∪现货族∪基币现货派生∪quarter 期货∪uly 指数键（订阅宇宙）
	// famErr 本轮族读败记账（T-0427 族级 fail-closed——FamSet 该族上抛
	// 让位不虚构空集；Symbols 面降级不连带照供；下轮 refresh 全量重记）。
	famErr map[string]error
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

// SwapSet 粗扫 SWAP 族谱域=live -USDT-SWAP 目录全集（md_tick_scan 原谱域
// 语义——T-0427 起粗扫谱域按族扩全，本出口保留兼容面）。
func (u *UniverseCatalog) SwapSet(ctx context.Context) (map[string]bool, error) {
	return u.FamSet("SWAP")(ctx)
}

// FamSet 族谱域闸集（T-0427——instType ∈ SWAP/SPOT/FUTURES 扫册三族；
// 该族目录读败/零 live=族错上抛，调用面该族让位不虚构，他族照常由扫臂
// 族循环保证——fail-closed 粒度到族；扫册外族 fail-closed 拒）。闭包形
// 同 SwapSet 装配惯例——粗扫 Gates 逐族挂线+T-0426 仿真层未来语义径
// 目录接口位（/public/instruments 实读按族取子集）。
func (u *UniverseCatalog) FamSet(instType string) func(context.Context) (map[string]bool, error) {
	return func(ctx context.Context) (map[string]bool, error) {
		if err := u.refresh(ctx); err != nil {
			return nil, err
		}
		u.mu.Lock()
		defer u.mu.Unlock()
		if err := u.famErr[instType]; err != nil {
			return nil, fmt.Errorf("目录 %s 族读败——该族 fail-closed 让位: %w", instType, err)
		}
		var list []string
		switch instType {
		case "SWAP":
			list = u.swap
		case "SPOT":
			list = u.spot
		case "FUTURES":
			list = u.fut
		default:
			return nil, fmt.Errorf("族 %s 非扫册（SWAP/SPOT/FUTURES 外 fail-closed 拒）", instType)
		}
		m := make(map[string]bool, len(list))
		for _, s := range list {
			m[s] = true
		}
		return m, nil
	}
}

// refresh TTL 内命中=直返；过期→三族实读（SWAP 锚族败=上抛不污旧账
// ——fail-closed；SPOT/FUTURES 族败=族级记账降级不连带——Symbols 照供、
// FamSet 该族上抛）。锁内取件串行化防 TTL 边界并发双打。
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
	var swap, spot, fut, all []string
	famErr := map[string]error{}
	seenSpot := map[string]bool{}
	for _, i := range sw {
		if i.State != "live" || !strings.HasSuffix(i.InstID, "-USDT-SWAP") {
			continue
		}
		swap = append(swap, i.InstID)
		all = append(all, i.InstID)
		// T-0129 根修沿用：宇宙放开须含现货腿面——SWAP 基币现货对同入
		// 订阅集（对冲腿现价源非指数价——index-tickers 不可代）。T-0427
		// SPOT 族并集为主、本派生臂保留兜底（SPOT 族读败日基币现货腿
		// 不失明）。
		base := strings.TrimSuffix(i.InstID, "-SWAP")
		if !seenSpot[base] {
			seenSpot[base] = true
			all = append(all, base)
		}
	}
	if len(swap) == 0 {
		return fmt.Errorf("instruments SWAP live 零行——目录面病灶 fail-closed")
	}
	// SPOT 族（T-0427）：live -USDT 对全集——quote 键域律同全域（杂币不采）。
	raw, err = u.get(ctx, "/api/v5/public/instruments?instType=SPOT")
	if err != nil {
		famErr["SPOT"] = err
	} else {
		var sp []struct {
			InstID string `json:"instId"`
			State  string `json:"state"`
		}
		if err := json.Unmarshal(raw, &sp); err != nil {
			famErr["SPOT"] = err
		} else {
			for _, i := range sp {
				if i.State == "live" && strings.HasSuffix(i.InstID, "-USDT") {
					spot = append(spot, i.InstID)
					all = append(all, i.InstID)
				}
			}
			if len(spot) == 0 {
				famErr["SPOT"] = fmt.Errorf("instruments SPOT live 零行——族面病灶 fail-closed")
			}
		}
	}
	// FUTURES 族（T-0189 形沿用）：quarter 到期期货全收+uly 剥离键并入集
	//（index-tickers USD 指数行覆盖——basisAnn JOIN 键）；族败=记账降级
	// 不连带（SWAP/SPOT 全集已得——universeExpander 原语义同守）。
	raw, err = u.get(ctx, "/api/v5/public/instruments?instType=FUTURES")
	if err != nil {
		famErr["FUTURES"] = err
	} else {
		var futResp []struct {
			InstID string `json:"instId"`
			Alias  string `json:"alias"`
			Uly    string `json:"uly"`
			State  string `json:"state"`
		}
		if err := json.Unmarshal(raw, &futResp); err != nil {
			famErr["FUTURES"] = err
		} else {
			for _, i := range futResp {
				if i.State == "live" && i.Alias == "quarter" && datedFutRe.MatchString(i.InstID) {
					fut = append(fut, i.InstID)
					all = append(all, i.InstID)
					all = append(all, i.Uly)
				}
			}
			if len(fut) == 0 {
				famErr["FUTURES"] = fmt.Errorf("instruments FUTURES quarter live 零行——族面病灶 fail-closed")
			}
		}
	}
	u.swap, u.spot, u.fut, u.all, u.famErr, u.at = swap, spot, fut, all, famErr, time.Now()
	return nil
}
