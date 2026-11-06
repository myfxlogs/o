// miner_skill.go —— skill_patch 产线（T-0387——canon 自修环矿工段，
// Memento/agent-os「弱 skill→改进→写回」环的 O 等价件）：矿工聚类命中
// skill 相关签名（坑条目 id/能力账键/RPC 名）→skill_patch 型 pending
// （target 域载荷扩非表变——票面钦 T-0385 pending 表构不动）。
// 名册=keyset 硬钉（capcheck 面清单同例）：skill→{能力账键/RPC 名 keys,
// 坑条目 id pits}，词界匹配+泛词不入册（okx/mt5/venue/Account 族泛词
// 入册=误路由病灶源）。坑引映射由 miner_skill_test 双向机钉（条目
// frontmatter skills: 标签须含名册 skill）——删条目/改 skill 名=册红
// 必修；新 skill/新坑条目不入册=降级 lesson 静默兜底（宁漏不误路由——
// 错投 canon 重于漏诊）。矿工只产草案——canon 写域名册钉归裁定席
// （skills/<name>/ 实存目录+skillpatch 机械五闸），矿工/LLM 零直改
// skills/**。
package hunger

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// skillSig skill 签名册行：keys=能力账键/RPC 名 token 集（特异词才入——
// 泛词入册=误路由）；pits=坑条目 id 集（条目 frontmatter skills: 标签
// 反查钦，测试机钉双向对拍）。
type skillSig struct {
	keys []string
	pits []string
}

// skillSigRoster skill 相关签名册（票面钦三类签名 token：坑条目 id/
// 能力账键/RPC 名；skill 目录名自身恒为命中 token 不入表）。
var skillSigRoster = map[string]skillSig{
	// mt5/mt4 面（mtapi 能力账——RPC 名取表列特异词，泛词 Account/
	// Mails/Connect/On* 不入册）
	"mt5-trading-mutating": {
		keys: []string{"OrderSend", "OrderModify", "OrderClose", "RetryFresh"},
		pits: []string{"exec-error-ambiguous-arbitrate-raw",
			"mt5-broker-symbol-trademode-closeonly"},
	},
	"mt5-orders-streams": {
		keys: []string{"OpenedOrder", "OrderHistory", "OrderHistoryPagination",
			"PendingOrderHistory", "MarketWatchMany", "SubscribeMarketWatch"},
		pits: []string{"probe-silent-stream-false-negative"},
	},
	"mt5-quotes-history": {
		keys: []string{"SymbolParams", "GetQuote", "PriceHistory",
			"GetTickValueMany", "TickValueWithSize"},
		pits: []string{"mt5-broker-symbol-trademode-closeonly"},
	},
	"mt5-account": {
		keys: []string{"AccountSummary", "ServerTimezone", "RequiredMargin"},
		pits: []string{"probe-silent-stream-false-negative"},
	},
	"mt5-gateway-admin": {keys: []string{"CheckConnect"}},
	"mt-gateway":        {},
	"mt-accounts":       {},
	// okx 面（能力账键+特异字段/参数字典键）
	"okx-exec": {
		keys: []string{"clOrdId", "FreshClient"},
		pits: []string{"cex-close-sz-fee-in-base", "cex-market-buy-sz-quote-ccy",
			"exec-error-ambiguous-arbitrate-raw", "exec-notional-ctval-missing",
			"lotsz-granularity-vs-hedge-band", "okx-clordid-charset-alnum-only",
			"probe-restore-scope-overreach", "rollback-close-direction",
			"unwind-limit-floor-protected"},
	},
	"okx-api-spec": {
		keys: []string{"docs-v5"},
		pits: []string{"cex-market-buy-sz-quote-ccy", "exec-notional-ctval-missing"},
	},
	"okx-account": {
		keys: []string{"availBal", "max-size"},
		pits: []string{"cex-close-sz-fee-in-base", "earn-amt-pending-doublecount",
			"unwind-limit-floor-protected"},
	},
	"okx-asset-funding": {
		keys: []string{"clientId"},
		pits: []string{"dcd-quote-reject-invisible-armedttl-kill",
			"earn-amt-pending-doublecount", "okx-dcd-endpoint-dialect",
			"venue-holding-surface-blind-spot"},
	},
	"okx-market-ws":     {keys: []string{"books5", "mark-price", "index-tickers"}},
	"okx-private-ws":    {keys: []string{"balance_and_position"}},
	"okx-rest-boundary": {pits: []string{"probe-restore-scope-overreach"}},
	// 数据/基建面
	"pg-capacity": {},
	"ctx-sources": {keys: []string{"ffcal"}},
}

