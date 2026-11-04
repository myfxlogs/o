// Package execfam —— 执行径族册唯一真源（T-0330 提案闸前置拒）：
// fin_opp_eval.template→执行径在册否——「有径」语义=lifecycle paper 开户→
// probe→live 晋升链与 cmd open 派发 executor 全链贯通者。
//
//	未建族提案=评估侧前置拒（eval verdict=rejected+reason=exec_family_
//	missing 如实，族码=eval.template 列归因可查），不进提案队列淤积
//	等死；lifecycle admitProposals 模板门闸同读本册——扩族=本表一处
//	登记，评估/生命周期双侧零漂移。
package execfam

var set = map[string]bool{
	"funding-hedge": true, "dcd_call": true,
	"vol_sell_strangle": true, "vol_sell_single_call": true,
	"quarter_basis_long": true, "perp_quarter_spread": true} // T-0160/T-0215/T-0216 历次落位同集（原 lifecycle.paperTemplates 册归并——quarter_basis_short=spot_short 腿缺 margin 能力面不在册）
// xvenue-carry：T-0145 执行径未派案在档（armed→open 显式禁）——不在册；
// 建径票落=本表登记即全链通，提案面/门闸面无需再改。

// Has 模板族执行径在册否。
func Has(template string) bool { return set[template] }
