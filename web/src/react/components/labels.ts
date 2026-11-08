// labels.ts —— T-0441 机读态→人读词映射层（业主令「表达尽量准确无黑话」
// 全谱审计落点：全部枚举/态标/字段名词表归一此件——组件禁自带词表，
// 未知枚举值一律落人话兜底不裸透（映射层内部引用除外——验收⑤口径）。
// 词表以票面词为准：armed→候选就绪/待触发、proposed→待开仓、
// adopted→已采纳 等既有中文词沿用不另造。

// cn 词表查询助手：命中=中文词；未命中=人话兜底（未知枚举不裸出机器词，
// 原值留 title 悬停由调用方自决——兜底词保证可见面零黑话）。
export function cn(map: Record<string, string>, v: string | undefined | null, fallback = "未识别"): string {
	if (!v) return "—";
	return map[v] ?? fallback;
}

// —— 提案/策略生命周期方向（fin_proposal.direction 闭集四值）——
export const DIRECTION_CN: Record<string, string> = {
	long_perp_short_spot: "永续多·现货空",
	short_perp_long_spot: "永续空·现货多",
	long_perp_short_cfd: "永续多·CFD空",
	short_perp_long_cfd: "永续空·CFD多",
};

// —— 驾驶舱平台分列（CockpitPlatformRow.platform）——
export const PLATFORM_CN: Record<string, string> = {
	crypto: "OKX 加密",
	MT5: "MT5",
	future: "期货",
};

// —— CMC 平台预警态（HedgeStructure.platform_alert_state）——
export const PLATFORM_ALERT_CN: Record<string, string> = {
	normal: "正常",
	warn: "预警",
	breach: "强平触碰",
};

// —— 管道步/行结果与当前态（CockpitPipelineStep.result /
// CockpitPipelineRow.current_state——后者含 waiting_audit 未在 STEP_RESULT）——
export const STEP_RESULT_CN: Record<string, string> = {
	working: "在干",
	waiting_audit: "等审计",
	closed: "完成移交",
	passed: "通过",
	rejected: "驳回",
	blocked: "挂起",
	error: "异常",
};
export const STEP_RESULT_FALLBACK = "处理中";

// —— 无头行五态（HeadlessRow.state——服务端已发中文标时直通；
// 英文原键兼容映射防旧帧漏词）——
export const HEADLESS_STATE_CN: Record<string, string> = {
	alive: "在飞",
	done: "完工",
	dead: "死亡",
	stalled: "停滞",
	overdue: "逾期",
	exit: "已退",
	在飞: "在飞",
	完工: "完工",
	死亡: "死亡",
	停滞: "停滞",
	逾期: "逾期",
};

// —— 复盘报告指标键（ReportMetric.k——未知键不裸出字段名）——
export const METRIC_CN: Record<string, string> = {
	net: "净额",
	cum_nav: "累计实现盈亏",
	funding_income: "资金费收入",
	earn_balance: "Earn 停泊",
	opened: "开仓",
	closed: "平仓",
	proposed: "评",
	rejected: "拒",
	active: "在册策略",
	alerts: "风控拒",
	capital_flows: "资金流",
	annualized: "年化",
	capital_base: "基数",
	retired: "退役",
	net_vs_earn: "净利−Earn基线",
	earn_baseline_snaps: "基线快照数",
};
export const METRIC_FALLBACK = "未识别指标";

// —— 现金流类（CashflowRow.kind / HedgeFlowItem.kind 同族——
// 未知类落「其他」不裸出枚举）——
export const CASHFLOW_KIND_CN: Record<string, string> = {
	interest: "活期利息",
	funding_settle: "资金费",
	trade_pnl: "平仓盈亏",
	fee: "交易成本",
	deposit: "入金",
	withdraw: "出金",
	spot_buy: "现货买入",
	spot_sell: "现货卖出",
	recon_adjust: "校账调整",
	rollback_cost: "回滚成本",
	earn_in: "Earn 转入",
	earn_out: "Earn 转出",
	// HedgeFlowItem.kind 族（trx 账本流水）
	rental_income: "租金收入",
	vote_claim: "奖励提取",
	stake: "质押",
	unstake: "解押",
	overnight: "隔夜息",
	permission_update: "权限变更",
	unstake_cancel: "取消解冻",
	vote: "投票",
	delegate: "委托资源",
	undelegate: "收回委托",
	close_trade: "平仓",
	sell_order: "卖出委托",
	order_cancelled: "撤单",
};
export const KIND_FALLBACK = "其他";