// minerPatchDailyCap skill_patch 型日额（票面钦 ≤2——canon 是高权文本面，
// 比 lesson 日额 5 更严；越额簇降级 lesson 产不丢信号）。
const minerPatchDailyCap = 2

// skillHit 命中记（病灶锚素材——token@命中面）。
type skillHit struct {
	token, where string
	skill        string
}

// matchSkillSig 簇→skill 签名命中判定：全簇行 note+esc_class+esc_scope
// +ticket 扫册 token 词界命中；目标 skill=特异 token 命中数最多者
// （平手取字典序小者——定排可复现），零命中=lesson 面。
func matchSkillSig(c *mineCluster) (skill string, hits []skillHit) {
	bySkill := map[string][]skillHit{}
	for i := range c.rows {
		r := &c.rows[i]
		faces := []struct {
			where, text string
		}{
			{"note", r.note}, {"esc_scope", r.escScope},
			{"esc_class", r.escClass}, {"ticket", r.ticket},
		}
		for _, f := range faces {
			if f.text == "" {
				continue
			}
			for sk, sig := range skillSigRoster {
				for _, tok := range sig.tokens(sk) {
					if !hasToken(f.text, tok) {
						continue
					}
					bySkill[sk] = append(bySkill[sk], skillHit{tok, f.where, sk})
				}
			}
		}
	}
	best, bestN := "", 0
	var names []string
	for sk := range bySkill {
		names = append(names, sk)
	}
	sort.Strings(names)
	for _, sk := range names {
		seen := map[string]bool{}
		for _, h := range bySkill[sk] {
			seen[h.token] = true
		}
		if len(seen) > bestN {
			best, bestN = sk, len(seen)
		}
	}
	if best == "" {
		return "", nil
	}
	// 命中记定排（token 序——病灶锚可复现）。
	hits = bySkill[best]
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].token != hits[j].token {
			return hits[i].token < hits[j].token
		}
		return hits[i].where < hits[j].where
	})
	return best, hits
}

// tokens skill 全部签名 token（目录名自身恒在——canon 名直称即命中）。
func (s skillSig) tokens(name string) []string {
	out := make([]string, 0, len(s.keys)+len(s.pits)+1)
	out = append(out, name)
	out = append(out, s.keys...)
	return append(out, s.pits...)
}

// hasToken 词界匹配：token 两侧须非 [0-9A-Za-z_]（含串端——横线当界：
// canon 名含横线，t/0400-okx-exec-drift 式连缀要中；xokx-exec 式
// 字母粘连不中）。
func hasToken(text, tok string) bool {
	for i := 0; ; {
		j := strings.Index(text[i:], tok)
		if j < 0 {
			return false
		}
		j += i
		if wordEdge(text, j-1) && wordEdge(text, j+len(tok)) {
			return true
		}
		i = j + 1
	}
}

// wordEdge 位置在词界=真（越界/非词字符——词字符只 [0-9A-Za-z_]）。
func wordEdge(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	c := s[i]
	return !(c == '_' ||
		(c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'))
}

// lesionAnchor 病灶锚（命中 token@面 列——canon 内定位锚素材，截 8 件防胀）。
func lesionAnchor(hits []skillHit) string {
	var parts []string
	seen := map[string]bool{}
	for _, h := range hits {
		k := h.token + "@" + h.where
		if seen[k] {
			continue
		}
		seen[k] = true
		parts = append(parts, k)
		if len(parts) >= 8 {
			break
		}
	}
	return strings.Join(parts, ",")
}

// minerPatchCapLeft skill_patch 日额闸（票面钦 ≤2——比 lesson 更严；
// 同日额读径：当日已产 skill_patch 行随查询自见勿重复计）。
func (s *Sensor) minerPatchCapLeft(ctx context.Context) (bool, error) {
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_eng_lesson
		WHERE ts>=date_trunc('day',now()) AND numbers->>'target'='skill_patch'`).Scan(&n); err != nil {
		return false, fmt.Errorf("矿工 skill_patch 日额读败: %w", err)
	}
	return n < minerPatchDailyCap, nil
}
