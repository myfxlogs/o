// flowpulse.tsx —— 活水段段链（T-0307：运行时机器流——票=组织流语义不混；
// 灯色服务端 StageLamp 派生；行数闸拆分件）。
import { Fragment } from "react";
import { client } from "../api";
import { useLive } from "../hooks";
import type { FlowStage } from "../../gen/dashboardv1/core_pb";
import { EmptyState, StateDot } from "./ui";
import { flowStageCopy, humanStatus } from "./pipeStatus";

// 活水段（T-0307：运行时机器流段链——票=组织流语义不混，
// 段=机器流；灯色服务端 StageLamp 派生——阈按各臂节拍常量钉）。
// T-0362 流动骨架：判词行（state 序列机械派生）+三段式卡+脊线箭头+人话滞后。
export function FlowPulseStrip() {
	const fp = useLive(["ev_funding", "ev_eval"], () => client.getFlowPulse({}));
	if (!fp.data) return fp.err ? <p className="dim">活水段数据缺（读败）</p> : null;
	return <FlowPulseView stages={fp.data.stages} rosterCount={fp.data.rosterCount} rosterWorstLagS={fp.data.rosterWorstLagS} />;
}

// FlowPulseView 纯渲染件（hook 外置——DOM 断言可直注合成态）。
export function FlowPulseView(props: { stages: FlowStage[]; rosterCount: number; rosterWorstLagS: bigint }) {
	const ago = (s: number) => (s < 60 ? "刚刚" : s < 3600 ? `${Math.floor(s / 60)}分钟前` : s < 86400 ? `${Math.floor(s / 3600)}小时前` : `${Math.floor(s / 86400)}天前`);
	const stages = props.stages;
	// 判词机械派生：首个黄/红段=断点（灰中性段跳过）；全绿灰而 eval 段带
	// 态标=饿殍期（阈未达类）；余=活水正常。不造神谕。
	const bad = stages.findIndex((st) => st.state === 1 || st.state === 2);
	// T-0366 人话面：态标一律过 humanStatus——机器原文（net_apr/阈/armed 等）
	// 禁入前端任何层；可见面=人话短句，悬停=人话详述带数值（评估实数同源）。
	const evalCount = Number(stages.find((st) => st.key === "eval")?.count24h ?? 0);
	const plain = (st: FlowStage) => humanStatus(st.key, st.status, evalCount);
	const evalStatus = bad < 0 ? stages.find((st) => st.key === "eval" && st.status) : undefined;
	const verdict = bad >= 0
		? `断在 ${bad > 0 ? stages[bad - 1].label + "→" : ""}${stages[bad].label}`
		: evalStatus
			? `饿殍期·${plain(evalStatus).short}`
			: "活水正常";
	const vcls = bad >= 0 ? (stages[bad].state === 2 ? "s2" : "s1") : "s0";
	const mid = (st: FlowStage) =>
		!st.ok ? "缺" : plain(st).idle ? "闲" : Number(st.count24h) > 0 ? String(st.count24h) : plain(st).short || "—";
	const lagText = (st: FlowStage) =>
		!st.ok ? "读取失败" : !st.lastTs ? "暂无数据" : ago(Number(st.lagS));
	const cardTitle = (st: FlowStage) => {
		const copy = flowStageCopy(st.key);
		const cadence = copy.eventDriven ? "事件驱动，无固定节奏" : copy.cadence;
		const recent = copy.eventDriven ? "" : st.ok
			? `，最近一次写入${st.lastTs ? ago(Number(st.lagS)) : "暂无数据"}`
			: "，最近一次写入读取失败，该段显示为缺";
		const count = st.ok
			? `近 24 小时 ${st.count24h} ${copy.unit}`
			: "近 24 小时数据读取失败";
		const detail = plain(st).detail;
		return `${st.label}——${copy.what}；${cadence}${recent}；${count}${detail ? `；${detail}` : ""}`;
	};
	return (
		<div className="pulse-strip">
			<div className={`pipe-verdict ${vcls}`}>{verdict}</div>
			<div className="pipe-flow">
				{stages.map((st, i) => {
					const p = plain(st);
					return (
						<Fragment key={st.key}>
							{i > 0 && (
								<span className="pipe-link" aria-hidden>
									<span className="pipe-line" />▸
								</span>
							)}
							<div className="pipe-card" title={cardTitle(st)}>
								<div className="pipe-card-top">
									<StateDot state={st.state} breathe={st.state === 0}
										title={`${st.label}：${!st.ok ? "读取失败——该段显示为缺" : !st.lastTs ? "暂无数据" : `最近一次写入${ago(Number(st.lagS))}`}`} />
									{st.label}
								</div>
								<div className={`pipe-card-num${p.idle ? " pipe-card-num-hungry" : ""}`}>{mid(st)}</div>
								<div className="pipe-card-lag">{lagText(st)}</div>
							</div>
						</Fragment>
					);
				})}
			</div>
			<div className="ink-3" style={{ marginTop: 6 }}>
				{props.rosterCount} 岗在岗 · 最坏滞后 {ago(Number(props.rosterWorstLagS))}
			</div>
		</div>
	);
}


// WorkItemsBlock 钱径工件行（T-0309——③卡下半段：工件从「被看见」走向
// 「变成利润」的在飞集合；行=ref+stage 徽标+在飞时长；服务端排序 proposal→
// position→venue cap10；空态「无在飞工件」）。
export function WorkItemsBlock() {
	const fp = useLive(["ev_funding", "ev_eval", "ev_proposal", "ev_position"], () => client.getFlowPulse({}));
	if (!fp.data) return fp.err ? <p className="dim">工件行数据缺（读败）</p> : null;
	const items = fp.data.workItems ?? [];
	if (items.length === 0) return <EmptyState title="无在飞工件" hint="钱径工件（提案、持仓、交易所度量）为零" />;
	const ago = (t?: { seconds: bigint }) => {
		if (!t) return "—";
		const s = Math.floor((Date.now() - Number(t.seconds) * 1000) / 1000);
		return s >= 3600 ? `${(s / 3600).toFixed(1)}h` : s >= 60 ? `${(s / 60).toFixed(0)}m` : `${s}s`;
	};
	const KIND: Record<string, string> = { proposal: "提案", position: "持仓", venue: "侦察" };
	return (
		<div className="tile-scroll">
			{items.map((w) => (
				<div className="pipe-row" key={w.kind + ":" + w.ref}>
					<span className="pipe-ticket">{KIND[w.kind] ?? w.kind}</span>
					<span className="pipe-title">{w.ref}</span>
					<span className="pipe-seat">{w.stage}</span>
					{w.progress && <span className="pipe-seat">{w.progress}</span>}
					<span className="pipe-elapsed">{ago(w.sinceTs)}</span>
				</div>
			))}
		</div>
	);
}
