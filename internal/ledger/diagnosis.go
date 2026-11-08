// diagnosis.go —— 根因诊断案装配机（T-0431——案卷驱动开放考古排班+判词归流，
// 工程径自循环末环：detect→升件→案卷→【本件补诊断】→判词→修法）。
//
// 病灶（业主判词）：机器会分类不会归因——postmortem 只产「哪类异常」码集，
// 「为什么异常」无座位（pos34 leg.sz 单位错配靠业主问+人肉考古 ~7h 定位）。
//
// 分工（人机分界钉死——草案=假设非判词，机器不自批）：
//
//   - 本件（Go，确定性）：案卷→诊断任务——异常案卷（postmortem 异常码+守卫环
//     +账实差未定性）自动生成 fin_econ_lesson kind='rootcause_diag' 诊断任务行
//     （与 trade_postmortem 同 ref 并行，UNIQUE 幂等一仓一案）：载荷=案卷+
//     证据查询锚集+假说族种子（先例谱确定性匹配——排班「先挖什么」）；
//   - 裁定席第四分支（skills/裁定席，LLM）：领任务+代码仓只读开放考古——
//     证伪/证实/扩族→诊断草案落位（numbers.diagnosis+status='drafted'）
//     +ops_event 诊断草案行（DM 可读判词行）；
//   - DM 裁（skills/CTO席）：裁决/驳回行——裁过→修法票自动草稿（GW-001
//     段D4 起归流执行=机械轨 scripts/meta/verdict_reflow.sh 确定性脚本，
//     复用 fin_ticket_draft→ticketdraft 物料化链）或并入在飞票判词；
//     裁驳→驳回因入案卷（diag_state='rejected'——再诊断携错因
//     =草案质量自学习），三轮驳回升不裁。
//
// 禁线（诊断面零钱径）：本件只写 fin_econ_lesson（案卷锚表）+rejects 计数
// ——venue 调用零、钱径表（fin_position/fin_order/fin_order_leg/
// fin_holding/fin_guard_close/fin_ledger/fin_cashflow）写径零；证据查询
// 锚=只读 SQL 文本（裁定席会话执行），本件不执行。
package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
)

// diagCaseFacts 案卷特征面（确定性族匹配的输入——postmortem 事实+守卫环）。
type diagCaseFacts struct {
	PosID      int64
	ExecID     int64
	InstID     string
	Anomalies  []string // 机械异常码集（fast_close/settled_over/trigger_unknown/bare_bag）
	BareBagUSD float64  // 残袋挂账额（账实差未定性显形）
	BareBagQty float64
	Attempts   int     // 守卫平仓尝试数（≥2=守卫环）
	Trigger    string  // 触发类（原貌）
	SettledPct float64 // 结算幅 %
	// GW-001 段D1 fast_close 复发闸入参（postmortem 调用面装配——
	// 单次秒平不烧诊断额度，复发∨亏损才入闸）：
	SettledUSD      float64 // 结算额原值（<0=秒平亏损单次即立案）
	FastCloseRepeat bool    // 复发实断（7d 窗同 strategy_ref∨inst_id 含本仓共 ≥N 次）
}

// hasAnomaly 码集命中。
func (f diagCaseFacts) hasAnomaly(code string) bool {
	for _, c := range f.Anomalies {
		if c == code {
			return true
		}
	}
	return false
}

// diagFamily 假说族（先例谱条目——KB 实证条目升谱：pos34 条目
// order-leg-sz-unit-mismatch-naked-bag 为首族；新族随票径扩谱非热改）。
type diagFamily struct {
	ID         string                   // 族标（机读——草案/裁决/修法票签名共用键）
	Name       string                   // 族名（人读）
	Match      func(diagCaseFacts) bool // 结构签名（案卷特征→族先验——种子非判词）
	Hypothesis string                   // 种子假设（族先验话术——开放考古的起点非终点）
	Evidence   []string                 // 证据查询锚（只读 SQL——裁定席会话执行；{pos}/{exec}/{inst} 命名占位）
	FixSurface string                   // 建议修法面（先例处置节指向）
	Lexicon    []string                 // 族命中词谱（文本级族匹配——红桩判据机读形）
}

