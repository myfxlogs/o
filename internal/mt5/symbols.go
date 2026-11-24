// symbols.go —— 按户订阅清单三源序（O-37448 ⑥类型化承重点）：
//  1. <cred_ns>.symbols 显式册（app_config——运营钦定优先）；
//  2. fin_symbol_dialect 该 (venue,account=login) venue_symbol 行——映射过=
//     确认符=订阅集（向导落行即采集自来——"加户零码"承重件：方言硬验行一落
//     订阅自动跟，无需任何配置）；
//  3. SymbolConfirm 全谱候选兜底（固定谱∪SymbolList 自报名 CFD 主流筛——
//     XM BTCUSD# 族零码命中，病灶「现候选只含 Exness 后缀 XM 零命中=采集死」
//     根修面）。
package mt5

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"
)

// symbolCandidates 全谱 CFD 主流候选（O-37448 扩谱——Exness m 尾实拍+
// #/.//+ 尾方言+裸名全族；SymbolConfirm venue 实认后启用，未认候选弃）。
const symbolCandidates = "BTCUSDm,ETHUSDm,XAUUSDm,XAGUSDm," +
	"BTCUSD#,ETHUSD#,XAUUSD#,XAGUSD#," +
	"BTCUSD,ETHUSD,XAUUSD,XAGUSD," +
	"BTCUSD.,ETHUSD.,BTCUSD+,ETHUSD+"

// cfdMajorsRe SymbolList 自报名 CFD 主流族筛（venue 实读名零猜——BTC/ETH/
// XAU/XAG 基底+报价段+方言尾全谱：m/#/./+ 尾、裸名、远月变体 .Z25 形；
// 匹配名原样回 candidate——存在性仍 SymbolConfirm venue 判）。
var cfdMajorsRe = regexp.MustCompile(`^(?i)(BTC|ETH|XAU|XAG)[A-Z0-9]{2,6}([.#+][A-Z0-9]{0,4})?$`)

// symbols 按户订阅清单三源序取集（spec ⑥钦定序——任一源非空即返，
// 全源空=SymbolConfirm 空集由 session() 上抛显形）。
// 显式册通配语义：清单含 "*" 元素=全 SymbolList 订阅（业主钦定面——
// 新品自跟不列死名；采集面只读零授权含义，执行面仍方言行 fail-closed）。
func (c *Collector) symbols(ctx context.Context, cl *Client, a *acctRun) []string {
	if raw, err := c.cfg.Get(a.acct.SymbolsKey()); err == nil {
		var syms []string
		if json.Unmarshal(raw, &syms) == nil && len(syms) > 0 {
			for _, s := range syms {
				if s == "*" {
					all, err := cl.SymbolList(ctx)
					if err != nil || len(all) == 0 {
						log.Printf("[mt5] %s 通配册 SymbolList 败（%v）——落三源序余径", a.acct.Venue, err)
						break
					}
					log.Printf("[mt5] %s 订阅集=通配全谱 %d 符", a.acct.Venue, len(all))
					return all
				}
			}
			return syms
		}
	}
	if a.acct.Login != "" {
		if syms, err := c.dialectSymbols(ctx, a); err != nil {
			log.Printf("[mt5] %s 方言订阅集读败（落候选兜底径）: %v", a.acct.Venue, err)
		} else if len(syms) > 0 {
			log.Printf("[mt5] %s 订阅集=方言确认符 %d 枚: %v", a.acct.Venue, len(syms), syms)
			return syms
		}
	}
	confirmed := cl.SymbolConfirm(ctx, c.confirmCandidates(ctx, cl))
	log.Printf("[mt5] %s 方言回确命中: %v", a.acct.Venue, confirmed)
	return confirmed
}

// dialectSymbols fin_symbol_dialect (venue,account) venue_symbol 行集——
// 映射过=venue 硬验确认符，订阅集=映射集（canonical 序定序）。
func (c *Collector) dialectSymbols(ctx context.Context, a *acctRun) ([]string, error) {
	rows, err := c.pool.Query(ctx, `SELECT venue_symbol FROM fin_symbol_dialect
		WHERE venue=$1 AND account=$2 ORDER BY canonical`, a.acct.Venue, a.acct.Login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// confirmCandidates 兜底候选集——固定全谱∪SymbolList 自报名 CFD 主流筛
// （去重保序；SymbolList 读败=仅固定谱——venue 名一律实读禁猜）。
func (c *Collector) confirmCandidates(ctx context.Context, cl *Client) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, s := range strings.Split(symbolCandidates, ",") {
		add(s)
	}
	if syms, err := cl.SymbolList(ctx); err == nil {
		for _, s := range syms {
			if cfdMajorsRe.MatchString(strings.ToUpper(s)) {
				add(s)
			}
		}
	}
	return out
}
