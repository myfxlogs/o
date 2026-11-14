// econrule.go —— 经济规约闸（fin_0175「canon 可执行化」件②：裁定席把
// 可机读教训落 fin_econ_rule 行,admitProposals 机械执行——「lethal 簇
// 禁再喂」从 runbook 劝会话变机器闸,规约绕不开契约也不依赖会话读）。
//
// kind=admit_block:match{template_ref?inst_id?direction?}AND 匹配的提案
// 不开 paper 户留 fin_proposed——同模板门闸待位语义(规在拦,规撤复进)。
// 闸读一拍一载;读败=空规则集放行不阻钱径(规约面失效=回到契约劝域,
// 非降级黑箱——[lifecycle] log 留证)。未知 match 键/空 match=规约不适用
// (fail-closed 不拦——写坏的规约不许扩大战果)。
package lifecycle

import (
	"context"
	"encoding/json"
	"log"
)

// admitRule 机读规约行——match 子集键逐项等值匹配。
type admitRule map[string]string

// loadAdmitBlockRules 载 active admit_block 规约——闭集键外的行丢弃
// (fail-closed:未知键语义不认=不拦)。
func (r *Runner) loadAdmitBlockRules(ctx context.Context) []admitRule {
	rows, err := r.pool.Query(ctx, `SELECT match FROM fin_econ_rule
		WHERE kind='admit_block' AND active`)
	if err != nil {
		log.Printf("[lifecycle] econ_rule 载败(空集放行=回契约劝域): %v", err)
		return nil
	}
	defer rows.Close()
	var rules []admitRule
	for rows.Next() {
		var raw []byte
		if rows.Scan(&raw) != nil {
			return nil
		}
		var m map[string]string
		if json.Unmarshal(raw, &m) != nil || len(m) == 0 {
			continue // 空/非对象 match=不适用
		}
		valid := true
		for k := range m {
			if k != "template_ref" && k != "inst_id" && k != "direction" {
				valid = false
				break // 未知键=语义不认不拦(fail-closed)
			}
		}
		if valid {
			rules = append(rules, admitRule(m))
		}
	}
	return rules
}

// admitBlocked 提案键过规约闸——一键对全 match 项等值命中=拦。
func admitBlocked(rules []admitRule, tmpl, inst, dir string) bool {
	for _, rule := range rules {
		hit := true
		for k, v := range rule {
			var got string
			switch k {
			case "template_ref":
				got = tmpl
			case "inst_id":
				got = inst
			case "direction":
				got = dir
			}
			if got != v {
				hit = false
				break
			}
		}
		if hit {
			return true
		}
	}
	return false
}
