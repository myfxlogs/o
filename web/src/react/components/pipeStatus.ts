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
}

// humanStatus 态标串→人话二态。evalCount=评估段 24h 行数（提案段详述带
// 「评估 N 个机会」实测数——数据源现字段零新增）。未知形态原文照传风险面
// 经 machineLeft 哨兵二次兜底。
export function humanStatus(stageKey: string, status: string, evalCount: number): PlainStatus {
	let s = status.trim();
	if (s === "") return { short: "", detail: "" };

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
		detail = `${n} 个提案已放行，待开仓或执行中`;
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
	return { short, detail };
}
