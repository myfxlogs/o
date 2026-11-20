// Package legset —— 策略腿行情键归一链唯一真源（T-0392）。
//
// 现货腿派生表达式三面消费：okxws activeSet 订阅集 / projector 镜像谓词 /
// hunger 腿-行情覆盖探针——派生臂式唯一真源（禁自造第二套归一链），状态
// 谓词=各面集语义自带（订阅集/镜像谓词/覆盖探针各钦自身谓词片段）。
// funding-hedge 族册=execfam.FundingHedge 同源，模板集扩缩零漂移。
package legset

import (
	"regexp"
	"strings"

	"o/internal/platform/execfam"
)

// DatedFutRe 季度期货形（'-' 尾 6 位到期日——mark-price 适用无 funding）。
var DatedFutRe = regexp.MustCompile(`-[0-9]{6}$`)

// IsUSDTInst USDT 计价闸（业主令：杂币不采不用不计——instId 形如 X-USDT[-SWAP]）。
// 自 okxws 归并唯一真源（T-0392——订阅闸与覆盖探针同判据）：
// USD 指数键（BTC-USD——index-tickers uly 键域）+到期期货形（-\d{6}$ 尾——
// BTC-USD-261225）放行豁免——「杂币不采」禁的是杂计价币面，USD 指数/币
// 本位期货系 basis 域钦定数据源非杂币。
func IsUSDTInst(instID string) bool {
	if strings.HasSuffix(instID, "-USD") || DatedFutRe.MatchString(instID) {
		return true
	}
	return strings.Contains(instID, "-USDT")
}

// fundingHedgeIN 族键 SQL IN 表（execfam 册唯一真源——字面量键无注入面）。
var fundingHedgeIN = "'" + strings.Join(execfam.FundingHedgeList(), "','") + "'"

// BasisSpotArm basis 两族现货腿派生臂（T-0216 归一链：期货键剥 -YYMMDD 尾
// →剥 -USD 尾→-USDT 现货对）。pred=fin_strategy s 行谓词片段（各面集语义
// 自带——订阅集/镜像谓词/探针各钦；调用面只准静态字面量，禁拼接外部输入）。
func BasisSpotArm(pred string) string {
	return `SELECT regexp_replace(regexp_replace(s.inst_id,'-[0-9]{6}$',''),'-USD$','-USDT')
		FROM fin_strategy s WHERE ` + pred + `
		AND s.template_ref IN ('quarter_basis_long','perp_quarter_spread')`
}

// DCDSpotArm dcd 族现货腿派生臂（T-0171 归一链：uly 剥 -USD(G)→-USDT，
// md_dcd_product JOIN 取 uly——与 research/dcd spot() 候选链同语义）。
func DCDSpotArm(pred string) string {
	return `SELECT regexp_replace(d.uly,'-USD(G)?$','-USDT') FROM fin_strategy s
		JOIN md_dcd_product d ON d.product_id=s.inst_id
		WHERE ` + pred + ` AND s.template_ref LIKE 'dcd%'`
}

// FundingHedgeSpotArm funding-hedge 族现货腿派生臂（T-0392 补臂：SWAP 键
// 剥 -SWAP 尾→现货对——short_perp_long_spot/long_perp_short_spot 向策略
// 的现货腿价源；模板属=execfam fundingHedge 册同源禁自造清单）。
func FundingHedgeSpotArm(pred string) string {
	return `SELECT regexp_replace(s.inst_id,'-SWAP$','') FROM fin_strategy s
		WHERE ` + pred + ` AND s.template_ref IN (` + fundingHedgeIN + `)`
}

// SubRequestArm 晋级临时订阅臂（T-0401 两级采集制②——评估臂提名候选符的
// 临时行情请求；fin_sub_request.state='open' 行入集=collector 流式采，
// 评估判词出闭请行退订，提案常驻=提案臂接管）。谓词自带无外参。
var SubRequestArm = `SELECT inst_id FROM fin_sub_request WHERE state='open'`

// ── inst 构型核（工单#15943 promotion 模板门闸——execfam.Has 只核「有径」
// 不核「inst 形合族腿构型」：DCD 期权合约（BETH-USDT-*-C 族）被盲 promotion
// 误建 funding-hedge paper 748 行实证（econ 矿 #273/#274，live 占位死）。
// 族→允许构型映射唯一真源，lifecycle admitProposals 门闸消费；
// 反向同律——SWAP/远期形 inst 不得进 dcd/basis 族（腿派生臂同源失明）。

