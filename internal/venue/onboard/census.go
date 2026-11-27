// census.go —— O-41378 件2：fin_venue_symbols 普查拍——venue 凭据在+会话通
// →SymbolList 实拉→逐符 SymbolSpecFacts 全谱（probePace 同尺限速）→upsert
// 行+last_seen 重涂；下架符=last_seen 止涂自然陈化。配对=表上纯函数
// （census×registry——拉数与判数分拍，本拍只拉数落库零判词）。
// family_guess=机分类参数分拣（业主令②：人工分不清合约只能错配——
// 名形族判只收窄 registry 别名匹配域，落闸判据=参数对拍一件，mint.go）。
package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"o/internal/mt5"
)

// censusSession 普查/mint 会话窄缝（*mt5.Client 满足；桩测注假——禁桩真
// venue 同律；proto 型不出缝——SpecFacts typed struct 纯字段，D-514 守）。
type censusSession interface {
	Connect(ctx context.Context, cr mt5.Credentials) error
	SymbolList(ctx context.Context) ([]string, error)
	SymbolSpecFacts(ctx context.Context, sym string) (mt5.SpecFacts, error)
	GetQuoteOne(ctx context.Context, sym string) (mt5.QuoteSnapshot, error)
	Disconnect(ctx context.Context)
	Close()
}

// censusEvery 普查重扫间隔（listing 增面慢变量——6h 节流；首见=零行必跑；
// var=桩测覆写锚）。censusBudget 单户拍死线（网关/venue IO 帽——逐符
// probePace 步进内自然兜底，177 符≈21s 远低于帽）。
var (
	censusEvery  = 6 * time.Hour
	censusBudget = 10 * time.Minute
)

// censusPass 一拍普查：凭据在册 mt5 族逐户——节流后全符扫落 fin_venue_symbols。
// 败=日志留痕不上抛（监测面同 probePass 律——不阻主链）。
func (r *Runner) censusPass(ctx context.Context, brokers map[string]credFamily) {
	bases := r.cryptoBases(ctx)
	for _, b := range brokers {
		if b.missing || (b.kind != "" && b.kind != "mt5") {
			continue
		}
		if b.fields["login"] == "" {
			continue // 多账户粒度键缺——无绑账户无普查面
		}
		fresh, err := r.censusFresh(ctx, b)
		if err != nil {
			log.Printf("WARN [onboard] %s 普查节流读败: %v", b.venue, err)
			continue
		}
		if fresh {
			continue // max(last_seen) 新鲜<censusEvery——listing 慢变量节流
		}
		if err := r.censusVenue(ctx, b, bases); err != nil {
			log.Printf("WARN [onboard] %s 普查拍败: %v", b.venue, err)
		}
	}
}

// censusFresh 本户普查新鲜判（max(last_seen) 距今<censusEvery=跳；零行
// =false 必跑——首见/新户不等节流）。
func (r *Runner) censusFresh(ctx context.Context, b credFamily) (bool, error) {
	var last *time.Time
	err := r.fin.QueryRow(ctx, `SELECT max(last_seen) FROM fin_venue_symbols
		WHERE venue=$1 AND account=$2`, b.venue, b.fields["login"]).Scan(&last)
	if err != nil {
		return false, err
	}
	return last != nil && r.now().Sub(*last) < censusEvery, nil
}

// censusVenue 单户普查拍：会话→SymbolList→逐符 SpecFacts→upsert
// （map_state/map_reason/canonical_ref/spread_live 列 census 永不触——
// 配对判词归 mint 拍，本拍只刷新自报事实面）。
func (r *Runner) censusVenue(ctx context.Context, b credFamily, cryptoBases map[string]bool) error {
	sess, err := r.dialCensus(ctx, b)
	if err != nil {
		return fmt.Errorf("拨号: %w", err)
	}
	defer sess.Close()
	syms, err := sess.SymbolList(ctx)
	if err != nil {
		return fmt.Errorf("SymbolList: %w", err)
	}
	var upserted, specFail int
	for i, sym := range syms {
		if i > 0 && probePace > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(probePace): // sched-exempt-ok：逐符限速步进（限频面守——probePace 同尺）
			}
		}
		base, suffix := mt5.SplitDialect(sym)
		fam := classifyFamily(base, suffix, cryptoBases)
		f, err := sess.SymbolSpecFacts(ctx, sym)
		if err != nil {
			specFail++
			r.upsertCensusBare(ctx, b, sym, base, suffix, fam) // venue 在册未认——记名+分类，参数列留旧/NULL
			continue
		}
		params, _ := json.Marshal(f)
		if err := r.upsertCensus(ctx, b, sym, base, suffix, fam, f, params); err != nil {
			return fmt.Errorf("upsert %s: %w", sym, err)
		}
		upserted++
	}
	log.Printf("[onboard] %s 普查落账 syms=%d spec_fail=%d", b.venue, upserted, specFail)
	return nil
}

