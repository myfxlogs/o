// pipeStatus.ts —— 段态标人话映射（T-0366：cockpit 可见面=人话面，机器原文
// 禁入前端任何层——映射表静态钉死随代码走，不接 LLM；net_apr/阈 等原文只留
// DB/日志面）。输入=服务端态标串，输出=可见面短句+悬停详述（可读句+数值）。
// 未识词一律落人话兜底——宁失细节不露机器词。

interface PlainReason {
	short: string; // 可见面人话短句
	detail: string; // 悬停人话详述（短句扩成可读原因）
}

// rejectPlain 拒因短标→人话（键=归一短标特征子串，序=特异度先判）。
const rejectPlain: [string, PlainReason][] = [
	["net_apr", { short: "无达标机会", detail: "测算净年化低于放行线" }],
	["below_earn", { short: "不及保底收益", detail: "测算净年化低于 Earn 保底收益" }],
	["basis_below_min", { short: "基差未过线", detail: "基差低于最小开仓线" }],
	["apr_est_missing", { short: "缺数据", detail: "年化数据缺失无法评估" }],
	["missing", { short: "缺数据", detail: "评估所需数据缺失" }],
	["stale", { short: "数据陈旧", detail: "评估所用数据过旧" }],
	["liquidity", { short: "深度不足", detail: "市场深度不够" }],
	["unprofitable", { short: "算不过账", detail: "扣完成本不盈利" }],
];

// machineLeft 残留机读词哨兵（人话输出过此闸——命中即整句兜底，宁可笼统不
// 露原文）：snake_case 词/armed 等机器态词/阈 字。
const machineLeft = /[a-z][a-z0-9]*_[a-z0-9_]+|\barmed\b|阈/i;

// reasonToPlain 归一拒因短标→人话；未识/截断残词→泛化兜底（不露原文）。
function reasonToPlain(short: string): PlainReason {
	for (const [key, p] of rejectPlain) {
		if (short.includes(key)) return p;
	}
	return { short: "未过放行线", detail: "评估判据未放行" };
}

export interface PlainStatus {
	short: string; // 可见面（态标位）人话短句
	detail: string; // 悬停人话详述（可读句+数值）
	idle: boolean; // 无在途事项：大格只显「闲」，详述仍保留在悬停
}

export interface FlowStageCopy {
	what: string;
	cadence: string;
	unit: string;
	eventDriven: boolean;
}

// 活水段语义由票面权威表钉定，避免依赖服务端机器 key/秒数翻译。
const flowStageCopies: Record<string, FlowStageCopy> = {
	collect: { what: "OKX 资金费率行情采集臂", cadence: "约每 5 分钟一次", unit: "条行情数据", eventDriven: false },
	bridge: { what: "行情投影入账桥（开发环境到生产镜像）", cadence: "约每 10 分钟一次", unit: "帧", eventDriven: false },
	eval: { what: "策略机会评估臂（净年化过放行线才出提案）", cadence: "约每 30 分钟一次", unit: "次评估", eventDriven: false },
	proposal: { what: "已放行提案（待开仓或候选就绪待触发）", cadence: "", unit: "个提案", eventDriven: true },
	strategy: { what: "策略沙箱开户与晋升（纸面户开平、判据晋升、退役）", cadence: "", unit: "次状态迁移", eventDriven: true },
	exec: { what: "真金白银流水（成交、利息、费用）", cadence: "", unit: "笔流水", eventDriven: true },
	report: { what: "每日结算报表", cadence: "约每天一次", unit: "份报表", eventDriven: false },
	scout: { what: "全球交易所进水口侦察度量", cadence: "约每天一次", unit: "个交易所", eventDriven: false },
};

export function flowStageCopy(stageKey: string): FlowStageCopy {
	return flowStageCopies[stageKey] ?? { what: "运行数据处理", cadence: "", unit: "条记录", eventDriven: false };
}

// humanStatus 态标串→人话二态。evalCount=评估段 24h 行数（提案段详述带
// 「评估 N 个机会」实测数——数据源现字段零新增）。未知形态原文照传风险面
// 经 machineLeft 哨兵二次兜底。
export function humanStatus(stageKey: string, status: string, evalCount: number): PlainStatus {
	let s = status.trim();
	if (s === "") return { short: "", detail: "", idle: false };

	// ①拒因尾标「（末因 <短标>）」剥离先翻——短标为机器归一词必翻不照传。
	let reason: PlainReason | undefined;
	const m = s.match(/^(.*)（末因 ([^）]+)）$/);
	if (m) {
		s = m[1];
		reason = reasonToPlain(m[2]);
	}

	// ②机读态词归一：armed→在飞提案（提案段=待开仓/执行中；执行段=真执行类流水）。
	s = s.replace(/活跃·(\d+) armed/, "活跃·$1 个在飞提案");
	if (/无 armed/.test(s)) {
		s = stageKey === "exec" ? "闲·无真执行流水" : "闲·无在飞提案";
	}
	// 「（阈未达）」类残标人话化。
	s = s.replace(/（阈未达）/, "（未过放行线）");

	// ③详述组装：拒因件把判据人话因+评估实数带进可读句。
	let detail = "";
	if (reason) {
		detail = evalCount > 0
			? `近 24 小时评估 ${evalCount} 个机会全部未过放行线；主因：${reason.detail}`
			: `评估全拒；主因：${reason.detail}`;
	} else if (/^活跃·(\d+) 个在飞提案/.test(s)) {
		const n = s.match(/^活跃·(\d+)/)?.[1];
		detail = `${n} 个提案已放行，待开仓或候选就绪待触发`;
	} else if (s === "闲·无真执行流水") {
		detail = "近 24 小时无成交/持仓类真金白银流水";
	} else if (s === "闲·无在飞提案") {
		detail = "当前无已放行提案在飞";
	} else {
		detail = s;
	}

	// ④哨兵：可见/详述仍含机读词→人话兜底（漏词不外泄）。
	if (machineLeft.test(s)) s = "评估未放行";
	const short = reason ? `${s}（${reason.short}）` : s;
	if (machineLeft.test(detail)) detail = "评估判据未放行";
	return { short, detail, idle: /^闲(?:·|$)/.test(s) };
}