var (
	// swapPerpRe 永续腿形（-SWAP$ 尾——funding-hedge/xvenue-carry 主腿，
	// 现货腿由 SWAP 键派生不直接持现货 inst）。
	swapPerpRe = regexp.MustCompile(`-SWAP$`)
	// dcdOptRe DCD 期权腿形（BASE-QUOTE-YYMMDD-STRIKE-{C,P} 段——QUOTE 域
	// USDT/RLUSD/USDC/USDG…不收窄；strangle 复合形 + 连接双腿同段各命中）。
	dcdOptRe = regexp.MustCompile(`-[0-9]{6}-[0-9]+(\.[0-9]+)?-[CP]`)
	// optLegRe 期权类型段界（-C+/−P$——复合形每腿各成一枚类型计）。
	optLegRe = regexp.MustCompile(`-([CP])(?:\+|$)`)
	// venueCcyRe stake_carry 假想卡 inst 形（venue:ccy——okx:TRX/cmc:TRX，
	// 双头族键=venue 空腿位+质押币对，非可交易 inst 键——工单#21612 参考模型族）。
	venueCcyRe = regexp.MustCompile(`^[a-z0-9]+:[A-Z0-9]+$`)
	// okxUsdgRe stable-park-usdg 族 inst 形（工单#26272 骨架——okusd
	// 申购/赎回标的=USDG 资产级伪键，非交易对；venue 前缀钉 okx 单源）。
	okxUsdgRe = regexp.MustCompile(`^okx:USDG$`)
	// okxStakeRe staking-defi-parking 族 inst 形（工单#26272 骨架——
	// 质押标的=ETH/SOL 资产级伪键，卡面钦定两币收窄；offers 面扩币
	// 另票开，fail-closed 偏严）。
	okxStakeRe = regexp.MustCompile(`^okx:(ETH|SOL)$`)
)

// instForm inst 构型分类（族腿构型判据的中间层）。
type instForm int

const (
	formOther       instForm = iota // 现货对/伪键/未知形
	formSwapPerp                    // -SWAP$ 永续
	formDatedFut                    // -\d{6}$ 季度远期（DatedFutRe 同源）
	formDCDCall                     // DCD 单腿看涨产品
	formDCDPut                      // DCD 单腿看跌产品
	formDCDStrangle                 // DCD strangle 复合形（C+P 双腿）
)

// classifyInst inst→构型分类。期权形优先判（复合形含 -P$ 尾不撞
// DatedFutRe；SWAP/远期形不含 -\d{6}-strike-C/P 段不撞 dcdOptRe——三族
// 构型互斥无需次序，但期权复合形先取保证 strangle 不落单腿）。
func classifyInst(instID string) instForm {
	if dcdOptRe.MatchString(instID) {
		types := map[byte]bool{}
		for _, m := range optLegRe.FindAllStringSubmatch(instID, -1) {
			types[m[1][0]] = true
		}
		switch {
		case types['C'] && types['P']:
			return formDCDStrangle
		case types['C']:
			return formDCDCall
		case types['P']:
			return formDCDPut
		}
		return formOther
	}
	if swapPerpRe.MatchString(instID) {
		return formSwapPerp
	}
	if DatedFutRe.MatchString(instID) {
		return formDatedFut
	}
	return formOther
}

