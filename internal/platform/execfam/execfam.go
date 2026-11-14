// Package execfam —— 执行径族册唯一真源（T-0330 提案闸前置拒）：
// fin_opp_eval.template→执行径在册否——「有径」语义=lifecycle paper 开户→
// probe→live 晋升链与 cmd open 派发 executor 全链贯通者。
//
//	未建族提案=评估侧前置拒（eval verdict=rejected+reason=exec_family_
//	missing 如实，族码=eval.template 列归因可查），不进提案队列淤积
//	等死；lifecycle admitProposals 模板门闸同读本册——扩族=本表一处
//	登记，评估/生命周期双侧零漂移。
package execfam

import "sort"

var set = map[string]bool{
	"funding-hedge": true, "dcd_call": true, "dcd_put": true,
	"vol_sell_strangle": true, "vol_sell_single_call": true,
	"quarter_basis_long": true, "perp_quarter_spread": true,
	"xvenue-carry": true,
	"dir_perp_mom": true} // T-0160/T-0215/T-0216 历次落位同集（原 lifecycle.paperTemplates 册归并——quarter_basis_short=spot_short 腿缺 margin 能力面不在册）；dcd_put=T-0353 件②④复役登记（USDT 计价腿=基金钱常径——executor DcdQuote/DcdTrade 方向无关、paper 摊提同式、open 派发同 case，全链贯通验证在案）；xvenue-carry=T-0377 paper 径贯通登记（T-0144 双腿开户全链可达：评估前置拒/模板门闸双闸开）——open 派发仍 opener 永拒显式禁（T-0144 armed→open 钉死，T-0145 真执行另票），登记语义=paper 结算域贯通非真单径；dir_perp_mom=方向族首发 paper-only 登记（棒1——eval→proposal→paper→MTM 记账链贯通；真单径=executor+单腿演习件另棒，Drillable=false 拦 probe→live 在位）

// Has 模板族执行径在册否。
func Has(template string) bool { return set[template] }

// All 执行径族册全量（序定排——excell 词表零占免演登记消费面）。
func All() []string {
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// fundingHedge funding-hedge 族册（perp 空+现货多家族——T-0392 自 lifecycle
// opener.fundingHedgeFamily 归并：族属判据唯一真源，开仓 funding 闸域/
// 行情腿派生臂/覆盖探针三面同源，禁各处自造清单）。
var fundingHedge = map[string]bool{
	"funding-hedge": true, "xvenue-carry": true,
}

// FundingHedge 模板属 funding-hedge 族否（short_perp_long_spot/
// long_perp_short_spot 向——funding 闸域与现货腿派生同判据）。
func FundingHedge(template string) bool { return fundingHedge[template] }

// FundingHedgeList 族键序（SQL IN 表/桩测面——序定排防断言漂移）。
func FundingHedgeList() []string {
	out := make([]string, 0, len(fundingHedge))
	for t := range fundingHedge {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// paperOnly paper 结算域族册（方向族面——登记语义=paper MTM/出场摊提域
// 成员（dirpaper 消费），非「真单径缺席」判：真单径缺席语义唯一真源=
// !excell.Drillable（棒5c 根修——dir_perp_mom 单腿径接通后 paper 域与
// live 域并存，册语义收窄回结算域，演习豁免判转 Drillable）。
var paperOnly = map[string]bool{
	"dir_perp_mom": true,
}

// PaperOnly 模板属 paper-only 裁定族否。
func PaperOnly(template string) bool { return paperOnly[template] }

// PaperOnlyList paper-only 族键序（SQL ANY 绑参/记账面集——序定排防漂移）。
func PaperOnlyList() []string {
	out := make([]string, 0, len(paperOnly))
	for t := range paperOnly {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
