// Package execfam —— 执行径族册唯一真源（T-0330 提案闸前置拒）：
// fin_opp_eval.template→执行径在册否——「有径」语义=lifecycle paper 开户→
// probe→live 晋升链与 cmd open 派发 executor 全链贯通者。
//
//	未建族提案=评估侧前置拒（eval verdict=rejected+reason=exec_family_
//	missing 如实，族码=eval.template 列归因可查），不进提案队列淤积
//	等死；lifecycle admitProposals 模板门闸同读本册——扩族=本表一处
//	登记，评估/生命周期双侧零漂移。
package execfam

import (
	"fmt"
	"sort"
)

var set = map[string]bool{
	"funding-hedge": true, "dcd_call": true, "dcd_put": true,
	"vol_sell_strangle": true, "vol_sell_single_call": true,
	"quarter_basis_long": true, "perp_quarter_spread": true,
	"quarter-calendar-spread": true, // 工单#24981 骨架登记——quarter_future双
	// 腿族,参照quarter_basis/perp_quarter册件;骨架语义=eval前置拒解禁+
	// 提案队列可达,basis记账臂模板集(basis.go:51)尚未覆盖=paper记账
	// 臂另棒(同quarter_basis_short缺margin册注先例),卡#531289复裁件
	"xvenue-carry": true,
	"stake-carry":  true,
	"dir_perp_mom": true, "perp-trend-daily": true, "mt5_dir_mom": true, // T-0160/T-0215/T-0216 历次落位同集（原 lifecycle.paperTemplates 册归并——quarter_basis_short=spot_short 腿缺 margin 能力面不在册）；dcd_put=T-0353 件②④复役登记（USDT 计价腿=基金钱常径——executor DcdQuote/DcdTrade 方向无关、paper 摊提同式、open 派发同 case，全链贯通验证在案）；xvenue-carry=T-0377 paper 径贯通登记（T-0144 双腿开户全链可达：评估前置拒/模板门闸双闸开）——open 派发仍 opener 永拒显式禁（T-0144 armed→open 钉死，T-0145 真执行另票），登记语义=paper 结算域贯通非真单径；dir_perp_mom=方向族首发 paper-only 登记（棒1——eval→proposal→paper→MTM 记账链贯通；真单径=executor+单腿演习件另棒，Drillable=false 拦 probe→live 在位）；perp-trend-daily=方向趋势族 paper-only 登记（工单#11481——Donchian 突破+winrate-CI 评估臂，eval→proposal→paper→MTM 链同棒1贯通；真单径另棒同裁定）；mt5_dir_mom=O-35163 CFD 方向动量族 paper-only 登记（模板#2074 legs=cfd_long/cfd_short settle=mtm——dirmom venue='mt5' 评估臂在案，eval→proposal→paper→dirpaper MTM 链贯通；live 构型位 fail-closed 桩：eq=0+Drillable 未产能，真单径不建）
	"grid_spot": true, // 工单#25840 OKX原生网格族登记——paper 域提案门开
	// （判词卡→提案→paper 链贯通）；真单径=okxgrid venue原生bot适配器非
	// execution.Executor 腿语义——live 晋升双闸在案：excell.Drillable 未
	// 产能（grid_spot 无 LegShape 映=probe→live 结构性拦）+okxgrid.live_
	// switch 业主首闸（缺/非 true fail-closed）——登记语义=paper 域贯通，
	// 真钱拉起两闸均未通。
	"sprd-atomic-basis": true, // 工单#26268 OKX Nitro Spread 价差组合族骨架册
	// ——提案卡#550052 parked 前提②消（裁定词「execfam 骨架未册收养骨检硬拒」）。
	// 登记语义=paper 域提案门开（判词卡→提案→paper 链贯通），**非真单语义**：
	// 真单径=OKX /api/v5/sprd/order 原子多腿执行臂，凭据私读 403 物理前提
	// 在案归业主（condreq 票 okx-read-scope），本票禁写任何 sprd API 调用码
	// （协议面取证纪律）；Drillable 未产能（无 LegShape 映=probe→live 结构
	// 性拦勿虚开）。inst 构型=sprdId 双腿 '_' 连接形（legset.InstFitOK
	// 同源判）。hunger 册免：族无自有 cfg 键——depenum/facesnap_whitelist
	// 「有键才册」票钦免册。
	"stable-park-usdg": true, // 工单#26272 parked卡#550053 骨架登记——paper 域
	// 提案门开；真径=OKX finance okusd subscribe/redeem(clOrdId幂等,非
	// execution.Executor腿语义);Drillable未册=probe→live结构性拦在位;
	// 凭据域=业主condreq在案(前提①),登记语义=骨架册非执行径;
	// hunger depenum/facesnap_whitelist 免册=族无自有cfg键(有键才册票钦)。
	"staking-defi-parking": true, // 工单#26272 parked卡#550055 骨架登记——
	// paper 域提案门开;真径=OKX finance staking-defi offers purchase/
	// redeem/cancel(三向停泊比价评估,非Executor腿语义);Drillable未册=
	// probe→live结构性拦在位;凭据域=业主condreq在案;hunger册免=
	// 族无自有cfg键(有键才册票钦)。
	"signal-bot-relay": true, // 工单#26272 parked卡#550054 骨架登记——
	// paper 域提案门开;真径=tradingBot signal API(create-signal+order-
	// algo+webhook,venue侧executor转发外部信号,消化#26159取证在案;弹
	// 性增量非新alpha);Drillable未册=probe→live结构性拦在位;凭据域=
	// 业主condreq;hunger册免=族无自有cfg键(有键才册票钦)。
	"xsec-perp-mom-basket": true, // 工单O-35489 假想卡#578380 骨架登记——
	// paper 域提案门开；真径=横截面动量篮多臂执行器未建（腿原语
	// perp_long_basket/perp_short_basket=篮复合非单 inst 腿形）；
	// Drillable 未册=probe→live 结构性拦在位；hunger 册免=族无自有
	// cfg 键（判据键 research.xsec_mom.* 在 fin_0241——有键才册先例
	// 适用但 depenum/facesnap 册语义=族执行 cfg 非判据 cfg，同#26272 族免）。
	"funding-extreme-fade": true, // 工单O-35489 假想卡#578381 骨架登记——
	// paper 域提案门开；真径=极端资金费率反打执行器未建（腿原语
	// perp_fade_dyn=方向动态单腿）；Drillable=ShapeOKXSingleLeg 册载
	// （单腿演习形在案——占用格出真演习格同 dir_perp_mom 先例）；
	// hunger 册免=族无自有 cfg 键同上。
	"mark-last-disloc-fade": true, // 工单O-35489 假想卡#578382 骨架登记——
	// paper 域提案门开；真径=标记/最新价错位反打执行器未建（腿原语
	// perp_fade_dyn 同上；延迟预算闸=卡 requirements.latency）；
	// Drillable=ShapeOKXSingleLeg 册载同上；hunger 册免同上。
}

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
	"dir_perp_mom": true, "perp-trend-daily": true, // 工单#11481 方向趋势族同棒1 paper-only 裁定
	"mt5_dir_mom": true, // O-35163 CFD 方向动量族——dirpaper MTM/出场摊提域成员（价源 md_ticks venue='mt5'，出场判词同查 fin_opp_eval direction 键）
	// O-35489 OSS 三族骨架——paper-only 裁定同上（真单径未建）；
	// dirpaper 消费面=template_ref∈本册 ∧ direction∈(long_perp,
	// short_perp)：两 fade 族 eval 将产方向行→MTM 链天然贯通；
	// xsec 篮族双向复合 direction 词表外→dirpaper 不收（篮 MTM 记账臂
	// 另件——funding 通用臂照记不虚构 pnl，fail-closed 不晋升）。
	"xsec-perp-mom-basket": true, "funding-extreme-fade": true,
	"mark-last-disloc-fade": true,
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