// upsertCensus 全谱行写（冲突更新参数面+last_seen——map_state 判词列不触）。
func (r *Runner) upsertCensus(ctx context.Context, b credFamily, sym, base, suffix, fam string,
	f mt5.SpecFacts, params []byte) error {
	_, err := r.fin.Exec(ctx, `INSERT INTO fin_venue_symbols(venue,account,venue_symbol,
		base_guess,suffix_guess,family_guess,ct_val,tick_sz,min_sz,lot_step,digits,
		trade_mode,profit_ccy,margin_ccy,spec_hash,params)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (venue,account,venue_symbol) DO UPDATE SET
		base_guess=EXCLUDED.base_guess,suffix_guess=EXCLUDED.suffix_guess,
		family_guess=EXCLUDED.family_guess,ct_val=EXCLUDED.ct_val,tick_sz=EXCLUDED.tick_sz,
		min_sz=EXCLUDED.min_sz,lot_step=EXCLUDED.lot_step,digits=EXCLUDED.digits,
		trade_mode=EXCLUDED.trade_mode,profit_ccy=EXCLUDED.profit_ccy,
		margin_ccy=EXCLUDED.margin_ccy,spec_hash=EXCLUDED.spec_hash,params=EXCLUDED.params,
		last_seen=now()`,
		b.venue, b.fields["login"], sym, base, suffix, fam,
		f.CtVal, f.TickSz, f.MinSz, f.LotStep, f.Digits, f.TradeMode,
		f.ProfitCcy, f.MarginCcy, f.SpecHash, params)
	return err
}

// upsertCensusBare spec 败行（venue 自报在册但 SymbolParams 未认——记名+
// 剥件+分类，参数列新行 NULL/旧行保留；last_seen 重涂=仍在自报表）。
func (r *Runner) upsertCensusBare(ctx context.Context, b credFamily, sym, base, suffix, fam string) {
	_, err := r.fin.Exec(ctx, `INSERT INTO fin_venue_symbols(venue,account,venue_symbol,
		base_guess,suffix_guess,family_guess) VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT (venue,account,venue_symbol) DO UPDATE SET
		base_guess=EXCLUDED.base_guess,suffix_guess=EXCLUDED.suffix_guess,
		family_guess=EXCLUDED.family_guess,last_seen=now()`,
		b.venue, b.fields["login"], sym, base, suffix, fam)
	if err != nil {
		log.Printf("WARN [onboard] %s 普查 bare upsert 败 %s: %v", b.venue, sym, err)
	}
}

// dialCensus 普查/mint 会话拨号（凭据族实测——r.dialCensusFn 缝 nil=mt5.Dial
// 生产径；censusBudget 死线——同 probeBudget 律）。
func (r *Runner) dialCensus(ctx context.Context, b credFamily) (censusSession, error) {
	addr, err := r.cfgStr(ctx, "mt5.gateway_addr")
	if err != nil {
		return nil, fmt.Errorf("mt5.gateway_addr: %w", err)
	}
	dial := r.dialCensusFn
	if dial == nil {
		dial = func(addr string) (censusSession, error) { return mt5.Dial(addr) }
	}
	sess, err := dial(addr)
	if err != nil {
		return nil, err
	}
	login, _ := strconv.ParseUint(b.fields["login"], 10, 64)
	cctx, cancel := context.WithTimeout(ctx, censusBudget)
	defer cancel()
	if err := sess.Connect(cctx, mt5.Credentials{
		Login:    login,
		Password: b.fields["password"],
		Server:   b.fields["server"],
		Host:     b.fields["host"]}); err != nil {
		sess.Close()
		return nil, fmt.Errorf("connect: %w", err)
	}
	return sess, nil
}