// diagFamilies 假说族先例谱。
var diagFamilies = []diagFamily{
	{
		ID:   "unit_mismatch",
		Name: "单位语义错配",
		Match: func(f diagCaseFacts) bool {
			// pos34 结构签名：残袋挂账（账实差未定性）× 结算幅异常/秒级开平
			// ——「平了但只清零头」族形；守卫环（attempts≥2）同族强化证据。
			return f.BareBagUSD > 0 && (f.hasAnomaly("settled_over") || f.hasAnomaly("fast_close"))
		},
		Hypothesis: "双制量字段消费面单位语义错配：sz_unit=quote（美元额）的 leg.sz 被" +
			"下游（平仓腿 qty 计算/漂移判官 notionalNow）当 base 币量消费——平/比只及零头，" +
			"产残袋挂账+假性 drift 杀环+账实背离（先例=pos34：开腿 sz=24.999 quote、" +
			"qty_base=1915.63 对，平腿同数按 base 发单只清 1.3%）",
		Evidence: []string{
			"SELECT leg_no,side,inst_id,sz,sz_unit,qty_base,state FROM fin_order_leg WHERE execution_id={exec} ORDER BY leg_no —— 开/平腿 sz×sz_unit×qty_base 三列对拍（quote 额被当 base 量的直接证据）",
			"SELECT state,attempts,last_error,last_attempt_at,trigger_kind FROM fin_guard_close WHERE position_id={pos} —— 守卫环：重试次数×败因×触发类原貌（幻觉 drift 杀环审计锚）",
			"SELECT kind,ccy,amount FROM fin_holding WHERE ccy=split_part('{inst}','-',1) AND kind IN ('spot_dust','earn_flexible') —— 账实差：残袋在 venue 的实存形（Earn 收息在案）",
		},
		FixSurface: "平仓腿 qty=持仓腿实得 qty_base（读腿/读余额），禁拿开仓 sz 美元额当 base 卖；sz_unit 双制字段全部消费面按制分派或一律读 qty_base（grep 消费谱对拍）——先例处置节=docs/工程经验库/条目/order-leg-sz-unit-mismatch-naked-bag.md",
		Lexicon:    []string{"sz_unit", "qty_base", "单位", "量纲", "quote", "base", "双制"},
	},
}

// matchDiagFamilies 族谱确定性匹配（种子排班——命中族=先挖方向非结论）。
func matchDiagFamilies(f diagCaseFacts) []diagFamily {
	var out []diagFamily
	for _, fam := range diagFamilies {
		if fam.Match(f) {
			out = append(out, fam)
		}
	}
	return out
}

// matchDiagFamilyText 文本级族命中（族谱词谱等值——「非必点名同词，族命中
// 即可」判据的机读形；红桩断言+LLM 草案族标校验同用此器）。
func matchDiagFamilyText(text string) []string {
	var out []string
	for _, fam := range diagFamilies {
		for _, w := range fam.Lexicon {
			if w != "" && strings.Contains(strings.ToLower(text), strings.ToLower(w)) {
				out = append(out, fam.ID)
				break
			}
		}
	}
	return out
}

// fastCloseWorthy fast_close 复发闸（GW-001 段D1——单次秒平不烧诊断
// 额度：复发实断（7d 窗同 strategy_ref∨inst_id 含本仓共 ≥N 次，
// N=热启键 guard.postmortem.fast_close_repeat_n）∨秒平亏损（settled<0）
// 二者任一才入闸；复发实断由 postmortem 调用面装配进 facts——本件保持
// 纯函数零 DB 依赖）。
func (f diagCaseFacts) fastCloseWorthy() bool {
	return f.hasAnomaly("fast_close") && (f.FastCloseRepeat || f.SettledUSD < 0)
}

