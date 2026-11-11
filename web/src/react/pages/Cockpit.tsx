// Cockpit.tsx —— 要我办啥（2026-11-11 激进极简版：页问收窄到「要我动什么」——
// 待开仓批件+工作流管道+行情提醒三段线性；原 KPI 条全撤：NAV/MTD/30d 归
// 机赚如何/钱怎么走的 正源，在岗席位归 哪病了，敞口越限由 cap_hit 升件
// 覆盖——单源律：事实一处渲,他页指路不复读。判态全服务端派生,前端零判据）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { onLive } from "../stream";
import { applyPipeline, hydrate } from "../store";
import { Component, type ReactNode } from "react";
import { EmptyState, Skeleton } from "../components/ui";
import { PipelineTile } from "../components/pipeline";
import { QuoteBoard } from "../components/QuoteBoard";
import { DIRECTION_CN, cn } from "../components/labels";
import type { GetCockpitResponse, CockpitArmed } from "../../gen/dashboardv1/core_pb";

const ts = (t?: { seconds: bigint }) =>
	t ? new Date(Number(t.seconds) * 1000).toLocaleString("zh-CN", { hour12: false }) : "—";

export function CockpitPage() {
	const [d, setD] = useState<GetCockpitResponse | null>(null);
	// 取数泵：pipeline 分片喂 PipelineTile（hydrate 首取+onLive 复连回补，
	// 帧驱动零定时器）；armed/asOf 走 getCockpit unary。
	useEffect(() => {
		hydrate("pipeline", () => client.getPipeline({}));
		let replayed = false;
		return onLive((v?: unknown) => {
			if (!replayed) { replayed = true; return; }
			if (v === true) client.getPipeline({}).then(applyPipeline).catch(() => {});
		});
	}, []);
	useEffect(() => {
		client.getCockpit({})
			.then((d) => {
				if (d.pipeline) applyPipeline(d.pipeline);
				setD(d);
			})
			.catch(() => setD(null));
	}, []);
	if (!d) return (
		<div className="page v2">
			<section className="v2-hero">
				<div className="v2-eyebrow">
					<span className="v2-livedot" />
					要我办啥 · LIVE
				</div>
				<h1 className="v2-title">—</h1>
			</section>
			<div className="v2-panel"><div className="v2-panel-head"><h2 className="v2-panel-title">待开仓</h2></div><Skeleton rows={3} /></div>
			<PipelineTile />
			<QuoteBoard />
		</div>
	);
	const miss = new Set(d.missing);
	const sec = (name: string, el: ReactNode) =>
		miss.has(name) ? <p className="dim">该段数据缺（查询败——不冒零）</p> :
			<SecBoundary>{el}</SecBoundary>;
	const armedMiss = miss.has("armed");
	return (
		<div className="page v2">
			<section className="v2-hero">
				<div className="v2-eyebrow">
					<span className="v2-livedot" />
					要我办啥 · LIVE
				</div>
				<h1 className="v2-title">
					{armedMiss ? "读件缺" : `${d.armed.length} 件待开仓`}
				</h1>
				<p className="v2-lede">更新 {ts(d.asOf)} · 钱面见 <a href="/">机赚如何</a> · 席位见 <a href="/device">哪病了</a></p>
			</section>

			{/* 待开仓——要我批的单（页问正主,首段） */}
			<div className="v2-panel">
				<div className="v2-panel-head">
					<h2 className="v2-panel-title">待开仓</h2>
					<span className="ink-3">{armedMiss ? "—" : `${d.armed.length} 单`}</span>
				</div>
				{sec("armed",
					d.armed.length === 0 ? <EmptyState title="无待开仓" hint="待开仓队列为空——当前没有待处理事项" /> :
					<div className="tile-scroll"><table className="tile-table"><thead><tr><th>#</th><th>标的</th><th>方向</th><th>建时</th><th>TTL余(h)</th></tr></thead>
						<tbody>{d.armed.map((a: CockpitArmed) => <tr key={a.id}><td>{a.id}</td><td>{a.instId}</td>
							<td title={a.direction}>{cn(DIRECTION_CN, a.direction, "对冲方向未识别")}</td><td>{ts(a.created)}</td><td>{a.ttlHoursLeft.toFixed(1)}</td></tr>)}</tbody></table></div>)}
			</div>

			{/* 工作流管道——机器在干什么/卡哪（1 票 1 行,流推 ≤10s） */}
			<PipelineTile />

			{/* 行情·价格提醒——观测面（单向告警非条件单） */}
			<QuoteBoard />
		</div>
	);
}

// SecBoundary 块级 error boundary（T-0248——一块抛错不掀全页，显缺注占位）。
class SecBoundary extends Component<{ children: ReactNode }, { err: boolean }> {
	state = { err: false };
	static getDerivedStateFromError() { return { err: true }; }
	render() {
		return this.state.err ? <p className="dim">该段数据缺（渲染败）</p> : this.props.children;
	}
}
