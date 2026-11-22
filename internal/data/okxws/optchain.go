// optchain.go —— OKX 期权链采集面（工单#32540——covered-call 研究
// #32274 数据前置：venue 不补历史延一天缺一天）。频道普查实证
// （2026-11-23 真连）：opt-summary WS 频道须 instFamily 键（instId
// 形无 ack 拒）增量推全链希腊字；tickers/mark-price/books5 对
// OPTION instId 全受理；instruments {instType:OPTION} 受理仅态变推。
// 四族=BTC/ETH×{USD 反向币本位,USD_UM 线性 USDT 保证金}——instFamily
// 枚举实读（instruments 须 instFamily∪uly 二选一，裸 instType=50015）。
// 落点纪律：加法件——新表域 md_opt_*（md_1101）+新频道 opt-summary/
// 期权形 tickers，既有频道语义/既存表零碰（票 ban 钉）。
package okxws

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// OptTTL 期权目录缓存 TTL（合约目录慢变面——新挂/到期小时级，
	// 6h 内命中零重读防 venue 打爆；同 UniverseTTL 量级放宽一档）。
	OptTTL = 6 * time.Hour
	// optTickExpiries 活跃链到期窗=最近 N 个不同到期日（covered-call
	// 研究域=近月链——远月希腊字近零边际）。
	optTickExpiries = 3
	// optTickDays 到期日上限（超出窗外不选——陈旧远月不配 tickers 带宽）。
	optTickDays = 60
	// optTickRange 行权价窗=现价 ±20%（ATM 域——深度虚值/实值
	// delta≈0/1 边际零，窗口外留 opt-summary 全链覆盖不丢）。
	optTickRange = 0.20
	// optTickMaxFam 单族活跃集上限（订阅预算闸——4 族×200=800 args
	// ≈8 帧，480 ops/h 重订预算内安全余量）。
	optTickMaxFam = 200
	// optSumMinInterval opt-summary per-inst 写节流（增量推 ~490 行/s/
	// 族洪——最新态 upsert 语义，60s 鲜度足够证据窗；粗粒度起）。
	optSumMinInterval = 60 * time.Second
	// optSnapEvery 周期快照拍距（活跃子集全量追加 md_opt_snapshot——
	// PG 体积纪律：60min×~800 行≈1.2M 行/90d，容量器官域可再降频）。
	optSnapEvery = time.Hour
)

// OptFamilies 期权族全集（普查实证四族全 live——BTC/ETH 主线×
// 币本位反向/USDT 线性两保证金形；alt 族后扩=加一行常量）。
var OptFamilies = []string{"BTC-USD", "BTC-USD_UM", "ETH-USD", "ETH-USD_UM"}

// optInstRe 期权 instId 形（<family>-<YYMMDD>-<strike>-<C|P>——family
// 自身含 '-' 故贪婪前缀+尾三段钉；SWAP/-\d{6}$ 期货/现货形零误中）。
var optInstRe = regexp.MustCompile(`^(.+)-(\d{6})-([0-9.]+)-([CP])$`)

// isOptInstID 期权合约键域判（订阅闸/帧分流同判据——非 USDT 计价
// 但系钦定采集域，IsUSDTInst 闸前须先放行本形）。
func isOptInstID(instID string) bool { return optInstRe.MatchString(instID) }

// OptInstrument 期权合约目录行（instruments 实读字段子集——enum
// 快照落 md_opt_instruments 全态；活链判据只取 state=live）。
type OptInstrument struct {
	InstID     string
	InstFamily string
	Uly        string
	OptType    string // C/P
	ExpTime    time.Time
	Strike     float64
	CtType     string // inverse/linear
	SettleCcy  string
	State      string
	TickSz     float64
	LotSz      float64
	MinSz      float64
	CtVal      float64
	CtValCcy   string
	ListTime   time.Time
}

// OptChain 期权目录供给件（TTL 常驻缓存——Refresh 败情上抛不污旧账，
// fail-closed 归调用面；活跃子集=近 N 到期日×现价±20%窗×族帽）。
type OptChain struct {
	get  ScanGet
	ttl  time.Duration
	mu   sync.Mutex
	at   time.Time
	inst map[string][]OptInstrument // family→全态目录行
	live map[string][]OptInstrument // family→live 行（活链判据集）
	act  []string                   // 活跃子集 instId（tickers 订阅集）
}

// NewOptChain 装配（get=REST 公开读件同 UniverseCatalog 先例；
// ttl<=0=OptTTL）。
func NewOptChain(get ScanGet, ttl time.Duration) *OptChain {
	if ttl <= 0 {
		ttl = OptTTL
	}
	return &OptChain{get: get, ttl: ttl}
}

// Active 活跃子集快照（tickers 订阅集+快照臂判集——拷贝防外改）。
func (o *OptChain) Active() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.act...)
}

// AllInstruments 全族目录行平铺（enum 快照 upsert 源——全态非仅 live）。
func (o *OptChain) AllInstruments() []OptInstrument {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []OptInstrument
	for _, fam := range OptFamilies {
		out = append(out, o.inst[fam]...)
	}
	return out
}