// diagnosisWorthy 诊断任务生成闸（三案卷源任一——账实差未定性/认知集外
// 触发/守卫环——外加 GW-001 段D1 fast_close 复发闸；纯市况幅超阈/单次
// 秒平不生成——裁定席既有三分支消化，诊断额度不烧市况案）。
func diagnosisWorthy(f diagCaseFacts) bool {
	return f.BareBagUSD > 0 || f.hasAnomaly("trigger_unknown") || f.Attempts >= 2 || f.fastCloseWorthy()
}

// attrFacts 单仓盈亏归因锚面（T-0464——机判序输入；锚=确定性读数非判词，
// 读败锚=零值回落 unattributed 不虚构归因）。
type attrFacts struct {
	BugShaped    bool    // 病形码件（bare_bag/trigger_unknown/守卫环——diagnosisWorthy 同源）
	LegErrs      int     // fin_order_leg 错因行（failed/ambiguous/venue_err——拒因异常）
	OpsDisrupt   int     // 窗内 ops_event 断链/断供/凭据/部署停滞/ENOSPC/网络行
	GuardVenueEr bool    // fin_guard_close.last_error 'venue:' 前缀（venue 期败=断供邻域）
	MktTrigger   bool    // 市场生命周期触发（funding_flip*/delta_drift——窗口闭合/对手价动）
	FundingNet   float64 // Σfunding_settle 净额（费率不利/有利锚）
	Settled      float64 // settled_amount（符号对齐判据）
}

// marketTriggers 市场生命周期触发集（窗口闭合/对手价动——max_hold/dd_30d/
// strategy_*/manual 不入：时间止损/风控闸/生命周期中性面不归市场锚）。
var marketTriggers = map[string]bool{
	"funding_flip": true, "funding_flip_preempt": true, "delta_drift": true,
}

// attributePnl 四值机判序（T-0464 钦定序：code→ops→market→unattributed——
// 零锚=unattributed 显式判词非沉默；人裁可改判，机判原值留案卷/decision 行）。
func attributePnl(f attrFacts) string {
	switch {
	case f.BugShaped || f.LegErrs > 0:
		return "code"
	case f.OpsDisrupt > 0 || f.GuardVenueEr:
		return "ops"
	case f.MktTrigger || fundingDominant(f.FundingNet, f.Settled):
		return "market"
	default:
		return "unattributed"
	}
}

// fundingDominant 费率锚（费率净额与结算结果同向且占结果半值以上=费率不利/
// 有利是盈亏主因——反向/微量不锚，防 funding carry 常态流量误吞归因）。
func fundingDominant(fundingNet, settled float64) bool {
	if fundingNet == 0 || settled == 0 || (fundingNet > 0) != (settled > 0) {
		return false
	}
	return math.Abs(fundingNet) >= math.Abs(settled)/2
}

// hits 机判锚命中集（判词留痕面——outcome payload/案卷/日志同源）。
func (f attrFacts) hits() []string {
	var out []string
	if f.BugShaped {
		out = append(out, "bug_shape")
	}
	if f.LegErrs > 0 {
		out = append(out, fmt.Sprintf("leg_errs=%d", f.LegErrs))
	}
	if f.OpsDisrupt > 0 {
		out = append(out, fmt.Sprintf("ops_disrupt=%d", f.OpsDisrupt))
	}
	if f.GuardVenueEr {
		out = append(out, "guard_venue_err")
	}
	if f.MktTrigger {
		out = append(out, "mkt_trigger")
	}
	if fundingDominant(f.FundingNet, f.Settled) {
		out = append(out, fmt.Sprintf("funding_net=%.4f", f.FundingNet))
	}
	return out
}

// diagSeed 种子（族谱命中→草案种子载荷——LLM 开放考古的排班起点）。
type diagSeed struct {
	Family     string   `json:"family"`
	FamilyName string   `json:"family_name"`
	Hypothesis string   `json:"hypothesis"`
	Evidence   []string `json:"evidence"`
	FixSurface string   `json:"fix_surface"`
}