// ── 亏算形态标（判例#1器官级物化——工单#26436）：ops_event
// seat=自裁 event=判例 ticket=okxgrid-shape 在案——本金线内偏好亏算封
// 顶工具形（现货网格/套利/earn 停泊：亏≤投入+带内浮亏，无爆仓无资金
// 费拖磨）；杠杆不禁用但逐案过仓位判据保证金面——杠杆把潜在亏损推出
// 本金总额线外=越线归业主物理前提。形态标=族级静态册（扩族登记时钦
// 定）；判例行读径留扩展位（判例攒多再抽查表，本票不提前抽象）。
const (
	ShapeBounded   uint8 = 1 << iota // 亏算封顶——现货网格/现货套利/earn 停泊形
	ShapeLeveraged                   // 杠杆含爆仓线——perp/margin/期权卖方形
	ShapeFunded                      // 背资金费/持续持有成本拖磨——SWAP 系
)

// riskShapes 族→形态标位集（册内族必有册；缺册=未登记——RiskShape
// ok=false，闸消费面 fail-closed 视 leveraged 从严）。
var riskShapes = map[string]uint8{
	"funding-hedge":           ShapeLeveraged | ShapeFunded, // perp 腿爆仓线+资金费拖磨
	"xvenue-carry":            ShapeLeveraged | ShapeFunded,
	"dir_perp_mom":            ShapeLeveraged | ShapeFunded, // 方向 perp——杠杆+资金费
	"perp-trend-daily":        ShapeLeveraged | ShapeFunded,
	"mt5_dir_mom":             ShapeLeveraged | ShapeFunded, // O-35163 CFD——保证金杠杆+swap 隔夜融资拖磨（杠杆把潜在亏推过本金线=越线归业主同判例#1）
	"signal-bot-relay":        ShapeLeveraged | ShapeFunded, // 卡#550054 骨架——信号载体=USDT 永续
	"quarter_basis_long":      ShapeLeveraged,               // 季度远期腿=期货保证金形（无资金费位）
	"perp_quarter_spread":     ShapeLeveraged | ShapeFunded, // perp 腿背资金费（basis.go:128 实证——perp_quarter_spread 专属 funding 记账臂）
	"quarter-calendar-spread": ShapeLeveraged,
	"sprd-atomic-basis":       ShapeLeveraged | ShapeFunded, // 腿可含 -SWAP 形（sprdLegRe 收）——保证金面+资金费拖磨双标
	"dcd_call":                ShapeLeveraged,               // 期权卖方——尾部风险超权利金，margin 形态从严
	"dcd_put":                 ShapeLeveraged,
	"vol_sell_single_call":    ShapeLeveraged,
	"vol_sell_strangle":       ShapeLeveraged,
	"stake-carry":             ShapeBounded,                 // 质押 carry=earn 停泊形（参考模型族）
	"grid_spot":               ShapeBounded,                 // 判例#1 钦定代表——现货网格亏算封顶
	"stable-park-usdg":        ShapeBounded,                 // 卡#550053 骨架——okusd 停泊=earn 形
	"staking-defi-parking":    ShapeBounded,                 // 卡#550055 骨架——staking 停泊=earn 形
	"xsec-perp-mom-basket":    ShapeLeveraged | ShapeFunded, // O-35489 卡#578380——perp 篮腿爆仓线+资金费拖磨（判例#1 杠杆逐案过仓位判据同 bit 语义）
	"funding-extreme-fade":    ShapeLeveraged | ShapeFunded, // 卡#578381——perp 单腿同上
	"mark-last-disloc-fade":   ShapeLeveraged | ShapeFunded, // 卡#578382——perp 单腿同上
}