// Refresh TTL 内命中=直返 fresh=false；过期→四族 instruments 实读+
// 指数参照价实读+活跃子集重选。任一族读败/零 live=上抛 fail-closed
// （不虚构目录——同 UniverseCatalog 供给件哲学）。
func (o *OptChain) Refresh(ctx context.Context) (bool, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.get == nil {
		return false, fmt.Errorf("opt get 未挂——供给件缺席 fail-closed")
	}
	if o.inst != nil && time.Since(o.at) < o.ttl {
		return false, nil
	}
	inst := map[string][]OptInstrument{}
	live := map[string][]OptInstrument{}
	for _, fam := range OptFamilies {
		raw, err := o.get(ctx, "/api/v5/public/instruments?instType=OPTION&instFamily="+fam)
		if err != nil {
			return false, fmt.Errorf("opt instruments %s 读败: %w", fam, err)
		}
		var rows []struct {
			InstID     string `json:"instId"`
			InstFamily string `json:"instFamily"`
			Uly        string `json:"uly"`
			OptType    string `json:"optType"`
			ExpTime    string `json:"expTime"`
			Stk        string `json:"stk"`
			CtType     string `json:"ctType"`
			SettleCcy  string `json:"settleCcy"`
			State      string `json:"state"`
			TickSz     string `json:"tickSz"`
			LotSz      string `json:"lotSz"`
			MinSz      string `json:"minSz"`
			CtVal      string `json:"ctVal"`
			CtValCcy   string `json:"ctValCcy"`
			ListTime   string `json:"listTime"`
		}
		if err := json.Unmarshal(raw, &rows); err != nil {
			return false, fmt.Errorf("opt instruments %s 解析败: %w", fam, err)
		}
		for _, r := range rows {
			in := OptInstrument{
				InstID: r.InstID, InstFamily: r.InstFamily, Uly: r.Uly,
				OptType: r.OptType, CtType: r.CtType, SettleCcy: r.SettleCcy,
				State: r.State, CtValCcy: r.CtValCcy,
				ExpTime: optMs(r.ExpTime), ListTime: optMs(r.ListTime),
				Strike: optF(r.Stk), TickSz: optF(r.TickSz),
				LotSz: optF(r.LotSz), MinSz: optF(r.MinSz), CtVal: optF(r.CtVal),
			}
			inst[fam] = append(inst[fam], in)
			if in.State == "live" {
				live[fam] = append(live[fam], in)
			}
		}
		if len(live[fam]) == 0 {
			return false, fmt.Errorf("opt instruments %s live 零行——族面病灶 fail-closed", fam)
		}
	}
	// 指数参照价（活跃窗锚——/public/index-tickers 实读；族→指数名
	// =instFamily 剥 _UM 尾：BTC-USD_UM→BTC-USD、ETH-USD_UM→ETH-USD，
	// 普查实证 uly 同锚）。读败=参照价缺席该族不选不虚构。
	ref := map[string]float64{}
	for _, fam := range OptFamilies {
		idx := strings.TrimSuffix(fam, "_UM")
		if _, ok := ref[idx]; ok {
			continue
		}
		raw, err := o.get(ctx, "/api/v5/market/index-tickers?instId="+idx)
		if err != nil {
			return false, fmt.Errorf("opt 参照价 %s 读败: %w", idx, err)
		}
		var rows []struct {
			IdxPx string `json:"idxPx"`
		}
		if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
			return false, fmt.Errorf("opt 参照价 %s 解析败/零行", idx)
		}
		ref[idx] = optF(rows[0].IdxPx)
	}
	o.inst, o.live, o.act, o.at = inst, live, selectOptActive(live, ref), time.Now()
	return true, nil
}

// selectOptActive 活跃子集判据（近 optTickExpiries 个不同到期日∧
// 现价±optTickRange 行权价窗∧族帽 optTickMaxFam——确定性排序：
// 到期日升序→|行权价-参照价|升序截帽；参照价缺席族=不选 fail-closed）。
func selectOptActive(live map[string][]OptInstrument, ref map[string]float64) []string {
	var out []string
	for _, fam := range OptFamilies {
		px := ref[strings.TrimSuffix(fam, "_UM")]
		if px <= 0 {
			continue
		}
		// 到期日分桶（expTime→UTC 日）——最近 N 个不同日。
		byDay := map[string][]OptInstrument{}
		var days []string
		for _, in := range live[fam] {
			if in.ExpTime.IsZero() {
				continue
			}
			day := in.ExpTime.UTC().Format("20060102")
			if _, ok := byDay[day]; !ok {
				days = append(days, day)
			}
			byDay[day] = append(byDay[day], in)
		}
		sort.Strings(days) // days 天然去重——升序首见=最近到期
		cutoff := time.Now().UTC().Add(optTickDays * 24 * time.Hour)
		var sel []OptInstrument
		for i, d := range days {
			if i >= optTickExpiries {
				break // 最近 N 个到期日封顶
			}
			dayT, _ := time.Parse("20060102", d)
			if dayT.After(cutoff) {
				break // 窗外到期日不选
			}
			sel = append(sel, byDay[d]...)
		}
		// 行权价窗过滤+距离升序截帽。
		var near []OptInstrument
		for _, in := range sel {
			if in.Strike > 0 && absF(in.Strike-px)/px <= optTickRange {
				near = append(near, in)
			}
		}
		sort.Slice(near, func(i, j int) bool {
			return absF(near[i].Strike-px) < absF(near[j].Strike-px)
		})
		if len(near) > optTickMaxFam {
			near = near[:optTickMaxFam]
		}
		for _, in := range near {
			out = append(out, in.InstID)
		}
	}
	sort.Strings(out)
	return out
}

func absF(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func optF(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func optMs(s string) time.Time {
	ms, _ := strconv.ParseInt(s, 10, 64)
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// SubscribeArgs 期权订阅面（opt-summary×4 族 instFamily 键 +
// tickers×活跃子集 instId 键——分帧节流走 subscribeAll 同律）。
func (o *OptChain) SubscribeArgs() []map[string]string {
	o.mu.Lock()
	defer o.mu.Unlock()
	var args []map[string]string
	for _, fam := range OptFamilies {
		args = append(args, map[string]string{"channel": "opt-summary", "instFamily": fam})
	}
	for _, id := range o.act {
		args = append(args, map[string]string{"channel": "tickers", "instId": id})
	}
	return args
}