// diagSources 诊断源标签（案卷源集——任务载荷+log 同源）。
func diagSources(f diagCaseFacts) []string {
	var out []string
	if f.BareBagUSD > 0 {
		out = append(out, "账实差未定性（残袋挂账）")
	}
	if f.hasAnomaly("trigger_unknown") {
		out = append(out, "认知集外触发")
	}
	if f.Attempts >= 2 {
		out = append(out, "守卫环（平仓重试循环）")
	}
	// GW-001 段D1：fast_close 复发闸两入径标签（复发实断/秒平亏损——
	// 单次秒平盈利不入闸故无标签）。
	if f.hasAnomaly("fast_close") {
		if f.FastCloseRepeat {
			out = append(out, "秒平复发（7d 窗同 strategy_ref∨inst_id ≥N 次）")
		}
		if f.SettledUSD < 0 {
			out = append(out, "秒平亏损（settled<0）")
		}
	}
	return out
}

// buildDiagTaskNumbers 诊断任务载荷装配（确定性——同案卷同载荷）。
func buildDiagTaskNumbers(f diagCaseFacts, caseNum json.RawMessage) map[string]any {
	var seeds []diagSeed
	for _, fam := range matchDiagFamilies(f) {
		ev := make([]string, 0, len(fam.Evidence))
		for _, q := range fam.Evidence {
			q = strings.ReplaceAll(q, "{pos}", fmt.Sprint(f.PosID))
			q = strings.ReplaceAll(q, "{exec}", fmt.Sprint(f.ExecID))
			q = strings.ReplaceAll(q, "{inst}", f.InstID)
			ev = append(ev, q)
		}
		seeds = append(seeds, diagSeed{
			Family: fam.ID, FamilyName: fam.Name,
			Hypothesis: fam.Hypothesis, Evidence: ev, FixSurface: fam.FixSurface})
	}
	if seeds == nil {
		seeds = []diagSeed{}
	}
	numbers := map[string]any{
		"case":    json.RawMessage(caseNum), // 案卷原貌（postmortem numbers 原样入卷）
		"sources": diagSources(f),
		"seeds":   seeds,
		"target_output": "最可能根因+证据链+建议修法面（草案——假设非判词，禁自批；" +
			"族标=seeds 命中族或 novel:<一句定性>）",
	}
	if len(seeds) == 0 {
		// 族谱外案=纯开放考古（种子缺席不是缺席诊断——案卷+守卫环锚照给）。
		numbers["evidence_queries"] = []string{
			fmt.Sprintf("SELECT state,attempts,last_error,last_attempt_at,trigger_kind "+
				"FROM fin_guard_close WHERE position_id=%d", f.PosID)}
	}
	return numbers
}

// emitDiagTask 诊断任务落账（结案判定机同拍挂接——UNIQUE(kind,entity_ref)
// 幂等一仓一案；写败计数+log 不阻钱径，与监察案写同律）。返回落账与否。
func emitDiagTask(ctx context.Context, r *Runner, f diagCaseFacts, caseNum json.RawMessage) bool {
	numbers, err := json.Marshal(buildDiagTaskNumbers(f, caseNum))
	if err != nil {
		r.rej.Inc("diag_task_err")
		log.Printf("WARN [ledger] 仓 #%d 诊断任务载荷序列化败: %v", f.PosID, err)
		return false
	}
	tag, err := r.pool.Exec(ctx, `INSERT INTO fin_econ_lesson(kind,entity_ref,numbers)
		VALUES('rootcause_diag',$1,$2::jsonb) ON CONFLICT(kind,entity_ref) DO NOTHING`,
		fmt.Sprintf("position#%d", f.PosID), string(numbers))
	if err != nil {
		r.rej.Inc("diag_task_err")
		log.Printf("WARN [ledger] 仓 #%d 诊断任务落账败——根因诊断失显形: %v", f.PosID, err)
		return false
	}
	if tag.RowsAffected() == 0 {
		return false // 已在案（幂等重放）——零产
	}
	log.Printf("[ledger] 仓 #%d 根因诊断案生成（%s）——诊断任务入裁定席 pending",
		f.PosID, strings.Join(diagSources(f), "×"))
	return true
}
