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

// ActiveSetUnion 活跃集腿行情键 UNION（订阅集与覆盖探针同式——inst 本键
// 提案/策略面+三族派生现货腿全谱+晋级临时订阅臂；T-0392 票面钦「归一链
// 复用 activeSet 同源表达式」即本串，禁第二套）。
// 谓词面：提案=fin_proposed/fin_armed；策略 inst=live/degraded；dcd 臂=
// 非 retired；basis 臂=paper/live/degraded；funding-hedge 臂=非 retired
// （票面「在仓/在途/live」——paper/probe/proposed 同含，退役即出）；
// 晋级臂=fin_sub_request open（T-0401——UNION 加臂非改旧臂，红线不动）。
var ActiveSetUnion = `SELECT inst_id FROM fin_proposal WHERE state IN ('fin_proposed','fin_armed')
	UNION SELECT inst_id FROM fin_strategy WHERE state IN ('live','degraded')
	UNION ` + DCDSpotArm(`s.state<>'retired'`) + `
	UNION ` + BasisSpotArm(`s.state IN ('paper','live','degraded')`) + `
	UNION ` + FundingHedgeSpotArm(`s.state<>'retired'`) + `
	UNION ` + SubRequestArm