// InstFitOK promotion 模板门闸 inst 构型核（工单#15943）：template 族
// 允许的 inst 构型唯一判据——不合=提案留 fin_proposed 不建策略行
// （与模板门闸同语义「看见了做不了」）。未注册族返 false=fail-closed
// （execfam 扩族漏登构型=闸拦响非静默放行；覆盖断言见 legset 测试）。
func InstFitOK(template, instID string) bool {
	switch template {
	case "funding-hedge", "xvenue-carry", "dir_perp_mom", "perp-trend-daily":
		// perp 腿形家族——主腿=USDT 本位永续（D-0018 只许 USDT 计价；
		// IsUSDTInst 同源拒 BTC-USD-SWAP 币本位形；dir_perp_mom=单腿方向
		// 位同 perp 键，dirpaper.go 价源 md_ticks 同读）。
		return classifyInst(instID) == formSwapPerp && IsUSDTInst(instID)
	case "quarter_basis_long", "perp_quarter_spread", "quarter-calendar-spread":
		// basis 两族——腿=季度远期形（-\d{6}$ 尾，币/ USDT 本位皆可）。
		// quarter-calendar-spread 双腿同季度远期键（工单#24981 骨架）。
		return classifyInst(instID) == formDatedFut
	case "dcd_call", "vol_sell_single_call":
		return classifyInst(instID) == formDCDCall
	case "dcd_put":
		return classifyInst(instID) == formDCDPut
	case "vol_sell_strangle":
		return classifyInst(instID) == formDCDStrangle
	case "stake-carry":
		// 双头族 inst=venue:ccy 假想卡键（非交易 inst——工单#21612 参考
		// 模型族：eval→proposal→paper 记账面贯通，真单径 venue 执行臂另案）。
		return venueCcyRe.MatchString(instID)
	case "grid_spot":
		// venue 原生现货网格族（工单#25840）——bot 腿=USDT 计价现货对
		// （BTC-USDT 形，formOther 域内圈定——-SWAP/-\d{6}/venue:ccy 皆拒）。
		return classifyInst(instID) == formOther && IsUSDTInst(instID)
	case "stable-park-usdg":
		// 卡#550053 骨架——inst=okx:USDG 资产级伪键（earn_deposit 腿标的；
		// 交易对/SWAP/venue:ccy 通形皆拒，okusd 面仅 USDG 一标的收窄）。
		return okxUsdgRe.MatchString(instID)
	case "staking-defi-parking":
		// 卡#550055 骨架——inst=okx:ETH/okx:SOL 资产级伪键（staking-defi
		// purchase 标的；他币/异 venue/交易对/SWAP 形皆拒，fail-closed）。
		return okxStakeRe.MatchString(instID)
	case "signal-bot-relay":
		// 卡#550054 骨架——信号载体=order-algo instIds 钦定 USDT 本位
		// 永续形（docs-v5 tradingBot/signal/order-algo 实证 BTC-USDT-
		// SWAP）；现货对/远期/期权/伪键形皆拒，fail-closed。
		return classifyInst(instID) == formSwapPerp && IsUSDTInst(instID)
	}
	return false
}

// ProposalArm 提案臂（fin_proposed/fin_armed 在订即入集——无策略/无
// position 罩的独键提案 tick 亦须过桥投影，armed 提案入场判据读 SG
// md_ticks 鲜度闸 fail-closed；升件#12229 双端谓词同源件——订阅集与
// 桥端投影同引本件，禁复制字面量）。谓词自带无外参。
var ProposalArm = `SELECT inst_id FROM fin_proposal WHERE state IN ('fin_proposed','fin_armed')`

// ActiveSetUnion 活跃集腿行情键 UNION（订阅集与覆盖探针同式——inst 本键
// 提案/策略面+三族派生现货腿全谱+晋级临时订阅臂；T-0392 票面钦「归一链
// 复用 activeSet 同源表达式」即本串，禁第二套）。
// 谓词面：提案=fin_proposed/fin_armed；策略 inst=live/degraded；dcd 臂=
// 非 retired；basis 臂=paper/live/degraded；funding-hedge 臂=非 retired
// （票面「在仓/在途/live」——paper/probe/proposed 同含，退役即出）；
// 晋级臂=fin_sub_request open（T-0401——UNION 加臂非改旧臂，红线不动）。
// ActiveSetUnionSansSubReq 非提名臂 UNION（需求源分解面——dep 探针
// 「提名臂独占键」判定引本件；谓词面同全串档逐臂在案，禁第二套）。
var ActiveSetUnionSansSubReq = ProposalArm + `
	UNION SELECT inst_id FROM fin_strategy WHERE state IN ('live','degraded')
	UNION ` + DCDSpotArm(`s.state<>'retired'`) + `
	UNION ` + BasisSpotArm(`s.state IN ('paper','live','degraded')`) + `
	UNION ` + FundingHedgeSpotArm(`s.state<>'retired'`)

var ActiveSetUnion = ActiveSetUnionSansSubReq + `
	UNION ` + SubRequestArm