// RiskShape 族形态标集（ok=false=册缺——消费面 fail-closed 视 leveraged
// 从严；扩族漏登形态标=闸按最严档处理非静默放行）。
func RiskShape(template string) (mask uint8, registered bool) {
	m, ok := riskShapes[template]
	return m, ok
}

// ShapeBoundedOnly 族形态=bounded 单标否（判例#1 直过判据——无
// leveraged/funded 位=亏算结构封顶，保证金面审免）。
func ShapeBoundedOnly(template string) bool {
	m, ok := riskShapes[template]
	return ok && m == ShapeBounded
}

// ShapeGateVerdict 判例#1形态闸判词（拉起闸消费件——策略 probe→live/
// bot 拉起面同读）：
//
//	bounded 单标→直过（判例钦定优先形）；
//	含 leveraged/funded 位或缺册（fail-closed 视 leveraged 从严）→保证金
//	面审：demand=notional/levAssumed（期望保证金占用）+notional×lossFrac
//	（极端情形潜在亏损）>principal（本金总额线）=硬拒——杠杆把潜在亏
//	损推出本金总额线外=越线归业主物理前提，升件记案归调用面；
//	参数非正（levAssumed/lossFrac/principal≤0）=判据缺 fail-closed 拒
//	（不虚构判据值——bounded 直过先于参数判，免审族不受参数面影响）。
//	第三返回=越本金线归业主标（true=调用面升件记案；判据缺/审过=false）。
func ShapeGateVerdict(template string, notional, principal, levAssumed, lossFrac float64) (bool, string, bool) {
	if ShapeBoundedOnly(template) {
		return true, fmt.Sprintf("族=%s bounded 单标直过（判例#1——亏算封顶形态免保证金面审）", template), false
	}
	mask, ok := riskShapes[template]
	if !ok {
		mask = ShapeLeveraged // 缺册 fail-closed 视 leveraged 从严
	}
	if levAssumed <= 0 || lossFrac <= 0 || principal <= 0 {
		return false, fmt.Sprintf("族=%s 形态=%08b 判据缺（lev=%.4g lossFrac=%.4g principal=%.4g）——fail-closed 不放行", template, mask, levAssumed, lossFrac, principal), false
	}
	margin := notional / levAssumed
	extreme := notional * lossFrac
	demand := margin + extreme
	if demand > principal {
		return false, fmt.Sprintf("族=%s 形态=%08b 硬拒：需求=%.2f（保证金%.2f+极端亏%.2f notional=%.2f）>本金线%.2f——杠杆潜在亏损越线归业主物理前提", template, mask, demand, margin, extreme, notional, principal), true
	}
	return true, fmt.Sprintf("族=%s 形态=%08b 保证金面审过：需求=%.2f（保证金%.2f+极端亏%.2f）≤本金线%.2f", template, mask, demand, margin, extreme, principal), false
}