// cryptoBases OKX 宇宙 base 码集（universeFn 缝实读——BTC/ETH/DOGE…动态取，
// 新币上线自动入族；缝缺=空集 crypto 族判不出归 other 不猜）。
func (r *Runner) cryptoBases(ctx context.Context) map[string]bool {
	out := map[string]bool{}
	if r.universeFn == nil {
		return out
	}
	insts, err := r.universeFn(ctx)
	if err != nil {
		return out
	}
	for _, inst := range insts {
		out[strings.ToUpper(strings.SplitN(inst, "-", 2)[0])] = true
	}
	return out
}

// ── family_guess 机分类（参数分拣不靠名为业主令锚——名形族判仅 index/
// energy/future 三族，其余全参数/码谱判；判不出=other 不猜。family_guess
// 只收窄 registry 别名匹配域，不做落闸判据——落闸=参数对拍一件 mint.go）──

// fiatCode 法币码谱（fx 族判据——base+quote 双侧皆法币=外汇对形）。
var fiatCode = map[string]bool{
	"USD": true, "EUR": true, "GBP": true, "JPY": true, "CNH": true,
	"AUD": true, "CAD": true, "CHF": true, "DKK": true, "HKD": true,
	"HUF": true, "MXN": true, "NOK": true, "NZD": true, "PLN": true,
	"SEK": true, "SGD": true, "TRY": true, "ZAR": true, "ILS": true,
	"RUB": true, "CZK": true, "THB": true, "INR": true, "KRW": true,
}

// metalPrefix 贵金属族名前缀（XAU/XAG/GAU/XPT/XPD+GOLD/SILVER 单名形——
// profit_ccy 无金本位码可依，名形族判与 index/energy 同律收窄用）。
var metalPrefix = []string{"XAU", "XAG", "GAU", "XPT", "XPD", "GOLD", "SILVER"}

// energyShape 商品码形（指数/能源 CFD 名形族判——OIL/NGAS/农软金属码）。
func energyShape(u string) bool {
	for _, k := range []string{"OIL", "NGAS", "BRENT", "GASOIL", "COCOA", "COFFE",
		"CORN", "COTTO", "SUGAR", "WHEAT", "SOYBEAN", "HGCOP", "NICKEL", "ZINC", "ALUM"} {
		if strings.Contains(u, k) {
			return true
		}
	}
	return false
}

// indexShape 指数码形（Cash#/指数名形——US100Cash#/DE30Cash# 族）。
func indexShape(u string) bool {
	for _, k := range []string{"CASH", "INDX", "INDEX", "SPX", "NDX", "DJI", "US30", "US500", "GER", "JP225", "STOXX", "FTSE", "HSI"} {
		if strings.Contains(u, k) {
			return true
		}
	}
	return false
}

// futureShape 期约形（-MONYY 尾——OIL-JAN27/US500-MAR26 族；月份码硬谱判）。
func futureShape(b string) bool {
	i := strings.LastIndex(b, "-")
	if i < 0 || len(b)-i-1 != 5 {
		return false
	}
	tail := b[i+1:]
	mon := tail[:3]
	switch mon {
	case "JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC":
	default:
		return false
	}
	for _, c := range tail[3:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// classifyFamily 族分类核（判序：fx 双法币→future 期约形→metal→crypto→
// index→energy→other；收窄用非落闸——错分只影响别名匹配域不影响判词）。
func classifyFamily(base, suffix string, cryptoBases map[string]bool) string {
	b := strings.ToUpper(strings.TrimSpace(base))
	s := strings.TrimSuffix(strings.ToUpper(strings.TrimSpace(suffix)), "M") // m尾=mini 方言标剥（USDM→USD）
	if b == "" {
		return "other"
	}
	if fiatCode[b] && s != "" && fiatCode[s] {
		return "fx"
	}
	if futureShape(b) {
		return "future_cfd"
	}
	for _, p := range metalPrefix {
		if strings.HasPrefix(b, p) {
			return "metal"
		}
	}
	if cryptoBases[b] {
		return "crypto"
	}
	u := b + s
	if indexShape(u) {
		return "index"
	}
	if energyShape(u) {
		return "energy"
	}
	return "other"
}
