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
	"funding-hedge": true, "dcd_call": true, "dcd_put": true,
	"vol_sell_strangle": true, "vol_sell_single_call": true,
	"quarter_basis_long": true, "perp_quarter_spread": true,
	"xvenue-carry": true} // T-0160/T-0215/T-0216 历次落位同集（原 lifecycle.paperTemplates 册归并——quarter_basis_short=spot_short 腿缺 margin 能力面不在册）；dcd_put=T-0353 件②④复役登记（USDT 计价腿=基金钱常径——executor DcdQuote/DcdTrade 方向无关、paper 摊提同式、open 派发同 case，全链贯通验证在案）；xvenue-carry=T-0377 paper 径贯通登记（T-0144 双腿开户全链可达：评估前置拒/模板门闸双闸开）——open 派发仍 opener 永拒显式禁（T-0144 armed→open 钉死，T-0145 真执行另票），登记语义=paper 结算域贯通非真单径

// Has 模板族执行径在册否。
func Has(template string) bool { return set[template] }
