// Package legset —— 策略腿行情键归一链唯一真源（T-0394）。
//
// 现货腿派生表达式多面消费：okxws activeSet 订阅集 / projector 镜像谓词 /
// hunger 依赖对活性枚举探针——派生臂式唯一真源（禁自造第二套归一链），
// 状态谓词=各面集语义自带（订阅集/镜像谓词/枚举探针各钦自身谓词片段）。
// 与在飞 T-0392 界碑：本件归并的是 master 现谱三臂（inst 本键∪dcd∪
// basis）；0392 的 funding-hedge 第四臂=其票面根修件，合入时在本件臂册
// 原位补臂——两面同一真源不漂移。
package legset

import (
	"regexp"
	"strings"
)

// DatedFutRe 季度期货形（'-' 尾 6 位到期日——mark-price 适用无 funding）。
var DatedFutRe = regexp.MustCompile(`-[0-9]{6}$`)

// IsUSDTInst USDT 计价闸（业主令：杂币不采不用不计——instId 形如 X-USDT[-SWAP]）。
// 自 okxws 归并唯一真源（T-0394——订阅闸与枚举探针同判据）：
// USD 指数键（BTC-USD——index-tickers uly 键域）+到期期货形（-\d{6}$ 尾——
// BTC-USD-261225）放行豁免——「杂币不采」禁的是杂计价币面，USD 指数/币
// 本位期货系 basis 域钦定数据源非杂币。
func IsUSDTInst(instID string) bool {
	if strings.HasSuffix(instID, "-USD") || DatedFutRe.MatchString(instID) {
		return true
	}
	return strings.Contains(instID, "-USDT")
}

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

// ActiveSetUnion 活跃集腿行情键 UNION（订阅集与枚举探针同式——inst 本键
// 提案/策略面+两族派生现货腿全谱；T-0394 票面钦「枚举从代码/注册表实扫
// 零硬编码」的腿键源即本串，禁第二套）。
// 谓词面：提案=fin_proposed/fin_armed；策略 inst=live/degraded；dcd 臂=
// 非 retired；basis 臂=paper/live/degraded——master 现谱语义零漂移
// （funding-hedge 第四臂归在飞 T-0392，合入时此串补臂单点改）。
var ActiveSetUnion = `SELECT inst_id FROM fin_proposal WHERE state IN ('fin_proposed','fin_armed')
	UNION SELECT inst_id FROM fin_strategy WHERE state IN ('live','degraded')
	UNION ` + DCDSpotArm(`s.state<>'retired'`) + `
	UNION ` + BasisSpotArm(`s.state IN ('paper','live','degraded')`)