// —— 现金流平台（CashflowRow.venue / HedgeFlowItem.source）——
export const VENUE_CN: Record<string, string> = { okx: "OKX", mt5: "MT5", trx: "TRX", cmc: "CMC" };

// —— 侦察榜（scoutv1.Board 行：家族/态/基线源）——
export const SCOUT_FAMILY_CN: Record<string, string> = {
	carry: "单所息差",
	xvenue_spread: "跨所价差",
	basis: "基差",
};
export const SCOUT_STATE_CN: Record<string, string> = {
	measuring: "度量中",
	candidate: "候选",
	recommended: "建议注册",
	registered: "已注册",
	rejected: "已拒",
	kyc_blocked: "KYC受限",
	adopted: "已采纳",
	dismissed: "已驳回",
	parked: "已停放",
};
export const SCOUT_BASE_SRC_CN: Record<string, string> = { earn: "Earn", okx_family: "OKX同族" };

// —— broker 接入态（BrokerOnboardRow.phase/state）——
export const BROKER_PHASE_CN: Record<string, string> = {
	registered: "已注册·候验连",
	connecting: "验连中",
	verified: "已验证",
	failed: "接入败",
};
export const BROKER_STATE_CN: Record<string, string> = {
	registered: "已注册·候探测",
	probed: "已探测",
	mapped: "映射就绪",
	paper: "paper 激活",
	candidate: "候选",
	measuring: "度量中",
	recommended: "建议注册",
	rejected: "已拒",
	kyc_blocked: "KYC受限",
	adopted: "已采纳",
	dismissed: "已驳回",
	parked: "已停放",
};

// —— 提醒行（PriceAlertRow.state/direction）——
export const ALERT_STATE_CN: Record<string, string> = {
	active: "盯梢中",
	fired: "已提醒",
	disabled: "已停用",
};
export const ALERT_DIR_CN: Record<string, string> = { ">=": "涨到 ≥", "<=": "跌破 ≤" };

// —— 工件行（WorkItem.kind/stage——stage 中文标服务端派生，
// 「执行中」在提案域=fin_armed 武装候选非真执行（T-0441 病根①——
// 后端根因 cmd/core/flow_pulse.go 在清单外落账，本层按真态词表校正））——
export const WORK_KIND_CN: Record<string, string> = {
	proposal: "提案",
	position: "持仓",
	venue: "侦察",
};
export const WORK_STAGE_CN: Record<string, string> = {
	"待开仓": "待开仓",
	"执行中": "候选就绪·待触发",
	"持仓中": "持仓中",
	"开仓挂起": "开仓挂起",
	"残仓": "残仓",
	"读件缺": "读件缺",
	"度量中": "度量中",
};

// —— 采集组件名（SystemComponent.component——13 件健康册闭集，
// 组件实名带信号 kind 机读尾缀，显示面剥词表人话）——
export const COMPONENT_CN: Record<string, string> = {
	"okxws WS 行情": "OKX 行情流",
	"funding WS": "资金费率流",
	"买五档 depth5": "买五档行情",
	"蒸馏 basis_ann": "蒸馏·基差信号",
	"蒸馏 funding_view": "蒸馏·资金费视图",
	"蒸馏 xvenue_spread": "蒸馏·跨所价差",
	"蒸馏 carry_diff": "蒸馏·利差指标",
	"蒸馏 mt5_swap": "蒸馏·MT5互换",
	"MT5 桥行情": "MT5 行情",
	"MT5 利润流": "MT5 盈亏流水",
	"OKX 私有 WS": "OKX 账户私有流",
	"earn 现率采样": "Earn 利率采样",
	"行情 ticks(OKX)": "OKX 行情流水",
	"okxws 采集臂": "OKX 采集臂",
};

// —— okxws 拒因键（GetSystemHealth.rejects——snake_case 机读键人话化；
// 未识键归并「其他拒因」不逐键裸出）——
export const REJECT_CN: Record<string, string> = {
	non_usdt_instid: "非USDT合约",
	seq_gap: "序号跳空",
	parse_fail: "解析失败",
	write_err: "写入失败",
	symbols_missing: "合约表缺",
	activeset_read_failed: "活跃集读败",
	depth5_no_ctval: "档五缺面值",
	scan_empty: "扫描空返",
	scan_fetch_err: "扫描拉取败",
	scan_write_err: "扫描写败",
};
